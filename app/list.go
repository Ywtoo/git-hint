package app

import (
	"errors"
	"sync"

	"git-hint/engine"
	"git-hint/engine/render"
	"git-hint/engine/tokenizer"
	"git-hint/registry"
	"git-hint/scraper"
	"git-hint/state"
)

// crawling tracks commands currently being indexed on demand, so a burst
// of keystrokes for the same command doesn't launch CrawlOne repeatedly
// while the first one is still running.
var (
	crawlingMu sync.Mutex
	crawling   = make(map[string]bool)
)

func List(buffer string, selected int, promptCol int, renderMode string) string {
	state.SetBuffer(buffer)

	matches, currentToken, err := engine.Suggestions(buffer)
	if errors.Is(err, engine.ErrNotIndexed) {
		return handleNotIndexed(buffer)
	}
	if err != nil {
		return ""
	}

	return render.FormatList(matches, selected, buffer, currentToken, promptCol, renderMode)
}

// handleNotIndexed kicks off a one-time background crawl for a command
// that's known to exist on the system but hasn't had its help tree
// crawled yet. Returns a placeholder immediately; the goroutine crawls
// the complete tree (MaxDepth=999) so level 1 is available fast and
// the rest finishes in background.
func handleNotIndexed(buffer string) string {
	parts := tokenizer.TokenizeBuffer(buffer)
	if len(parts) == 0 {
		return ""
	}
	commandName := parts[0]

	status, err := registry.Status(commandName)
	if err != nil || !status.Known || status.BinPath == "" {
		return ""
	}

	crawlingMu.Lock()
	if crawling[commandName] {
		crawlingMu.Unlock()
		return "⏳ indexing " + commandName + "..."
	}
	crawling[commandName] = true
	crawlingMu.Unlock()

	go func() {
		// Single call: level 1 returns quickly, continues until complete.
		_ = scraper.CrawlOne(commandName, status.BinPath, scraper.Options{MaxDepth: 999})

		crawlingMu.Lock()
		delete(crawling, commandName)
		crawlingMu.Unlock()
	}()

	return "⏳ indexing " + commandName + "..."
}
