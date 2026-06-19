package parser

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	filePath := "../../../data/git.json"

	cmd, err := ParseCommand(filePath)

	if err != nil {
		t.Fatalf("ParseCommand falhou inesperadamente: %v", err)
	}

	if cmd == nil {
		t.Fatal("ParseCommand retornou nil, mas deveria ter retornado um comando")
	}

	var commandTest CommandMatch
	for _, v := range cmd {
		commandTest = v
		break
	}

	lenghtSubC := len(commandTest.SubCommand)
	if lenghtSubC == 0 {
		t.Errorf("Esperava subcomandos, mas a lista estava vazia")
	}

}
