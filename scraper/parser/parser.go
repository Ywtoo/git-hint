package parser

import (
	"regexp"
	"strings"
)

// ============================================================================
// Types
// ============================================================================

// NodeKind classifies a parsed token by its role in the command's help text.
type NodeKind string

const (
	// KindSubcommand marks an exclusive child: selecting it moves to the
	// next command level and may trigger a new help invocation.
	KindSubcommand NodeKind = "subcommand"
	// KindOption marks a combinable flag. Never triggers a new help
	// invocation; its children (if any) are extracted from the same
	// help output as the flag itself.
	KindOption NodeKind = "option"
	// KindArgument marks a leaf value, such as a placeholder (<message>)
	// or a literal enum member (always/auto/never). Never triggers a new
	// help invocation.
	KindArgument NodeKind = "argument"
)

// ParsedNode represents one token in the parsed help output with its children.
// A flag like -m that takes a <message> will have <message> as a child node
// and its key listed in Requires. A flag with a fixed set of values (e.g.
// "--color[=WHEN]" where WHEN is one of always/auto/never) has one child per
// literal value instead.
type ParsedNode struct {
	Kind        NodeKind
	Description string
	Children    map[string]*ParsedNode
	// Requires lists keys from Children that become mandatory once this
	// node is selected (e.g. a flag requiring its value argument).
	Requires []string
}

// ParsedHelp is the result of parsing one "-h" output.
// Top-level keys are subcommand names, flags, or placeholders.
type ParsedHelp struct {
	Nodes map[string]*ParsedNode
}

func (p ParsedHelp) IsLeaf() bool {
	return len(p.Nodes) == 0
}

// ============================================================================
// Regex patterns
// ============================================================================

var (
	// validNameRe matches a valid subcommand name: starts with a letter,
	// then letters/digits/hyphens.
	validNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)

	// placeholderRe matches a single "<...>" placeholder.
	placeholderRe = regexp.MustCompile(`<[^<>]+>`)

	// bareWordRe matches a single uppercase-ish word with no punctuation,
	// e.g. "WHEN", "CONTROL", "N" — used to recognize a flag's value spec
	// as a normalizable placeholder rather than an enum or garbage.
	bareWordRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

	// listLineRe matches indented list lines (Strategy A):
	// "   subcommand  description text"
	listLineRe = regexp.MustCompile(`^   (\S+)\s{2,}(.+)$`)

	// flagWithDescRe matches a flag spec + description on the same line:
	// "-m, --message <message>    commit message"
	flagWithDescRe = regexp.MustCompile(`^\s*(-\S.*?)\s{2,}(\S.*)$`)

	// flagOnlyRe matches a flag spec alone (description on next line):
	// "-F, --file <file>"
	flagOnlyRe = regexp.MustCompile(`^\s*(-\S.*)$`)

	// wrappedDescRe matches deeply-indented continuation lines:
	// "                           read message from file"
	wrappedDescRe = regexp.MustCompile(`^\s{6,}(\S.*)$`)
)

// ============================================================================
// Shared helpers — token classification
// ============================================================================

func isFlag(tok string) bool        { return strings.HasPrefix(tok, "-") }
func isPlaceholder(tok string) bool { return strings.HasPrefix(tok, "<") }

// ============================================================================
// Shared helpers — flag name / value-spec extraction
//
// Flag specs come in a few shapes:
//
//	--format=WORD                        value is required
//	--color[=WHEN]                       value is optional
//	--no-track[=(direct|inherit)]        optional, with a fixed set of values
//	--[no-]include-untracked             negatable flag, no value at all
//	-m <message>                         required, space-separated
//
// stripFlagValue isolates the bare flag name. The cut must skip over an
// embedded "[no-]" negation marker first (see valueSpecIndex), otherwise
// "--[no-]include-untracked" gets truncated at its own "[" down to "--".
// ============================================================================

