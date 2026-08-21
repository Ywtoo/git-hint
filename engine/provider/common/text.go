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

	seen := make(map[string]bool)
	var matches []core.CommandMatch
	for _, m := range messages {
		rest := strings.TrimSpace(strings.TrimPrefix(m, commandName))
		if rest == "" {
			continue
		}

		msgOnly := ensureQuoted(rest)
		if seen[msgOnly] {
			continue
		}
		seen[msgOnly] = true

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
		// Strip known prefix and get the next token.
		rest := strings.TrimSpace(strings.TrimPrefix(line, commandName))
		if rest == "" {
			continue
		}
		// Extract only the next token (without dragging the rest of the line).
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
		return `""`
	}

	// If already enclosed in double or single quotes, strip them first
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		s = s[1 : len(s)-1]
	} else {
		s = strings.Trim(s, "\"'")
	}

	return `"` + s + `"`
}
