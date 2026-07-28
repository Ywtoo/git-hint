package git

import (
	"fmt"
	"os/exec"
	"strings"

	"git-hint/core"
)

func CommitProvider() []core.CommandMatch {
	output, err := exec.Command("git", "log", "-n", "10", "--format=%h|%s").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
	var matches []core.CommandMatch

	for i, line := range lines {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		hash, subject := parts[0], parts[1]

		label := "HEAD"
		if i > 0 {
			label = fmt.Sprintf("HEAD~%d", i)
		}

		matches = append(matches, core.CommandMatch{
			Name:        label,
			MatchKey:    hash,
			Description: fmt.Sprintf("(%s) %s", hash, subject),
		})
	}
	return matches
}
