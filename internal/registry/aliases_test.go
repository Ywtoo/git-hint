package registry_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-hint/internal/core"
	"git-hint/internal/registry"
	"git-hint/internal/state"
)

func TestMergeAliasesAddsLevelZeroSuggestions(t *testing.T) {
	state.SetAliases(map[string]string{
		"gs":  "git status",
		"gcm": "git commit -m",
		"ll":  "ls -lah",
	})
	t.Cleanup(func() { state.SetAliases(nil) })

	index := map[string]core.CommandMatch{
		"git": {Name: "git", Description: "the stupid content tracker"},
		"ls":  {Name: "ls", Path: "/bin/ls"},
	}

	merged := registry.MergeAliases(index)

	gs, ok := merged["gs"]
	if !ok {
		t.Fatalf("alias 'gs' missing from merged index")
	}
	if !strings.HasPrefix(gs.Description, "alias:") {
		t.Errorf("alias 'gs' description = %q, want prefix \"alias:\"", gs.Description)
	}
	if _, ok := merged["git"]; !ok {
		t.Error("index entry 'git' must survive the alias merge")
	}
	if _, ok := merged["ls"]; !ok {
		t.Error("index entry 'ls' must survive the alias merge")
	}
}

func TestMergeAliasesEmptyTableReturnsIndexUntouched(t *testing.T) {
	state.SetAliases(nil)

	index := map[string]core.CommandMatch{
		"git": {Name: "git"},
	}
	got := registry.MergeAliases(index)
	if len(got) != 1 {
		t.Errorf("MergeAliases with no aliases changed the index: %v", got)
	}
}

func TestSuggestFromIndexIncludesAliases(t *testing.T) {
	state.SetAliases(map[string]string{"gco": "git checkout"})
	t.Cleanup(func() { state.SetAliases(nil) })

	// Point the data dir at a temp dir so LoadIndex reads our test index.
	fakeBin := filepath.Join(t.TempDir(), "githint-test")
	if err := os.WriteFile(fakeBin, []byte{}, 0755); err != nil {
		t.Fatal(err)
	}
	cleanupExe := core.SetExecutablePathForTest(func() (string, error) {
		return fakeBin, nil
	})
	registry.ResetCacheForTest()
	t.Cleanup(func() {
		cleanupExe()
		registry.ResetCacheForTest()
	})

	dataDir := filepath.Join(filepath.Dir(fakeBin), "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	indexJSON := []byte(`{"git": {"name": "git", "description": "git"}}`)
	if err := os.WriteFile(filepath.Join(dataDir, "index.json"), indexJSON, 0644); err != nil {
		t.Fatal(err)
	}

	list, _, err := registry.SuggestFromIndex("g")
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, m := range list {
		if m.Name == "gco" {
			found = true
		}
	}
	if !found {
		t.Errorf("alias 'gco' not suggested at level 0, got %v", list)
	}
}
