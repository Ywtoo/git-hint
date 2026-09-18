package scraper_test

import (
	"encoding/json"
	"os"
	"testing"

	"git-hint/internal/core"
	"git-hint/scraper"
)

func TestCrawlOne_GitStructure(t *testing.T) {
	tempDir := t.TempDir()
	cleanup := core.SetExecutablePathForTest(func() (string, error) {
		return tempDir + "/test-bin", nil
	})
	t.Cleanup(cleanup)

	opts := scraper.Options{
		MaxDepth: 1, // Only root + immediate level (e.g. git commit)
		Verbose:  false,
	}

	err := scraper.CrawlOne("git", "/usr/bin/git", opts)
	if err != nil {
		t.Fatalf("CrawlOne failed: %v", err)
	}

	path, err := core.CommandDataPath("git")
	if err != nil {
		t.Fatalf("Failed to get command data path: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read generated git.json: %v", err)
	}

	var tree map[string]core.CommandMatch
	if err := json.Unmarshal(data, &tree); err != nil {
		t.Fatalf("Failed to unmarshal git.json: %v", err)
	}

	commitCmd, ok := tree["commit"]
	if !ok {
		t.Fatalf("git.json missing 'commit' command")
	}

	t.Logf("Commit description: %s", commitCmd.Description)
	t.Logf("Commit subCommand count: %d", len(commitCmd.SubCommand))
	t.Logf("Commit options count: %d", len(commitCmd.Options))

	if len(commitCmd.Options) == 0 {
		t.Errorf("Expected commitCmd.Options to be populated with flags, but was empty!")
	}

	// Check if -m is registered in Options and has requires
	if mFlag, found := commitCmd.Options["-m"]; found {
		t.Logf("Found -m flag: kind=%s, requires=%v", mFlag.Kind, mFlag.Requires)
		if mFlag.Kind != core.KindOption {
			t.Errorf("Expected -m kind to be 'option', got %s", mFlag.Kind)
		}
	} else {
		t.Errorf("-m flag not found in commitCmd.Options!")
	}
}
