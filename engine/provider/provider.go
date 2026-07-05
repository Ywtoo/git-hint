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

func FlagCheck(flag string) string {
	if len(flag) >= 2 && flag[0] == '<' && flag[len(flag)-1] == '>' {
		return flag[1 : len(flag)-1]
	}
	return ""
}

// TODO: Implement specialized providers:
// 1. BranchProvider: Improve logic to handle "last used" branch for better ranking.
// 2. CommitProvider: Integrate with history ranking instead of just returning top 10.
// 3. RemoteProvider: Expand to handle remote-tracking branches.
// 4. MsgProvider: Extract common commit messages from .zsh_history.
// 5. StashProvider: Implement logic to split 'stash@{n}' (Name) from the description (everything after ':').
// 6. AuthorProvider: Run 'git log --format=%an' and deduplicate.
// 7. ConfigProvider: Static list of common git config keys (user.name, core.editor, etc).
// 8. FileProviders (Context-Aware):
//   - <file-staged>: 'git diff --name-only --cached'
//   - <file-modified>: 'git diff --name-only'
//   - <file-untracked>: 'git ls-files --others --exclude-standard'
//
// 9. ConflictProvider: Parse 'git status --porcelain' for 'UU' markers.
func Provider(flag string) []parser.CommandMatch {
	cache := state.LoadCache()

	cacheKey := flag
	if flag == "msg" {
		// msg depende do comando sendo completado (ex: "git commit"),
		// não só do nome do placeholder. Sem isso, o cache trava no
		// primeiro resultado (mesmo vazio) e nunca mais atualiza.
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
