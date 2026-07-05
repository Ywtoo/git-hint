package provider

import "git-hint/engine/parser"

// TODO: Implement StashProvider
// This provider should list git stashes with clear separation between ID and message.
// Command: 'git stash list'
// Logic:
// 1. Get output lines (e.g., "stash@{0}: WIP on main...")
// 2. Split each line at the first ':'
// 3. Name = "stash@{n}", Description = the rest of the line.
func StashProvider() []parser.CommandMatch {
	return nil
}
