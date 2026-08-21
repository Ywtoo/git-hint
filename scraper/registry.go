package scraper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"git-hint/core"
	"git-hint/engine/ranking"
)

// maxConcurrentCrawls bounds how many "-h" subprocesses run at once
// during the eager-crawl phase. Bounded because each exec is I/O-bound
// (waiting on the subprocess), so a modest pool speeds things up a lot
// without hammering the machine like an unbounded fan-out would.
const maxConcurrentCrawls = 8

// BuildIndex discovers all binaries in $PATH and zsh builtins, writes index.json,
// and returns the index map.
func BuildIndex() (map[string]core.CommandMatch, error) {
	index := make(map[string]core.CommandMatch)

	commands := DiscoverPathBinaries()
	commands = append(commands, DiscoverZshBuiltins()...)

	for _, cmd := range commands {
		index[cmd.Name] = core.CommandMatch{Name: cmd.Name, Path: cmd.Path}
	}

	if err := WriteIndex(index); err != nil {
		return nil, err
	}
	return index, nil
}

// Warmup runs on the very first execution of git-hint (when index.json does not exist).
// It creates index.json with all system commands and eagerly crawls the top 10
// most frequently used root commands from the user's shell history.
func Warmup(opts Options) error {
	indexPath, err := core.IndexPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(indexPath); err == nil {
		// index.json already exists: warmup already ran before.
		return nil
	}

	index, err := BuildIndex()
	if err != nil {
		return fmt.Errorf("warmup failed to build index: %w", err)
	}

	var candidates []core.CommandMatch
	for _, match := range index {
		if match.Path != "" {
			candidates = append(candidates, match)
		}
	}

	ranked, err := ranking.RankSuggestions("", candidates)
	if err != nil {
		return fmt.Errorf("warmup failed to rank history commands: %w", err)
	}

	var topCommands []DiscoveredCommand
	for _, m := range ranked {
		if m.NUsed == 0 {
			break
		}
		if entry, ok := index[m.Name]; ok && entry.Path != "" {
			topCommands = append(topCommands, DiscoveredCommand{Name: m.Name, Path: entry.Path})
		}
		if len(topCommands) >= 10 {
			break
		}
	}

	if len(topCommands) == 0 {
		return nil
	}

	crawlCommandsConcurrently(topCommands, opts)
	return nil
}

// Rebuild refreshes index.json (finding new or removed binaries) and recrawls
// all commands that were previously crawled (i.e. those with an existing JSON file in data/).
func Rebuild(opts Options) error {
	dir, err := core.DataDir()
	if err != nil {
		return err
	}

	// Identify commands that already had a crawled json file
	existingCrawled := make(map[string]bool)
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" && e.Name() != "index.json" {
				cmdName := strings.TrimSuffix(e.Name(), ".json")
				existingCrawled[cmdName] = true
			}
		}
	}

	index, err := BuildIndex()
	if err != nil {
		return fmt.Errorf("failed to rebuild index: %w", err)
	}

	var toCrawl []DiscoveredCommand
	for cmdName := range existingCrawled {
		if entry, ok := index[cmdName]; ok && entry.Path != "" {
			toCrawl = append(toCrawl, DiscoveredCommand{Name: cmdName, Path: entry.Path})
		}
	}

	// If no commands were crawled yet, fallback to top commands from history
	if len(toCrawl) == 0 {
		var candidates []core.CommandMatch
		for _, match := range index {
			if match.Path != "" {
				candidates = append(candidates, match)
			}
		}
		if ranked, err := ranking.RankSuggestions("", candidates); err == nil {
			for _, m := range ranked {
				if m.NUsed == 0 {
					break
				}
				if entry, ok := index[m.Name]; ok && entry.Path != "" {
					toCrawl = append(toCrawl, DiscoveredCommand{Name: m.Name, Path: entry.Path})
				}
				if len(toCrawl) >= 10 {
					break
				}
			}
		}
	}

	crawlCommandsConcurrently(toCrawl, opts)
	return nil
}

func crawlCommandsConcurrently(commands []DiscoveredCommand, opts Options) {
	opts.Progress = &ProgressState{Total: len(commands)}

	var (
		wg       sync.WaitGroup
		sem      = make(chan struct{}, maxConcurrentCrawls)
		progMu   sync.Mutex
		visited  = make(map[string]bool)
		visitedM sync.Mutex
	)

	for _, cmd := range commands {
		wg.Add(1)
		sem <- struct{}{}
		go func(cmd DiscoveredCommand) {
			defer wg.Done()
			defer func() { <-sem }()

			visitedM.Lock()
			localVisited := make(map[string]bool, len(visited))
			for k, v := range visited {
				localVisited[k] = v
			}
			visitedM.Unlock()

			tree := crawlCommand([]string{cmd.Path}, opts, localVisited)
			_ = WriteCommand(cmd.Name, tree.SubCommand)

			progMu.Lock()
			opts.Progress.Current++
			if opts.OnProgress != nil {
				opts.OnProgress(opts.Progress.Current, opts.Progress.Total, cmd.Name)
			}
			progMu.Unlock()
		}(cmd)
	}
	wg.Wait()
}
