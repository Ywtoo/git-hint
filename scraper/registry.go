package scraper

import (
	"fmt"
	"sync"

	"git-hint/core"
)

// maxConcurrentCrawls bounds how many "-h" subprocesses run at once
// during the eager-crawl phase. Bounded because each exec is I/O-bound
// (waiting on the subprocess), so a modest pool speeds things up a lot
// without hammering the machine like an unbounded fan-out would.
const maxConcurrentCrawls = 8

// Rebuild discovers every command on the system ($PATH binaries + zsh
// builtins) and writes a full index.json (name + origin path, no exec
// involved — just directory listing). It then eagerly crawls the help
// tree only for commands in the common allowlist (see allowlist.go);
// everything else is left indexed-but-uncrawled and gets its tree filled
// in later, on demand, via CrawlOne — see daemon's lookup path.
//
// This keeps a fresh install/rebuild fast (seconds) instead of exec'ing
// -h against every single binary on $PATH, which can be thousands of
// commands and take a very long time (some of which may hang waiting on
// stdin, e.g. REPLs invoked with unexpected flags).
func Rebuild(opts Options) error {
	if err := ResetDataDir(); err != nil {
		return fmt.Errorf("failed to reset data dir: %w", err)
	}
	index := make(map[string]core.CommandMatch)

	commands := DiscoverPathBinaries()
	commands = append(commands, DiscoverZshBuiltins()...)

	// Phase 1: index everything, no subprocess calls.
	for _, cmd := range commands {
		index[cmd.Name] = core.CommandMatch{Name: cmd.Name, Path: cmd.Path}
	}
	if err := WriteIndex(index); err != nil {
		return err
	}

	// Phase 2: eagerly crawl only the allowlist, concurrently.
	var toCrawl []DiscoveredCommand
	for _, cmd := range commands {
		if cmd.Path != "" && IsCommon(cmd.Name) {
			toCrawl = append(toCrawl, cmd)
		}
	}

	opts.Progress = &ProgressState{Total: len(toCrawl)}

	var (
		wg       sync.WaitGroup
		sem      = make(chan struct{}, maxConcurrentCrawls)
		progMu   sync.Mutex
		visited  = make(map[string]bool)
		visitedM sync.Mutex
	)

	for _, cmd := range toCrawl {
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

	return nil
}
