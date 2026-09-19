package engine

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"git-hint/internal/core"
	"git-hint/internal/engine/provider"
	"git-hint/internal/engine/provider/common"
	"git-hint/internal/engine/ranking"
	"git-hint/internal/engine/tokenizer"
	"git-hint/internal/registry"
	"git-hint/internal/state"
)

var ErrNotIndexed = errors.New("command known but not yet indexed")

// applySpecialRules rewrites commands whose suggestions should not come from
// the generic spec tree. Today: `source` (and `.`), which always complete a
// FILE PATH. The <arg>/<file> placeholders of its Fig spec are remapped to
// "file-path": the provider merges the user's sourced-file history (most
// recent first, so history is the first option) with the real filesystem as
// the user keeps typing a path.
func applySpecialRules(commandName string) {
	if commandName == "source" || commandName == "." {
		setState(commandName, "file-path")
	}
}

// setState rewrites the top-level placeholder entries of a special command's
// cached spec in place: every <placeholder> key is renamed to the provider
// key (<file-path>) and gets the friendly description shown in the group
// header. Idempotent: renames are collected before mutating the map so the
// renamed key is never revisited and deleted by its own pass.
func setState(commandName, flag string) {
	tree, err := resolveCommandTree(commandName)
	if err != nil || tree == nil {
		return
	}
	target := "<" + flag + ">"

	var renames []string
	for name := range tree {
		if provider.FlagCheck(name) != "" && name != target {
			renames = append(renames, name)
		}
	}
	for _, name := range renames {
		cmd := tree[name]
		cmd.Name = target
		cmd.Description = "arquivo para source"
		tree[target] = cmd
		delete(tree, name)
	}
}

// Suggestions returns the list of suggestions for the current input buffer,
// along with the current token being typed (used for cursor replacement)
// and any error encountered along the way.
func Suggestions(input string) ([]core.CommandMatch, string, error) {
	parts := tokenizer.TokenizeBuffer(input)
	if len(parts) == 0 {
		return nil, "", nil
	}

	commandName := parts[0]
	remainingInput := parts[1:]

	// Level 0: user is still typing the command name itself (e.g. "gi").
	if len(remainingInput) == 0 {
		list, currentToken, err := registry.SuggestFromIndex(commandName)
		if err != nil {
			return nil, "", err
		}
		if len(list) == 0 {
			return nil, currentToken, nil
		}
		list, err = ranking.RankSuggestions(commandName, list)
		if err != nil {
			return nil, "", fmt.Errorf("❌ Failed to rank commands: %v", err)
		}
		return list, currentToken, nil
	}

	// `source <Space>` (and `.`): remap the spec's file placeholders to the
	// file-path provider before the tree is used. Unknown/unindexed commands
	// resolve to a nil tree inside setState and are left untouched.
	applySpecialRules(commandName)

	// Level 1+: resolve the full subcommand tree for the given command.
	commands, err := resolveCommandTree(commandName)
	if err != nil {
		return nil, "", err
	}
	if commands == nil {
		return nil, "", nil
	}

	matches, currentToken, err := FindCommands(remainingInput, commands)
	if err != nil {
		return nil, "", fmt.Errorf("❌ Error finding commands: %v", err)
	}
	if matches == nil {
		return nil, "", nil
	}

	list, dynamicFlagSeen := buildSuggestionList(matches, currentToken)
	if len(list) == 0 {
		return nil, currentToken, nil
	}

	list, err = rankAndGroup(input, list, dynamicFlagSeen)
	if err != nil {
		return nil, "", err
	}

	// Open message quote: a bare `"` closes the message. Offered as the very
	// last row (only when the quote is open) so it completes the buffer
	// without competing with real suggestions.
	if OpenQuoteToken(input) {
		list = append(list, core.CommandMatch{
			Name:        "\"",
			Description: "fechar mensagem",
		})
	}

	return list, currentToken, nil
}

// resolveCommandTree loads and parses the subcommand tree for commandName,
// checking along the way whether it's known and already indexed.
func resolveCommandTree(commandName string) (map[string]core.CommandMatch, error) {
	return resolveCommandTreeVisited(commandName, map[string]bool{})
}

