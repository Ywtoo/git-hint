package provider

import (
	"fmt"
	"git-hint/engine/history"
	"git-hint/engine/parser"
	"git-hint/engine/state"
	"git-hint/engine/tokenizer"
	"os/exec"
	"strings"
)

// SupportedFlags é o conjunto de flags que têm um provider real implementado
// no switch de Provider(). Qualquer placeholder não listado aqui cai em
// default → nil (o usuário digita o valor manualmente).
//
// Mantenha esta variável sincronizada com os cases do switch em Provider().
var SupportedFlags = map[string]bool{
	"branch": true,
	"commit": true,
	"remote": true,
	"msg":    true,
	"name":   true,
	"stash":  true,
	"tag":    true,
	"url":    true,
}

func FlagCheck(flag string) string {
	if len(flag) >= 2 && flag[0] == '<' && flag[len(flag)-1] == '>' {
		return flag[1 : len(flag)-1]
	}
	return ""
}

func Provider(flag string) []parser.CommandMatch {
	cache := state.LoadCache()

	cacheKey := flag
	if flag == "msg" || flag == "name" || flag == "url" {
		// These flags depend on the full command context (e.g. "git remote add"),
		// not just the placeholder name — so we key the cache by context too.
		parts := tokenizer.TokenizeBuffer(state.Buffer)
		if len(parts) > 1 {
			cacheKey = flag + "|" + strings.Join(parts[:len(parts)-1], " ")
		}
	}

	if cache.CurrentPlaceholder == cacheKey {
		return cache.ExpandedList
	}

	var results []parser.CommandMatch
	switch flag {
	case "branch":
		results = GittoList([]string{"branch", "--format=%(refname:short)"}, 0)
	case "commit":
		results = CommitProvider()
	case "remote":
		results = GittoList([]string{"remote"}, 0)
	case "msg":
		results = MsgProvider()
	case "name", "url":
		results = FreeTextProvider()
	case "stash":
		results = GittoList([]string{"stash", "list"}, 1)
	case "tag":
		results = GittoList([]string{"tag"}, 0)
	default:
		results = nil
	}

	cache.CurrentPlaceholder = cacheKey
	cache.ExpandedList = results
	state.SaveCache(cache)

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

func MsgProvider() []parser.CommandMatch {
	parts := tokenizer.TokenizeBuffer(state.Buffer)
	if len(parts) <= 1 {
		return nil
	}
	commandName := strings.Join(parts[:len(parts)-1], " ")

	messages, err := history.FindHistoryCommands(commandName)
	if err != nil {
		return nil
	}

	var matches []parser.CommandMatch
	for _, m := range messages {
		idx := strings.Index(m, `"`)
		if idx == -1 {
			continue
		}
		msgOnly := m[idx:]

		matches = append(matches, parser.CommandMatch{
			Name: msgOnly,
		})
	}
	return matches
}

func FreeTextProvider() []parser.CommandMatch {
	parts := tokenizer.TokenizeBuffer(state.Buffer)
	if len(parts) <= 1 {
		return nil
	}
	commandName := strings.Join(parts[:len(parts)-1], " ")

	lines, err := history.FindHistoryCommands(commandName)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var matches []parser.CommandMatch
	for _, line := range lines {
		// Remove o prefixo conhecido e pega o próximo token.
		rest := strings.TrimSpace(strings.TrimPrefix(line, commandName))
		if rest == "" {
			continue
		}
		// Pega apenas o próximo token (sem arrastar o resto da linha).
		token := strings.Fields(rest)[0]
		if token == "" || seen[token] {
			continue
		}
		seen[token] = true
		matches = append(matches, parser.CommandMatch{
			Name: token,
		})
	}
	return matches
}

func CommitProvider() []parser.CommandMatch {
	output, err := exec.Command("git", "log", "-n", "10", "--format=%h|%s").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
	var matches []parser.CommandMatch

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

		matches = append(matches, parser.CommandMatch{
			Name:        label,
			MatchKey:    hash,
			Description: fmt.Sprintf("(%s) %s", hash, subject),
		})
	}
	return matches
}
