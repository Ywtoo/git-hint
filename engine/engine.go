package engine

import (
	"fmt"
	"strings"

	"git-hint/engine/parser"
	"git-hint/engine/provider"
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
		var dynamicFlag string
		for name := range matches {
			if flag := provider.FlagCheck(name); flag != "" {
				dynamicFlag = flag
				break
			}
		}

		if dynamicFlag != "" {
			expanded := provider.Provider(dynamicFlag)
			parts := strings.Split(input, " ")
			lastToken := parts[len(parts)-1]

			var filtered []parser.CommandMatch
			for _, s := range expanded {
				if s.Name != "" && strings.HasPrefix(s.Name, lastToken) {
					filtered = append(filtered, s)
				}
			}
			return filtered, nil
		}

		for _, cmd := range matches {
			list = append(list, cmd)
		}
		list, err := ranking.RankSuggestions(input, list)
		if err != nil {
			return nil, fmt.Errorf("❌ Erro ao ordenar comandos: %v\n", err)
		}
		return list, nil
	}
	return nil, nil
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
