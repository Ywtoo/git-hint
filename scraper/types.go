package scraper

import "git-hint/internal/core"

// ============================================================================
// Shared types
// ============================================================================

// ProgressState tracks crawl progress for the UI.
type ProgressState struct {
	Total   int
	Current int
}

// Options controls how crawling behaves.
type Options struct {
	MaxDepth   int
	Verbose    bool
	OnProgress func(current, total int, cmd string)
	Progress   *ProgressState
}

// DiscoveredCommand represents a binary found on $PATH or a shell builtin.
type DiscoveredCommand struct {
	Name string
	Path string
}

// ensure DiscoveredCommand satisfies the interface at compile time
var _ = core.CommandMatch{}
