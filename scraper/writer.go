package scraper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"git-hint/internal/core"
)

// ResetDataDir removes every file in DataDir. Rebuild calls this before
// writing anything, so stale command files from a previous run (e.g. one
// that used a wider crawl scope, or a command later removed from the
// allowlist) don't linger and get mistaken for current data.
func ResetDataDir() error {
	dir, err := core.DataDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("failed to remove stale file %s: %w", e.Name(), err)
		}
	}
	return nil
}

// WriteCommand saves a single command's tree to data/<name>.json,
// next to the running binary.
func WriteCommand(name string, tree map[string]core.CommandMatch) error {
	path, err := core.CommandDataPath(name)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", name, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// WriteIndex saves index.json — a flat map of root command name to its
// entry. The filename each command lives in is always derivable as
// "<name>.json" (see core.CommandDataPath), so it's not stored here;
// Path (when set) records whether the command came from $PATH or is a
// shell builtin.
func WriteIndex(index map[string]core.CommandMatch) error {
	path, err := core.IndexPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
