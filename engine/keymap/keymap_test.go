package keymap

import (
	"os"
	"path/filepath"
	"testing"

	"git-hint/core"
	"git-hint/registry"
	"git-hint/state"
)

func setupTestEnvironment(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexJSON := `{
		"git": {
			"name": "git",
			"description": "the stupid content tracker",
			"path": "/usr/bin/git"
		}
	}`
	if err := os.WriteFile(filepath.Join(dataDir, "index.json"), []byte(indexJSON), 0644); err != nil {
		t.Fatal(err)
	}

	gitJSON := `{
		"commit": {
			"name": "commit",
			"description": "Record changes to the repository",
			"subCommand": {
				"-m": {
					"name": "-m",
					"description": "Use the given message as the commit message"
				},
				"-a": {
					"name": "-a",
					"description": "Automatically stage modified files"
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(dataDir, "git.json"), []byte(gitJSON), 0644); err != nil {
		t.Fatal(err)
	}

	fakeBinary := filepath.Join(tmpDir, "git-hint")
	if err := os.WriteFile(fakeBinary, []byte{}, 0755); err != nil {
		t.Fatal(err)
	}

	cleanupExe := core.SetExecutablePathForTest(func() (string, error) {
		return fakeBinary, nil
	})
	registry.ResetCacheForTest()

	t.Cleanup(func() {
		cleanupExe()
		registry.ResetCacheForTest()
	})
}

func TestKeyHandler(t *testing.T) {
	setupTestEnvironment(t)
	tests := []struct {
		name         string
		key          string
		buffer       string
		selected     int
		wantWidget   string
		wantSelected int
	}{
		{
			name:         "Arrow Up - stay in list",
			key:          "arrowUP",
			buffer:       "git commit",
			selected:     1,
			wantWidget:   "",
			wantSelected: 0,
		},
		{
			name:         "Arrow Up - go to shell history",
			key:          "arrowUP",
			buffer:       "git commit",
			selected:     0,
			wantWidget:   "up-line-or-history",
			wantSelected: -1,
		},
		{
			name:         "Arrow Down - move down",
			key:          "arrowDOWN",
			buffer:       "git commit",
			selected:     0,
			wantWidget:   "",
			wantSelected: 1,
		},
		{
			name:         "TAB - complete",
			key:          "TAB",
			buffer:       "git com",
			selected:     0,
			wantWidget:   "",
			wantSelected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state.SetSelected(tt.selected)
			state.SetBuffer(tt.buffer)
			gotWidget, gotSelected := KeyHandler(tt.key)
			if gotWidget != tt.wantWidget {
				t.Errorf("KeyHandler() widget = %v, want %v", gotWidget, tt.wantWidget)
			}
			if gotSelected != tt.wantSelected {
				t.Errorf("KeyHandler() selected = %v, want %v", gotSelected, tt.wantSelected)
			}
		})
	}
}
