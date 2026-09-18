package scraper

import "sync"

// ============================================================================
// Builtin crawl failure cache
//
// Builtins whose documentation could not be found are recorded here so
// repeated keystrokes don't re-run the lookup forever (e.g. on Windows,
// which has neither bash help nor zsh HELPDIR files).
// ============================================================================

var (
	builtinCrawlFailedMu sync.Mutex
	builtinCrawlFailed   = make(map[string]bool)
)

func builtinMarkFailed(name string) {
	builtinCrawlFailedMu.Lock()
	builtinCrawlFailed[name] = true
	builtinCrawlFailedMu.Unlock()
}

func builtinIsFailed(name string) bool {
	builtinCrawlFailedMu.Lock()
	defer builtinCrawlFailedMu.Unlock()
	return builtinCrawlFailed[name]
}
