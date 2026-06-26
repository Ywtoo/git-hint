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

	// Decidimos se rankeamos a palavra atual (completando) ou a próxima (nova palavra)
	targetIdx := commandLen // Default: próxima palavra

	isCompleting := false
	for _, s := range suggestions {
		// Se o token é um prefixo da sugestão, mas não é a sugestão completa, estamos completando
		if strings.HasPrefix(s.Name, lastToken) && s.Name != lastToken {
			isCompleting = true
			break
		}
	}

	if isCompleting {
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
	if err := history.Err(); err != nil {
		return nil, fmt.Errorf("❌ Erro ao ler historico: %v", err)
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
