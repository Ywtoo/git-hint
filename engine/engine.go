package engine

import (
	"fmt"
	"strings"

	"git-hint/engine/parser"
	"git-hint/engine/ranking"
	"git-hint/engine/registry"
)

func Suggestions(input string) ([]parser.CommandMatch, error) {
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

		list, err := ranking.RankSuggestions(commandName, list)
		if err != nil {
			return nil, fmt.Errorf("❌ Erro ao ordenar comandos: %v\n", err)
		}
		return list, nil
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
	jumped := false
	if len(input) >= 1 && len(newCommands) == 1 {
		for name, cmd := range newCommands {
			if name == input[0] {
				subCommands := make(map[string]parser.CommandMatch)
				for subname, subcmd := range cmd.SubCommand {
					subcmd.Name = subname
					subCommands[subname] = subcmd
				}
				newCommands = subCommands
				jumped = true
			}
		}
	}

	if jumped && len(input) == 1 {
		return newCommands, nil
	}

	return FindCommands(input[1:], newCommands)
}

func CompleteBuffer(buffer string, selectedIndex int) string {
	suggestions, err := Suggestions(buffer)
	if err != nil || selectedIndex < 0 || selectedIndex >= len(suggestions) {
		return buffer
	}

	selectedSuggestion := suggestions[selectedIndex].Name

	// 1. Extract the last token from the buffer
	var lastToken string
	if !strings.HasSuffix(buffer, " ") {
		parts := strings.Fields(buffer)
		if len(parts) > 0 {
			lastToken = parts[len(parts)-1]
		}
	}

	// 2. Check if the last token is a prefix of any current suggestion
	isPrefix := false
	if lastToken != "" {
		for _, s := range suggestions {
			if strings.HasPrefix(s.Name, lastToken) {
				isPrefix = true
				break
			}
		}
	}

	// 3. Decide: Replace or Append
	if isPrefix {
		// Replace the last token with the selected suggestion
		idx := strings.LastIndex(buffer, lastToken)
		if idx == -1 {
			return selectedSuggestion
		}
		return buffer[:idx] + selectedSuggestion
	}

	// Append the suggestion
	if strings.HasSuffix(buffer, " ") {
		return buffer + selectedSuggestion
	}
	return buffer + " " + selectedSuggestion
}
