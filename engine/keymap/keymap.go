package keymap

import (
	"git-hint/engine"
	"git-hint/state"
)

func KeyHandler(key string) (string, int) {
	matches, currentToken, _ := engine.Suggestions(state.Buffer)
	listSize := len(matches)

	switch key {
	case "arrowUP":
		if state.Selected < 0 {
			state.Selected--
			return "up-line-or-history", state.Selected
		}
		state.Selected--
		if state.Selected < 0 {
			return "up-line-or-history", state.Selected
		}
		return "", state.Selected

	case "arrowDOWN":
		if state.Selected < 0 {
			state.Selected++
			return "down-line-or-history", state.Selected
		}
		state.Selected++
		if state.Selected >= listSize {
			state.Selected = listSize - 1
			if state.Selected < 0 {
				state.Selected = 0
			}
			// When reaching the end of the list, we stop here and don't
			// trigger the shell history to avoid jumping out of the list.
			return "", state.Selected
		}
		return "", state.Selected

	case "TAB":
		if state.Selected < 0 || listSize == 0 {
			return "expand-or-complete", state.Selected
		}
		if state.Selected >= 0 && state.Selected < listSize {
			state.Buffer = engine.CompleteBuffer(state.Buffer, state.Selected, matches, currentToken)
			return "", state.Selected
		}
		return "expand-or-complete", state.Selected

	default:
		return "", state.Selected
	}
}