// noPrefixMarker is the GNU convention for a flag with a negated form,
// e.g. "--[no-]include-untracked". It must not be mistaken for a value
// spec bracket like "--color[=WHEN]".
const noPrefixMarker = "[no-]"

// valueSpecIndex finds where a value spec ("[=WHEN]", "=FOO") begins,
// skipping over an embedded "[no-]" negation marker first so it isn't
// mistaken for the start of a value bracket. Returns -1 if there is no
// value spec.
func valueSpecIndex(tok string) int {
	search := tok
	offset := 0
	if idx := strings.Index(tok, noPrefixMarker); idx != -1 {
		offset = idx + len(noPrefixMarker)
		search = tok[offset:]
	}
	idx := strings.IndexAny(search, "[=")
	if idx == -1 {
		return -1
	}
	return offset + idx
}

// stripFlagValue removes the value spec from a flag token, leaving just
// the bare flag name (e.g. "--color[=WHEN]" → "--color").
func stripFlagValue(tok string) string {
	if idx := valueSpecIndex(tok); idx != -1 {
		return tok[:idx]
	}
	return tok
}

// stripWrapping trims surrounding usage-line punctuation (brackets,
// parens, pipes) and then isolates the flag name via stripFlagValue,
// so a raw usage-line token like "[--color[=WHEN]]" collapses to
// "--color" instead of leaving a dangling bracket in the key.
func stripWrapping(tok string) string {
	return stripFlagValue(strings.Trim(tok, "[]()|="))
}

// expandNoPrefix expands a "[no-]" marker into its positive and negative
// forms. E.g. "--[no-]include-untracked" → ["--include-untracked", "--no-include-untracked"].
func expandNoPrefix(flag string) []string {
	idx := strings.Index(flag, noPrefixMarker)
	if idx == -1 {
		return []string{flag}
	}
	without := flag[:idx] + flag[idx+len(noPrefixMarker):]
	with := flag[:idx] + "no-" + flag[idx+len(noPrefixMarker):]
	return []string{without, with}
}

// extractValueChildren pulls the value spec out of a raw flag token (the
// part after the first real value-spec bracket/equals — see
// valueSpecIndex, which skips over an embedded "[no-]" marker) and
// classifies it into child nodes:
//   - "(a|b|c)" or "a|b|c"  → one literal child per option (enum)
//   - "<name>"              → that placeholder as-is
//   - "WORD" (bare word)    → normalized to "<word>"
//   - anything else         → nil (no children)
func extractValueChildren(tok, desc string) map[string]*ParsedNode {
	idx := valueSpecIndex(tok)
	if idx == -1 {
		return nil
	}
	rest := tok[idx:]
	rest = strings.TrimPrefix(rest, "[")
	rest = strings.TrimPrefix(rest, "=")
	rest = strings.TrimSuffix(rest, "]")
	rest = strings.TrimSpace(rest)
	rest = strings.Trim(rest, "()")

	if rest == "" || rest == "..." {
		return nil
	}
	// Leftover brackets mean the value could not be cleanly isolated
	// (e.g. nested "[host:]port") — bail out instead of emitting garbage.
	if strings.ContainsAny(rest, "[]") {
		return nil
	}

	children := make(map[string]*ParsedNode)

	if strings.Contains(rest, "|") {
		for _, opt := range strings.Split(rest, "|") {
			opt = strings.TrimSpace(strings.Trim(opt, "()"))
			if opt == "" {
				continue
			}
			children[opt] = &ParsedNode{Kind: KindArgument, Description: desc}
		}
		if len(children) == 0 {
			return nil
		}
		return children
	}

	if m := placeholderRe.FindString(rest); m != "" {
		children[m] = &ParsedNode{Kind: KindArgument, Description: desc}
		return children
	}

	if bareWordRe.MatchString(rest) {
		placeholder := "<" + strings.ToLower(rest) + ">"
		children[placeholder] = &ParsedNode{Kind: KindArgument, Description: desc}
		return children
	}

	return nil
}

