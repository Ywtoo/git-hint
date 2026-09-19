package common

import (
	"regexp"
	"strings"

	"git-hint/internal/core"
	"git-hint/internal/engine/history"
	"git-hint/internal/engine/tokenizer"
	"git-hint/internal/state"
)

var commitMsgRegex = regexp.MustCompile(`(?:-(?:[a-zA-Z]*m|F|C)|--message(?:=|\s+))\s*(?:"([^"]*)"?|'([^']*)'?|(\S+.*))`)

func MsgProvider() []core.CommandMatch {
	buf := state.GetBuffer()
	baseCmd, msgFlag := extractBaseCmdAndMsgFlag(buf)
	if baseCmd == "" || msgFlag == "" {
		return nil
	}

	// Search history for entries that start with the base command
	// (e.g. "git commit") and contain the message flag somewhere.
	lines, err := history.FindHistoryCommands(baseCmd)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var matches []core.CommandMatch
	for _, line := range lines {
		msg := extractMsgAfterFlag(line, msgFlag)
		if msg == "" {
			continue
		}
		// A message with an unbalanced quote (e.g. a history line broken
		// mid-string: `commit -m "feat: add Open\`) would render a suggestion
		// that itself opens a new quote context — corrupting every later list.
		if strings.Count(msg, "\"")%2 != 0 {
			continue
		}
		msgOnly := ensureQuoted(msg)
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
	commandName := extractCommandPrefix(state.GetBuffer())
	if commandName == "" {
		return nil
	}

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

// IsMsgFlag reports whether a token introduces a message argument.
// Exported for the engine's quote-bootstrap logic.
func IsMsgFlag(flag string) bool {
	return isMsgFlag(flag)
}

func isMsgFlag(flag string) bool {
	if flag == "-m" || flag == "-F" || flag == "-C" || flag == "-c" || flag == "--message" {
		return true
	}
	if strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && strings.HasSuffix(flag, "m") {
		return true
	}
	return false
}

// extractBaseCmdAndMsgFlag parses buf and returns:
//   - baseCmd: all non-flag tokens joined (e.g. "git commit")
//   - msgFlag: the last message-introducing flag seen (e.g. "-m")
//
// Returns ("", "") if no message flag is found.
func extractBaseCmdAndMsgFlag(buf string) (string, string) {
	parts := tokenizer.TokenizeBuffer(buf)

	var baseTokens []string
	lastMsgFlag := ""

	for _, p := range parts {
		if p == "" {
			continue
		}
		if isMsgFlag(p) {
			lastMsgFlag = p
			continue
		}
		if strings.HasPrefix(p, "-") {
			// Other flags: skip entirely (don't add to base, don't treat as msg flag)
			continue
		}
		baseTokens = append(baseTokens, p)
	}

	// The last token is the message currently being typed (`-m f`), not part
	// of the base command. Leaving it in made the history lookup search for
	// `git commit f...` lines, which never match, so the message suggestions
	// disappeared as soon as the user typed the first character.
	if len(baseTokens) > 0 && !isMsgFlag(parts[len(parts)-1]) {
		baseTokens = baseTokens[:len(baseTokens)-1]
	}

	if lastMsgFlag == "" {
		return "", ""
	}
	return strings.Join(baseTokens, " "), lastMsgFlag
}

// extractMsgAfterFlag scans a history line and returns the commit message text.
func extractMsgAfterFlag(line, _ string) string {
	matches := commitMsgRegex.FindStringSubmatch(line)
	if len(matches) > 0 {
		for i := 1; i < len(matches); i++ {
			if matches[i] != "" {
				return strings.TrimSpace(matches[i])
			}
		}
	}
	return ""
}

// extractCommandPrefix returns the prefix of the buffer up to (but not
// including) the last token. Used by FreeTextProvider for subcommand context.
func extractCommandPrefix(buf string) string {
	parts := tokenizer.TokenizeBuffer(buf)
	if len(parts) <= 1 {
		return ""
	}

	last := parts[len(parts)-1]
	if last == "" {
		// e.g. "git remote add "
		return strings.TrimSpace(strings.Join(parts[:len(parts)-1], " "))
	}
	if strings.HasPrefix(last, "-") {
		// e.g. "git remote add -"
		return strings.TrimSpace(strings.Join(parts, " "))
	}
	return strings.TrimSpace(strings.Join(parts[:len(parts)-1], " "))
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
