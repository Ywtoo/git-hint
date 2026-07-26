package provider

import "git-hint/core"

// TODO: Implement Context-Aware File Providers
// These providers should return different file lists based on the placeholder:
//
// <file-staged>: 'git diff --name-only --cached'
// <file-modified>: 'git diff --name-only'
// <file-untracked>: 'git ls-files --others --exclude-standard'
func FileProvider(context string) []core.CommandMatch {
	return nil
}
