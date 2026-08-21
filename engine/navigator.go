package engine

import (
	"fmt"
	"strings"

	"git-hint/core"
	"git-hint/engine/provider"
)

// FindCommands returns (matches, currentToken, error).
// currentToken is the text that is still "open" at the final returned level:
// "" when the user just jumped to a new level (jump) or typed a space;
// the raw token when still filtering within the SAME level.
func FindCommands(input []string, commands map[string]core.CommandMatch) (map[string]core.CommandMatch, string, error) {
	// Logic: If a command name is wrapped in quotes, it should be treated as a literal string
	// that still triggers dynamic expansion, but potentially avoids some of the
	// "proactive jump" logic or standard prefix filtering.
	if len(input) == 0 || input[0] == "" {
		return commands, "", nil
	}

	if commands == nil {
		return nil, "", fmt.Errorf("❌ Commands map is nil")
	}

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
						return subCommands, "", nil // jump: 100% match, show next level suggestions
					}
					return FindCommands(input[1:], subCommands)
				}
				continue // terminal placeholder, already filled, nothing more to suggest
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

	if hasExactStatic {
		cmd := newCommands[exactStaticName]
		if cmd.SubCommand != nil {
			subCommands := make(map[string]core.CommandMatch, len(cmd.SubCommand))
			for subname, subcmd := range cmd.SubCommand {
				subcmd.Name = subname
				subCommands[subname] = subcmd
			}
			if len(input) == 1 {
				// If the user typed the exact command name and the buffer ends with a space,
				// or if there is already a following token, jump directly to the subcommand!
				return subCommands, "", nil
			}
			return FindCommands(input[1:], subCommands)
		}
	}

	if len(input) == 1 {
		// Last token, no jump: newCommands is still being filtered by it.
		return newCommands, input[0], nil
	}
	return FindCommands(input[1:], newCommands)
}
