package state

import (
	"encoding/json"
	"os"
	"sync"

	"git-hint/internal/core"
)

// cacheMu serializes all disk access to cachePath. Multiple daemon
// connections can call LoadCache/SaveCache concurrently; without this,
// two simultaneous writes could interleave and corrupt the JSON file.
var cacheMu sync.Mutex

var cachePath = "/tmp/githint-session.json"

type SessionCache struct {
	CurrentPlaceholder string              `json:"current_placeholder"`
	ExpandedList       []core.CommandMatch `json:"expanded_list"`
	LastInput          string              `json:"last_input"`
}

func LoadCache() *SessionCache {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return &SessionCache{}
	}

	var cache SessionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return &SessionCache{}
	}

	return &cache
}

func SaveCache(cache *SessionCache) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	data, err := json.Marshal(cache)
	if err != nil {
		return
	}

	// Atomic write: write to a temp file first, then rename over the
	// real path. os.Rename on the same filesystem is atomic at the OS
	// level, so any reader either sees the old file or the new one
	// completely — never a half-written file, even if the process is
	// killed mid-write.
	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return
	}
	_ = os.Rename(tmpPath, cachePath)
}

func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	_ = os.Remove(cachePath)
}
