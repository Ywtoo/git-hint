package common

import (
	"strings"

	"git-hint/core"
	"git-hint/engine/history"
	"git-hint/engine/tokenizer"
	"git-hint/state"
)

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
		msgOnly := ensureQuoted(m[idx:])

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

func ensureQuoted(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	hasStart := s[0] == '"'
	hasEnd := s[len(s)-1] == '"'

	if !hasStart {
		s = `"` + s
	}
	if !hasEnd {
		s = s + `"`
	}

	return s
}
