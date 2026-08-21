package parser

import (
	"regexp"
	"strings"
)

// ParsedNode represents one token in the parsed help output with its children.
// A flag like -m that takes a <message> will have <message> as a child node.
// A flag with a fixed set of values (e.g. "--color[=WHEN]" where WHEN is
// one of always/auto/never) has one child per literal value instead.
type ParsedNode struct {
	Description string
	Children    map[string]*ParsedNode // for flags: their value/placeholder children
}

// ParsedHelp is the result of parsing one "-h" output.
// Top-level keys are subcommand names, flags, or placeholders.
type ParsedHelp struct {
	Nodes map[string]*ParsedNode
}

func (p ParsedHelp) IsLeaf() bool {
	return len(p.Nodes) == 0
}

// A valid subcommand name: starts with a letter, then letters/digits/hyphens.
var validNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)

// placeholderRe matches a single "<...>" placeholder.
var placeholderRe = regexp.MustCompile(`<[^<>]+>`)

// bareWordRe matches a single uppercase-ish word with no punctuation,
// e.g. "WHEN", "CONTROL", "N" — used to recognize a flag's value spec
// as a normalizable placeholder rather than an enum or garbage.
var bareWordRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// ---------------------------------------------------------------------
// Flag name / value-spec extraction
//
// Flag specs come in a few shapes:
//   --format=WORD           value is required
//   --color[=WHEN]          value is optional (flag works bare too)
//   --no-track[=(direct|inherit)]   optional, with a fixed set of values
//   --[no-]include-untracked        negatable flag, no value at all
//   -m <message>            required, space-separated (handled elsewhere,
//                           see collectTokens' second pass / parseFlagSpec)
//
// stripFlagValue isolates the bare flag name. Using strings.Trim on a
// cutset (the old approach) only strips characters from the START/END of
// the string, so it never touches an unclosed "[" sitting in the middle —
// that's what let "--color[=WHEN]" through as the corrupted key
// "--color[=WHEN" instead of the clean "--color". Cutting at the first
// "[" or "=" instead handles every case uniformly, including nested
// brackets like node's "--inspect-brk[=[host:]port]".
//
// IMPORTANT: the cut must skip over an embedded "[no-]" negation marker
// first (see valueSpecIndex), otherwise "--[no-]include-untracked" gets
// truncated at its own "[" down to a bare "--".
// ---------------------------------------------------------------------

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

func stripFlagValue(tok string) string {
	if idx := valueSpecIndex(tok); idx != -1 {
		return tok[:idx]
	}
	return tok
}

// extractValueChildren pulls the value spec out of a raw flag token (the
// part after the first real value-spec bracket/equals — see
// valueSpecIndex, which skips over an embedded "[no-]" marker) and
// classifies it into child nodes:
//   - "(a|b|c)" or "a|b|c"  -> one literal child per option (enum)
//   - "<name>"              -> that placeholder as-is
//   - "WORD" (bare word)    -> normalized to "<word>", same convention
//     used for GNU positional args
//   - anything else (unclosed brackets, "...", empty) -> nil, no children
//     rather than guessing and creating a corrupted placeholder.
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
	// Leftover brackets mean we couldn't cleanly isolate the value
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
			children[opt] = &ParsedNode{Description: desc}
		}
		if len(children) == 0 {
			return nil
		}
		return children
	}

	if m := placeholderRe.FindString(rest); m != "" {
		children[m] = &ParsedNode{Description: desc}
		return children
	}

	if bareWordRe.MatchString(rest) {
		placeholder := "<" + strings.ToLower(rest) + ">"
		children[placeholder] = &ParsedNode{Description: desc}
		return children
	}

	return nil
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

// ---------------------------------------------------------------------
// STRATEGY A: indented list with description (git root level: git -h / git help -a)
// ---------------------------------------------------------------------

var listLineRe = regexp.MustCompile(`^   (\S+)\s{2,}(.+)$`)

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

		nodes[name] = &ParsedNode{Description: desc}
	}

	return ParsedHelp{Nodes: nodes}
}

// ---------------------------------------------------------------------
// STRATEGY B: "usage:" / "or:" lines
// ---------------------------------------------------------------------

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

