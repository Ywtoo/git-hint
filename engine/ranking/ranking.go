package ranking

import (
	"bufio"
	"git-hint/engine/history"
	"git-hint/engine/parser"
	"io"
	"slices"
	"strings"
)

func RankSuggestions(commandName string, suggestions []parser.CommandMatch) ([]parser.CommandMatch, error) {
	// To keep RankSuggestions compatible and simple, we fetch the history first.
	// However, since FindHistoryCommands returns a slice, we can convert it to a reader
	// or just refactor RankSuggestionsWithReader to take a slice.
	// For the sake of the tests already written in ranking_test.go, we'll use a reader approach.

	commandHistory, err := history.FindHistoryCommands(commandName)
	if err != nil {
		return nil, err
	}

	historyText := strings.Join(commandHistory, "\n")
	return RankSuggestionsWithReader(commandName, suggestions, strings.NewReader(historyText))
}

func RankSuggestionsWithReader(commandName string, suggestions []parser.CommandMatch, reader io.Reader) ([]parser.CommandMatch, error) {
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

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// The reader might receive raw .zsh_history lines (with timestamps)
		// or already processed command strings. Handle both.
		cmd := line
		if strings.Contains(line, ";") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) >= 2 {
				cmd = parts[1]
			}
		}

		historyFields := strings.Fields(cmd)
		if len(historyFields) <= targetIdx {
			continue
		}

		// Prefix validation: strict for the past, flexible for the present
		matches := true
		for i := 0; i < commandLen; i++ {
			if i >= len(historyFields) {
				matches = false
				break
			}
			if i < commandLen-1 {
				// Previous words must be identical
				if historyFields[i] != commandsFields[i] {
					matches = false
					break
				}
			} else {
				// The last word must be a prefix
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

	if err := scanner.Err(); err != nil {
		return nil, err
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
