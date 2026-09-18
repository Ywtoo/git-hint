package data

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed fig/*.json
var specsFS embed.FS

// ExtractAll extracts all embedded command JSON specs into targetDir
// if they do not already exist on disk.
func ExtractAll(targetDir string) (int, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, err
	}

	entries, err := fs.ReadDir(specsFS, "fig")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		targetPath := filepath.Join(targetDir, entry.Name())
		// If already exists on disk, preserve user's local version or crawled version
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		content, err := specsFS.ReadFile(filepath.Join("fig", entry.Name()))
		if err != nil {
			continue
		}

		if err := os.WriteFile(targetPath, content, 0644); err == nil {
			count++
		}
	}

	return count, nil
}

// ReadSpec reads an embedded spec by command name (without .json extension).
func ReadSpec(commandName string) ([]byte, error) {
	return specsFS.ReadFile(filepath.Join("fig", commandName+".json"))
}

// HasSpec reports whether a command spec is present in the embedded dataset.
func HasSpec(commandName string) bool {
	_, err := specsFS.Open(filepath.Join("fig", commandName+".json"))
	return err == nil
}