// resolveCommandTreeVisited resolves through alias definitions (an alias
// completes as its underlying command: typing "gst " continues with
// "git status" flags) while guarding against self-referencing aliases.
func resolveCommandTreeVisited(commandName string, visited map[string]bool) (map[string]core.CommandMatch, error) {
	if visited[commandName] {
		return nil, nil
	}
	visited[commandName] = true

	if definition, ok := state.LookupAlias(commandName); ok {
		fields := strings.Fields(definition)
		if len(fields) > 0 && fields[0] != commandName {
			return resolveCommandTreeVisited(fields[0], visited)
		}
	}
	status, err := registry.Status(commandName)
	if err != nil {
		return nil, fmt.Errorf("❌ Error checking command '%s': %v", commandName, err)
	}
	if !status.Known {
		return nil, nil
	}
	if !status.Indexed {
		return nil, ErrNotIndexed
	}

	data, err := registry.ResolveCommandData(commandName)
	if err != nil {
		return nil, fmt.Errorf("❌ Error finding command '%s': %v", commandName, err)
	}
	if data == nil {
		return nil, nil
	}

	commands, err := registry.ParseCommand(commandName, data)
	if err != nil {
		return nil, fmt.Errorf("❌ Error reading file: %v", err)
	}
	return commands, nil
}

// buildSuggestionList turns raw matches into the final flat suggestion list,
// expanding dynamic flags (branch, remote, commit, msg...) into their
// provider-generated values. It also returns the last dynamic flag seen,
// used by the caller to decide whether ranking should be skipped.
func buildSuggestionList(matches map[string]core.CommandMatch, currentToken string) ([]core.CommandMatch, string) {
	var list []core.CommandMatch
	var dynamicFlagSeen string

	for name, cmd := range matches {
		flag := provider.FlagCheck(name)
		if flag == "" {
			// Static command: Description already comes from cmd itself, always displayed.
			list = append(list, cmd)
			continue
		}

		dynamicFlagSeen = flag
		list = append(list, expandDynamicFlag(name, flag, cmd, currentToken)...)
	}

	return list, dynamicFlagSeen
}

// expandDynamicFlag builds the group header item plus every matching value
// returned by the flag's provider (e.g. all local branches for "branch").
func expandDynamicFlag(name, flag string, cmd core.CommandMatch, currentToken string) []core.CommandMatch {
	providerFn, hasProvider := provider.LookupProvider(flag)
	if !hasProvider {
		// Placeholder with no provider (e.g. <arg>, <alias>): no suggestions.
		// Emitting it as an item made TAB insert the literal "<arg>" into the
		// buffer; an empty list falls through to the shell's own completion.
		return nil
	}

	groupPlaceholder := name
	if cmd.Description != "" {
		groupPlaceholder = name + "  " + cmd.Description
	}

	// Group header: not selectable as a static command, just a section label.
	header := cmd
	header.Placeholder = groupPlaceholder
	header.Name = ""

	out := []core.CommandMatch{header}

	quoteInsensitive := provider.IsQuoteInsensitive(flag)
	for _, s := range providerFn() {
		if s.Name == "" || !tokenMatches(s.Name, currentToken, quoteInsensitive) {
			continue
		}
		s.Placeholder = groupPlaceholder
		applyDescriptionRule(&s, flag, cmd.Description)
		out = append(out, s)
	}

	return out
}

// tokenMatches reports whether a suggestion name matches the token being
// typed. For quote-insensitive groups (messages, names, urls) the match
// ignores surrounding quotes in the suggestion: typing `git commit -m g`
// matches the suggestion `"fix git bug"`, and completing replaces the token
// with the quoted form — the user never types the quotes by hand.
func tokenMatches(name, token string, quoteInsensitive bool) bool {
	if strings.HasPrefix(name, token) {
		return true
	}
	if quoteInsensitive {
		bare := strings.Trim(name, "\"'")
		return strings.HasPrefix(bare, token)
	}
	return false
}

// applyDescriptionRule decides how a dynamically-provided suggestion's
// Description should be displayed, based on which flag generated it.
func applyDescriptionRule(s *core.CommandMatch, flag, parentDescription string) {
	switch {
	case provider.HasNoDescription(flag):
		// msg, message: no comment needed, the value itself is the info.
		s.Description = ""

	case provider.ShouldSkipRanking(flag):
		// commit: keep the specific Description the provider already built
		// (hash + subject), always displayed — does not inherit from parent,
		// not conditioned to "only when selected".

	default:
		// branch, remote, tag, stash: shared description from parent,
		// displayed only when the item is selected.
		s.Description = parentDescription
		s.ShowOnlyWhenSelected = true
	}
}

