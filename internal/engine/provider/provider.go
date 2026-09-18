package provider

import (
	"os"
	"path/filepath"
	"strings"

	"git-hint/internal/core"
	"git-hint/internal/engine/provider/common"
	"git-hint/internal/engine/provider/git"
	"git-hint/internal/engine/tokenizer"
	"git-hint/internal/state"
)

// Providers maps placeholder names to completion providers.
//
// Only placeholders listed here have an implemented provider.
// Any other placeholder falls back to the default behavior,
// allowing the user to type the value manually.
var Providers = map[string]func() []core.CommandMatch{

	// ---------------------------------------------------------------------
	// Git references
	// ---------------------------------------------------------------------

	"branch":     git.BranchProvider,
	"old-branch": git.BranchProvider,

	"commit":     git.CommitProvider,
	"commit-ish": git.CommitProvider,
	"tree-ish":   git.CommitProvider,
	"head":       git.CommitProvider,

	"ref":     git.RefProvider,
	"refname": git.RefProvider,

	"upstream": git.UpstreamProvider,

	"remote": git.RemoteProvider,
	"stash":  git.StashProvider,
	"tag":    git.TagProvider,

	// ---------------------------------------------------------------------
	// Files
	// ---------------------------------------------------------------------

	"file":           git.TrackedFileProvider,
	"path":           git.TrackedFileProvider,
	"file-staged":    func() []core.CommandMatch { return git.FileProvider("file-staged") },
	"file-modified":  func() []core.CommandMatch { return git.FileProvider("file-modified") },
	"file-untracked": func() []core.CommandMatch { return git.FileProvider("file-untracked") },
	"author":         git.AuthorProvider,
	"conflict":       git.ConflictProvider,
	"config":         git.ConfigProvider,

	// ---------------------------------------------------------------------
	// Message
	// ---------------------------------------------------------------------

	// Free text with automatic quoting.
	"msg":     common.MsgProvider,
	"message": common.MsgProvider,

	// ---------------------------------------------------------------------
	// Free text
	// ---------------------------------------------------------------------

	"url":         common.FreeTextProvider,
	"name":        common.FreeTextProvider,
	"new-branch":  common.FreeTextProvider,
	"branch-name": common.FreeTextProvider,
	"target":      common.MakeTargetProvider,
	"script":      common.NPMScriptProvider,
	"dir":         common.DirectoryProvider,

	// ---------------------------------------------------------------------
	// Shell builtins
	// ---------------------------------------------------------------------

	// source / . complete any filesystem path: files from the working
	// directory (or the directory being typed), not git-tracked files only.
	"file-path": common.FilePathProvider,
}

// LookupProvider returns the provider function registered for a placeholder
// flag and whether one exists. Placeholders without a provider fall back to
// free text instead of rendering a dead-end group.
func LookupProvider(flag string) (func() []core.CommandMatch, bool) {
	fn, ok := Providers[flag]
	return fn, ok
}

// ResolveDirFlag reports the directory a path provider should read for the
// given placeholder flag, derived from the current buffer's last token.
// Returns "" when the token carries no directory part (complete relative to
// the working directory).
func ResolveDirFlag(flag string) string {
	if flag != "file-path" && flag != "file" && flag != "path" {
		return ""
	}
	parts := tokenizer.TokenizeBuffer(state.GetBuffer())
	if len(parts) == 0 {
		return ""
	}
	token := parts[len(parts)-1]
	idx := strings.LastIndex(token, "/")
	if idx < 0 {
		return ""
	}
	dir := token[:idx+1]
	if dir == "" {
		return ""
	}
	if strings.HasPrefix(dir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(dir, "~/"))
		}
	}
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	if wd := state.GetWorkingDir(); wd != "" {
		return filepath.Join(wd, dir)
	}
	return dir
}

// Provider returns the suggestion list for a placeholder flag, consulting
// the session cache when the flag's results are context-independent.
func Provider(flag string) []core.CommandMatch {
	// Directory contents depend on both PWD and the token being typed. Caching
	// this provider by placeholder alone makes "cd st" and "cd storage/"
	// incorrectly reuse the first result.
	if flag == "dir" {
		if fn, ok := Providers[flag]; ok {
			return fn()
		}
		return nil
	}
	// file-path depends on the token being typed the same way "dir" does —
	// "source sc" and "source scripts/" must never share a cached result.
	if flag == "file-path" {
		if fn, ok := Providers[flag]; ok {
			return fn()
		}
		return nil
	}
	cache := state.LoadCache()

	cacheKey := flag
	if flag == "target" || flag == "script" {
		cacheKey = flag + "|" + state.GetWorkingDir()
	}
	if flag == "msg" || flag == "message" || flag == "name" || flag == "url" {
		// These flags depend on the full command context (e.g. "git remote add"),
		// not just the placeholder name — so we key the cache by context too.
		parts := tokenizer.TokenizeBuffer(state.GetBuffer())
		if len(parts) > 1 {
			cacheKey = flag + "|" + strings.Join(parts[:len(parts)-1], " ")
		}
	}

	if cache.CurrentPlaceholder == cacheKey {
		return cache.ExpandedList
	}

	var results []core.CommandMatch

	fn, ok := Providers[flag]
	if ok {
		results = fn()
	}

	cache.CurrentPlaceholder = cacheKey
	cache.ExpandedList = results
	state.SaveCache(cache)

	return results
}
