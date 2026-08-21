package engine

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"git-hint/core"
	"git-hint/engine/provider"
	"git-hint/engine/ranking"
	"git-hint/engine/tokenizer"
	"git-hint/registry"
)

var noDescriptionFlags = map[string]bool{
	"msg": true,
}

var skipRankingFlags = map[string]bool{
	"commit": true,
	"msg":    true,
}

var ErrNotIndexed = errors.New("command known but not yet indexed")

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

	return list, currentToken, nil
}

// resolveCommandTree loads and parses the subcommand tree for commandName,
// checking along the way whether it's known and already indexed.
func resolveCommandTree(commandName string) (map[string]core.CommandMatch, error) {
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
	groupPlaceholder := name
	if cmd.Description != "" {
		groupPlaceholder = name + "  " + cmd.Description
	}

	// Group header: not selectable as a static command, just a section label.
	header := cmd
	header.Placeholder = groupPlaceholder
	header.Name = ""

	out := []core.CommandMatch{header}

	for _, s := range provider.Provider(flag) {
		if s.Name == "" || !strings.HasPrefix(s.Name, currentToken) {
			continue
		}
		s.Placeholder = groupPlaceholder
		applyDescriptionRule(&s, flag, cmd.Description)
		out = append(out, s)
	}

	return out
}

// applyDescriptionRule decides how a dynamically-provided suggestion's
// Description should be displayed, based on which flag generated it.
func applyDescriptionRule(s *core.CommandMatch, flag, parentDescription string) {
	switch {
	case noDescriptionFlags[flag]:
		// msg: no comment needed, the value itself is the info.
		s.Description = ""

	case skipRankingFlags[flag]:
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
	if !skipRankingFlags[dynamicFlagSeen] {
		ranked, err := ranking.RankSuggestions(input, list)
		if err != nil {
			return nil, fmt.Errorf("❌ Failed to rank commands: %v", err)
		}
		list = ranked
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Placeholder < list[j].Placeholder
	})

	return list, nil
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
