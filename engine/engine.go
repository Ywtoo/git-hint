package engine

import (
	"fmt"
	"sort"
	"strings"

	"git-hint/core"
	"git-hint/engine/provider"
	"git-hint/engine/ranking"
	"git-hint/engine/tokenizer"
	"git-hint/registry"
)

var noDescriptionFlags = map[string]bool{
	"msg": true,
}

var skipRankingFlags = map[string]bool{
	"commit": true,
	"msg":    true,
}

func Suggestions(input string) ([]core.CommandMatch, string, error) {
	parts := tokenizer.TokenizeBuffer(input)
	if len(parts) == 0 {
		return nil, "", nil
	}
	commandName := parts[0]
	remainingInput := parts[1:]
	var list []core.CommandMatch

	data, err := registry.ResolveCommandData(commandName)
	if err != nil {
		return nil, "", fmt.Errorf("❌ Error finding command '%s': %v", commandName, err)
	}
	if data == nil {
		return nil, "", nil
	}

	commands, err := registry.ParseCommand(data)
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
			groupPlaceholder := name // ex: "<msg>"
			if cmd.Description != "" {
				groupPlaceholder = name + "  " + cmd.Description // ex: "<msg>  mensagem do commit"
			}

			// Item placeholder principal (não selecionável na UI como comando estático, serve de cabeçalho do grupo)
			placeholderItem := cmd
			placeholderItem.Placeholder = groupPlaceholder
			placeholderItem.Name = "" // Sem nome selecionável para a linha do cabeçalho
			list = append(list, placeholderItem)

			dynamicFlagSeen = flag
			expanded := provider.Provider(flag)

			for _, s := range expanded {
				if s.Name == "" || !strings.HasPrefix(s.Name, currentToken) {
					continue
				}
				s.Placeholder = groupPlaceholder

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

	// Reagrupa por placeholder mantendo a ordem de ranking dentro de cada grupo.
	// Itens estáticos (Placeholder == "") ficam juntos no topo, antes dos dinâmicos.
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Placeholder < list[j].Placeholder
	})

	return list, currentToken, nil
}

func CompleteBuffer(buffer string, selectedIndex int, suggestions []core.CommandMatch, currentToken string) string {
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
