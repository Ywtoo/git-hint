package parser

import (
	"os"
	"testing"
)

func TestParseCommand(t *testing.T) {
	// Create a temporary JSON file to avoid relative path issues
	content := `{
		"test-cmd": {
			"description": "test description",
			"subCommand": {
				"sub-1": { "description": "sub desc" }
			}
		}
	}`
	tmpFile, err := os.CreateTemp("", "git_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cmd, err := ParseCommand(tmpFile.Name())

	if err != nil {
		t.Fatalf("ParseCommand falhou inesperadamente: %v", err)
	}

	if cmd == nil {
		t.Fatal("ParseCommand retornou nil, mas deveria ter retornado um comando")
	}

	if _, ok := cmd["test-cmd"]; !ok {
		t.Error("Esperava encontrar 'test-cmd' no mapa de comandos")
	}

	testCmd := cmd["test-cmd"]
	if len(testCmd.SubCommand) == 0 {
		t.Errorf("Esperava subcomandos, mas a lista estava vazia")
	}
}
