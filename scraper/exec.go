package scraper

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const helpTimeout = 5 * time.Second

// helpFlags are tried in order until one produces output that actually
// looks like help text. Not every CLI treats "-h" as help — GNU and BSD
// coreutils (ls, cp, mv...) reserve "-h" for "human-readable sizes" and
// only respond to real help on "--help", so relying on "-h" alone silently
// produces empty trees for a huge chunk of common commands.
var helpFlags = []string{"--help", "-h", "help"}
var gitHelpFlags = []string{"-h", "--help", "help"}

// looksLikeHelp is a cheap heuristic to reject output that technically
// came back without error but clearly isn't a help page — e.g. "ls -h"
// silently listing the current directory instead of failing.
var sectionHeaderRe = regexp.MustCompile(`(?m)^[A-Za-z][A-Za-z /]*Commands\s*$`)

func looksLikeHelp(output string) bool {
	if strings.TrimSpace(output) == "" {
		return false
	}
	lower := strings.ToLower(output)
	return strings.Contains(lower, "usage:") ||
		strings.Contains(lower, "options:") ||
		strings.Contains(lower, "commands:") ||
		strings.Contains(lower, "flags:") ||
		sectionHeaderRe.MatchString(output)
}

// runWithFlag runs `<path...> <flag>` and returns its combined output.
// We ignore the exec error itself (many CLIs exit non-zero on help) and
// only treat a timeout as a real failure — everything else is left for
// the caller to judge via looksLikeHelp.
func runWithFlag(path []string, flag string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), helpTimeout)
	defer cancel()

	args := append(append([]string{}, path[1:]...), flag)
	cmd := exec.CommandContext(ctx, path[0], args...)
	out, _ := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timeout running %s %s", strings.Join(path, " "), flag)
	}
	return string(out), nil
}

// executeHelp tries each flag in helpFlags in order, returning the first
// output that looks like real help text. If none look right, it falls
// back to returning whatever the last attempt produced (better than
// nothing for CLIs with unusual but still parseable help formats), unless
// a real timeout occurred, which is propagated as an error.
func executeHelp(path []string) (string, error) {
	flags := helpFlags
	if filepath.Base(path[0]) == "git" {
		flags = gitHelpFlags
	}

	var lastOut string
	for _, flag := range flags {
		out, err := runWithFlag(path, flag)
		if err != nil {
			return "", err
		}
		if looksLikeHelp(out) {
			return out, nil
		}
		if lastOut == "" {
			lastOut = out
		}
	}

	return lastOut, nil
}

// executeRootHelp tries `<binary> help -a` first (richer root listing on
// some CLIs), falling back to executeHelp's cascade if that produced
// nothing useful.
func executeRootHelp(binary string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), helpTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "help", "-a")
	out, err := cmd.CombinedOutput()
	if err == nil && looksLikeHelp(string(out)) {
		return string(out), nil
	}

	return executeHelp([]string{binary})
}
