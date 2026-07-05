package engine

import (
	"fmt"
	"strings"

	"git-hint/engine/parser"
	"git-hint/engine/provider"
)

func FindCommands(input []string, commands map[string]parser.CommandMatch) (map[string]parser.CommandMatch, error) {
	if len(input) == 0 || input[0] == "" {
		return commands, nil
	}
	if commands == nil {
		return nil, fmt.Errorf("❌ Commands map is nil")
	}

	newCommands := make(map[string]parser.CommandMatch)
	var exactStaticName string
	hasExactStatic := false

	for name, cmd := range commands {
		if provider.FlagCheck(name) != "" {
			// Placeholder dinâmico (<remote>, <branch>, etc).
			// Se AINDA há input depois deste token, significa que o usuário
			// já "preencheu" o valor (ex: digitou "origin" e deu espaço).
			// Nesse caso avançamos para o SubCommand do placeholder.
			if len(input) > 1 {
				if cmd.SubCommand != nil {
					subCommands := make(map[string]parser.CommandMatch, len(cmd.SubCommand))
					for subname, subcmd := range cmd.SubCommand {
						subcmd.Name = subname
						subCommands[subname] = subcmd
					}
					return FindCommands(input[1:], subCommands)
				}
				// Placeholder sem próximo nível: não há mais nada a sugerir aqui.
				continue
			}
			// Ainda é o último token (usuário digitando/filtrando o valor):
			// mantemos como candidato para expandir via provider.
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

	// Auto jump para comando estático com match exato e subcomandos.
	if hasExactStatic {
		cmd := newCommands[exactStaticName]
		if cmd.SubCommand != nil {
			subCommands := make(map[string]parser.CommandMatch, len(cmd.SubCommand))
			for subname, subcmd := range cmd.SubCommand {
				subcmd.Name = subname
				subCommands[subname] = subcmd
			}
			if len(input) == 1 {
				return subCommands, nil
			}
			return FindCommands(input[1:], subCommands)
		}
	}

	return FindCommands(input[1:], newCommands)
}
