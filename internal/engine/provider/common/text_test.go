package common_test

import (
	"os"
	"testing"

	"git-hint/internal/engine/history"
	"git-hint/internal/engine/provider/common"
	"git-hint/internal/state"
)

func setupTestHistory(t *testing.T, historyContent string) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "zsh_history_common_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	if _, err := tmpFile.Write([]byte(historyContent)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cleanup := history.SetHistoryPathForTest(func() (string, error) {
		return tmpFile.Name(), nil
	})
	t.Cleanup(cleanup)
}

func TestMsgProvider(t *testing.T) {
	content := ": 1712345678:0;git commit -m \"feat: double quotes\"\n" +
		": 1712345679:0;git commit -m 'fix: single quotes'\n" +
		": 1712345680:0;git commit -m plain unquoted message\n" +
		": 1712345681:0;git commit -m \"unclosed quote message\n" +
		": 1712345682:0;git commit --amend -m \"docs: other command\"\n"

	setupTestHistory(t, content)

	t.Run("Single token buffer returns nil", func(t *testing.T) {
		state.SetBuffer("git")
		matches := common.MsgProvider()
		if matches != nil {
			t.Errorf("MsgProvider() = %v, want nil", matches)
		}
	})

	t.Run("No message flag in buffer returns nil", func(t *testing.T) {
		state.SetBuffer("git commit ")
		matches := common.MsgProvider()
		if matches != nil {
			t.Errorf("MsgProvider() = %v, want nil when no -m flag", matches)
		}
	})

	t.Run("Extracts messages and always ensures double quotes at start and end", func(t *testing.T) {
		state.SetBuffer("git commit -m ")
		matches := common.MsgProvider()
		// Expects 5: unclosed, plain, fix, feat, docs (--amend also matches base "git commit")
		if len(matches) != 5 {
			t.Fatalf("MsgProvider() len = %d, want 5\n  got: %v", len(matches), matches)
		}

		for i, m := range matches {
			got := m.Name
			if len(got) < 2 || got[0] != '"' || got[len(got)-1] != '"' {
				t.Errorf("matches[%d].Name (%q) is not properly wrapped with double quotes", i, got)
			}
		}
	})

	t.Run("Interleaved flags: git commit -a -m finds same messages as git commit -m", func(t *testing.T) {
		state.SetBuffer("git commit -m ")
		matchesNormal := common.MsgProvider()

		state.SetBuffer("git commit -a -m ")
		matchesInterleaved := common.MsgProvider()

		if len(matchesNormal) != len(matchesInterleaved) {
			t.Errorf("interleaved flags: got %d matches, want %d (same as without -a)",
				len(matchesInterleaved), len(matchesNormal))
		}

		state.SetBuffer("git commit -a --all -m")
		matchesMulti := common.MsgProvider()
		if len(matchesNormal) != len(matchesMulti) {
			t.Errorf("multi-flag without space: got %d matches, want %d",
				len(matchesMulti), len(matchesNormal))
		}
	})

	t.Run("Matches combined -am flag and attached quotes from history", func(t *testing.T) {
		extraContent := ": 1712345683:0;git commit -am \"feat: combined flag\"\n" +
			": 1712345684:0;git commit -m\"attached quotes\"\n" +
			": 1712345685:0;git commit --message=\"equals flag\"\n"
		setupTestHistory(t, extraContent)

		state.SetBuffer("git commit -m ")
		matches := common.MsgProvider()
		if len(matches) != 3 {
			t.Fatalf("expected 3 matches from extra history, got %d: %v", len(matches), matches)
		}
	})
}

func TestFreeTextProvider(t *testing.T) {
	content := ": 1712345678:0;git remote add origin https://github.com/foo/bar.git\n" +
		": 1712345679:0;git remote add upstream https://github.com/foo/bar.git\n" +
		": 1712345680:0;git remote add origin https://github.com/another/repo.git\n"

	setupTestHistory(t, content)

	t.Run("Single token buffer returns nil", func(t *testing.T) {
		state.SetBuffer("git")
		matches := common.FreeTextProvider()
		if matches != nil {
			t.Errorf("FreeTextProvider() = %v, want nil", matches)
		}
	})

	t.Run("Extracts next token and deduplicates", func(t *testing.T) {
		state.SetBuffer("git remote add ")
		matches := common.FreeTextProvider()
		if len(matches) != 2 {
			t.Fatalf("FreeTextProvider() len = %d, want 2", len(matches))
		}

		// Most recent appears first: origin, upstream
		if matches[0].Name != "origin" {
			t.Errorf("matches[0].Name = %q, want 'origin'", matches[0].Name)
		}
		if matches[1].Name != "upstream" {
			t.Errorf("matches[1].Name = %q, want 'upstream'", matches[1].Name)
		}
	})
}
