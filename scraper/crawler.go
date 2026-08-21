package scraper

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git-hint/core"
	"git-hint/scraper/parser"
)

type ProgressState struct {
	Total   int
	Current int
}

type Options struct {
	MaxDepth   int
	Verbose    bool
	OnProgress func(current, total int, cmd string)
	Progress   *ProgressState
}

// --- temporary instrumentation for diagnostics ---
var (
	execCount int64
	slowMu    sync.Mutex
	slowLog   []slowEntry
)

type slowEntry struct {
	path     string
	duration time.Duration
}

func recordDuration(path string, d time.Duration) {
	atomic.AddInt64(&execCount, 1)
	if d > 300*time.Millisecond {
		slowMu.Lock()
		slowLog = append(slowLog, slowEntry{path, d})
		slowMu.Unlock()
	}
}

func runWithTimeout(d time.Duration, fn func() (string, error)) (string, error, bool) {
	type result struct {
		output string
		err    error
	}
	ch := make(chan result, 1)

	go func() {
		out, err := fn()
		ch <- result{out, err}
	}()

	select {
	case r := <-ch:
		return r.output, r.err, false
	case <-time.After(d):
		return "", fmt.Errorf("timeout"), true
	}
}

func PrintDiagnostics() {
	fmt.Printf("\n=== DIAGNOSTICS ===\n")
	fmt.Printf("Total exec.Command calls: %d\n", atomic.LoadInt64(&execCount))

	slowMu.Lock()
	defer slowMu.Unlock()
	sort.Slice(slowLog, func(i, j int) bool { return slowLog[i].duration > slowLog[j].duration })

	limit := 30
	if len(slowLog) < limit {
		limit = len(slowLog)
	}
	fmt.Printf("Top %d slowest (>300ms):\n", limit)
	for i := 0; i < limit; i++ {
		fmt.Printf("  %v  %s\n", slowLog[i].duration, slowLog[i].path)
	}
}

// --- end instrumentation ---

func CrawlOne(name, path string, opts Options) error {
	visited := make(map[string]bool)
	tree := crawlCommand([]string{path}, opts, visited)
	return WriteCommand(name, tree.SubCommand)
}

func isTerminalToken(name string) bool {
	return strings.HasPrefix(name, "-") || strings.HasPrefix(name, "<")
}

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

	namePath := append([]string{filepath.Base(path[0])}, path[1:]...)
	parsed := parser.ParseCommand(output, namePath)
	if parsed.IsLeaf() {
		return node
	}

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
				child = core.NewCommandMatch("", "")
				child.Description = pNode.Description
				for chName, chNode := range pNode.Children {
					ph := core.NewCommandMatch("", "")
					ph.Description = chNode.Description
					child.SubCommand[chName] = ph
				}
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

			mu.Lock()
			node.SubCommand[subName] = child
			mu.Unlock()
		}(subName, pNode)
	}
	wg.Wait()

	return node
}
