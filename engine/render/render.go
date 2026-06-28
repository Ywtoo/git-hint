package render

import (
	"git-hint/engine/parser"
	"strings"
)

// calculateOffset determina a posição relativa dentro do buffer onde as sugestões devem começar.
func calculateOffset(buffer string, matches []parser.CommandMatch) int {
	// Limpa espaços no final para evitar que o menu "pule" ao digitar espaço
	trimmedBuffer := strings.TrimRight(buffer, " ")

	var lastToken string
	parts := strings.Fields(trimmedBuffer)
	if len(parts) > 0 {
		lastToken = parts[len(parts)-1]
	}

	// 2. Check if the last token is a prefix of any current suggestion
	isPrefix := false
	if lastToken != "" {
		for _, s := range matches {
			if strings.HasPrefix(s.Name, lastToken) {
				isPrefix = true
				break
			}
		}
	}

	// 3. Decide: Replace or Append
	if isPrefix {
		// Replace the last token with the selected suggestion
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

	// Itens da janela
	for i := start; i < end; i++ {
		m := matches[i]

		prefix := "  "
		if i == selected {
			prefix = "> "
		}

		// Adiciona seta para cima no primeiro item se houver mais acima
		if i == start && start > 0 {
			prefix = "↑" + prefix[1:] // substitui o primeiro espaço por ↑
		}
		// Adiciona seta para baixo no último item se houver mais abaixo
		if i == end-1 && end < total {
			prefix = "↓" + prefix[1:] // substitui o primeiro espaço por ↓
		}

		formatted = append(formatted, indent+prefix+m.Name)
	}

	return strings.Join(formatted, "\n")
}
