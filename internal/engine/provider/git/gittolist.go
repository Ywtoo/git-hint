package git

import (
	"os/exec"
	"strings"

	"git-hint/internal/core"
)

func gitLines(command []string) []core.CommandMatch {
	output, err := exec.Command("git", command...).Output()
	if err != nil {
		return nil
	}

	var list []string
	text := string(output)

	text = strings.TrimRight(text, "\n")
	if text != "" {
		list = strings.Split(text, "\n")
	}

	var matches []core.CommandMatch

	for _, name := range list {
		matches = append(matches, core.CommandMatch{
			Name: name,
		})
	}
	return matches
}

func gitFields(command []string) []core.CommandMatch {
	output, err := exec.Command("git", command...).Output()
	if err != nil {
		return nil
	}
	var matches []core.CommandMatch
	for _, name := range strings.Fields(string(output)) {
		matches = append(matches, core.CommandMatch{Name: name})
	}
	return matches
}

func RemoteProvider() []core.CommandMatch {
	return gitFields([]string{"remote"})
}

func StashProvider() []core.CommandMatch {
	return gitLines([]string{"stash", "list"})
}

func TagProvider() []core.CommandMatch {
	return gitFields([]string{"tag"})
}

func TrackedFileProvider() []core.CommandMatch {
	return gitLines([]string{"ls-files"})
}

func RefProvider() []core.CommandMatch {
	return gitFields([]string{"for-each-ref", "--format=%(refname:short)"})
}

func UpstreamProvider() []core.CommandMatch {
	return gitFields([]string{"for-each-ref", "--format=%(refname:short)", "refs/remotes"})
}

func BranchProvider() []core.CommandMatch {
	return gitFields([]string{"branch", "--format=%(refname:short)"})
}