// rankAndGroup ranks the suggestion list by relevance to the input (unless
// the last dynamic flag seen opts out of ranking) and then groups items by
// Placeholder, keeping ranking order stable within each group.
func rankAndGroup(input string, list []core.CommandMatch, dynamicFlagSeen string) ([]core.CommandMatch, error) {
	if !provider.ShouldSkipRanking(dynamicFlagSeen) {
		ranked, err := ranking.RankSuggestions(input, list)
		if err != nil {
			return nil, fmt.Errorf("❌ Failed to rank commands: %v", err)
		}
		list = ranked
	}

	sort.SliceStable(list, func(i, j int) bool {
		iPlaceholder := list[i].Placeholder != ""
		jPlaceholder := list[j].Placeholder != ""
		if iPlaceholder != jPlaceholder {
			return iPlaceholder
		}
		return list[i].Placeholder < list[j].Placeholder
	})
	return limitGroups(list), nil
}

func limitGroups(list []core.CommandMatch) []core.CommandMatch {
	group, limit := state.GetExpansion()
	if group != "" {
		return expandedGroup(list, group)
	}
	if limit < 10 {
		limit = 10
	}

	// Born-expanded: when the ENTIRE list is a single placeholder group
	// (e.g. `cd` or `source ` where every suggestion comes from one
	// provider), the group screen is pointless — show the items directly,
	// no header row, no "sair". With more than one group (or static items
	// mixed in) keep the normal header + selection flow.
	onlyGroup := ""
	multiGroup := false
	hasStatic := false
	groupItemCount := 0
	for _, item := range list {
		if item.Placeholder == "" {
			hasStatic = true
			continue
		}
		if onlyGroup == "" {
			onlyGroup = item.Placeholder
		} else if item.Placeholder != onlyGroup {
			multiGroup = true
		}
		if item.Name != "" {
			groupItemCount++
		}
	}
	if onlyGroup != "" && !multiGroup && !hasStatic && groupItemCount > 0 {
		out := make([]core.CommandMatch, 0, groupItemCount+1)
		// Header stays visible as a label at the TOP, but NormalizeSelected
		// keeps it unselectable — the cursor lands on the first real item.
		var header *core.CommandMatch
		for _, item := range list {
			if item.Name == "" && header == nil {
				h := item
				header = &h
				continue
			}
			if item.Name != "" {
				out = append(out, item)
			}
		}
		if len(out) > limit {
			out = out[:limit]
		}
		if header != nil {
			out = append([]core.CommandMatch{*header}, out...)
		}
		return out
	}

	// Buffer consecutive group members so each placeholder group can be
	// emitted as a whole: header exactly once, then its items. Singleton
	// groups drop the header entirely — one selectable row between the user
	// and the value is pure noise (e.g. `cd` finding a single directory).
	// Groups with a header but zero items still emit the header: it is the
	// affordance that tells the user the placeholder exists (e.g. <msg>
	// without history suggestions).
	type bufferedGroup struct {
		header *core.CommandMatch
		items  []core.CommandMatch
	}
	out := make([]core.CommandMatch, 0, len(list))
	var order []string
	groups := map[string]*bufferedGroup{}

	flush := func() {
		for _, p := range order {
			g := groups[p]
			if len(g.items) == 1 {
				out = append(out, g.items[0])
				continue
			}
			if g.header != nil {
				out = append(out, *g.header)
			}
			for i, it := range g.items {
				if i < limit {
					out = append(out, it)
				}
			}
		}
		order = nil
		groups = map[string]*bufferedGroup{}
	}

	for _, item := range list {
		if item.Placeholder == "" {
			flush()
			out = append(out, item)
			continue
		}
		p := item.Placeholder
		if _, ok := groups[p]; !ok {
			groups[p] = &bufferedGroup{}
			order = append(order, p)
		}
		if item.Name == "" {
			if groups[p].header == nil {
				header := item
				groups[p].header = &header
			}
			continue
		}
		groups[p].items = append(groups[p].items, item)
	}
	flush()
	return out
}