// ============================================================================
// Shared helpers — node manipulation
// ============================================================================

// getOrCreate returns the existing node for key, or creates one with the
// given kind. The kind of an existing node is never downgraded — the
// first classification for a given key is authoritative.
func getOrCreate(nodes map[string]*ParsedNode, key string, kind NodeKind) *ParsedNode {
	if n, ok := nodes[key]; ok {
		if n.Kind == "" {
			n.Kind = kind
		}
		return n
	}
	n := &ParsedNode{Kind: kind}
	nodes[key] = n
	return n
}

// mergeChildren copies src into dst (creating dst if needed), keeping
// whichever entry already exists on conflict.
func mergeChildren(node *ParsedNode, src map[string]*ParsedNode) {
	if len(src) == 0 {
		return
	}
	if node.Children == nil {
		node.Children = make(map[string]*ParsedNode)
	}
	for name, child := range src {
		if _, exists := node.Children[name]; !exists {
			node.Children[name] = child
		}
	}
}

// requireChild records that key becomes mandatory once node is selected,
// without introducing a duplicate entry in Requires.
func requireChild(node *ParsedNode, key string) {
	for _, existing := range node.Requires {
		if existing == key {
			return
		}
	}
	node.Requires = append(node.Requires, key)
}

// ============================================================================
// Strategy A — indented list with description
//
// Used at root level only (git -h / git help -a). Lines like:
//
//	   commit    create a commit for the staged changes
//	   add       add file contents to the index
// ============================================================================

func parseIndentedList(output string) ParsedHelp {
	nodes := make(map[string]*ParsedNode)

	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, " \t")

		match := listLineRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		name := match[1]
		desc := strings.TrimSpace(match[2])

		if strings.HasPrefix(name, "-") {
			continue // flag, not subcommand
		}
		if !validNameRe.MatchString(name) {
			continue
		}

		nodes[name] = &ParsedNode{Kind: KindSubcommand, Description: desc}
	}

	return ParsedHelp{Nodes: nodes}
}

// ============================================================================
// Strategy B — "usage:" / "or:" lines
//
// Parses the usage synopsis to extract the command structure (flags,
// placeholders, subcommands). This gives the skeleton; descriptions
// come from Strategy C.
// ============================================================================

// collectUsageLines returns the full text of each "usage:"/"or:" statement,
// with wrapped continuation lines joined back in.
func collectUsageLines(output string) []string {
	raw := strings.Split(output, "\n")
	var statements []string

	for i := 0; i < len(raw); i++ {
		line := strings.TrimSpace(raw[i])

		var text string
		switch {
		case strings.HasPrefix(line, "usage:"):
			text = strings.TrimSpace(strings.TrimPrefix(line, "usage:"))
		case strings.HasPrefix(line, "or:"):
			text = strings.TrimSpace(strings.TrimPrefix(line, "or:"))
		default:
			continue
		}

		for i+1 < len(raw) {
			nextLine := raw[i+1]
			trimmedNext := strings.TrimSpace(nextLine)
			if trimmedNext == "" {
				break
			}
			if strings.HasPrefix(trimmedNext, "usage:") || strings.HasPrefix(trimmedNext, "or:") {
				break
			}
			if flagWithDescRe.MatchString(nextLine) || flagOnlyRe.MatchString(nextLine) {
				break
			}
			if strings.HasPrefix(nextLine, " ") || strings.HasPrefix(nextLine, "\t") ||
				strings.HasPrefix(trimmedNext, "[") || strings.HasPrefix(trimmedNext, "(") ||
				strings.HasPrefix(trimmedNext, "-") || strings.HasPrefix(trimmedNext, "<") {
				text += " " + trimmedNext
				i++
			} else {
				break
			}
		}

		statements = append(statements, text)
	}

	return statements
}

