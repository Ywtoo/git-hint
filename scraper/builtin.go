package scraper

import (
	"os"
	"strings"

	"git-hint/internal/core"
	"git-hint/scraper/parser"
)

// ============================================================================
// Builtin command scraping
//
// Builtins have no binary to run "--help" against. Their usage is sourced
// from available documentation, tried in this order:
//
//  1. HELPDIR (typically /usr/share/zsh/help) — zsh's own help files
//  2. zshbuiltins man page — fallback on macOS / Linux
//  3. bash's help builtin (bash -c 'help <name>') — last resort
//
// HELPDIR and man page give zsh-specific flags (e.g. cd's -q, -s),
// while bash help gives bash flags (cd's -L, -P). Since this is a
// zsh plugin, zsh sources must come first.
//
// If no source yields a parseable synopsis, the builtin is marked as
// failed and never retried (graceful degradation).
//
// Source files:
//   - man_help.go   — HELPDIR + man page sources
//   - bash_help.go  — bash help source (fallback)
//   - cache.go      — failure cache
//   - types.go      — shared types
//   - diagnostics.go — instrumentation
// ============================================================================

// CrawlBuiltin builds and writes the data file for a zsh builtin from
// the shell's own documentation. Returns an error when no source could
// produce a parseable synopsis.
func CrawlBuiltin(name string) error {
	if builtinIsFailed(name) {
		return os.ErrNotExist
	}

	for _, src := range []func(string) (string, bool){
		builtinHelpFromHelpDir,
		builtinHelpFromManPage,
		builtinHelpFromBashHelp,
	} {
		synopsis, ok := src(name)
		if !ok {
			continue
		}
		tree := parseBuiltinSynopsis(name, synopsis)
		if len(tree) == 0 {
			continue
		}
		return WriteCommand(name, tree)
	}

	builtinMarkFailed(name)
	return os.ErrNotExist
}

// ============================================================================
// Synopsis normalization
//
// zsh and bash use different notations in their usage lines that the
// generic parser doesn't understand. These functions normalize them into
// equivalent forms the parser can handle.
// ============================================================================

// parseBuiltinSynopsis feeds the builtin's usage lines through the same
// parser used for regular --help text (shaped as usage:/or: statements,
// which parseUsageLines understands) and returns the tree in the same
// shape CrawlOne writes for PATH commands.
func parseBuiltinSynopsis(name, synopsis string) map[string]core.CommandMatch {
	synopsis = normalizeBuiltinSynopsis(name, synopsis)
	synopsis = normalizeZshGroups(synopsis)

	var b strings.Builder
	for _, line := range strings.Split(synopsis, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("usage: ")
		} else {
			b.WriteString("or: ")
		}
		b.WriteString(strings.TrimSpace(line))
		b.WriteByte('\n')
	}
	if b.Len() == 0 {
		return nil
	}

	parsed := parser.ParseCommand(b.String(), []string{name})
	if parsed.IsLeaf() {
		return nil
	}

	var root core.CommandMatch
	for nodeName, node := range parsed.Nodes {
		child := core.NewCommandMatch("", "")
		child.Description = node.Description
		child.Kind = core.NodeKind(node.Kind)
		child.Requires = node.Requires
		for chName, chNode := range node.Children {
			ph := core.NewCommandMatch("", "")
			ph.Description = chNode.Description
			ph.Kind = core.NodeKind(chNode.Kind)
			ph.Requires = chNode.Requires
			attachChildNode(&child, chName, ph)
		}
		attachChildNode(&root, nodeName, child)
	}
	return root.SubCommand
}

