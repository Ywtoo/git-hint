package engine

import (
	"os"
	"path/filepath"
	"testing"

	"git-hint/internal/core"
	"git-hint/internal/registry"
	"git-hint/internal/state"
)

func setupCDTest(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	indexJSON := `{"cd": {"name": "cd"}}`
	if err := os.WriteFile(filepath.Join(dataDir, "index.json"), []byte(indexJSON), 0644); err != nil {
		t.Fatal(err)
	}
	fakeBin := filepath.Join(tmp, "githint")
	if err := os.WriteFile(fakeBin, []byte{}, 0755); err != nil {
		t.Fatal(err)
	}
	cleanupExe := core.SetExecutablePathForTest(func() (string, error) { return fakeBin, nil })
	registry.ResetCacheForTest()
	t.Cleanup(func() {
		cleanupExe()
		registry.ResetCacheForTest()
		state.SetWorkingDir("")
		state.SetBuffer("")
	})
}

// "cd st" must find a symlinked directory (~/storage -> /media/...): the
// IsDir check must follow symlinks, otherwise the list comes back empty and
// Tab lands on a bare "sair (Tab)" screen.
func TestCDFindsSymlinkedDirectory(t *testing.T) {
	setupCDTest(t)

	tmp := t.TempDir()
	target := filepath.Join(tmp, "media", "storage_fixed")
	if err := os.MkdirAll(filepath.Join(target, "Programming"), 0755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(home, "storage")); err != nil {
		t.Fatal(err)
	}
	state.SetWorkingDir(home)
	state.SetBuffer("cd st")

	list, _, err := Suggestions("cd st")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range list {
		if m.Name == "storage/" {
			found = true
		}
	}
	if !found {
		t.Errorf("symlinked dir 'storage/' missing from suggestions, got %v", list)
	}
}

// A group must render exactly one header, always before its items — even
// when usage ranking would rank an item above the header.
func TestGroupRendersSingleHeaderBeforeItems(t *testing.T) {
	setupCDTest(t)

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "alpha"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "beta"), 0755); err != nil {
		t.Fatal(err)
	}
	state.SetWorkingDir(tmp)
	state.SetBuffer("cd ")

	list, _, err := Suggestions("cd ")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 3 {
		t.Fatalf("expected header + 2 items, got %v", list)
	}

	if list[0].Name != "" {
		t.Fatalf("first entry must be the group header, got name=%q", list[0].Name)
	}
	headerCount := 0
	for _, m := range list {
		if m.Name == "" {
			headerCount++
		}
	}
	if headerCount != 1 {
		t.Errorf("expected exactly 1 header, got %d: %v", headerCount, list)
	}
}

// Typing a directory prefix keeps working through the trailing-slash form.
// A single-item group renders without its header: one selectable row between
// the user and the value is pure noise.
func TestCDIntoTypedDirectoryListsNested(t *testing.T) {
	setupCDTest(t)

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "sub1", "deep"), 0755); err != nil {
		t.Fatal(err)
	}
	state.SetWorkingDir(tmp)
	state.SetBuffer("cd sub1/")

	list, _, err := Suggestions("cd sub1/")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected single item without header for singleton group, got %v", list)
	}
	if list[0].Name != "sub1/deep/" {
		t.Errorf("unexpected suggestions: %v", list)
	}
}