// pathMatches checks that fields starts with the given path segments.
func pathMatches(fields []string, path []string) bool {
	if len(fields) < len(path) {
		return false
	}
	for i, p := range path {
		if fields[i] != p {
			return false
		}
	}
	return true
}

// collectTokens walks usage-line tokens after the command path.
//   - Flags and placeholders are added to nodes.
//   - Flags followed immediately by a placeholder (next token) register the
//     placeholder as a required child of the flag.
//   - Flags carrying a bracketed value spec (enum or bare word) get that
//     value expanded into children too.
//   - Flags carrying a "[no-]" negation marker get expanded into both
//     their positive and negative forms, same as Strategy C.
//   - Real subcommand names stop the scan (the rest belongs to that subcommand).
func collectTokens(fields []string, afterIndex int, nodes map[string]*ParsedNode) {
	i := afterIndex
	for i < len(fields) {
		raw := strings.ReplaceAll(fields[i], "|", " ")
		rawTokens := strings.Fields(raw)

		for _, rawTok := range rawTokens {
			tok := stripWrapping(rawTok)
			if tok == "" {
				continue
			}

			switch {
			case isFlag(tok):
				for _, expanded := range expandNoPrefix(tok) {
					flagNode := getOrCreate(nodes, expanded, KindOption)

					inlinePhs := placeholderRe.FindAllString(rawTok, -1)
					for _, ph := range inlinePhs {
						mergeChildren(flagNode, map[string]*ParsedNode{ph: {Kind: KindArgument}})
					}
					mergeChildren(flagNode, extractValueChildren(rawTok, ""))
				}

			case isPlaceholder(tok):
				getOrCreate(nodes, tok, KindArgument)

			case validNameRe.MatchString(tok):
				getOrCreate(nodes, tok, KindSubcommand)
				return // stop: rest belongs to this subcommand
			}
		}
		i++
	}

	// Second pass: link flags to their placeholder siblings when the flag
	// is immediately followed by a standalone placeholder in the usage line,
	// e.g. "git commit [-m <message>]" -> -m requires <message>.
	for idx := afterIndex; idx < len(fields)-1; idx++ {
		cur := stripWrapping(strings.ReplaceAll(fields[idx], "|", " "))
		nxt := stripWrapping(fields[idx+1])
		curToks := strings.Fields(cur)
		nxtToks := strings.Fields(nxt)
		if len(curToks) == 0 || len(nxtToks) == 0 {
			continue
		}
		lastCur := curToks[len(curToks)-1]
		firstNxt := nxtToks[0]
		if isFlag(lastCur) && isPlaceholder(firstNxt) {
			for _, expanded := range expandNoPrefix(lastCur) {
				if flagNode, ok := nodes[expanded]; ok {
					mergeChildren(flagNode, map[string]*ParsedNode{firstNxt: {Kind: KindArgument}})
					requireChild(flagNode, firstNxt)
				}
			}
			// The placeholder belongs under the flag, not at the top level.
			delete(nodes, firstNxt)
		}
	}
}

func parseUsageLines(output string, path []string) ParsedHelp {
	nodes := make(map[string]*ParsedNode)

	for _, line := range collectUsageLines(output) {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != path[0] {
			continue
		}
		if !pathMatches(fields, path) {
			continue
		}
		collectTokens(fields, len(path), nodes)
	}

	return ParsedHelp{Nodes: nodes}
}

// ============================================================================
// Strategy C — flag list with description
//
// Parses the detailed flag descriptions, e.g.:
//
//	-m, --message <message>    commit message
//	-F, --file <file>
//	                           read message from file
//	--color[=WHEN]              colorize the output
//	-u, --[no-]include-untracked  include untracked files in the stash
//
// Each flag spec may list a short alias and a long alias separated by ", ".
// Any value spec (placeholder, bracketed optional, or enum) attaches as
// children to every alias flag in the spec. A "[no-]" marker on any alias
// expands that alias into its positive and negative forms.
// ============================================================================

