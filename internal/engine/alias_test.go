package engine

import (
	"strings"
	"testing"

	"git-hint/internal/state"
)

// Typing "gst " (an alias for "git status") must continue with git's
// subcommand tree instead of dying in an empty spec tree.
func TestAliasResolvesToUnderlyingCommandTree(t *testing.T) {
	setupTestEnvironment(t)

	state.SetAliases(map[string]string{"gst": "git status"})
	t.Cleanup(func() { state.SetAliases(nil) })

	list, _, err := Suggestions("gst ")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("alias 'gst ' returned no suggestions")
	}

	// The exact underlying tree depends on the spec in setupTestEnvironment;
	// what matters is that SOMETHING from git's level was suggested, not an
	// empty list or an "indexing" spinner.
	found := false
	for _, m := range list {
		if m.Name == "commit" || m.Name == "checkout" || m.Name == "rebase" || m.Name == "push" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected git subcommands behind alias 'gst', got %v", list)
	}
}

// A self-referencing alias (alias ls='ls -lah') must not hang or error.
func TestSelfReferencingAliasDoesNotLoop(t *testing.T) {
	setupTestEnvironment(t)

	state.SetAliases(map[string]string{"ls": "ls -lah"})
	t.Cleanup(func() { state.SetAliases(nil) })

	_, _, err := Suggestions("ls ")
	if err != nil {
		t.Fatalf("self-referencing alias should resolve gracefully, got error: %v", err)
	}
}

// Level 0: typing the alias prefix suggests the alias itself.
func TestAliasSuggestedAtLevelZero(t *testing.T) {
	setupTestEnvironment(t)

	state.SetAliases(map[string]string{"gst": "git status"})
	t.Cleanup(func() { state.SetAliases(nil) })

	list, _, err := Suggestions("gs")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range list {
		if m.Name == "gst" && strings.HasPrefix(m.Description, "alias:") {
			found = true
		}
	}
	if !found {
		t.Errorf("alias 'gst' not suggested at level 0, got %v", list)
	}
}
