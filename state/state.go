// Package state holds the current interaction state (buffer text and
// selected suggestion index). Access goes through Get/Set functions
// protected by a mutex, since the daemon may handle multiple connections
// concurrently and this state is shared across goroutines.
package state

import "sync"

var mu sync.RWMutex

var buffer string
var selected int

// SetBuffer updates the current command-line buffer.
func SetBuffer(b string) {
	mu.Lock()
	defer mu.Unlock()
	buffer = b
}

// GetBuffer returns the current command-line buffer.
func GetBuffer() string {
	mu.RLock()
	defer mu.RUnlock()
	return buffer
}

// SetSelected updates the currently selected suggestion index.
func SetSelected(s int) {
	mu.Lock()
	defer mu.Unlock()
	selected = s
}

// GetSelected returns the currently selected suggestion index.
func GetSelected() int {
	mu.RLock()
	defer mu.RUnlock()
	return selected
}