// parseFlagSpec parses a flag spec like "-m, --message <message>" or
// "--color[=WHEN]" or "-u, --[no-]include-untracked" and records each
// alias into nodes. Any value spec (placeholder, enum, or normalized bare
// word) becomes a child of each flag alias node. A "[no-]" marker expands
// that alias into both its positive and negative forms.
func parseFlagSpec(spec, desc string, nodes map[string]*ParsedNode) {
	parts := strings.Split(spec, ",")

	var flags []string
	valueChildren := make(map[string]*ParsedNode)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		toks := strings.Fields(part)
		for _, tok := range toks {
			clean := stripFlagValue(tok)
			switch {
			case isPlaceholder(clean):
				valueChildren[clean] = &ParsedNode{Kind: KindArgument, Description: desc}
			case isFlag(clean):
				flags = append(flags, expandNoPrefix(clean)...)
				for name, child := range extractValueChildren(tok, desc) {
					valueChildren[name] = child
				}
			case placeholderRe.MatchString(tok):
				// Inline placeholder glued onto the flag, e.g. "-u<mode>".
				phs := placeholderRe.FindAllString(tok, -1)
				flagPart := stripFlagValue(placeholderRe.ReplaceAllString(tok, ""))
				if isFlag(flagPart) {
					flags = append(flags, expandNoPrefix(flagPart)...)
				}
				for _, ph := range phs {
					valueChildren[ph] = &ParsedNode{Kind: KindArgument, Description: desc}
				}
			}
		}
	}

	for _, f := range flags {
		flagNode := getOrCreate(nodes, f, KindOption)
		if flagNode.Description == "" {
			flagNode.Description = desc
		}
		mergeChildren(flagNode, valueChildren)
		for vName := range valueChildren {
			requireChild(flagNode, vName)
		}
	}
}

func parseFlagList(output string) map[string]*ParsedNode {
	nodes := make(map[string]*ParsedNode)
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if m := flagWithDescRe.FindStringSubmatch(line); m != nil {
			parseFlagSpec(m[1], strings.TrimSpace(m[2]), nodes)
			continue
		}

		if m := flagOnlyRe.FindStringSubmatch(line); m != nil {
			candidate := strings.TrimSpace(m[1])
			if !strings.HasPrefix(candidate, "-") {
				continue
			}
			desc := ""
			if i+1 < len(lines) {
				if wm := wrappedDescRe.FindStringSubmatch(lines[i+1]); wm != nil {
					desc = strings.TrimSpace(wm[1])
					i++
				}
			}
			parseFlagSpec(candidate, desc, nodes)
		}
	}

	return nodes
}

// ============================================================================
// Entry point
// ============================================================================

// ParseCommand parses the help output for a command at the given path
// and returns a tree of nodes (subcommands, options, arguments).
func ParseCommand(output string, path []string) ParsedHelp {
	// At root level only, try the indented-list strategy first (it carries
	// descriptions that the usage-line strategy does not).
	if len(path) == 1 {
		if fromList := parseIndentedList(output); !fromList.IsLeaf() {
			return fromList
		}
	}

	// Merge: Strategy B (usage lines) gives structure; Strategy C (flag list)
	// gives descriptions. Strategy C wins on descriptions (richer text).
	nodes := parseUsageLines(output, path).Nodes
	for name, node := range parseFlagList(output) {
		if existing, ok := nodes[name]; ok {
			if existing.Description == "" && node.Description != "" {
				existing.Description = node.Description
			}
			if existing.Kind == "" {
				existing.Kind = node.Kind
			}
			if len(node.Children) > 0 {
				mergeChildren(existing, node.Children)
				for chName := range node.Children {
					delete(nodes, chName)
				}
			}
			for _, req := range node.Requires {
				requireChild(existing, req)
			}
		} else {
			nodes[name] = node
			for chName := range node.Children {
				delete(nodes, chName)
			}
		}
	}

	return ParsedHelp{Nodes: nodes}
}
