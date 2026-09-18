package state

import (
	"os"
	"testing"

	"git-hint/internal/core"
)

func TestSessionCache(t *testing.T) {
	// Override cachePath for testing
	originalPath := cachePath
	tmpFile, err := os.CreateTemp("", "githint_cache_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cachePath = tmpFile.Name()
	defer func() { cachePath = originalPath }()

	// Test 1: Save and Load
	originalCache := &SessionCache{
		CurrentPlaceholder: "branch",
		ExpandedList: []core.CommandMatch{
			{Name: "main"},
			{Name: "develop"},
		},
		LastInput: "git checkout ",
	}

	SaveCache(originalCache)
	loadedCache := LoadCache()

	if loadedCache.CurrentPlaceholder != originalCache.CurrentPlaceholder {
		t.Errorf("Expected placeholder %s, got %s", originalCache.CurrentPlaceholder, loadedCache.CurrentPlaceholder)
	}

	if len(loadedCache.ExpandedList) != len(originalCache.ExpandedList) {
		t.Errorf("Expected list length %d, got %d", len(originalCache.ExpandedList), len(loadedCache.ExpandedList))
	}

	if loadedCache.LastInput != originalCache.LastInput {
		t.Errorf("Expected last input %s, got %s", originalCache.LastInput, loadedCache.LastInput)
	}

	// Test 2: Clear Cache
	ClearCache()
	clearedCache := LoadCache()
	if clearedCache.CurrentPlaceholder != "" {
		t.Errorf("Expected empty placeholder after ClearCache, got %s", clearedCache.CurrentPlaceholder)
	}

}
