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
		return ""
	}
	if err != nil {
		return ""
	}

	return render.FormatList(matches, selected, buffer, currentToken, promptCol, renderMode)
}

// TODO:Fix this is not working at all
// handleNotIndexed kicks off a one-time background crawl for a command
// that's known to exist on the system but hasn't had its help tree
// crawled yet, and returns an immediate response the zsh-plugin can show
// while that happens. The crawl itself runs in a goroutine so it never
// blocks the daemon's response to this keystroke.
func handleNotIndexed(buffer string) string {
	parts := tokenizer.TokenizeBuffer(buffer)
	if len(parts) == 0 {
		return ""
	}
	commandName := parts[0]

	status, err := registry.Status(commandName)
	if err != nil || !status.Known || status.BinPath == "" {
		return "" // shouldn't happen if engine already said Known+!Indexed, but be safe
	}

	crawlingMu.Lock()
	alreadyCrawling := crawling[commandName]
	if !alreadyCrawling {
		crawling[commandName] = true
	}
	crawlingMu.Unlock()

	if !alreadyCrawling {
		go func() {
			_ = scraper.CrawlOne(commandName, status.BinPath, scraper.Options{MaxDepth: 4})

			crawlingMu.Lock()
			delete(crawling, commandName)
			crawlingMu.Unlock()
		}()
	}

	return "⏳ indexando " + commandName + "..."
}
