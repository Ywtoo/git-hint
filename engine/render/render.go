package render

import (
	"git-hint/engine/parser"
	"strings"
)

const (
	colorGreenStart = "\x01"
	colorGreenEnd   = "\x02"
	colorGrayStart  = "\x03"
	colorGrayEnd    = "\x04"
)

func calculateOffset(buffer string, currentToken string) int {
	if currentToken == "" {
		trimmed := strings.TrimRight(buffer, " ")
		return len(trimmed) + 1
	}
	if strings.HasSuffix(buffer, currentToken) {
		return len(buffer) - len(currentToken)
	}
	return len(buffer) + 1
}

// FormatList monta a lista de sugestões.
// groupDescription: comentário herdado do placeholder pai (ex: "Selecione a branch"),
// usado só quando a lista é dinâmica E não auto-descritiva — aparece só na linha selecionada.
// Para listas estáticas ou auto-descritivas, groupDescription vem "" e cada item usa sua própria m.Description.
func FormatList(matches []parser.CommandMatch, selected int, buffer string, currentToken string, promptCol int, renderMode string) string {
	total := len(matches)
	if total == 0 {
		return ""
	}

	var indent string
	if renderMode == "basic" {
		indent = ""
	} else {
		internalOffset := calculateOffset(buffer, currentToken)
		totalOffset := promptCol + internalOffset
		indent = strings.Repeat(" ", totalOffset)
	}

	windowSize := 4
	if total < windowSize {
		windowSize = total
	}

	start := selected - 1
	end := start + windowSize

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

	// Largura máxima do nome, só dentro da janela visível, pra alinhar os comentários em coluna.
	maxNameLen := 0
	for i := start; i < end; i++ {
		n := len(displayName(matches[i].Name))
		if n > maxNameLen {
			maxNameLen = n
		}
	}

	var formatted []string

	for i := start; i < end; i++ {
		m := matches[i]
		name := displayName(m.Name)
		isSelected := i == selected

		if !isSelected {
			name = colorGrayStart + name + colorGrayEnd
		}

		charA := " "
		charB := " "
		if isSelected {
			charB = ">"
		}
		if i == start && start > 0 {
			charA = "↑"
		} else if i == end-1 && end < total {
			charA = "↓"
		}

		comment := resolveComment(m, isSelected)

		line := indent + charA + charB + name
		if comment != "" {
			visualLen := len(displayName(m.Name))
			padding := strings.Repeat(" ", maxNameLen-visualLen+2)
			line += padding + colorGreenStart + "# " + comment + colorGreenEnd
		}

		formatted = append(formatted, line)
	}

	return strings.Join(formatted, "\n")
}

func resolveComment(m parser.CommandMatch, isSelected bool) string {
	if m.Description == "" {
		return "" // self-descriptive (msg) ou sem descrição mesmo
	}
	if m.ShowOnlyWhenSelected && !isSelected {
		return ""
	}
	return m.Description
}

func displayName(name string) string {
	if len(name) >= 2 && strings.HasPrefix(name, `"`) && strings.HasSuffix(name, `"`) {
		return name[1 : len(name)-1]
	}
	return name
}
