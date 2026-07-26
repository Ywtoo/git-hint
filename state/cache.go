package state

import (
	"encoding/json"
	"os"

	"git-hint/core"
)

var cachePath = "/tmp/githint-session.json"

type SessionCache struct {
	CurrentPlaceholder string              `json:"current_placeholder"`
	ExpandedList       []core.CommandMatch `json:"expanded_list"`
	LastInput          string              `json:"last_input"`
}

func LoadCache() *SessionCache {
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
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	_ = os.WriteFile(cachePath, data, 0644)
}

func ClearCache() {
	_ = os.Remove(cachePath)
}
