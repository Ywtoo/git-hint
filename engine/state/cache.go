package state

import (
	"encoding/json"
	"git-hint/engine/parser"
	"os"
)

var cachePath = "/tmp/githint-session.json"

type SessionCache struct {
	CurrentPlaceholder string                `json:"current_placeholder"`
	ExpandedList       []parser.CommandMatch `json:"expanded_list"`
	LastInput          string                `json:"last_input"`
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
