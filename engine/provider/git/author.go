package git

import "git-hint/core"

// TODO: Implement AuthorProvider
// This provider should suggest authors who have committed to the repository.
// Command: 'git log --format=%an'
// Logic:
// 1. Get the list of all author names from the git log.
// 2. Deduplicate the list to ensure unique entries.
func AuthorProvider() []core.CommandMatch {
	return nil
}
