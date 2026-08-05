package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"git-hint/core"
)

var (
	indexCacheMu sync.RWMutex
	indexCache   map[string]core.CommandMatch
)

// loadIndex reads and parses index.json once, caching the result in memory.
func loadIndex() (map[string]core.CommandMatch, error) {
	indexCacheMu.RLock()
	if indexCache != nil {
		defer indexCacheMu.RUnlock()
		return indexCache, nil
	}
	indexCacheMu.RUnlock()

	path, err := core.IndexPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("index.json não encontrado: %w", err)
	}

	var index map[string]core.CommandMatch
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("index.json inválido: %w", err)
	}

	indexCacheMu.Lock()
	indexCache = index
	indexCacheMu.Unlock()

	return index, nil
}

// SuggestFromIndex returns known top-level commands whose name starts
// with prefix (level 0: git, docker, npm...).
func SuggestFromIndex(prefix string) ([]core.CommandMatch, string, error) {
	index, err := loadIndex()
	if err != nil {
		return nil, "", err
	}
	return filterByPrefix(index, prefix), prefix, nil
}

// filterByPrefix is shared by index (level 0) and command (level 1)
// lookups — both are just "map of name -> CommandMatch, filtered".
func filterByPrefix(m map[string]core.CommandMatch, prefix string) []core.CommandMatch {
	var out []core.CommandMatch
	for name, cmd := range m {
		if strings.HasPrefix(name, prefix) {
			out = append(out, cmd)
		}
	}
	return out
}
