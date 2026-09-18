package scraper

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// Diagnostics — instrumentation for measuring crawl performance
// ============================================================================

var (
	execCount int64
	slowMu    sync.Mutex
	slowLog   []slowEntry
)

type slowEntry struct {
	path     string
	duration time.Duration
}

// recordDuration tracks an exec.Command call and logs slow ones (>300ms).
func recordDuration(path string, d time.Duration) {
	atomic.AddInt64(&execCount, 1)
	if d > 300*time.Millisecond {
		slowMu.Lock()
		slowLog = append(slowLog, slowEntry{path, d})
		slowMu.Unlock()
	}
}

// PrintDiagnostics prints a summary of exec.Command calls and the slowest ones.
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

// runWithTimeout runs fn in a goroutine and returns its result, or a
// timeout error if it doesn't complete within d.
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
