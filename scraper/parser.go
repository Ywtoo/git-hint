package scraper

import (
	"regexp"
	"strings"
)

// ParsedNode represents one token in the parsed help output with its children.
// A flag like -m that takes a <message> will have <message> as a child node.
type ParsedNode struct {
	Description string
	Children    map[string]*ParsedNode // for flags: their placeholder children
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
			// Continuation line: indented or starts with bracket/flag/placeholder
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

func stripWrapping(tok string) string {
	return strings.Trim(tok, "[]()|=")
}

func isFlag(tok string) bool        { return strings.HasPrefix(tok, "-") }
func isPlaceholder(tok string) bool { return strings.HasPrefix(tok, "<") }

func expandNoPrefix(flag string) []string {
	const marker = "[no-]"
	idx := strings.Index(flag, marker)
	if idx == -1 {
		return []string{flag}
	}
	without := flag[:idx] + flag[idx+len(marker):]
	with := flag[:idx] + "no-" + flag[idx+len(marker):]
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

// addFlagWithPlaceholder adds a flag entry; if the flag carries an inline
// placeholder (e.g. "-u<mode>"), the placeholder becomes a child of the flag.
func addFlagWithPlaceholder(tok, desc string, nodes map[string]*ParsedNode) {
	tok = strings.Trim(tok, "[]()|=")
	if tok == "" {
		return
	}

	// Collect any placeholders embedded in this token (e.g. "-u<mode>")
	placeholders := placeholderRe.FindAllString(tok, -1)

	// Strip placeholders to get the pure flag
	flagRaw := strings.Trim(placeholderRe.ReplaceAllString(tok, ""), "[]()|=")

	if isPlaceholder(tok) {
		// Bare placeholder (no flag) — just register at this level
		n := getOrCreate(nodes, tok)
		if n.Description == "" {
			n.Description = desc
		}
		return
	}

	if !strings.HasPrefix(flagRaw, "-") {
		return
	}

	for _, f := range expandNoPrefix(flagRaw) {
		flagNode := getOrCreate(nodes, f)
		if flagNode.Description == "" {
			flagNode.Description = desc
		}
		// Placeholders embedded in this flag become children of the flag node
		for _, ph := range placeholders {
			if flagNode.Children == nil {
				flagNode.Children = make(map[string]*ParsedNode)
			}
			phNode := &ParsedNode{Description: desc}
			if _, exists := flagNode.Children[ph]; !exists {
				flagNode.Children[ph] = phNode
			}
		}
	}
}

// collectTokens walks usage-line tokens after the command path.
// - Flags and placeholders are added to nodes.
// - Flags followed immediately by a placeholder (next token) register the
//   placeholder as a child of the flag.
// - Real subcommand names stop the scan (the rest belongs to that subcommand).
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
				// Check if the NEXT field (after stripping) is a bare placeholder
				// e.g. "-m <message>" or "--file <file>"
				flagNode := getOrCreate(nodes, tok)

				// Look ahead for placeholder(s) glued onto this flag or as next token
				inlinePhs := placeholderRe.FindAllString(tok, -1)
				if len(inlinePhs) > 0 {
					for _, ph := range inlinePhs {
						if flagNode.Children == nil {
							flagNode.Children = make(map[string]*ParsedNode)
						}
						if _, exists := flagNode.Children[ph]; !exists {
							flagNode.Children[ph] = &ParsedNode{}
						}
					}
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
			if flagNode, ok := nodes[lastCur]; ok {
				if flagNode.Children == nil {
					flagNode.Children = make(map[string]*ParsedNode)
				}
				if _, exists := flagNode.Children[firstNxt]; !exists {
					flagNode.Children[firstNxt] = &ParsedNode{}
				}
				// Remove the placeholder from the top-level (it belongs under the flag)
				delete(nodes, firstNxt)
			}
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
//
// Each flag spec may list a short alias and a long alias separated by ", ".
// The placeholder (if any) is the LAST token in the flag spec and becomes
// a child of every alias flag.
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
// "-F, --file <file>" and records each alias into nodes. The trailing
// placeholder (if any) becomes a child of each flag alias node.
func parseFlagSpec(spec, desc string, nodes map[string]*ParsedNode) {
	parts := strings.Split(spec, ",")

	var flags []string
	var placeholder string // last placeholder seen in the spec

	for _, part := range parts {
		part = strings.TrimSpace(part)
		toks := strings.Fields(part)
		for _, tok := range toks {
			clean := strings.Trim(tok, "[]()|=")
			if isPlaceholder(clean) {
				placeholder = clean
			} else if isFlag(clean) {
				for _, expanded := range expandNoPrefix(clean) {
					flags = append(flags, expanded)
				}
			} else if placeholderRe.MatchString(tok) {
				// inline placeholder glued onto flag e.g. "-u<mode>"
				phs := placeholderRe.FindAllString(tok, -1)
				flagPart := strings.Trim(placeholderRe.ReplaceAllString(tok, ""), "[]()|=")
				if isFlag(flagPart) {
					for _, expanded := range expandNoPrefix(flagPart) {
						flags = append(flags, expanded)
					}
				}
				for _, ph := range phs {
					placeholder = ph
				}
			}
		}
	}

	for _, f := range flags {
		flagNode := getOrCreate(nodes, f)
		if flagNode.Description == "" {
			flagNode.Description = desc
		}
		if placeholder != "" {
			if flagNode.Children == nil {
				flagNode.Children = make(map[string]*ParsedNode)
			}
			if _, exists := flagNode.Children[placeholder]; !exists {
				flagNode.Children[placeholder] = &ParsedNode{Description: desc}
			}
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
			// Update description if richer
			if existing.Description == "" && node.Description != "" {
				existing.Description = node.Description
			}
			// Merge children (placeholders from flag list are authoritative)
			if len(node.Children) > 0 {
				if existing.Children == nil {
					existing.Children = make(map[string]*ParsedNode)
				}
				for chName, chNode := range node.Children {
					if _, exists := existing.Children[chName]; !exists {
						existing.Children[chName] = chNode
					}
				}
				// Remove top-level placeholder if it's now a child of a flag
				for chName := range node.Children {
					delete(nodes, chName)
				}
			}
		} else {
			nodes[name] = node
			// Remove top-level placeholders that are children of this new flag
			for chName := range node.Children {
				delete(nodes, chName)
			}
		}
	}

	return ParsedHelp{Nodes: nodes}
}
