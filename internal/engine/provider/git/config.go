package git

import "git-hint/internal/core"

// TODO: Implement ConfigProvider
// This provider should return a static list of commonly used git configuration keys.
// Examples: "user.name", "user.email", "core.editor", "pull.rebase", "init.defaultBranch".
// Since this is a static list, it doesn't require an external git command.
func ConfigProvider() []core.CommandMatch {
	keys := []string{"user.name", "user.email", "core.editor", "core.autocrlf", "init.defaultBranch", "pull.rebase", "push.default", "fetch.prune", "rerere.enabled", "commit.gpgsign"}
	result := make([]core.CommandMatch, 0, len(keys))
	for _, key := range keys {
		result = append(result, core.CommandMatch{Name: key})
	}
	return result
}
