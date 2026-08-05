package git

import (
	"os/exec"
	"strings"

	"git-hint/core"
)

func GittoList(command []string, mode int) []core.CommandMatch {
	output, err := exec.Command("git", command...).Output()
	if err != nil {
		return nil
	}

	var list []string
	text := string(output)

	switch mode {
	case 1:
		text = strings.TrimRight(text, "\n")
		if text != "" {
			list = strings.Split(text, "\n")
		}
	default:
		list = strings.Fields(text)
	}

	var matches []core.CommandMatch

	for _, name := range list {
		matches = append(matches, core.CommandMatch{
			Name: name,
		})
	}
	return matches
}

func RemoteProvider() []core.CommandMatch {
	return GittoList([]string{"remote"}, 0)
}

func StashProvider() []core.CommandMatch {
	return GittoList([]string{"stash", "list"}, 1)
}

func TagProvider() []core.CommandMatch {
	return GittoList([]string{"tag"}, 0)
}

func TrackedFileProvider() []core.CommandMatch {
	return GittoList([]string{"ls-files"}, 1)
}

func RefProvider() []core.CommandMatch {
	return GittoList([]string{"for-each-ref", "--format=%(refname:short)"}, 0)
}

func UpstreamProvider() []core.CommandMatch {
	return GittoList([]string{"for-each-ref", "--format=%(refname:short)", "refs/remotes"}, 0)
}

func BranchProvider() []core.CommandMatch {
	return GittoList([]string{"branch", "--format=%(refname:short)"}, 0)
}
