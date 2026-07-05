package engine

import (
	"fmt"
	"strings"

	"git-hint/engine/parser"
	"git-hint/engine/provider"
)

// FindCommands retorna (matches, currentToken, error).
// currentToken é o texto que ainda está "em aberto" no nível final retornado:
// "" quando o usuário acabou de pular pra um novo nível (jump) ou digitou espaço;
// o token cru quando ainda está filtrando dentro do MESMO nível.
func FindCommands(input []string, commands map[string]parser.CommandMatch) (map[string]parser.CommandMatch, string, error) {
	// TODO: Handle quoted placeholders ("<...>")
	// Logic: If a command name is wrapped in quotes, it should be treated as a literal string
	// that still triggers dynamic expansion, but potentially avoids some of the
	// "proactive jump" logic or standard prefix filtering.
	if len(input) == 0 || input[0] == "" {
		return commands, "", nil
	}
	if commands == nil {
		return nil, "", fmt.Errorf("❌ Commands map is nil")
	}

	newCommands := make(map[string]parser.CommandMatch)
	var exactStaticName string
	hasExactStatic := false

	for name, cmd := range commands {
		if flag := provider.FlagCheck(name); flag != "" {
			completed := len(input) > 1
			if !completed {
				for _, e := range provider.Provider(flag) {
					if e.Name == input[0] || (e.MatchKey != "" && e.MatchKey == input[0]) {
						completed = true
						break
					}
				}
			}

			if completed {
				if cmd.SubCommand != nil {
					subCommands := make(map[string]parser.CommandMatch, len(cmd.SubCommand))
					for subname, subcmd := range cmd.SubCommand {
						subcmd.Name = subname
						subCommands[subname] = subcmd
					}
					if len(input) == 1 {
						return subCommands, "", nil // jump: nada digitado ainda no novo nível
					}
					return FindCommands(input[1:], subCommands)
				}
				continue // placeholder terminal, já preenchido, nada mais a sugerir
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
			subCommands := make(map[string]parser.CommandMatch, len(cmd.SubCommand))
			for subname, subcmd := range cmd.SubCommand {
				subcmd.Name = subname
				subCommands[subname] = subcmd
			}
			if len(input) == 1 {
				return subCommands, "", nil // jump
			}
			return FindCommands(input[1:], subCommands)
		}
	}

	if len(input) == 1 {
		// Último token, sem jump: newCommands ainda está sendo filtrado por ele.
		return newCommands, input[0], nil
	}
	return FindCommands(input[1:], newCommands)
}
