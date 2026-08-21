package provider_test

import (
	"testing"

	"git-hint/engine/provider"
	"git-hint/state"
)

func TestFlagCheck(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<branch>", "branch"},
		{"<commit>", "commit"},
		{"<msg>", "msg"},
		{"branch", ""},
		{"<", ""},
		{">", ""},
		{"<>", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := provider.FlagCheck(tt.input)
			if got != tt.expected {
				t.Errorf("FlagCheck(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestProvider_UnknownFlag(t *testing.T) {
	state.ClearCache()
	res := provider.Provider("unknown-nonexistent-flag")
	if res != nil {
		t.Errorf("Provider(unknown) = %v, want nil", res)
	}
}

func TestProvider_Caching(t *testing.T) {
	state.ClearCache()

	// First call populates cache
	res1 := provider.Provider("branch")
	cache := state.LoadCache()
	if cache.CurrentPlaceholder != "branch" {
		t.Errorf("cache.CurrentPlaceholder = %q, want 'branch'", cache.CurrentPlaceholder)
	}

	// Second call returns cached slice directly
	res2 := provider.Provider("branch")
	if len(res1) != len(res2) {
		t.Errorf("Provider cache mismatch: %d vs %d", len(res1), len(res2))
	}
}