// expandedGroup is the dedicated screen for a placeholder. It never limits
// the provider and never inserts another "see more" control.
func expandedGroup(list []core.CommandMatch, group string) []core.CommandMatch {
	out := make([]core.CommandMatch, 0, len(list)+2)
	for _, item := range list {
		if item.Placeholder != group {
			continue
		}
		if item.Name == "" {
			continue
		}
		item.ExpandedGroup = true
		out = append(out, item)
	}
	out = append(out, core.CommandMatch{
		Name:          "sair (Tab)",
		Placeholder:   group,
		ExitControl:   true,
		ExpandedGroup: true,
	})
	return out
}

// NormalizeSelected moves the selection off group headers: headers are pure
// visual labels, never selectable. An incoming index that points at a header
// (the default 0, which is usually the first row) is advanced to the first
// real item, or pulled back to the last one if only headers follow. -1 (no
// selection) and empty lists pass through unchanged.
func NormalizeSelected(matches []core.CommandMatch, selected int) int {
	if len(matches) == 0 || selected < 0 {
		return -1
	}
	if selected >= len(matches) {
		selected = len(matches) - 1
	}
	if matches[selected].Name != "" {
		return selected
	}
	for i := selected + 1; i < len(matches); i++ {
		if matches[i].Name != "" {
			return i
		}
	}
	for i := selected - 1; i >= 0; i-- {
		if matches[i].Name != "" {
			return i
		}
	}
	return -1
}

// BootstrapQuote standardizes quoting for message flags: with `git commit -m`
// and an open message flag, TAB inserts the opening quote and leaves the user
// typing inside it — quoting becomes the default shape without the user ever
// typing a quote. Applied when nothing has been typed after the flag yet.
func BootstrapQuote(buffer string) (string, bool) {
	parts := tokenizer.TokenizeBuffer(buffer)
	if len(parts) == 0 {
		return buffer, false
	}
	// Nothing typed after the message flag means: the flag is either the last
	// token itself (`-m`) or followed only by whitespace (`-m ` — the trailing
	// space yields an empty token). Anything else being typed lets normal
	// completion decide.
	nothingTyped := false
	if parts[len(parts)-1] == "" {
		nothingTyped = true
	} else if common.IsMsgFlag(parts[len(parts)-1]) {
		nothingTyped = true
	}
	if !nothingTyped {
		return buffer, false
	}
	// Find the last message flag, skipping empties.
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "" {
			continue
		}
		if common.IsMsgFlag(parts[i]) {
			return buffer + " \"", true
		}
		if strings.HasPrefix(parts[i], "-") {
			continue // unrelated flag, keep scanning left
		}
		return buffer, false
	}
	return buffer, false
}

// OpenQuoteToken reports whether the buffer ends inside an open double quote
// (e.g. `git commit -m "feat`): the message context started but was never
// closed. When open, a bare `"` closes the message and must be offered as the
// LAST suggestion — it completes the buffer, it is not a value.
func OpenQuoteToken(buffer string) bool {
	parts := tokenizer.TokenizeBuffer(buffer)
	for i := len(parts) - 1; i >= 0; i-- {
		p := parts[i]
		if p == "" {
			continue
		}
		inMsg := false
		// Is this token part of a message context? Scan left for a msg flag.
		for j := i; j >= 0; j-- {
			if common.IsMsgFlag(parts[j]) {
				inMsg = true
				break
			}
		}
		if !inMsg {
			return false
		}
		// Count unescaped double quotes in this token.
		count := strings.Count(p, "\"")
		return count%2 == 1
	}
	return false
}

// CompleteBuffer replaces the current token in buffer with the selected
// suggestion, or appends it if there's nothing to replace.
func CompleteBuffer(buffer string, selectedIndex int, suggestions []core.CommandMatch, currentToken string) string {
	if selectedIndex < 0 || selectedIndex >= len(suggestions) {
		return buffer
	}

	selectedSuggestion := suggestions[selectedIndex].Name

	if currentToken == "" {
		if strings.HasSuffix(buffer, " ") {
			return buffer + selectedSuggestion
		}
		return buffer + " " + selectedSuggestion
	}

	if strings.HasSuffix(buffer, currentToken) {
		idx := len(buffer) - len(currentToken)
		return buffer[:idx] + selectedSuggestion
	}

	return buffer + " " + selectedSuggestion
}
