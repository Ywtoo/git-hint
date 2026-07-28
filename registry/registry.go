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

// ResolveCommandData reads the JSON data for the given command name from
// disk, at data/<commandName>.json next to the running binary.
func ResolveCommandData(commandName string) ([]byte, error) {
	path, err := core.CommandDataPath(commandName)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("comando não encontrado: %s", commandName)
	}
	return data, nil
}

// ParseCommand parses the JSON data for a command, caching the result
// so repeated calls for the same commandName skip json.Unmarshal entirely.
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

// registry/registry.go

// CommandStatus tells the caller whether a command is known to the system
// at all, and whether its help tree has been crawled yet.
type CommandStatus struct {
	Known   bool   // true if present in index.json (exists on $PATH / is a builtin)
	Indexed bool   // true if data/<name>.json exists (help tree already crawled)
	BinPath string // absolute path, empty for builtins
}

func Status(commandName string) (CommandStatus, error) {
	index, err := loadIndex() // reads index.json once, could be cached
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

// loadIndex reads and parses index.json, returning a map of command name
// to its CommandMatch entry (Name + Path).
func loadIndex() (map[string]core.CommandMatch, error) {
	path, err := core.IndexPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("index.json não encontrado: %w", err)
	}

	var index map[string]core.CommandMatch
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("index.json inválido: %w", err)
	}

	return index, nil
}
