package app

import (
	"os"
	"path/filepath"
	"testing"

	"git-hint/internal/core"
	"git-hint/internal/registry"
)

func setupListTestEnvironment(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// "git" has a binary path (crawlable); "githintfakecmd" has no path and
	// is not a real zsh builtin, so it can never be crawled — this is the
	// eternal-spinner regression. "cd" is a real zsh builtin (crawlable via
	// its documentation).
	indexJSON := `{
		"git": {"name": "git", "path": "/usr/bin/git"},
		"githintfakecmd": {"name": "githintfakecmd"},
		"cd": {"name": "cd"}
	}`
	if err := os.WriteFile(filepath.Join(dataDir, "index.json"), []byte(indexJSON), 0644); err != nil {
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

// A no-path command that zsh does not know as a builtin can never be
// crawled, so handleNotIndexed must return empty instead of a spinner
// that would never terminate.
func TestHandleNotIndexed_BuiltinNeverShowsSpinner(t *testing.T) {
	setupListTestEnvironment(t)

	got := handleNotIndexed("githintfakecmd")
	if got != "" {
		t.Errorf("non-zsh no-path command should not show a spinner, got %q", got)
	}
}

// A PATH binary that is known but not yet indexed keeps the friendly
// "indexing..." indicator while its background crawl runs.
func TestHandleNotIndexed_CrawlableShowsSpinner(t *testing.T) {
	setupListTestEnvironment(t)

	got := handleNotIndexed("git")
	want := "⏳ indexing git..."
	if got != want {
		t.Errorf("expected %q for crawlable command, got %q", want, got)
	}
}

// isCrawlable mirrors registry.Status + zsh builtin membership: PATH
// binaries and real zsh builtins are crawlable; the rest is not.
func TestIsCrawlable(t *testing.T) {
	setupListTestEnvironment(t)

	// "cd" is a real zsh builtin (present in zsh -c 'print -l ${(k)builtins}'),
	// so it is crawlable via documentation even without a binary path.
	if !isCrawlable("cd") {
		t.Error("zsh builtin 'cd' must be crawlable via its documentation")
	}
	if isCrawlable("githintfakecmd") {
		t.Error("no-path non-builtin must not be crawlable")
	}
	if isCrawlable("unknown-command") {
		t.Error("unknown command must not be crawlable")
	}
	if !isCrawlable("git") {
		t.Error("'git' has a binary path and must be crawlable")
	}
}
