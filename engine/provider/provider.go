package provider

import (
	"git-hint/engine/parser"
	"os/exec"
	"strings"
)

func FlagCheck(flag string) string {
	if len(flag) >= 2 && flag[0] == '<' && flag[len(flag)-1] == '>' {
		return flag[1 : len(flag)-1]
	}
	return ""
}

// TODO: BranchProvider, CommitProvider, RemoteProvider, MsgProvider
func Provider(flag string) []parser.CommandMatch {
	cache := LoadCache()

	if cache.CurrentPlaceholder == flag {
		return cache.ExpandedList
	}

	var results []parser.CommandMatch
	switch flag {
	case "branch":
		results = GittoList([]string{"branch", "--format=%(refname:short)"}, 0)
	case "commit":
		results = GittoList([]string{"log", "-n", "10", "--format=%h"}, 1)
	case "remote":
		results = GittoList([]string{"remote"}, 0)
	case "msg":
		results = nil //MsgProvider()
	case "stash":
		results = GittoList([]string{"stash", "list"}, 1)
	case "tag":
		results = GittoList([]string{"tag"}, 0)
	default:
		results = nil
	}

	cache.CurrentPlaceholder = flag
	cache.ExpandedList = results
	SaveCache(cache)

	return results
}

func GittoList(command []string, mode int) []parser.CommandMatch {
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

	var matches []parser.CommandMatch

	for _, name := range list {
		matches = append(matches, parser.CommandMatch{
			Name: name,
		})
	}
	return matches
}
