package keymap

import "git-hint/engine"

var Selected int
var Buffer string

func KeyHandler(key string) (string, int) {
	matches, _ := engine.Suggestions(Buffer)
	listSize := len(matches)

	switch key {
	case "arrowUP":
		if Selected < 0 {
			Selected--
			return "up-line-or-history", Selected
		}
		Selected--
		if Selected < 0 {
			return "up-line-or-history", Selected
		}
		return "", Selected

	case "arrowDOWN":
		if Selected < 0 {
			Selected++
			return "down-line-or-history", Selected
		}
		Selected++
		if Selected >= listSize {
			Selected = listSize - 1
			if Selected < 0 {
				Selected = 0
			}
			// When reaching the end of the list, we stop here and don't
			// trigger the shell history to avoid jumping out of the list.
			return "", Selected
		}
		return "", Selected

	case "TAB":
		// Pass-through: if in history mode or no suggestions, use Zsh default autocomplete
		if Selected < 0 || listSize == 0 {
			return "expand-or-complete", Selected
		}

		if Selected >= 0 && Selected < listSize {
			Buffer = engine.CompleteBuffer(Buffer, Selected)
			return "", Selected
		}
		return "expand-or-complete", Selected

	default:
		return "", Selected
	}
}
