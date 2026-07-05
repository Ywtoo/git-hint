package provider

import "git-hint/engine/parser"

// TODO: Implement ConflictProvider
// This provider should identify files currently in a conflicted state.
// Command: 'git status --porcelain'
// Logic: Filter for lines starting with 'UU' (Unmerged).
func ConflictProvider() []parser.CommandMatch {
	return nil
}
