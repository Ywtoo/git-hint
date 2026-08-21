package history

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"git-hint/registry"
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

// TopRootCommands reads shell history and returns the most frequently
// used root commands (level 0: git, docker, kubectl, etc.) that are
// known to the registry.
func TopRootCommands(limit int) ([]string, error) {
	path, err := historyPathFunc()
	if err != nil {
		return nil, err
	}

	commands, err := parseHistoryCommands(path)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, cmd := range commands {
		fields := strings.Fields(cmd)
		if len(fields) == 0 {
			continue
		}
		root := fields[0]
		counts[root]++
	}

	// Load registry index to filter only known commands
	index, err := registry.LoadIndex()
	if err != nil {
		return nil, err
	}

	// Convert to slice and sort by frequency
	type commandCount struct {
		command string
		count   int
	}
	var commandCounts []commandCount
	for cmd, count := range counts {
		if _, known := index[cmd]; known {
			commandCounts = append(commandCounts, commandCount{command: cmd, count: count})
		}
	}

	sort.Slice(commandCounts, func(i, j int) bool {
		return commandCounts[i].count > commandCounts[j].count
	})

	if len(commandCounts) > limit {
		commandCounts = commandCounts[:limit]
	}

	result := make([]string, len(commandCounts))
	for i, cc := range commandCounts {
		result[i] = cc.command
	}

	return result, nil
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
