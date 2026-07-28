package scraper

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiscoveredCommand is a command found on the system, not yet crawled.
type DiscoveredCommand struct {
	Name string
	Path string // absolute path for $PATH binaries; empty for shell builtins
}

// DiscoverPathBinaries walks every directory in $PATH and collects every
// executable name found, deduped by name (first match wins, same
// precedence order as $PATH itself).
func DiscoverPathBinaries() []DiscoveredCommand {
	seen := make(map[string]bool)
	var found []DiscoveredCommand

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // unreadable dir (permissions, doesn't exist) — skip
		}
		for _, e := range entries {
			if e.IsDir() || seen[e.Name()] {
				continue
			}
			full := filepath.Join(dir, e.Name())
			if info, err := os.Stat(full); err == nil && info.Mode()&0111 != 0 {
				seen[e.Name()] = true
				found = append(found, DiscoveredCommand{Name: e.Name(), Path: full})
			}
		}
	}
	return found
}

// DiscoverZshBuiltins lists builtin command names known to zsh.
// Builtins have no executable on disk and no machine-readable -h/--help,
// so only Name is populated for now — Path stays empty and Description
// is left blank until a man-page (zshbuiltins) based enrichment step is added.
func DiscoverZshBuiltins() []DiscoveredCommand {
	out, err := exec.Command("zsh", "-c", "print -l ${(k)builtins}").Output()
	if err != nil {
		return nil
	}

	var found []DiscoveredCommand
	for _, name := range strings.Fields(string(out)) {
		found = append(found, DiscoveredCommand{Name: name})
	}
	return found
}
