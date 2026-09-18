// Package state holds the current interaction state (buffer text and
// selected suggestion index). Access goes through Get/Set functions
// protected by a mutex, since the daemon may handle multiple connections
// concurrently and this state is shared across goroutines.
package state

import "sync"

var mu sync.RWMutex

var buffer string
var selected int
var workingDir string
var expandedGroup string
var expandedLimit = 10
var expandedParentSelected int
var expandedBuffer string

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

// SetWorkingDir records the shell directory that originated the request. The
// daemon is long-lived, so its own process directory cannot be used for path
// completion.
func SetWorkingDir(dir string) {
	mu.Lock()
	defer mu.Unlock()
	workingDir = dir
}

func GetWorkingDir() string {
	mu.RLock()
	defer mu.RUnlock()
	return workingDir
}

func SetExpansion(group string, limit int) {
	mu.Lock()
	defer mu.Unlock()
	expandedGroup, expandedLimit = group, limit
}

// EnterExpansion switches to a dedicated group screen and remembers the
// parent selection so the caller can return to exactly the same place.
func EnterExpansion(group string, parentSelected int) {
	mu.Lock()
	defer mu.Unlock()
	expandedGroup = group
	expandedLimit = 0
	expandedParentSelected = parentSelected
}

func ExitExpansion() int {
	mu.Lock()
	defer mu.Unlock()
	parentSelected := expandedParentSelected
	expandedGroup = ""
	expandedLimit = 10
	expandedParentSelected = 0
	return parentSelected
}
func GetExpansion() (string, int) {
	mu.RLock()
	defer mu.RUnlock()
	return expandedGroup, expandedLimit
}

// SetExpansionBuffer records the buffer that produced the current expansion
// screen, so List can detect that the user navigated away (typed something
// else) and reset the expansion instead of leaking its header into an
// unrelated command's suggestion list.
func SetExpansionBuffer(buffer string) {
	mu.Lock()
	defer mu.Unlock()
	expandedBuffer = buffer
}

func GetExpansionBuffer() string {
	mu.RLock()
	defer mu.RUnlock()
	return expandedBuffer
}

// ClearExpansion returns the list to its collapsed state without touching the
// normal selection index kept by the keymap.
func ClearExpansion() { ExitExpansion() }
