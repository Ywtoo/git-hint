package parser

import (
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------
// Family detection
//
// Different CLI ecosystems format their -h/--help output differently.
// Git has its own format (handled by the STRATEGY A/B/C parsers above).
// This file adds two more common families:
//
//   - Cobra (Go): docker, kubectl, gh, terraform, hugo, and most modern
//     Go CLIs — they share the library, so they share the format.
//   - GNU coreutils: ls, rm, cp, mv, grep, and most native Unix tools —
//     these are leaf commands (no subcommands), just "Usage: prog [OPTION]..."
//     plus a flat option list.
//
// detectFamily is a cheap heuristic: it looks for section headers/markers
// that are distinctive of each family, without trying to fully parse
// anything yet. If nothing matches, we fall back to git's strategies,
// which is a reasonable default since flag-list parsing (Strategy C) is
// generic enough to pick up *something* from most CLIs anyway.
// ---------------------------------------------------------------------

type cliFamily int

const (
	familyGit cliFamily = iota
	familyCobra
	familyGNU
)

var (
	// Matches ANY "...Commands:" section header — "Available Commands:",
	// "Management Commands:", or the bare "Commands:" that docker's root
	// help uses. Any of these is a strong, distinctive Cobra signal.
	cobraCommandsHeaderRe = regexp.MustCompile(`(?m)^[A-Za-z ]*Commands:\s*$`)
	// Matches a "Flags:", "Global Flags:", or "Options:" section header —
	// docker uses "Options:" while kubectl/most Cobra CLIs use "Flags:".
	cobraFlagsHeaderRe = regexp.MustCompile(`(?m)^(Flags|Global Flags|Options):\s*$`)
	// Matches a capitalized "Usage:" line with content on it (same line,
	// docker-style) or as its own line (kubectl-style, content follows
	// on the next line).
	cobraUsageRe = regexp.MustCompile(`(?m)^Usage:(\s*)$|^Usage:\s+\S`)

	gnuUsageLineRe = regexp.MustCompile(`(?m)^Usage:\s+\S+.*$`)
)

// detectFamily inspects the raw -h/--help output and guesses which CLI
// family produced it.
//
//   - Any "...Commands:" section header is an unambiguous Cobra signal on
//     its own (git and GNU tools never print one, since git uses its own
//     man-page-style indented list and GNU tools have no subcommands at all).
//   - A capitalized "Usage:" line together with a "Flags:"/"Options:" header
//     is Cobra too, even without a Commands section (leaf subcommands like
//     `docker commit -h` only show Usage + Options, no Commands list).
//   - A capitalized "Usage: prog ..." line with no lowercase "usage:"
//     anywhere is GNU coreutils — but only when path has a single element,
//     since GNU tools are always invoked as a single leaf binary (`rm -h`),
//     never through a subcommand path like Cobra CLIs are.
func detectFamily(output string, path []string) cliFamily {
	switch {
	case cobraCommandsHeaderRe.MatchString(output):
		return familyCobra
	case cobraUsageRe.MatchString(output) && cobraFlagsHeaderRe.MatchString(output):
		return familyCobra
	case len(path) == 1 && gnuUsageLineRe.MatchString(output) && !strings.Contains(output, "usage:"):
		// Git always prints its usage lines lowercase ("usage:"/"or:").
		return familyGNU
	default:
		return familyGit
	}
}

// parseByFamily dispatches to the family-specific parser. Returns ok=false
// if the family parser found nothing useful, so the caller can fall back
// to git's strategies as a last resort instead of returning an empty tree.
func parseByFamily(family cliFamily, output string, path []string) (ParsedHelp, bool) {
	switch family {
	case familyCobra:
		parsed := parseCobraHelp(output)
		return parsed, !parsed.IsLeaf()
	case familyGNU:
		parsed := parseGNUHelp(output)
		return parsed, !parsed.IsLeaf()
	default:
		return ParsedHelp{}, false
	}
}

// ---------------------------------------------------------------------
// Cobra family
//
// Typical shape:
//
//	Usage:
//	  docker commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]
//
//	Available Commands:
//	  build       Build an image from a Dockerfile
//	  commit      Create a new image from a container's changes
//
//	Flags:
//	  -a, --author string    Author (e.g., "name <email>")
//	  -m, --message string   Commit message
//
//	Global Flags:
//	      --config string   Location of client config files
// ---------------------------------------------------------------------

// cobraSectionLineRe matches one line inside a Commands section:
// two-or-more spaces, a bare subcommand name, two-or-more spaces, description.
var cobraSectionLineRe = regexp.MustCompile(`^\s{2,}([a-zA-Z][a-zA-Z0-9-]*)\s{2,}(\S.*)$`)

// anyHeaderRe matches any top-level section header line (e.g. "Flags:",
// "Usage:", "Examples:", "Available Commands:") — used to know when a
// Commands section has ended even without a blank line separating it
// from the next section.
var anyHeaderRe = regexp.MustCompile(`^[A-Za-z][A-Za-z ]*:\s*$`)

// parseCobraCommands extracts every "...Commands:" section (docker alone
// uses three different headers across its own help output: "Commands:",
// "Management Commands:", and occasionally "Available Commands:" — Cobra
// CLIs don't agree on a single name for this section) and merges them
// into one flat map of subcommand name → description.
func parseCobraCommands(output string) map[string]*ParsedNode {
	nodes := make(map[string]*ParsedNode)
	lines := strings.Split(output, "\n")
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")

		if cobraCommandsHeaderRe.MatchString(trimmed) {
			inSection = true
			continue
		}

		if !inSection {
			continue
		}

		if trimmed == "" || (anyHeaderRe.MatchString(trimmed) && !cobraCommandsHeaderRe.MatchString(trimmed)) {
			inSection = false
			continue
		}

		m := cobraSectionLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, desc := m[1], strings.TrimSpace(m[2])
		if name == "help" || name == "completion" {
			// Present on virtually every Cobra CLI, rarely useful for
			// autocomplete purposes — skip to keep the tree focused.
			continue
		}
		if _, exists := nodes[name]; !exists {
			nodes[name] = &ParsedNode{Description: desc}
		}
	}

	return nodes
}

