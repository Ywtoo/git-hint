package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// executablePath resolves the running binary's path. It's a variable
// (not a direct os.Executable() call) so tests can override it to point
// DataDir at a temp directory instead of the real test binary location.
var executablePath = os.Executable

// SetExecutablePathForTest overrides executablePath for testing purposes
// and returns a cleanup function to restore the original value.
func SetExecutablePathForTest(fn func() (string, error)) func() {
	original := executablePath
	executablePath = fn
	return func() { executablePath = original }
}

// DataDir returns the path to the data/ directory that sits next to the
// running binary (e.g. /opt/git-hint/data). This is where scraper.Rebuild
// writes json files and where registry reads them from — no embed, since
// the data is meant to be regenerated in place without recompiling.
func DataDir() (string, error) {
	exe, err := executablePath()
	if err != nil {
		return "", fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Resolve symlinks (e.g. /usr/local/bin/git-hint -> real install dir)
	// so data/ lands next to the actual binary, not a symlink location.
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}

	dir := filepath.Join(filepath.Dir(resolved), "data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data dir: %w", err)
	}
	return dir, nil
}

// CommandDataPath returns the full path to a single command's json file
// inside DataDir, e.g. data/git.json.
func CommandDataPath(commandName string) (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, commandName+".json"), nil
}

// IndexPath returns the full path to index.json inside DataDir.
func IndexPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.json"), nil
}
