package provider

import "git-hint/engine/parser"

// TODO: Implement ConfigProvider
// This provider should return a static list of commonly used git configuration keys.
// Examples: "user.name", "user.email", "core.editor", "pull.rebase", "init.defaultBranch".
// Since this is a static list, it doesn't require an external git command.
func ConfigProvider() []parser.CommandMatch {
	return nil
}
