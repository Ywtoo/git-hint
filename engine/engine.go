package engine

import (
	"fmt"
	"strings"

	"git-hint/engine/parser"
	"git-hint/engine/provider"
	"git-hint/engine/ranking"
	"git-hint/engine/registry"
	"git-hint/engine/tokenizer"
)

var noDescriptionFlags = map[string]bool{
	"msg": true,
}

var skipRankingFlags = map[string]bool{
	"commit": true,
	"msg":    true,
}

func Suggestions(input string) ([]parser.CommandMatch, string, error) {
	parts := tokenizer.TokenizeBuffer(input)
	if len(parts) == 0 {
		return nil, "", nil
	}
	commandName := parts[0]
	remainingInput := parts[1:]
	var list []parser.CommandMatch

	data, err := registry.ResolveCommandData(commandName)
	if err != nil {
		return nil, "", fmt.Errorf("❌ Error finding command '%s': %v", commandName, err)
	}
	if data == nil {
		return nil, "", nil
	}

	commands, err := parser.ParseCommand(data)
	if err != nil {
		return nil, "", fmt.Errorf("❌ Error reading file: %v\n", err)
	}
	if commands == nil {
		return nil, "", nil
	}

	matches, currentToken, err := FindCommands(remainingInput, commands)
	if err != nil {
		return nil, "", fmt.Errorf("❌ Error finding commands: %v\n", err)
	}
	if matches == nil {
		return nil, "", nil
	}

	var dynamicFlagSeen string // used only to decide whether to skip ranking

	for name, cmd := range matches {
		if flag := provider.FlagCheck(name); flag != "" {
			dynamicFlagSeen = flag
			expanded := provider.Provider(flag)

			for _, s := range expanded {
				if s.Name == "" || !strings.HasPrefix(s.Name, currentToken) {
					continue
				}

				switch {
				case noDescriptionFlags[flag]:
					s.Description = "" // msg: sem comentário, o valor já é a info

				case skipRankingFlags[flag]:
					// commit: keep the specific Description that the provider already built
					// (hash + subject), always displayed — does not inherit from parent,
					// not conditioned to "only when selected".

				default:
					// branch, remote, tag, stash: shared description from parent,
					// displayed only on the selected line.
					s.Description = cmd.Description
					s.ShowOnlyWhenSelected = true
				}

				list = append(list, s)
			}
		} else {
			// static: Description already comes from the cmd itself, always displayed.
			list = append(list, cmd)
		}
	}

	if len(list) == 0 {
		return nil, currentToken, nil
	}

	if !skipRankingFlags[dynamicFlagSeen] {
		list, err = ranking.RankSuggestions(input, list)
		if err != nil {
			return nil, "", fmt.Errorf("❌ Erro ao ordenar comandos: %v\n", err)
		}
	}

	return list, currentToken, nil
}

func CompleteBuffer(buffer string, selectedIndex int, suggestions []parser.CommandMatch, currentToken string) string {
	if selectedIndex < 0 || selectedIndex >= len(suggestions) {
		return buffer
	}

	selectedSuggestion := suggestions[selectedIndex].Name

	if currentToken == "" {
		if strings.HasSuffix(buffer, " ") {
			return buffer + selectedSuggestion
		}
		return buffer + " " + selectedSuggestion
	}

	if strings.HasSuffix(buffer, currentToken) {
		idx := len(buffer) - len(currentToken)
		return buffer[:idx] + selectedSuggestion
	}

	return buffer + " " + selectedSuggestion
}
