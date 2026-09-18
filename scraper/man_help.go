package scraper

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ============================================================================
// Source 2: zsh HELPDIR files
//
// Typically /usr/share/zsh/help — one plain-text file per builtin, the
// same files zsh's run-help reads. The leading lines hold the usage
// synopsis, e.g. "cd [ -qsLP ] [ arg ]".
// ============================================================================

// helpDirCandidates returns the directories that may hold per-builtin
// help files, mirroring zsh's run-help default plus common prefixes.
func helpDirCandidates() []string {
	if dir := os.Getenv("HELPDIR"); dir != "" {
		return []string{dir}
	}
	return []string{
		"/usr/share/zsh/help",
		"/usr/local/share/zsh/help",
		"/opt/homebrew/share/zsh/help",
	}
}

// builtinLookupName maps builtin names to their help-file names the same
// way zsh's run-help does.
func builtinLookupName(name string) string {
	switch name {
	case ".":
		return "dot"
	case ":":
		return "colon"
	}
	return name
}

// builtinHelpFromHelpDir reads the plain-text help file for one builtin
// and extracts its synopsis lines.
func builtinHelpFromHelpDir(name string) (string, bool) {
	lookup := builtinLookupName(name)

	for _, dir := range helpDirCandidates() {
		data, err := os.ReadFile(filepath.Join(dir, lookup))
		if err != nil {
			continue
		}
		synopsis := firstSynopsisLines(string(data), lookup)
		if synopsis == "" {
			continue
		}
		return synopsis, true
	}
	return "", false
}

// firstSynopsisLines extracts the synopsis lines from a HELPDIR file:
// the leading consecutive lines that start with the builtin's name.
// A file may hold several usage forms, one per line (e.g. cd's three
// variants). Blank lines, indented description prose, or a differently
// named entry (e.g. "chdir  Same as cd." inside cd's file) end the block.
func firstSynopsisLines(text, lookup string) string {
	var synopsis []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !synopsisStartsWith(trimmed, lookup) {
			break
		}
		synopsis = append(synopsis, trimmed)
	}
	return strings.Join(synopsis, "\n")
}

// ============================================================================
// Source 3: zshbuiltins man page
//
// When stdout is not a TTY, man formats the page and writes plain text
// (no groff escapes). This ships on macOS (where the HELPDIR files are
// absent) and on Linux. Read at most once per process.
// ============================================================================

var (
	manPageOnce sync.Once
	manPageText string
	manPageOK   bool
)

// execManZshbuiltins runs "man zshbuiltins" and returns its output.
// Extracted for testability (the real-man test calls it directly).
func execManZshbuiltins() ([]byte, error) {
	return exec.Command("man", "zshbuiltins").Output()
}

// builtinHelpFromManPage renders the zshbuiltins man page to plain text
// once per process and extracts the given builtin's synopsis lines.
func builtinHelpFromManPage(name string) (string, bool) {
	manPageOnce.Do(func() {
		out, err := execManZshbuiltins()
		if err != nil || len(out) == 0 {
			return
		}
		manPageText = string(out)
		manPageOK = true
	})
	if !manPageOK {
		return "", false
	}
	return extractRenderedSynopsis(manPageText, name)
}

// extractRenderedSynopsis pulls every usage line of one builtin from the
// rendered man page. Entries are indented 7 spaces; a builtin's usage
// block is the leading consecutive lines starting with "<name> ["/"name "
// right after the 7-space indent. Description prose is indented deeper
// (14 spaces), so requiring the 7-space indent + name prefix isolates it.
func extractRenderedSynopsis(manSrc, name string) (string, bool) {
	lines := strings.Split(manSrc, "\n")
	const entryIndent = "       " // 7 spaces, man's standard entry indent

	var synopsis []string
	started := false

	for _, line := range lines {
		if !strings.HasPrefix(line, entryIndent) {
			if started {
				break // entry over (blank line or section change)
			}
			continue
		}
		body := line[len(entryIndent):]
		trimmed := strings.TrimSpace(body)

		if len(synopsis) == 0 && !started {
			if !synopsisStartsWith(trimmed, name) {
				continue
			}
			started = true
			synopsis = append(synopsis, trimmed)
			continue
		}

		// Inside the entry: continuation synopsis lines share the same
		// shallow indent and name prefix; anything deeper is prose.
		if !synopsisStartsWith(trimmed, name) {
			// A long synopsis may wrap onto a deeper-indented line that
			// starts with a bracket (e.g. print's second usage line).
			if strings.HasPrefix(body, " ") && strings.HasPrefix(trimmed, "[") {
				synopsis[len(synopsis)-1] += " " + trimmed
				continue
			}
			break
		}
		synopsis = append(synopsis, trimmed)
	}

	if len(synopsis) == 0 {
		return "", false
	}
	return strings.Join(synopsis, "\n"), true
}

// ============================================================================
// Shared: synopsis line detection
// ============================================================================

// synopsisStartsWith reports whether a plain usage line belongs to the
// given builtin: exactly the name, or the name followed by a space or an
// opening bracket. Prevents "print" from matching "printf" and "cd" from
// matching "chdir".
func synopsisStartsWith(line, name string) bool {
	return line == name ||
		strings.HasPrefix(line, name+" ") ||
		strings.HasPrefix(line, name+"[")
}
