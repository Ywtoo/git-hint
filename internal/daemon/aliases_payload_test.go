package daemon

import "testing"

func TestParseAliasesPayload(t *testing.T) {
	// `alias -L` output style: definitions arrive quoted.
	got := parseAliasesPayload("gs\t'git status'\tll\t\"ls -lah\"")
	if len(got) != 2 {
		t.Fatalf("expected 2 aliases, got %v", got)
	}
	if got["gs"] != "git status" {
		t.Errorf("gs = %q, want \"git status\"", got["gs"])
	}
	if got["ll"] != "ls -lah" {
		t.Errorf("ll = %q, want \"ls -lah\"", got["ll"])
	}
}

func TestParseAliasesPayloadEmptyAndOdd(t *testing.T) {
	if got := parseAliasesPayload(""); len(got) != 0 {
		t.Errorf("empty payload should yield empty map, got %v", got)
	}
	// Odd trailing token (name without definition) is ignored.
	got := parseAliasesPayload("gs\tgit status\tbroken")
	if len(got) != 1 {
		t.Errorf("odd payload should ignore dangling name, got %v", got)
	}
}
