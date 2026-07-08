package parser

import (
	"encoding/json"
	"fmt"
)

type CommandMatch struct {
	Name                 string
	MatchKey             string                  `json:"-"`
	ShowOnlyWhenSelected bool                    `json:"-"`

	Description          string                  `json:"description"`
	CompleteDescription  string                  `json:"completeDescription"`
	NUsed                int                     `json:"nUsed"`
	SubCommand           map[string]CommandMatch `json:"subCommand"`
}

func ParseCommand(data []byte) (command map[string]CommandMatch, err error) {
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