// normalizeZshGroups rewrites zsh synopsis notations the generic parser
// doesn't know into equivalent forms it does:
//
//   - grouped short flags inside brackets, "[ -qsLP ]", become separate
//     optional flags "[ -q ] [ -s ] [ -L ] [ -P ]" (zsh semantics: the
//     group is optional as a whole, each letter is its own flag);
//   - the brace alternative "{+|-}n" becomes "-n" (the + variant is
//     dropped: the parser only models "-" flags, and the negative form
//     is by far the common one to type);
//   - a name[=value] pair loses its attached =value part, which adds no
//     hint value to the suggestion list.
func normalizeZshGroups(synopsis string) string {
	var b strings.Builder
	for _, line := range strings.Split(synopsis, "\n") {
		b.WriteString(normalizeZshGroupsLine(line))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// normalizeZshGroupsLine applies normalizeZshGroups to a single line.
func normalizeZshGroupsLine(line string) string {
	line = strings.ReplaceAll(line, "[=value]", "") // name[=value] pairs
	line = strings.ReplaceAll(line, "{+|-}", "-")   // brace alternatives
	return expandBracketGroups(line)
}

// expandBracketGroups walks the line expanding flag-only bracket groups,
// innermost pair first so nested groups like "[ -R [ -en ]]" are
// handled. Groups that aren't clustered short flags (placeholders,
// multi-word lists) are kept verbatim.
func expandBracketGroups(line string) string {
	start := 0
	for {
		open, close, ok := innermostBracket(line, start)
		if !ok {
			return line
		}
		if g := expandFlagGroup(line[open+1 : close]); g != "" {
			line = line[:open] + g + line[close+1:]
			start = 0 // text shifted; rescan (singles are kept verbatim)
		} else {
			start = close + 1 // skip the verbatim pair
		}
	}
}

// innermostBracket returns the first bracket pair at or after start that
// contains no nested '[' inside, so callers process inner groups before
// outers.
func innermostBracket(s string, start int) (open, close int, ok bool) {
	i := start
	for i < len(s) {
		if s[i] != '[' {
			i++
			continue
		}
		end := strings.IndexByte(s[i:], ']')
		if end == -1 {
			return 0, 0, false
		}
		close = i + end
		inner := s[i+1 : close]
		if j := strings.IndexByte(inner, '['); j >= 0 {
			i = i + 1 + j // descend into the nested bracket
			continue
		}
		return i, close, true
	}
	return 0, 0, false
}

// expandFlagGroup rewrites the inside of a bracket group holding only
// clustered short flags (e.g. " -qsLP ") into one bracket per flag
// (" [ -q ] [ -s ] [ -L ] [ -P ] "). Single flags (" -e ") and anything
// else (placeholders, multiple words) return "" meaning: keep verbatim.
func expandFlagGroup(inner string) string {
	trimmed := strings.TrimSpace(inner)
	if trimmed == "" || !strings.HasPrefix(trimmed, "-") {
		return ""
	}
	letters := trimmed[1:]
	if len(letters) < 2 {
		return ""
	}
	for _, r := range letters {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return ""
		}
	}
	groups := make([]string, 0, len(letters))
	for _, r := range letters {
		groups = append(groups, "[ -"+string(r)+" ]")
	}
	return strings.Join(groups, " ")
}

// isBareArgWord reports whether a usage token is a bare variable name
// (arg, old, n...) rather than a flag, bracket or placeholder.
func isBareArgWord(tok string) bool {
	if tok == "" {
		return false
	}
	switch tok[0] {
	case '-', '<', '[', '{', ']', '}', '+':
		return false
	}
	for _, r := range tok {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// normalizeBuiltinSynopsis wraps bare argument words (arg, old, new...)
// in <placeholders> so the parser treats them as arguments instead of
// subcommands. Builtins never have subcommands, so every bare word in
// their usage lines is a variable.
func normalizeBuiltinSynopsis(name, synopsis string) string {
	var out []string
	for _, line := range strings.Split(synopsis, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		for i, tok := range fields {
			if i == 0 && tok == name {
				continue // the command name itself
			}
			if isBareArgWord(tok) {
				fields[i] = "<" + tok + ">"
			}
		}
		out = append(out, strings.Join(fields, " "))
	}
	return strings.Join(out, "\n")
}
