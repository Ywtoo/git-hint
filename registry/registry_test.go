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
	_, err := ResolveCommandData("command-that-definitely-does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent command")
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
		t.Fatalf("ParseCommand failed unexpectedly: %v", err)
	}
	if cmd == nil {
		t.Fatal("ParseCommand returned nil, expected command map")
	}
	if _, ok := cmd["test-cmd"]; !ok {
		t.Error("Expected to find 'test-cmd' in commands map")
	}

	testCmd := cmd["test-cmd"]
	if len(testCmd.SubCommand) == 0 {
		t.Errorf("Expected subcommands, but list was empty")
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

	differentContent := `{"push":{"description":"this should not appear"}}`
	second, err := ParseCommand("commit", []byte(differentContent))
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if _, ok := second["commit"]; !ok {
		t.Fatal("expected cached result (commit), cache did not work")
	}
	if _, ok := second["push"]; ok {
		t.Fatal("cache did not work: reparsed new JSON instead of using cache")
	}
	if len(first) != len(second) {
		t.Fatalf("first and second should be identical, lengths differ: %d vs %d", len(first), len(second))
	}

	cacheMu.RLock()
	_, cached := cache["commit"]
	cacheMu.RUnlock()
	if !cached {
		t.Fatal("commit should be in cache variable after ParseCommand")
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
		t.Fatal("commitResult should contain 'commit'")
	}
	if _, ok := pushResult["push"]; !ok {
		t.Fatal("pushResult should contain 'push'")
	}
}

func TestParseCommand_InvalidJSONNotCached(t *testing.T) {
	cacheMu.Lock()
	cache = make(map[string]map[string]core.CommandMatch)
	cacheMu.Unlock()

	_, err := ParseCommand("broken", []byte("{this is not valid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, but ParseCommand returned no error")
	}

	cacheMu.RLock()
	_, cached := cache["broken"]
	cacheMu.RUnlock()
	if cached {
		t.Fatal("error result should not have been cached")
	}
}
