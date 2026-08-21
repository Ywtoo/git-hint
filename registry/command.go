package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"git-hint/core"
)

var (
	cacheMu sync.RWMutex
	cache   = make(map[string]map[string]core.CommandMatch)
)

func ResolveCommandData(commandName string) ([]byte, error) {
	path, err := core.CommandDataPath(commandName)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("command not found: %s", commandName)
	}
	return data, nil
}

func ParseCommand(commandName string, data []byte) (map[string]core.CommandMatch, error) {
	cacheMu.RLock()
	if cached, ok := cache[commandName]; ok {
		cacheMu.RUnlock()
		return cached, nil
	}
	cacheMu.RUnlock()

	var command map[string]core.CommandMatch
	if err := json.Unmarshal(data, &command); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	for name, cmd := range command {
		cmd.Name = name
		command[name] = cmd
	}

	cacheMu.Lock()
	cache[commandName] = command
	cacheMu.Unlock()

	return command, nil
}

type CommandStatus struct {
	Known   bool
	Indexed bool
	BinPath string
}

func Status(commandName string) (CommandStatus, error) {
	index, err := LoadIndex()
	if err != nil {
		return CommandStatus{}, err
	}
	entry, known := index[commandName]
	if !known {
		return CommandStatus{Known: false}, nil
	}

	_, err = ResolveCommandData(commandName)
	return CommandStatus{
		Known:   true,
		Indexed: err == nil,
		BinPath: entry.Path,
	}, nil
}
