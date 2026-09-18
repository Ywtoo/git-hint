package common

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"git-hint/internal/core"
	"git-hint/internal/engine/history"
	"git-hint/internal/engine/tokenizer"
	"git-hint/internal/state"
)

// FilePathProvider completes one path level at a time — files AND folders —
// relative to the shell PWD or to the directory being typed. Used by shell
// builtins like `source`, whose argument is any filesystem path (no git).
//
// Suggestion order comes from usage ranking: paths the user actually sourced
// before (read from shell history) carry a usage count and float above fresh
// directory listings; as the user types a path, the directory walk narrows
// the options naturally.
func FilePathProvider() []core.CommandMatch {
	workingDir := workingDirectory()
	token := lastBufferToken()

	result := historyPathSuggestions("source", token)

	dir, outputPrefix, namePrefix := splitPathToken(workingDir, token)
	historyCount := len(result)
	entries, err := os.ReadDir(dir)
	if err == nil {
		seen := make(map[string]bool, len(result))
		for _, m := range result {
			seen[m.Name] = true
		}

		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(namePrefix)) {
				continue
			}
			entryName := outputPrefix + name
			// Stat follows symlinks so a linked directory (~/storage ->
			// /media/...) still gets its trailing slash.
			info, err := os.Stat(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			if info.IsDir() {
				entryName += "/"
			}
			if seen[entryName] {
				continue
			}
			result = append(result, core.CommandMatch{Name: entryName})
		}
	}

	// History entries (result[:historyCount]) keep their most-recent-first
	// order and stay above the directory listing — they are the paths the
	// user actually uses. The listing is alphabetical, directories first.
	listing := result[historyCount:]
	sort.Slice(listing, func(i, j int) bool {
		aDir := strings.HasSuffix(listing[i].Name, "/")
		bDir := strings.HasSuffix(listing[j].Name, "/")
		if aDir != bDir {
			return aDir
		}
		return listing[i].Name < listing[j].Name
	})
	return append(result[:historyCount:historyCount], listing...)
}

// historyPathSuggestions turns paths found in shell history lines that start
// with commandName (e.g. "source ~/.zshrc") into suggestions, most recent
// first, filtered by the token being typed.
//
// Paths are normalized before display: quotes are stripped (zsh records
// source "$HOME/.sdkman/..." verbatim), $HOME expands to the real home and
// is displayed as ~/, and relative paths are dropped — they depended on the
// directory the command ran from, so resolving them now would be a guess.
// Normalization also deduplicates the same file recorded under two forms
// (zsh-plugin/x.zsh vs ./zsh-plugin/x.zsh) down to a single suggestion.
func historyPathSuggestions(commandName, token string) []core.CommandMatch {
	lines, err := history.FindHistoryCommands(commandName)
	if err != nil {
		return nil
	}

	home, _ := os.UserHomeDir()
	seen := make(map[string]bool)
	var out []core.CommandMatch
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		path := strings.Trim(fields[1], "\"'")
		path = strings.TrimPrefix(path, "./")
		if path == "" || strings.HasPrefix(path, "-") {
			continue
		}
		if strings.HasPrefix(path, "$HOME") {
			path = home + strings.TrimPrefix(path, "$HOME")
		}
		if strings.HasPrefix(path, "~") {
			if home == "" {
				continue
			}
			path = home + strings.TrimPrefix(path, "~")
		}
		// Only absolute or home-anchored paths are trustworthy.
		if !strings.HasPrefix(path, "/") {
			continue
		}
		display := path
		if home != "" && (path == home || strings.HasPrefix(path, home+"/")) {
			display = "~" + strings.TrimPrefix(path, home)
		}
		if seen[display] || !strings.HasPrefix(display, token) {
			continue
		}
		seen[display] = true
		out = append(out, core.CommandMatch{Name: display, Description: "do histórico"})
	}
	return out
}

func workingDirectory() string {
	if dir := state.GetWorkingDir(); dir != "" {
		return dir
	}
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "."
}

func lastBufferToken() string {
	tokens := tokenizer.TokenizeBuffer(state.GetBuffer())
	if len(tokens) == 0 {
		return ""
	}
	return tokens[len(tokens)-1]
}

// splitPathToken resolves the directory to list, the prefix to prepend to
// each entry name and the prefix to filter by — shared by the directory and
// file-path providers.
func splitPathToken(workingDir, token string) (dir, outputPrefix, namePrefix string) {
	lastSlash := strings.LastIndex(token, "/")
	if lastSlash < 0 {
		return workingDir, "", token
	}

	outputPrefix = token[:lastSlash+1]
	namePrefix = token[lastSlash+1:]
	parent := outputPrefix
	if strings.HasPrefix(parent, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(parent, "~/")), outputPrefix, namePrefix
		}
	}
	if filepath.IsAbs(parent) {
		return filepath.Clean(parent), outputPrefix, namePrefix
	}
	return filepath.Join(workingDir, parent), outputPrefix, namePrefix
}
