package engine

import (
	"fmt"
	"strings"

	"git-hint/engine/engine/parser"
	"git-hint/engine/engine/ranking"
	"git-hint/engine/engine/registry"
)

func Execute(input string) ([]parser.CommandMatch, error) {
	parts := strings.Split(input, " ")
	commandName := parts[0]
	remainingInput := parts[1:]
	var list []parser.CommandMatch

	//TODO: SQLite
	//File path resolve
	filePath, err := registry.ResolveCommandPath(commandName)
	if err != nil {
		return nil, fmt.Errorf("❌ Erro ao encontrar o comando '%s': %v", commandName, err)
	}

	if filePath == "" {
		return nil, nil
	}

	//Find comands
	commands, err := parser.ParseCommand(filePath)
	if err != nil {
		return nil, fmt.Errorf("❌ Erro ao ler o arquivo: %v\n", err)
	}

	if commands == nil {
		return nil, nil
	}

	matches, err := FindCommands(remainingInput, commands)
	if err != nil {
		return nil, fmt.Errorf("❌ Erro ao encontrar comandos: %v\n", err)
	}

	if matches != nil {
		for _, cmd := range matches {
			list = append(list, cmd)
		}
		return ranking.RankSuggestions(list), nil
	} else {
		return nil, nil
	}
}

func FindCommands(input []string, commands map[string]parser.CommandMatch) (map[string]parser.CommandMatch, error) {
	newCommands := make(map[string]parser.CommandMatch)

	// Case 1
	if len(input) == 0 || input[0] == "" {
		return commands, nil
	}
	if commands == nil {
		return nil, fmt.Errorf("❌ Commands map is nil")
	}

	// 1. Filter
	for name, cmd := range commands {
		if strings.HasPrefix(name, input[0]) {
			newCommands[name] = cmd
		}
	}

	// 2. Auto Jump
	jumped := false // <--- Inicializa a variável aqui para o compilador não reclamar!
	if len(input) >= 1 && len(newCommands) == 1 {
		for name, cmd := range newCommands {
			if name == input[0] {
				subCommands := make(map[string]parser.CommandMatch)
				for subname, subcmd := range cmd.SubCommand {
					subcmd.Name = subname // <-- Consertou o bug do teste!
					subCommands[subname] = subcmd
				}
				newCommands = subCommands
				jumped = true
			}
		}
	}

	// Se você acabou de saltar para as subflags e o input do usuário terminou aqui,
	// pare imediatamente e entregue as opções!
	if jumped && len(input) == 1 {
		return newCommands, nil
	}

	return FindCommands(input[1:], newCommands)
}
