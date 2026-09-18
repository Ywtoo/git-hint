package engine

import (
	"fmt"
	"strings"

	"git-hint/internal/core"
	"git-hint/internal/engine/provider"
)

// FindCommands returns (matches, currentToken, error).
// currentToken is the text that is still "open" at the final returned level:
// "" when the user just jumped to a new level (jump) or typed a space;
// the raw token when still filtering within the SAME level.
// FindCommands returns (matches, currentToken, error).
// currentToken is the text that is still "open" at the final returned level:
// "" when the user just jumped to a new level (jump) or typed a space;
// the raw token when still filtering within the SAME level.
func FindCommands(input []string, commands map[string]core.CommandMatch) (map[string]core.CommandMatch, string, error) {
	if len(input) == 0 || input[0] == "" {
		return commands, "", nil
	}

	if commands == nil {
		return nil, "", fmt.Errorf("❌ Commands map is nil")
	}

	// 1. Try resolving nested subcommands first (e.g. "git remote add" or "git commit").
	// Flags (starting with "-") are handled by navigateCommandContext, not as exclusive subcommands.
	if !strings.HasPrefix(input[0], "-") {
		if exactCmd, ok := commands[input[0]]; ok && len(exactCmd.SubCommand) > 0 {
			subCommands := make(map[string]core.CommandMatch, len(exactCmd.SubCommand))
			for subname, subcmd := range exactCmd.SubCommand {
				subcmd.Name = subname
				subCommands[subname] = subcmd
			}

			if len(input) == 1 {
				return subCommands, "", nil
			}
			return FindCommands(input[1:], subCommands)
		}
	}

	// 2. Check if the current context has combinable flags or positional required args.
	hasFlagsOrPositional := false
	for _, cmd := range commands {
		if strings.HasPrefix(cmd.Name, "-") || provider.FlagCheck(cmd.Name) != "" {
			hasFlagsOrPositional = true
			break
		}
	}

	if hasFlagsOrPositional {
		return navigateCommandContext(input, commands)
	}

	return navigateTree(input, commands)
}

// isClosedQuoted checks if a token is wrapped in complete matching quotes: "..." or '...'
func isClosedQuoted(token string) bool {
	if len(token) >= 2 {
		if (token[0] == '"' && token[len(token)-1] == '"') ||
			(token[0] == '\'' && token[len(token)-1] == '\'') {
			return true
		}
	}
	return false
}

// navigateCommandContext handles the state of an active command level where
// flags and positional arguments can be given in flexible order.
//
// Categories:
//  1. Required-Now: An active flag expecting an argument (e.g. -m <msg>).
//     Only the expected argument is suggested until fulfilled.
//  2. Required (Positional): Positional placeholders (e.g. <branch>, <commit>).
//     Always visible and prioritized when not actively typing a flag prefix ("-").
//  3. Optional Flags: Available flags that have not been consumed yet.
func navigateCommandContext(input []string, commands map[string]core.CommandMatch) (map[string]core.CommandMatch, string, error) {
	usedFlags := make(map[string]bool)
	var pendingFlag *core.CommandMatch

	// Process all completed tokens (everything except the last one being typed)
	for i := 0; i < len(input)-1; i++ {
		token := input[i]
		if token == "" {
			continue
		}

		if pendingFlag != nil {
			// Current token fulfills the required-now argument
			pendingFlag = nil
			continue
		}

		if cmd, ok := commands[token]; ok && strings.HasPrefix(token, "-") {
			usedFlags[token] = true
			if requiresArgument(cmd) {
				cp := cmd
				pendingFlag = &cp
			}
			continue
		}
	}

	lastToken := input[len(input)-1]

	// Category: Required-Now
	// If a flag was just specified and requires an argument, suggest ONLY that argument.
	if pendingFlag != nil {
		if isClosedQuoted(lastToken) {
			usedFlags[pendingFlag.Name] = true
			pendingFlag = nil
		} else {
			requiredMatches := extractRequiredChildren(*pendingFlag)
			if len(requiredMatches) > 0 {
				filtered := filterByToken(requiredMatches, lastToken)
				return filtered, lastToken, nil
			}
		}
	}

	// If user is currently typing a flag (e.g. "-"), filter to available flags
	if strings.HasPrefix(lastToken, "-") {
		flagMatches := make(map[string]core.CommandMatch)
		for name, cmd := range commands {
			if strings.HasPrefix(name, "-") && !usedFlags[name] {
				if strings.HasPrefix(name, lastToken) {
					flagMatches[name] = cmd
				}
			}
		}

		// An exact flag is complete even when longer flags share its prefix
		// (e.g. -a and -am). Advance past the exact token; the longer flag
		// remains available on the next level.
		if exactCmd, ok := flagMatches[lastToken]; ok {
			usedFlags[lastToken] = true
			if requiresArgument(exactCmd) {
				requiredMatches := extractRequiredChildren(exactCmd)
				if len(requiredMatches) > 0 {
					return requiredMatches, "", nil
				}
			} else {
				matches := make(map[string]core.CommandMatch)
				for name, cmd := range commands {
					if strings.HasPrefix(name, "-") && usedFlags[name] {
						continue
					}
					matches[name] = cmd
				}
				return matches, "", nil
			}
		}

		return flagMatches, lastToken, nil
	}

	// If the last token is a completed argument (closed quote), suggest next tokens with currentToken = ""
	tokenForPrefix := lastToken
	if isClosedQuoted(lastToken) {
		tokenForPrefix = ""
	}

	// Otherwise, combine:
	// 1. Required positional arguments (<branch>, <commit>, etc.)
	// 2. Unused flags
	matches := make(map[string]core.CommandMatch)
	for name, cmd := range commands {
		isFlag := strings.HasPrefix(name, "-")
		if isFlag && usedFlags[name] {
			continue
		}

		if strings.HasPrefix(name, tokenForPrefix) || provider.FlagCheck(name) != "" {
			matches[name] = cmd
		}
	}

	if isClosedQuoted(lastToken) {
		return matches, "", nil
	}

	return matches, lastToken, nil
}

