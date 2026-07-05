package ranking

import (
	"git-hint/engine/history"
	"git-hint/engine/parser"
	"slices"
	"strings"
)

func RankSuggestions(commandName string, suggestions []parser.CommandMatch) ([]parser.CommandMatch, error) {
	usedCommands := make(map[string]int)
	commandsFields := strings.Fields(commandName)
	commandLen := len(commandsFields)
	if commandLen < 1 {
		return suggestions, nil
	}

	lastToken := commandsFields[commandLen-1]

	targetIdx := commandLen

	isCompleting := false
	for _, s := range suggestions {
		if strings.HasPrefix(s.Name, lastToken) && s.Name != lastToken {
			isCompleting = true
			break
		}
	}

	if isCompleting {
		targetIdx = commandLen - 1
	}

	commandHistory, err := history.FindHistoryCommands(commandName)
	if err != nil {
		return nil, err
	}

	for i := range commandHistory {
		historyFields := strings.Fields(commandHistory[i])
		if len(historyFields) <= targetIdx {
			continue
		}

		// Validação do prefixo: rigoroso no passado, flexível no presente
		matches := true
		for i := 0; i < commandLen; i++ {
			if i >= len(historyFields) {
				matches = false
				break
			}
			if i < commandLen-1 {
				// Palavras anteriores devem ser idênticas
				if historyFields[i] != commandsFields[i] {
					matches = false
					break
				}
			} else {
				// A última palavra deve ser um prefixo
				if !strings.HasPrefix(historyFields[i], commandsFields[i]) {
					matches = false
					break
				}
			}
		}

		if !matches {
			continue
		}

		word := historyFields[targetIdx]
		if word != "" {
			usedCommands[word]++
		}
	}

	for i := range suggestions {
		if count, exists := usedCommands[suggestions[i].Name]; exists {
			suggestions[i].NUsed = count
		} else {
			suggestions[i].NUsed = 0
		}
	}

	slices.SortFunc(suggestions, func(a parser.CommandMatch, b parser.CommandMatch) int {
		if a.NUsed > b.NUsed {
			return -1
		}
		if a.NUsed < b.NUsed {
			return 1
		}

		return strings.Compare(a.Name, b.Name)
	})

	return suggestions, nil
}
