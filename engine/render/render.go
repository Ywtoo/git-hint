package render

import (
	"git-hint/engine/parser"
	"strings"
)

// FormatList recebe a lista de matches e o índice selecionado,
// retornando a string formatada para exibição no shell.
func FormatList(matches []parser.CommandMatch, selected int) string {
	if len(matches) == 0 {
		return ""
	}

	var formatted []string
	for i, m := range matches {
		if i == selected {
			formatted = append(formatted, "> "+m.Name)
		} else {
			formatted = append(formatted, "  "+m.Name)
		}
	}

	return strings.Join(formatted, "\n")
}
