package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCommandPath(t *testing.T) {
	// Setup a temporary project structure
	tmpDir, err := os.MkdirTemp("", "githint_registry_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod to mark the project root
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module git-hint"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create data directory and a dummy json file
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.Mkdir(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	cmdFile := filepath.Join(dataDir, "git.json")
	if err := os.WriteFile(cmdFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change working directory to the temp dir to simulate project root
	originalWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWd)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid command",
			input:    "git",
			expected: cmdFile,
			wantErr:  false,
		},
		{
			name:     "Invalid command",
			input:    "nonexistent",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveCommandData(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCommandData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && len(got) == 0 {
				t.Errorf("ResolveCommandData() = %v, want non-empty data", got)
			}
		})
	}
}
