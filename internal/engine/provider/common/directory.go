package common

import (
	"os"
	"path/filepath"
	"strings"

	"git-hint/internal/core"
	"git-hint/internal/engine/tokenizer"
	"git-hint/internal/state"
)

// DirectoryProvider completes one directory level at a time relative to the
// shell PWD supplied with the request. It deliberately does not walk the
// whole tree: that would be slow and would return paths the user did not ask
// to traverse.
func DirectoryProvider() []core.CommandMatch {
	workingDir := state.GetWorkingDir()
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}

	tokens := tokenizer.TokenizeBuffer(state.GetBuffer())
	token := ""
	if len(tokens) > 0 {
		token = tokens[len(tokens)-1]
	}

	dir, outputPrefix, namePrefix := directoryContext(workingDir, token)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	result := make([]core.CommandMatch, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasPrefix(strings.ToLower(entry.Name()), strings.ToLower(namePrefix)) {
			continue
		}
		// entry.IsDir() is lstat-based: a symlink to a directory (common for
		// mounts like ~/storage -> /media/...) reports false and silently
		// disappears from the list. Stat follows the link instead.
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := os.Stat(filepath.Join(dir, entry.Name()))
			if err != nil || !resolved.IsDir() {
				continue
			}
		} else if !info.IsDir() {
			continue
		}
		result = append(result, core.CommandMatch{Name: outputPrefix + entry.Name() + "/"})
	}
	return result
}

// directoryContext resolves the directory to list for the current token.
// Kept as a named alias of splitPathToken so each provider reads naturally;
// the implementation is shared to avoid duplicated logic.
func directoryContext(workingDir, token string) (dir, outputPrefix, namePrefix string) {
	return splitPathToken(workingDir, token)
}
