package scraper

import (
	"fmt"
	"os"
	"path/filepath"

	"git-hint/data"
	"git-hint/internal/core"
)

// BuildIndex discovers all binaries in $PATH and zsh builtins, writes index.json,
// and returns the index map.
func BuildIndex() (map[string]core.CommandMatch, error) {
	index := make(map[string]core.CommandMatch)

	commands := DiscoverPathBinaries()
	commands = append(commands, DiscoverZshBuiltins()...)

	for _, cmd := range commands {
		index[cmd.Name] = core.CommandMatch{Name: cmd.Name, Path: cmd.Path}
	}

	if err := WriteIndex(index); err != nil {
		return nil, err
	}
	return index, nil
}

// Warmup runs on the very first execution of git-hint (when index.json does not exist).
// It:
//  1. Extracts all 754 embedded Fig specs into the data/ directory so they are
//     visible on disk and the dynamic crawler is never triggered for them.
//  2. Builds index.json by discovering $PATH binaries and zsh builtins on this machine.
func Warmup(opts Options) error {
	indexPath, err := core.IndexPath()
	if err != nil {
		return err
	}

	dataDir, err := core.DataDir()
	if err != nil {
		return err
	}

	// Always extract embedded specs (skip files that already exist on disk,
	// so user-edited or crawler-generated files are never overwritten).
	if _, extractErr := data.ExtractAll(filepath.Join(dataDir, "fig")); extractErr != nil {
		// Non-fatal: the embedded lookup in registry still works even if
		// extraction fails (e.g. read-only filesystem).
		_ = extractErr
	}

	if _, err := os.Stat(indexPath); err == nil {
		// index.json already exists: warmup already ran before.
		return nil
	}

	_, err = BuildIndex()
	if err != nil {
		return fmt.Errorf("warmup failed to build index: %w", err)
	}
	return nil
}

// Rebuild refreshes index.json (finding new or removed binaries on this machine).
// Command data (subcommands, flags) comes from the embedded Fig bundle — no crawling.
func Rebuild(opts Options) error {
	_, err := BuildIndex()
	if err != nil {
		return fmt.Errorf("failed to rebuild index: %w", err)
	}
	return nil
}
