package provider

import (
	"strings"

	"git-hint/core"
	"git-hint/engine/provider/common"
	"git-hint/engine/provider/git"
	"git-hint/engine/tokenizer"
	"git-hint/state"
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

	"file": git.TrackedFileProvider,
	"path": git.TrackedFileProvider,

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
	"author":      common.FreeTextProvider,
}

func FlagCheck(flag string) string {
	if len(flag) >= 2 && flag[0] == '<' && flag[len(flag)-1] == '>' {
		return flag[1 : len(flag)-1]
	}
	return ""
}

func Provider(flag string) []core.CommandMatch {
	cache := state.LoadCache()

	cacheKey := flag
	if flag == "msg" || flag == "name" || flag == "url" {
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
