package git

import "git-hint/internal/core"

// TODO: Implement Context-Aware File Providers
// These providers should return different file lists based on the placeholder:
//
// <file-staged>: 'git diff --name-only --cached'
// <file-modified>: 'git diff --name-only'
// <file-untracked>: 'git ls-files --others --exclude-standard'
func FileProvider(context string) []core.CommandMatch {
	var args []string
	switch context {
	case "file-staged":
		args = []string{"diff", "--name-only", "--cached"}
	case "file-modified":
		args = []string{"diff", "--name-only"}
	case "file-untracked":
		args = []string{"ls-files", "--others", "--exclude-standard"}
	default:
		return TrackedFileProvider()
	}
	return gitLines(args)
}
