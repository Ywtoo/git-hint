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

	parts := tokenizer.TokenizeBuffer(buffer)
	if len(parts) > 0 && parts[0] != "" {
		maybeCrawlInBackground(parts[0])
	}

	matches, currentToken, err := engine.Suggestions(buffer)
	if errors.Is(err, engine.ErrNotIndexed) {
		return handleNotIndexed(buffer)
	}
	if err != nil {
		return ""
	}

	return render.FormatList(matches, selected, buffer, currentToken, promptCol, renderMode)
}

func maybeCrawlInBackground(commandName string) {
	status, err := registry.Status(commandName)
	if err != nil || !status.Known || status.Indexed || status.BinPath == "" {
		return
	}

	crawlingMu.Lock()
	if crawling[commandName] {
		crawlingMu.Unlock()
		return
	}
	crawling[commandName] = true
	crawlingMu.Unlock()

	go func() {
		defer func() {
			crawlingMu.Lock()
			delete(crawling, commandName)
			crawlingMu.Unlock()
		}()
		_ = scraper.CrawlOne(commandName, status.BinPath, scraper.Options{MaxDepth: 999})
	}()
}

// handleNotIndexed returns a friendly indicator while the background crawl finishes.
func handleNotIndexed(buffer string) string {
	parts := tokenizer.TokenizeBuffer(buffer)
	if len(parts) == 0 {
		return ""
	}
	commandName := parts[0]
	maybeCrawlInBackground(commandName)
	return "⏳ indexing " + commandName + "..."
}
