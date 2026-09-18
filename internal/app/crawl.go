package app

import (
	"os/exec"
	"strings"
	"sync"

	"git-hint/internal/engine/tokenizer"
	"git-hint/internal/registry"
	"git-hint/scraper"
)

var (
	crawlingMu sync.Mutex
	crawling   = make(map[string]bool)
)

func maybeCrawlInBackground(commandName string) {
	status, err := registry.Status(commandName)
	if err != nil || !status.Known || status.Indexed {
		return
	}
	if status.BinPath == "" && !isZshBuiltin(commandName) {
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
		defer func() { crawlingMu.Lock(); delete(crawling, commandName); crawlingMu.Unlock() }()
		_ = scraper.CrawlOne(commandName, status.BinPath, scraper.Options{MaxDepth: 999})
	}()
}

func handleNotIndexed(buffer string) string {
	parts := tokenizer.TokenizeBuffer(buffer)
	if len(parts) == 0 {
		return ""
	}
	commandName := parts[0]
	if isCrawlable(commandName) {
		maybeCrawlInBackground(commandName)
		return "⏳ indexing " + commandName + "..."
	}
	return ""
}

func isCrawlable(commandName string) bool {
	status, err := registry.Status(commandName)
	if err != nil || !status.Known {
		return false
	}
	return status.BinPath != "" || isZshBuiltin(commandName)
}

var (
	zshBuiltinsOnce sync.Once
	zshBuiltins     = make(map[string]bool)
)

func isZshBuiltin(commandName string) bool {
	status, err := registry.Status(commandName)
	if err != nil || !status.Known || status.BinPath != "" {
		return false
	}
	zshBuiltinsOnce.Do(func() {
		out, err := exec.Command("zsh", "-c", "print -l ${(k)builtins}").Output()
		if err != nil {
			return
		}
		for _, name := range strings.Fields(string(out)) {
			zshBuiltins[name] = true
		}
	})
	return zshBuiltins[commandName]
}
