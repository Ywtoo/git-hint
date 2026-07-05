package render

import (
	"git-hint/engine/parser"
	"strings"
)

func calculateOffset(buffer string, matches []parser.CommandMatch) int {
	trimmedBuffer := strings.TrimRight(buffer, " ")

	var lastToken string
	if !strings.HasSuffix(buffer, " ") {
		parts := strings.Fields(trimmedBuffer)
		if len(parts) > 0 {
			lastToken = parts[len(parts)-1]
		}
	}

	isPrefix := false
	if lastToken != "" {
		for _, s := range matches {
			if strings.HasPrefix(s.Name, lastToken) {
				isPrefix = true
				break
			}
		}
	}

	if isPrefix {
		idx := strings.LastIndex(trimmedBuffer, lastToken)
		if idx == -1 {
			return len(trimmedBuffer) + 1
		}
		return len(trimmedBuffer[:idx])
	}
	return len(trimmedBuffer) + 1
}

// FormatList recebe a lista de matches, o índice selecionado, o buffer atual, a coluna do prompt,
// e o modo de renderização, retornando a string formatada para exibição no shell com uma janela deslizante de 4 itens.
func FormatList(matches []parser.CommandMatch, selected int, buffer string, promptCol int, renderMode string) string {
	total := len(matches)
	if total == 0 {
		return ""
	}

	// Cálculo da indentação
	var indent string
	if renderMode == "basic" {
		indent = ""
	} else {
		internalOffset := calculateOffset(buffer, matches)
		totalOffset := promptCol + internalOffset
		indent = strings.Repeat(" ", totalOffset)
	}

	// Tamanho da janela
	windowSize := 4
	if total < windowSize {
		windowSize = total
	}

	// Calculamos o início da janela para tentar manter o selecionado centralizado (pos 1)
	start := selected - 1
	end := start + windowSize

	// Ajustes de borda
	if start < 0 {
		start = 0
		end = windowSize
	}
	if end > total {
		end = total
		start = end - windowSize
	}
	if start < 0 {
		start = 0
	}

	var formatted []string

	for i := start; i < end; i++ {
		m := matches[i]

		arrow := " "
		if i == start && start > 0 {
			arrow = "↑"
		} else if i == end-1 && end < total {
			arrow = "↓"
		}

		marker := " "
		if i == selected {
			marker = ">"
		}

		formatted = append(formatted, indent+arrow+marker+" "+m.Name)
	}

	return strings.Join(formatted, "\n")
}
