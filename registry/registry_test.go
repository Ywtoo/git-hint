package registry

import (
	"testing"

	"git-hint/core"
)

// TestResolveCommandData_NotFound verifies that requesting a command with
// no corresponding json file returns an error. Path resolution itself
// (DataDir following the binary, symlinks, etc.) is core's responsibility
// and is already covered in core/paths_test.go — no need to duplicate it.
func TestResolveCommandData_NotFound(t *testing.T) {
	_, err := ResolveCommandData("comando-que-nao-existe-com-certeza")
	if err == nil {
		t.Fatal("esperava erro para comando inexistente")
	}
}

func TestParseCommand(t *testing.T) {
	content := `{
		"test-cmd": {
			"description": "test description",
			"subCommand": {
				"sub-1": { "description": "sub desc" }
			}
		}
	}`

	cmd, err := ParseCommand("git_test", []byte(content))
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

func TestParseCommand_CachesResult(t *testing.T) {
	cacheMu.Lock()
	cache = make(map[string]map[string]core.CommandMatch)
	cacheMu.Unlock()

	content := `{"commit":{"description":"Record changes to the repository"}}`

	first, err := ParseCommand("commit", []byte(content))
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	differentContent := `{"push":{"description":"isso não deveria aparecer"}}`
	second, err := ParseCommand("commit", []byte(differentContent))
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if _, ok := second["commit"]; !ok {
		t.Fatal("esperava resultado cacheado (commit), cache não funcionou")
	}
	if _, ok := second["push"]; ok {
		t.Fatal("cache não funcionou: reparseou o JSON novo em vez de usar o cache")
	}
	if len(first) != len(second) {
		t.Fatalf("first e second deveriam ser idênticos, tamanhos diferem: %d vs %d", len(first), len(second))
	}

	cacheMu.RLock()
	_, cached := cache["commit"]
	cacheMu.RUnlock()
	if !cached {
		t.Fatal("commit deveria estar na variável cache após ParseCommand")
	}
}

func TestParseCommand_DifferentCommandNamesDontCollide(t *testing.T) {
	cacheMu.Lock()
	cache = make(map[string]map[string]core.CommandMatch)
	cacheMu.Unlock()

	commitData := `{"commit":{"description":"a"}}`
	pushData := `{"push":{"description":"b"}}`

	commitResult, err := ParseCommand("commit", []byte(commitData))
	if err != nil {
		t.Fatal(err)
	}
	pushResult, err := ParseCommand("push", []byte(pushData))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := commitResult["commit"]; !ok {
		t.Fatal("commitResult deveria conter 'commit'")
	}
	if _, ok := pushResult["push"]; !ok {
		t.Fatal("pushResult deveria conter 'push'")
	}
}

func TestParseCommand_InvalidJSONNotCached(t *testing.T) {
	cacheMu.Lock()
	cache = make(map[string]map[string]core.CommandMatch)
	cacheMu.Unlock()

	_, err := ParseCommand("broken", []byte("{isso não é json válido"))
	if err == nil {
		t.Fatal("esperava erro para JSON inválido, mas ParseCommand não retornou erro")
	}

	cacheMu.RLock()
	_, cached := cache["broken"]
	cacheMu.RUnlock()
	if cached {
		t.Fatal("resultado com erro não deveria ter sido cacheado")
	}
}
