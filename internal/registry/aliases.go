package registry

import (
	"sort"
	"strings"

	"git-hint/internal/core"
	"git-hint/internal/state"
)

// MergeAliases overlays the user's shell aliases on top of index.json so
// aliases are suggested at level 0 (typing "gs" suggests the alias, not the
// binary it shadows). Aliases only add entries — an index command hidden by
// the prefix filter is not removed by a shorter alias name.
func MergeAliases(index map[string]core.CommandMatch) map[string]core.CommandMatch {
	userAliases := state.GetAliases()
	if len(userAliases) == 0 {
		return index
	}

	merged := make(map[string]core.CommandMatch, len(index)+len(userAliases))
	for name, cmd := range index {
		merged[name] = cmd
	}

	names := make([]string, 0, len(userAliases))
	for name := range userAliases {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		definition := userAliases[name]
		if name == "" || strings.HasPrefix(name, "-") {
			continue
		}
		merged[name] = core.CommandMatch{
			Name:        name,
			Description: aliasDescription(definition),
		}
	}
	return merged
}

// aliasDescription renders a one-line description for an alias, keeping the
// full definition available in the expanded screen (completeDescription).
func aliasDescription(definition string) string {
	trimmed := strings.Join(strings.Fields(definition), " ")
	if trimmed == "" {
		return "shell alias"
	}
	if len(trimmed) > 60 {
		trimmed = trimmed[:57] + "..."
	}
	return "alias: " + trimmed
}
