package ranking

import (
	"bufio"
	"fmt"
	"git-hint/engine/parser"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func RankSuggestions(commandName string, suggestions []parser.CommandMatch) ([]parser.CommandMatch, error) {
	path, err := findHistory()
	if err != nil {
		return nil, fmt.Errorf("❌ Erro ao encontrar home: %v", err)
	}

	arquive, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("❌ Erro ao abrir historico: %v", err)
	}
	defer arquive.Close()

	history := bufio.NewScanner(arquive)
	usedCommands := make(map[string]int)

	commandsFields := strings.Fields(commandName)
	commandLen := len(commandsFields)
	if commandLen < 1 {
		return suggestions, nil
	}

	for history.Scan() {
		line := history.Text()
		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			continue
		}

		if strings.HasPrefix(parts[1], commandName) {
			historyFields := strings.Fields(parts[1])

			limit := commandLen - 1
			if len(historyFields) < limit {
				limit = len(historyFields)
			}

			if len(historyFields) > limit {
				word := historyFields[limit]
				if word != "" {
					usedCommands[word]++
				}
			}
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

func findHistory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zsh_history"), nil
}