// stripWrapping trims surrounding usage-line punctuation (brackets,
// parens, pipes) and then isolates the flag name via stripFlagValue,
// so a raw usage-line token like "[--color[=WHEN]]" collapses to
// "--color" instead of leaving a dangling bracket in the key.
func stripWrapping(tok string) string {
	return stripFlagValue(strings.Trim(tok, "[]()|="))
}

func isFlag(tok string) bool        { return strings.HasPrefix(tok, "-") }
func isPlaceholder(tok string) bool { return strings.HasPrefix(tok, "<") }

func expandNoPrefix(flag string) []string {
	idx := strings.Index(flag, noPrefixMarker)
	if idx == -1 {
		return []string{flag}
	}
	without := flag[:idx] + flag[idx+len(noPrefixMarker):]
	with := flag[:idx] + "no-" + flag[idx+len(noPrefixMarker):]
	return []string{without, with}
}

// getOrCreate returns the existing node for key, or creates a new one.
func getOrCreate(nodes map[string]*ParsedNode, key string) *ParsedNode {
	if n, ok := nodes[key]; ok {
		return n
	}
	n := &ParsedNode{}
	nodes[key] = n
	return n
}

// collectTokens walks usage-line tokens after the command path.
//   - Flags and placeholders are added to nodes.
//   - Flags followed immediately by a placeholder (next token) register the
//     placeholder as a child of the flag.
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
					flagNode := getOrCreate(nodes, expanded)

					inlinePhs := placeholderRe.FindAllString(rawTok, -1)
					for _, ph := range inlinePhs {
						mergeChildren(flagNode, map[string]*ParsedNode{ph: {}})
					}
					mergeChildren(flagNode, extractValueChildren(rawTok, ""))
				}

			case isPlaceholder(tok):
				getOrCreate(nodes, tok)

			case validNameRe.MatchString(tok):
				getOrCreate(nodes, tok)
				return // stop: rest belongs to this subcommand
			}
		}
		i++
	}

	// Second pass: link flags to their placeholder siblings when the flag
	// is immediately followed by a standalone placeholder in the usage line.
	// e.g. "git commit [-m <message>]" → -m has child <message>
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
					mergeChildren(flagNode, map[string]*ParsedNode{firstNxt: {}})
				}
			}
			// Remove the placeholder from the top-level (it belongs under the flag)
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

// ---------------------------------------------------------------------
// STRATEGY C: flag list with description
// e.g.:
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
// ---------------------------------------------------------------------

var (
	// flag spec + description on the same line, separated by 2+ spaces.
	flagWithDescRe = regexp.MustCompile(`^\s*(-\S.*?)\s{2,}(\S.*)$`)
	// flag spec alone (description on next line).
	flagOnlyRe = regexp.MustCompile(`^\s*(-\S.*)$`)
	// deeply-indented continuation (description for the previous flag-only line).
	wrappedDescRe = regexp.MustCompile(`^\s{6,}(\S.*)$`)
)

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
				valueChildren[clean] = &ParsedNode{Description: desc}
			case isFlag(clean):
				flags = append(flags, expandNoPrefix(clean)...)
				for name, child := range extractValueChildren(tok, desc) {
					valueChildren[name] = child
				}
			case placeholderRe.MatchString(tok):
				// inline placeholder glued onto flag e.g. "-u<mode>"
				phs := placeholderRe.FindAllString(tok, -1)
				flagPart := stripFlagValue(placeholderRe.ReplaceAllString(tok, ""))
				if isFlag(flagPart) {
					flags = append(flags, expandNoPrefix(flagPart)...)
				}
				for _, ph := range phs {
					valueChildren[ph] = &ParsedNode{Description: desc}
				}
			}
		}
	}

	for _, f := range flags {
		flagNode := getOrCreate(nodes, f)
		if flagNode.Description == "" {
			flagNode.Description = desc
		}
		mergeChildren(flagNode, valueChildren)
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

// ---------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------

func ParseCommand(output string, path []string) ParsedHelp {
	// At root level only, try the indented-list strategy first (has descriptions).
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
			if len(node.Children) > 0 {
				mergeChildren(existing, node.Children)
				for chName := range node.Children {
					delete(nodes, chName)
				}
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
