package render

import (
	"strings"

	"git-hint/core"
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

	// Maximum name length within the visible window to align comments in a column.
	maxNameLen := 0
	for i := start; i < end; i++ {
		n := len(matches[i].Name)
		if n > maxNameLen {
			maxNameLen = n
		}
	}

	var formatted []string
	lastPlaceholder := "" // controls when to display a new group header

	for i := start; i < end; i++ {
		m := matches[i]

		// Group header: displayed when group placeholder changes. It is NOT a selectable item.
		if m.Placeholder != "" && m.Placeholder != lastPlaceholder {
			formatted = append(formatted, indent+"  "+colorGreenStart+m.Placeholder+colorGreenEnd)
			lastPlaceholder = m.Placeholder
		}

		if m.Name == "" {
			continue
		}

		name := m.Name
		isSelected := i == selected // selected index maps directly to matches slice

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
			visualLen := len(m.Name)
			padding := strings.Repeat(" ", maxNameLen-visualLen+2)
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
	return m.Description
}
