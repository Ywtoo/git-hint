package keymap

import (
	"git-hint/engine"
	"git-hint/state"
)

func KeyHandler(key string) (string, int) {
	buffer := state.GetBuffer()
	selected := state.GetSelected()

	matches, currentToken, _ := engine.Suggestions(buffer)
	listSize := len(matches)

	switch key {
	case "arrowUP":
		if selected < 0 {
			selected--
			return "up-line-or-history", selected
		}
		selected--
		if selected < 0 {
			return "up-line-or-history", selected
		}
		return "", selected

	case "arrowDOWN":
		if selected < 0 {
			selected++
			return "down-line-or-history", selected
		}
		selected++
		if selected >= listSize {
			selected = listSize - 1
			if selected < 0 {
				selected = 0
			}
			// When reaching the end of the list, we stop here and don't
			// trigger the shell history to avoid jumping out of the list.
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
