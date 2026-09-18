package scraper

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"git-hint/internal/core"
	"git-hint/scraper/parser"
)

// ============================================================================
// Core crawling logic
//
// CrawlOne is the public entry point for crawling a single command.
// crawlCommand recursively builds the command tree by executing help
// flags and parsing the output.
// ============================================================================

// CrawlOne crawls a single command and writes its tree to data/<name>.json.
// If path is empty, the command is a shell builtin and is sourced from
// documentation instead of running --help.
func CrawlOne(name, path string, opts Options) error {
	if path == "" {
		return CrawlBuiltin(name)
	}

	visited := make(map[string]bool)
	tree := crawlCommand([]string{path}, opts, visited)
	return WriteCommand(name, tree.SubCommand)
}

// isTerminalToken reports whether a token is a flag or placeholder
// (leaf nodes that don't trigger further help invocations).
func isTerminalToken(name string) bool {
	return strings.HasPrefix(name, "-") || strings.HasPrefix(name, "<")
}

// crawlCommand recursively builds the command tree for a command at the
// given path. It executes help flags, parses the output, and recursively
// crawls subcommands.
func crawlCommand(path []string, opts Options, visited map[string]bool) core.CommandMatch {
	node := core.NewCommandMatch("", "")

	key := strings.Join(path, " ")
	if visited[key] {
		return node
	}
	visited[key] = true

	if isTerminalToken(path[len(path)-1]) {
		return node
	}

	depth := len(path) - 1
	if depth > opts.MaxDepth {
		return node
	}

	// Execute help and measure duration
	start := time.Now()
	var output string
	var err error
	var timedOut bool
	if depth == 0 {
		output, err, timedOut = runWithTimeout(30*time.Second, func() (string, error) {
			return executeRootHelp(path[0])
		})
	} else {
		output, err, timedOut = runWithTimeout(30*time.Second, func() (string, error) {
			return executeHelp(path)
		})
	}
	recordDuration(key, time.Since(start))

	if timedOut {
		fmt.Println("  TIMEOUT (30s):", key)
		return node
	}
	if err != nil {
		if opts.Verbose {
			fmt.Println("  error:", err)
		}
		return node
	}

	// Parse the help output
	namePath := append([]string{filepath.Base(path[0])}, path[1:]...)
	parsed := parser.ParseCommand(output, namePath)
	if parsed.IsLeaf() {
		return node
	}

	// Recursively crawl subcommands concurrently
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, 8)
	)

	for subName, pNode := range parsed.Nodes {
		wg.Add(1)
		sem <- struct{}{}
		go func(subName string, pNode *parser.ParsedNode) {
			defer wg.Done()
			defer func() { <-sem }()

			childPath := append(append([]string{}, path...), subName)

			var child core.CommandMatch
			if isTerminalToken(subName) {
				child = buildTerminalChild(pNode)
			} else {
				mu.Lock()
				localVisited := make(map[string]bool, len(visited))
				for k, v := range visited {
					localVisited[k] = v
				}
				mu.Unlock()

				child = crawlCommand(childPath, opts, localVisited)
				child.Description = pNode.Description
			}

			child.Kind = core.NodeKind(pNode.Kind)
			child.Requires = pNode.Requires

			mu.Lock()
			attachChildNode(&node, subName, child)
			mu.Unlock()
		}(subName, pNode)
	}
	wg.Wait()

	return node
}

// ============================================================================
// Node construction helpers
// ============================================================================

// buildTerminalChild constructs a leaf CommandMatch for placeholders
// and terminal options.
func buildTerminalChild(pNode *parser.ParsedNode) core.CommandMatch {
	child := core.NewCommandMatch("", "")
	child.Description = pNode.Description
	child.Kind = core.NodeKind(pNode.Kind)
	child.Requires = pNode.Requires

	for chName, chNode := range pNode.Children {
		ph := core.NewCommandMatch("", "")
		ph.Description = chNode.Description
		ph.Kind = core.NodeKind(chNode.Kind)
		ph.Requires = chNode.Requires
		attachChildNode(&child, chName, ph)
	}
	return child
}

// attachChildNode routes a child into Options or SubCommand based on
// its Kind. SubCommands remain in SubCommand, while Options and
// Arguments go to Options (and SubCommand for backwards-compat).
func attachChildNode(parent *core.CommandMatch, name string, child core.CommandMatch) {
	if parent.SubCommand == nil {
		parent.SubCommand = make(map[string]core.CommandMatch)
	}
	if parent.Options == nil {
		parent.Options = make(map[string]core.CommandMatch)
	}

	// Always populate SubCommand for backwards compatibility
	parent.SubCommand[name] = child

	// If it is an option (flag) or argument, also place it into Options pool
	if child.Kind == core.KindOption || child.Kind == core.KindArgument || strings.HasPrefix(name, "-") {
		parent.Options[name] = child
	}
}
