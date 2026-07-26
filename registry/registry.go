package registry

import (
	"embed"
	"encoding/json"
	"fmt"

	"git-hint/core"
)

//go:embed data/*.json
var dataFS embed.FS

// ResolveCommandData returns the JSON data for the given command name.
func ResolveCommandData(commandName string) ([]byte, error) {
	// The files are embedded relative to the directory of this file.
	path := fmt.Sprintf("data/%s.json", commandName)
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("comando não encontrado: %s", commandName)
	}
	return data, nil
}

func ParseCommand(data []byte) (command map[string]core.CommandMatch, err error) {
	err = json.Unmarshal(data, &command)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	for name, cmd := range command {
		cmd.Name = name
		command[name] = cmd
	}

	return command, nil
}
