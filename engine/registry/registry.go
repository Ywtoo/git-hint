package registry

import (
	"embed"
	"fmt"
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
