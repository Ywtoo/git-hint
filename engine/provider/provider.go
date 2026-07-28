package provider

import (
	"os/exec"
	"strings"

	"git-hint/core"
	"git-hint/engine/history"
	"git-hint/engine/provider/git"
	"git-hint/engine/tokenizer"
	"git-hint/state"
)

// SupportedFlags é o conjunto de flags que têm um provider real implementado
// no switch de Provider(). Qualquer placeholder não listado aqui cai em
// default → nil (o usuário digita o valor manualmente).
//
// Mantenha esta variável sincronizada com os cases do switch em Provider().
// Providers maps placeholder names to completion providers.
//
// Only placeholders listed here have an implemented provider.
// Any other placeholder falls back to the default behavior,
// allowing the user to type the value manually.
var Providers = map[string]func() []core.CommandMatch{

	// ---------------------------------------------------------------------
	// Git references
	// ---------------------------------------------------------------------

	"branch":     branchProvider,
	"old-branch": branchProvider,

	"commit":     commitProvider,
	"commit-ish": commitProvider,
	"tree-ish":   commitProvider,
	"head":       commitProvider,

	"ref":     refProvider,
	"refname": refProvider,

	"upstream": upstreamProvider,

	"remote": remoteProvider,
	"stash":  stashProvider,
	"tag":    tagProvider,

	// ---------------------------------------------------------------------
	// Files
	// ---------------------------------------------------------------------

	"file": trackedFileProvider,
	"path": trackedFileProvider,

	// ---------------------------------------------------------------------
	// Message
	// ---------------------------------------------------------------------

	// Free text with automatic quoting.
	"msg":     MsgProvider,
	"message": MsgProvider,

	// ---------------------------------------------------------------------
	// Free text
	// ---------------------------------------------------------------------

	"url":         freeTextProvider,
	"name":        freeTextProvider,
	"new-branch":  freeTextProvider,
	"branch-name": freeTextProvider,
	"author":      freeTextProvider,
}

func FlagCheck(flag string) string {
	if len(flag) >= 2 && flag[0] == '<' && flag[len(flag)-1] == '>' {
		return flag[1 : len(flag)-1]
	}
	return ""
}

func Provider(flag string) []core.CommandMatch {
	cache := state.LoadCache()

	cacheKey := flag
	if flag == "msg" || flag == "name" || flag == "url" {
		// These flags depend on the full command context (e.g. "git remote add"),
		// not just the placeholder name — so we key the cache by context too.
		parts := tokenizer.TokenizeBuffer(state.GetBuffer())
		if len(parts) > 1 {
			cacheKey = flag + "|" + strings.Join(parts[:len(parts)-1], " ")
		}
	}

	if cache.CurrentPlaceholder == cacheKey {
		return cache.ExpandedList
	}

	var results []core.CommandMatch

	fn, ok := Providers[flag]
	if ok {
		results = fn()
	}

	cache.CurrentPlaceholder = cacheKey
	cache.ExpandedList = results
	state.SaveCache(cache)

	return results
}

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

func MsgProvider() []core.CommandMatch {
	parts := tokenizer.TokenizeBuffer(state.GetBuffer())
	if len(parts) <= 1 {
		return nil
	}
	commandName := strings.Join(parts[:len(parts)-1], " ")

	messages, err := history.FindHistoryCommands(commandName)
	if err != nil {
		return nil
	}

	var matches []core.CommandMatch
	for _, m := range messages {
		idx := strings.Index(m, `"`)
		if idx == -1 {
			continue
		}
		msgOnly := m[idx:]

		matches = append(matches, core.CommandMatch{
			Name: msgOnly,
		})
	}
	return matches
}

func FreeTextProvider() []core.CommandMatch {
	parts := tokenizer.TokenizeBuffer(state.GetBuffer())
	if len(parts) <= 1 {
		return nil
	}
	commandName := strings.Join(parts[:len(parts)-1], " ")

	lines, err := history.FindHistoryCommands(commandName)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var matches []core.CommandMatch
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
		matches = append(matches, core.CommandMatch{
			Name: token,
		})
	}
	return matches
}

func branchProvider() []core.CommandMatch {
	return GittoList([]string{"branch", "--format=%(refname:short)"}, 0)
}

func commitProvider() []core.CommandMatch {
	return git.CommitProvider()
}

func remoteProvider() []core.CommandMatch {
	return GittoList([]string{"remote"}, 0)
}

func stashProvider() []core.CommandMatch {
	return GittoList([]string{"stash", "list"}, 1)
}

func tagProvider() []core.CommandMatch {
	return GittoList([]string{"tag"}, 0)
}

func trackedFileProvider() []core.CommandMatch {
	return GittoList([]string{"ls-files"}, 1)
}

func refProvider() []core.CommandMatch {
	return GittoList([]string{"for-each-ref", "--format=%(refname:short)"}, 0)
}

func upstreamProvider() []core.CommandMatch {
	return GittoList([]string{"for-each-ref", "--format=%(refname:short)", "refs/remotes"}, 0)
}

func freeTextProvider() []core.CommandMatch {
	return FreeTextProvider()
}
