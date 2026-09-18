// genbuiltins — DEV-ONLY tool. Never ships to users.
//
// Runs the builtins pipeline (HELPDIR → man zshbuiltins → bash help) for
// every builtin known to zsh and writes the result into data/<name>.json,
// committed to the repo so end users never crawl builtins at runtime.
//
// Usage (from repo root, on a machine that has zsh docs installed):
//
//	go run ./tools/builtin/importer
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"git-hint/internal/core"
	"git-hint/scraper"
)

func main() {
	// CrawlBuiltin writes through core.DataDir (exe dir + "/data"), so we
	// fake the executable inside a temp dir, generate there, then copy
	// everything into the real ./data.
	tmp, err := os.MkdirTemp("", "genbuiltins-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			fmt.Fprintln(os.Stderr, "cleanup:", err)
		}
	}()

	restore := core.SetExecutablePathForTest(func() (string, error) {
		return filepath.Join(tmp, "githint-gen"), nil
	})
	defer restore()

	builtins := scraper.DiscoverZshBuiltins()
	if len(builtins) == 0 {
		fmt.Fprintln(os.Stderr, "no zsh builtins discovered (is zsh installed?)")
		os.Exit(1)
	}

	ok, missing := 0, 0
	for _, b := range builtins {
		if err := scraper.CrawlBuiltin(b.Name); err != nil {
			missing++
			continue
		}
		ok++
	}

	// Copy the generated files into the repo's data/.
	genDir := filepath.Join(tmp, "data")
	entries, err := os.ReadDir(genDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	copied := 0
	for _, e := range entries {
		src := filepath.Join(genDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join("data", e.Name()), data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "  copy %s: %v\n", e.Name(), err)
			continue
		}
		copied++
	}

	fmt.Printf("genbuiltins: wrote %d builtin files to data/ (%d without docs)\n", copied, missing)
	_ = ok
}