// requiresArgument determines if a flag needs an immediate argument.
func requiresArgument(cmd core.CommandMatch) bool {
	if len(cmd.Requires) > 0 {
		return true
	}
	for subName := range cmd.SubCommand {
		if provider.FlagCheck(subName) != "" || strings.HasPrefix(subName, "<") {
			return true
		}
	}
	return false
}

// extractRequiredChildren returns the matches for a flag's required arguments.
// Every entry has Name set to its map key so that buildSuggestionList can
// identify dynamic placeholders (e.g. <message>) via FlagCheck(name).
func extractRequiredChildren(flag core.CommandMatch) map[string]core.CommandMatch {
	out := make(map[string]core.CommandMatch)
	for subName, subCmd := range flag.SubCommand {
		subCmd.Name = subName
		out[subName] = subCmd
	}
	for _, req := range flag.Requires {
		if _, exists := out[req]; !exists {
			out[req] = core.CommandMatch{Name: req}
		}
	}
	return out
}

func filterByToken(items map[string]core.CommandMatch, token string) map[string]core.CommandMatch {
	if token == "" {
		return items
	}
	filtered := make(map[string]core.CommandMatch)
	for name, item := range items {
		if provider.FlagCheck(name) != "" || strings.HasPrefix(name, token) {
			filtered[name] = item
		}
	}
	return filtered
}

// navigateTree is the standard fallback navigation for nested subcommands.
func navigateTree(input []string, commands map[string]core.CommandMatch) (map[string]core.CommandMatch, string, error) {
	newCommands := make(map[string]core.CommandMatch)
	var exactStaticName string
	hasExactStatic := false

	for name, cmd := range commands {
		if flag := provider.FlagCheck(name); flag != "" {
			completed := false
			for _, e := range provider.Provider(flag) {
				if e.Name == input[0] || (e.MatchKey != "" && e.MatchKey == input[0]) {
					completed = true
					break
				}
			}

			if completed {
				if cmd.SubCommand != nil {
					subCommands := make(map[string]core.CommandMatch, len(cmd.SubCommand))
					for subname, subcmd := range cmd.SubCommand {
						subcmd.Name = subname
						subCommands[subname] = subcmd
					}
					if len(input) == 1 {
						return subCommands, "", nil
					}
					return FindCommands(input[1:], subCommands)
				}
				continue
			}

			newCommands[name] = cmd
			continue
		}

		if strings.HasPrefix(name, input[0]) {
			newCommands[name] = cmd
			if name == input[0] {
				exactStaticName = name
				hasExactStatic = true
			}
		}
	}

	if hasExactStatic && len(newCommands) == 1 {
		cmd := newCommands[exactStaticName]
		if cmd.SubCommand != nil {
			subCommands := make(map[string]core.CommandMatch, len(cmd.SubCommand))
			for subname, subcmd := range cmd.SubCommand {
				subcmd.Name = subname
				subCommands[subname] = subcmd
			}
			if len(input) == 1 {
				return subCommands, "", nil
			}
			return FindCommands(input[1:], subCommands)
		}
	}

	if len(input) == 1 {
		return newCommands, input[0], nil
	}
	return FindCommands(input[1:], newCommands)
}
