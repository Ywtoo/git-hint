package ranking

import (
	"git-hint/engine/engine/parser"
	"slices"
)

func RankSuggestions(suggestions []parser.CommandMatch) []parser.CommandMatch {
	slices.SortFunc(suggestions, func(a parser.CommandMatch, b parser.CommandMatch) int {
		if a.NUsed < b.NUsed {
			return 1
		}
		if a.NUsed > b.NUsed {
			return -1
		}
		return 0

	})
	return suggestions
}
