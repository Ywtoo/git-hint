package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type CommandMatch struct {
	Name                string
	Required            *string                 `json:"required"`
	Description         string                  `json:"description"`
	CompleteDescription string                  `json:"completeDescription"`
	NUsed               int                     `json:"nUsed"`
	SubCommand          map[string]CommandMatch `json:"subCommand"`
}

func ParseCommand(filePath string) (command map[string]CommandMatch, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer file.Close()

	readedFile, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	err = json.Unmarshal(readedFile, &command)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	for name, cmd := range command {
		cmd.Name = name
		command[name] = cmd
	}

	return command, nil
}
