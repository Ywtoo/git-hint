package git

import (
	"git-hint/internal/core"
	"sort"
)

// TODO: Implement AuthorProvider
// This provider should suggest authors who have committed to the repository.
// Command: 'git log --format=%an'
// Logic:
// 1. Get the list of all author names from the git log.
// 2. Deduplicate the list to ensure unique entries.
func AuthorProvider() []core.CommandMatch {
	items := gitLines([]string{"log", "--format=%an"})
	seen := make(map[string]bool)
	out := items[:0]
	for _, item := range items {
		if item.Name != "" && !seen[item.Name] {
			seen[item.Name] = true
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
