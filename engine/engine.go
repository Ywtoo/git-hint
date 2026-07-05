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

	filePath, err := registry.ResolveCommandPath(commandName)
	if err != nil {
		return nil, "", fmt.Errorf("❌ Erro ao encontrar o comando '%s': %v", commandName, err)
	}
	if filePath == "" {
		return nil, "", nil
	}

	commands, err := parser.ParseCommand(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("❌ Erro ao ler o arquivo: %v\n", err)
	}
	if commands == nil {
		return nil, "", nil
	}

	matches, currentToken, err := FindCommands(remainingInput, commands)
	if err != nil {
		return nil, "", fmt.Errorf("❌ Erro ao encontrar comandos: %v\n", err)
	}
	if matches == nil {
		return nil, "", nil
	}

	var dynamicFlagSeen string // usado só pra decidir se pula o ranking

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
					// commit: mantém a Description própria que o provider já montou
					// (hash + assunto), sempre exibida — não herda do pai, não
					// fica condicionada a "só quando selecionado".

				default:
					// branch, remote, tag, stash: descrição compartilhada do pai,
					// exibida só na linha selecionada.
					s.Description = cmd.Description
					s.ShowOnlyWhenSelected = true
				}

				list = append(list, s)
			}
		} else {
			// estático: Description já vem do próprio cmd, sempre exibida.
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
