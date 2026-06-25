package ranking

import (
	"bufio"
	"fmt"
	"git-hint/engine/parser"
	"io"
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

	return RankSuggestionsWithReader(commandName, suggestions, arquive)
}

func RankSuggestionsWithReader(commandName string, suggestions []parser.CommandMatch, r io.Reader) ([]parser.CommandMatch, error) {
	history := bufio.NewScanner(r)
	usedCommands := make(map[string]int)

	commandsFields := strings.Fields(commandName)
	commandLen := len(commandsFields)
	if commandLen < 1 {
		return suggestions, nil
	}

	lastToken := commandsFields[commandLen-1]

	// Decidimos o índice do histórico baseado no estado da última palavra
	targetIdx := commandLen // Default: próxima palavra (ex: git commit -> flag)

	isPartial := false
	isComplete := false
	for _, s := range suggestions {
		if s.Name == lastToken {
			isComplete = true
		} else if strings.HasPrefix(s.Name, lastToken) {
			isPartial = true
		}
	}

	// Se é um prefixo mas NÃO é a palavra completa, estamos completando a palavra atual
	if isPartial && !isComplete {
		targetIdx = commandLen - 1
	}

	for history.Scan() {
		line := history.Text()
		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			continue
		}

		historyFields := strings.Fields(parts[1])
		if len(historyFields) <= targetIdx {
			continue
		}

		// Verificação de segurança: se estamos buscando a próxima palavra,
		// a palavra anterior no histórico deve bater com a última do buffer
		if targetIdx >= commandLen {
			if historyFields[commandLen-1] != lastToken {
				continue
			}
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

func findHistory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zsh_history"), nil
}
