package scraper

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// isForeignMount reports whether dir belongs to another OS's filesystem
// mounted inside this one (e.g. /mnt/c under WSL). We skip these — a
// Windows binary living under /mnt/c isn't a "real" Linux command, it's
// Windows leaking into $PATH.
func isForeignMount(dir string) bool {
	if runtime.GOOS == "windows" {
		return false // native Windows: nothing to skip
	}
	return strings.HasPrefix(dir, "/mnt/")
}

func isUnixExecutable(info os.FileInfo) bool {
	return !info.IsDir() && info.Mode()&0111 != 0
}

func DiscoverPathBinaries() []DiscoveredCommand {
	seen := make(map[string]bool)
	var found []DiscoveredCommand

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if isForeignMount(dir) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if e.IsDir() || seen[e.Name()] {
				continue
			}
			full := filepath.Join(dir, e.Name())
			info, err := os.Stat(full)
			if err != nil || !isUnixExecutable(info) {
				continue
			}
			seen[e.Name()] = true
			found = append(found, DiscoveredCommand{Name: e.Name(), Path: full})
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
