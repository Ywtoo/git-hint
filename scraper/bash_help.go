package scraper

import (
	"os/exec"
	"strings"
)

// ============================================================================
// Source 1: bash's help builtin (primary)
//
// bash is widely available on Linux and macOS. Its help output is clean
// and structured — a synopsis line followed by description and an
// Options section with per-flag descriptions.
//
//   cd: cd [-L|[-P [-e]] [-@]] [dir]
//       Change the shell working directory.
//       ...
//       Options:
//         -L    force symbolic links to be followed
//         -P    use the physical directory structure
// ============================================================================

// execBashHelp runs "bash -c 'help <name>'" and returns its output.
// Extracted for testability.
func execBashHelp(name string) (string, error) {
	out, err := exec.Command("bash", "-c", "help "+name).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// builtinHelpFromBashHelp extracts the synopsis from bash's help builtin.
func builtinHelpFromBashHelp(name string) (string, bool) {
	output, err := execBashHelp(name)
	if err != nil || len(output) == 0 {
		return "", false
	}
	return parseBashHelpOutput(name, output)
}

// parseBashHelpOutput extracts the synopsis from bash's help output.
func parseBashHelpOutput(name, output string) (string, bool) {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return "", false
	}

	// First line: "name: name [flags] [args]"
	firstLine := strings.TrimSpace(lines[0])
	synopsis := extractBashHelpSynopsis(firstLine, name)
	if synopsis == "" {
		return "", false
	}

	return synopsis, true
}

// extractBashHelpSynopsis pulls the synopsis from the first line of
// bash help output. The line has the form "name: name [flags] [args]"
// and we strip the "name: " prefix to get just the usage part.
func extractBashHelpSynopsis(line, name string) string {
	// Try "name: rest" format
	prefix := name + ": "
	if strings.HasPrefix(line, prefix) {
		return strings.TrimSpace(line[len(prefix):])
	}
	// Fallback: the line might just be the synopsis without the prefix
	if synopsisStartsWith(line, name) {
		return line
	}
	return ""
}
