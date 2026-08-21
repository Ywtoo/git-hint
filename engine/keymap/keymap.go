package keymap

import (
	"git-hint/engine"
	"git-hint/state"
)

func KeyHandler(key string) (string, int) {
	buffer := state.GetBuffer()
	selected := state.GetSelected()

	matches, currentToken, err := engine.Suggestions(buffer)
	if err != nil {
		return "", selected
	}
	listSize := len(matches)

	switch key {
	case "arrowUP":
		if selected <= 0 {
			return "up-line-or-history", -1
		}
		selected--
		return "", selected

	case "arrowDOWN":
		if selected < 0 {
			return "down-line-or-history", -1
		}
		selected++
		if selected >= listSize {
			selected = listSize - 1
			if selected < 0 {
				selected = 0
			}
			return "", selected
		}
		return "", selected

	case "TAB":
		if selected < 0 || listSize == 0 {
			return "expand-or-complete", selected
		}
		if selected >= 0 && selected < listSize {
			buffer = engine.CompleteBuffer(buffer, selected, matches, currentToken)
			state.SetBuffer(buffer)
			return "", selected
		}
		return "expand-or-complete", selected

	default:
		return "", selected
	}
}
