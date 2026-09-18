package git

import (
	"os/exec"
	"strings"

	"git-hint/internal/core"
)

// TODO: Implement ConflictProvider
// This provider should identify files currently in a conflicted state.
// Command: 'git status --porcelain'
// Logic: Filter for lines starting with 'UU' (Unmerged).
func ConflictProvider() []core.CommandMatch {
	out, err := exec.Command("git", "status", "--porcelain=v1").Output()
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var matches []core.CommandMatch
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 3 || !strings.Contains("U", line[:2]) {
			continue
		}
		name := strings.TrimSpace(line[3:])
		if i := strings.Index(name, " -> "); i >= 0 {
			name = name[i+4:]
		}
		if name != "" && !seen[name] {
			seen[name] = true
			matches = append(matches, core.CommandMatch{Name: name})
		}
	}
	return matches
}