// parseCobraHelp merges subcommands (Available Commands) with flags
// (Flags: / Global Flags:, reusing the existing generic flag-list parser
// since Cobra's "  -x, --flag string   desc" shape is a subset of what
// parseFlagList already understands).
func parseCobraHelp(output string) ParsedHelp {
	nodes := parseCobraCommands(output)

	for name, node := range parseFlagList(output) {
		if _, exists := nodes[name]; !exists {
			nodes[name] = node
		}
	}

	return ParsedHelp{Nodes: nodes}
}

// ---------------------------------------------------------------------
// GNU coreutils family
//
// Typical shape:
//
//	Usage: rm [OPTION]... [FILE]...
//	Remove (unlink) the FILE(s).
//
//	  -f, --force           ignore nonexistent files and arguments, never prompt
//	  -i                     prompt before every removal
//
// These tools are leaves — no subcommands, just flags and positional
// arguments. Positional args are written as bare UPPERCASE words (FILE,
// DIRECTORY, SOURCE, DEST) rather than git's <placeholder> convention, so
// we normalize them into that convention for consistency with the rest
// of the tree.
// ---------------------------------------------------------------------

// gnuPositionalRe matches a bare uppercase token, optionally wrapped in
// brackets and/or followed by "...", e.g. "[FILE]...", "SOURCE", "DEST".
var gnuPositionalRe = regexp.MustCompile(`^\[?([A-Z][A-Z0-9_]*)\]?(\.\.\.)?$`)

// parseGNUPositionals pulls positional argument placeholders out of the
// "Usage: prog ..." line and normalizes them to git-style <placeholder>.
func parseGNUPositionals(output string) map[string]*ParsedNode {
	nodes := make(map[string]*ParsedNode)

	m := gnuUsageLineRe.FindString(output)
	if m == "" {
		return nodes
	}

	for _, tok := range strings.Fields(m) {
		if tok == "Usage:" {
			continue
		}
		clean := strings.Trim(tok, "[]().")
		match := gnuPositionalRe.FindStringSubmatch(tok)
		if match == nil {
			continue
		}
		if clean == "" || clean == "OPTION" {
			continue
		}
		placeholder := "<" + strings.ToLower(match[1]) + ">"
		if _, exists := nodes[placeholder]; !exists {
			nodes[placeholder] = &ParsedNode{}
		}
	}

	return nodes
}

// parseGNUHelp merges the positional placeholders from the usage line
// with the flat flag list (reusing the same generic flag-list parser
// used by git and Cobra — GNU's "  -f, --force   description" shape
// matches it directly).
func parseGNUHelp(output string) ParsedHelp {
	nodes := parseGNUPositionals(output)

	for name, node := range parseFlagList(output) {
		if _, exists := nodes[name]; !exists {
			nodes[name] = node
		}
	}

	return ParsedHelp{Nodes: nodes}
}
