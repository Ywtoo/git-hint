package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"git-hint/data"
	"git-hint/internal/core"
)

var (
	cacheMu sync.RWMutex
	cache   = make(map[string]map[string]core.CommandMatch)
)

// ResolveCommandData reads a command's JSON spec using a two-layer lookup:
//  1. Embedded Fig bundle: the canonical spec for commands covered by Fig.
//  2. Disk: data/<name>.json next to the binary, used for local/crawler data
//     only when the command is not present in the Fig bundle.
//
// Fig must win over disk: a stale crawler-generated file must never replace
// the community-maintained specification. Embedded specs are also written to
// disk as a cache/materialized copy for inspection.
//
// The crawler (CrawlOne) is only triggered by app/list.go when this function
// returns an error — meaning the command is in index.json but not in either
// the disk cache or the embedded bundle.
func ResolveCommandData(commandName string) ([]byte, error) {
	diskPath, pathErr := core.CommandDataPath(commandName)

	// Layer 1: embedded Fig bundle (canonical source)
	raw, err := data.ReadSpec(commandName)
	if err == nil {
		if pathErr == nil {
			_ = os.WriteFile(diskPath, raw, 0644)
		}
		return raw, nil
	}

	// Layer 2: disk fallback (local edits or crawler-generated data).
	if pathErr == nil {
		if raw, readErr := os.ReadFile(diskPath); readErr == nil {
			return raw, nil
		}
	}

	return nil, fmt.Errorf("command not found: %s", commandName)
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

// Status reports whether a command is known (present in index.json) and
// indexed (has a spec available — either on disk or in the embedded bundle).
// Commands from the embedded Fig bundle are always Indexed from the first run,
// so the dynamic crawler is never triggered for them.
func Status(commandName string) (CommandStatus, error) {
	index, err := LoadIndex()
	if err != nil {
		return CommandStatus{}, err
	}
	entry, known := index[commandName]
	if !known {
		return CommandStatus{Known: false}, nil
	}

	// A spec is available if it exists on disk OR in the embedded bundle.
	diskPath, _ := core.CommandDataPath(commandName)
	onDisk := false
	if diskPath != "" {
		_, err := os.Stat(diskPath)
		onDisk = err == nil
	}
	indexed := onDisk || data.HasSpec(commandName)

	return CommandStatus{
		Known:   true,
		Indexed: indexed,
		BinPath: entry.Path,
	}, nil
}
