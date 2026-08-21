package history

import (
	"os"
	"path/filepath"
	"strings"
)

var historyPathFunc = findHistory

// SetHistoryPathForTest overrides historyPathFunc for testing purposes
// and returns a cleanup function to restore the original value.
func SetHistoryPathForTest(fn func() (string, error)) func() {
	original := historyPathFunc
	historyPathFunc = fn
	return func() { historyPathFunc = original }
}

func FindHistoryCommands(commandName string) ([]string, error) {
	path, err := historyPathFunc()
	if err != nil {
		return nil, err
	}

	commands, err := parseHistoryCommands(path)
	if err != nil {
		return nil, err
	}

	var historyList []string
	for _, cmd := range commands {
		if len(historyList) >= 500 {
			break
		}
		if strings.HasPrefix(cmd, commandName) {
			historyList = append(historyList, cmd)
		}
	}

	return historyList, nil
}

// findHistory returns the path to the user's zsh history file.
func findHistory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zsh_history"), nil
}

// parseHistoryCommands reads the history file and returns the commands
// (the part after ';'), from most recent to oldest.
func parseHistoryCommands(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var commands []string

	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if line == "" {
			continue
		}

		// ": 1712345678:0;git commit -m..."
		parts := strings.SplitN(line, ";", 2)
		if len(parts) < 2 {
			continue
		}

		commands = append(commands, parts[1])
	}

	return commands, nil
}
