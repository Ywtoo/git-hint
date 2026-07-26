package provider

import "git-hint/core"

// TODO: Implement ConflictProvider
// This provider should identify files currently in a conflicted state.
// Command: 'git status --porcelain'
// Logic: Filter for lines starting with 'UU' (Unmerged).
func ConflictProvider() []core.CommandMatch {
	return nil
}
