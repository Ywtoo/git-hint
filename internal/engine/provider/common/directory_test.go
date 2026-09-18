package common_test

import (
	"os"
	"path/filepath"
	"testing"

	"git-hint/internal/core"
	"git-hint/internal/engine/provider/common"
	"git-hint/internal/state"
)

func TestDirectoryProviderCompletesOneLevelFromShellDirectory(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"alpha/nested", "beta"} {
		if err := os.MkdirAll(filepath.Join(root, path), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "not-a-directory"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	state.SetWorkingDir(root)
	t.Cleanup(func() { state.SetWorkingDir("") })

	state.SetBuffer("cd ")
	if got := directoryNames(common.DirectoryProvider()); !sameStrings(got, []string{"alpha/", "beta/"}) {
		t.Fatalf("root completion = %v, want [alpha/ beta/]", got)
	}

	state.SetBuffer("cd AL")
	if got := directoryNames(common.DirectoryProvider()); !sameStrings(got, []string{"alpha/"}) {
		t.Fatalf("case-insensitive completion = %v, want [alpha/]", got)
	}

	state.SetBuffer("cd alpha/")
	if got := directoryNames(common.DirectoryProvider()); !sameStrings(got, []string{"alpha/nested/"}) {
		t.Fatalf("nested completion = %v, want [alpha/nested/]", got)
	}
}

func directoryNames(matches []core.CommandMatch) []string {
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match.Name)
	}
	return names
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
