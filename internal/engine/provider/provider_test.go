package provider_test

import (
	"os"
	"path/filepath"
	"testing"

	"git-hint/internal/engine/history"
	"git-hint/internal/engine/provider"
	"git-hint/internal/state"
)

func TestResolveDirFlag(t *testing.T) {
	state.ClearCache()
	state.SetBuffer("source sc")
	state.SetWorkingDir("/tmp/proj")
	t.Cleanup(func() { state.SetBuffer("") })

	if got := provider.ResolveDirFlag("file-path"); got != "" {
		// "sc" has no slash: relative to the working dir (no dir part).
		t.Errorf("ResolveDirFlag(no slash) = %q, want empty (cwd-relative)", got)
	}

	state.SetBuffer("source scripts/de")
	if got := provider.ResolveDirFlag("file-path"); got != filepath.Join("/tmp/proj", "scripts/") {
		t.Errorf("ResolveDirFlag(scripts/de) = %q, want /tmp/proj/scripts", got)
	}

	state.SetBuffer("source /etc/ho")
	if got := provider.ResolveDirFlag("file-path"); got != "/etc" {
		t.Errorf("ResolveDirFlag(abs) = %q, want /etc", got)
	}

	state.SetBuffer("git commit -m x")
	if got := provider.ResolveDirFlag("file-path"); got != "" {
		t.Errorf("ResolveDirFlag(no slash in last token) = %q, want \"\"", got)
	}
}

func TestFilePathProviderListsFilesAndDirs(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "setup.sh"), []byte("#!/bin/sh\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(tmp, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}

	// Hermetic history: an empty file so directory assertions are stable.
	emptyHist := filepath.Join(t.TempDir(), "hist")
	if err := os.WriteFile(emptyHist, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	restore := history.SetHistoryPathForTest(func() (string, error) { return emptyHist, nil })
	t.Cleanup(restore)

	state.ClearCache()
	state.SetWorkingDir(tmp)
	state.SetBuffer("source ")
	t.Cleanup(func() {
		state.SetBuffer("")
		state.SetWorkingDir("")
	})

	got := provider.Provider("file-path")

	var names []string
	for _, m := range got {
		names = append(names, m.Name)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 entries (dir+file), got %v", names)
	}
	// Directories sort before files.
	if names[0] != "scripts/" {
		t.Errorf("expected directory first with trailing slash, got %v", names)
	}
	if names[1] != "setup.sh" {
		t.Errorf("expected setup.sh, got %v", names)
	}

	// Typing filters within the same level.
	state.SetBuffer("source se")
	got = provider.Provider("file-path")
	if len(got) != 1 || got[0].Name != "setup.sh" {
		t.Errorf("typing filter failed, got %v", got)
	}
}

func TestFilePathProviderHistoryFirst(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "fresh.sh"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// History contains two sourced paths; the most recent goes first even
	// though it is not in the current directory.
	hist := filepath.Join(t.TempDir(), "hist")
	content := ": 1700000000:0;source /old/env.sh\n: 1700000100:0;source " + filepath.Join(tmp, "sourced-antes.sh") + "\n"
	if err := os.WriteFile(hist, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	restore := history.SetHistoryPathForTest(func() (string, error) { return hist, nil })
	t.Cleanup(restore)

	state.ClearCache()
	state.SetWorkingDir(tmp)
	state.SetBuffer("source ")
	t.Cleanup(func() {
		state.SetBuffer("")
		state.SetWorkingDir("")
	})

	got := provider.Provider("file-path")
	if len(got) < 3 {
		t.Fatalf("expected history entries + directory entry, got %v", got)
	}
	// History paths come first, ordered by usage then name — both appear
	// once each; assert the pair, not the internal order.
	first := got[0].Name
	second := got[1].Name
	pair := map[string]bool{first: true, second: true}
	if !pair[filepath.Join(tmp, "sourced-antes.sh")] || !pair["/old/env.sh"] {
		t.Errorf("history-first violated: first two = %q, %q", first, second)
	}
	// ...and fresh directory files after the history block.
	found := false
	for _, m := range got[2:] {
		if m.Name == "fresh.sh" {
			found = true
		}
	}
	if !found {
		t.Errorf("directory listing missing after history block: %v", got)
	}
}
