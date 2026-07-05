package history

import (
	"os"
	"path/filepath"
	"strings"
)

var historyPathFunc = findHistory

func FindHistoryCommands(commandName string) ([]string, error) {
	path, err := historyPathFunc()
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var historyList []string

	for i := len(lines) - 1; i >= 0; i-- {
		if len(historyList) >= 500 {
			break
		}

		line := lines[i]
		if line == "" {
			continue
		}

		// ": 1712345678:0;git commit -m..."
		parts := strings.SplitN(line, ";", 2)
		if len(parts) < 2 {
			continue
		}

		cmd := parts[1]
		if strings.HasPrefix(cmd, commandName) {
			historyList = append(historyList, cmd)
		}
	}

	return historyList, nil
}

func findHistory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zsh_history"), nil
}
