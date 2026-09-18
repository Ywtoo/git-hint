package render

import (
	"strings"

	"git-hint/internal/config"
	"git-hint/internal/core"
)

const (
	colorGreenStart           = "\x01"
	colorGreenEnd             = "\x02"
	colorGrayStart            = "\x03"
	colorGrayEnd              = "\x04"
	colorSelectedStart        = "\x05"
	colorSelectedEnd          = "\x06"
	colorSelectedCommentStart = "\x07"
	colorSelectedCommentEnd   = "\x08"
	// Keep the post-display compact. A long Fig/help description would otherwise
	// wrap over several terminal rows and visually detach from its suggestion.
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

// FormatList assembles the formatted suggestion list for the terminal.
func FormatList(matches []core.CommandMatch, selected int, buffer string, currentToken string, promptCol int, renderMode string) string {
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

	windowSize := config.Load().Items
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

	// Maximum name length within the visible window to align comments in a column.
	maxNameLen := 0
	for i := start; i < end; i++ {
		n := len(matches[i].Name)
		if n > maxNameLen {
			maxNameLen = n
		}
	}

	var formatted []string
	if matches[0].ExpandedGroup {
		formatted = append(formatted, indent+"  "+colorGreenStart+matches[0].Placeholder+colorGreenEnd)
	}

	for i := start; i < end; i++ {
		m := matches[i]
		isSelected := i == selected // selected index maps directly to matches slice

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

		// Group headers are real list entries. Rendering them from their own
		// index keeps keyboard selection and the visible cursor in sync.
		if m.Name == "" {
			if isSelected {
				formatted = append(formatted, indent+colorSelectedStart+charA+charB+m.Placeholder+colorSelectedEnd)
			} else {
				formatted = append(formatted, indent+charA+charB+colorGreenStart+m.Placeholder+colorGreenEnd)
			}
			continue
		}

		name := m.Name
		if !isSelected {
			name = colorGrayStart + name + colorGrayEnd
		}

		comment := resolveComment(m, isSelected)

		line := indent + charA + charB + name
		if m.Placeholder != "" {
			line = indent + "    " + charA + charB + name
		}
		if isSelected {
			prefix := indent
			if m.Placeholder != "" {
				prefix += "    "
			}
			line = prefix + colorSelectedStart + charA + charB + name + colorSelectedEnd
		}
		visualLen := len(m.Name)
		padding := strings.Repeat(" ", maxNameLen-visualLen+2)
		if isSelected {
			prefix := indent
			if m.Placeholder != "" {
				prefix += "    "
			}
			line = prefix + colorSelectedStart + charA + charB + m.Name + padding + "  " + colorSelectedEnd
			if comment != "" {
				line += colorSelectedCommentStart + "# " + comment + colorSelectedCommentEnd
			}
		} else if comment != "" {
			line += padding + colorGreenStart + "# " + comment + colorGreenEnd
		}

		formatted = append(formatted, line)
	}

	return strings.Join(formatted, "\n")
}

func resolveComment(m core.CommandMatch, isSelected bool) string {
	if m.Description == "" {
		return ""
	}
	if m.ShowOnlyWhenSelected && !isSelected {
		return ""
	}
	// Descriptions from Fig/help often contain embedded newlines. The list is
	// rendered one suggestion per terminal row, so flatten them instead of
	// allowing a long description to create phantom rows and break selection.
	return truncateRunes(strings.Join(strings.Fields(m.Description), " "), config.Load().CommentLimit)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit || limit <= 0 {
		return value
	}
	if limit == 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}
