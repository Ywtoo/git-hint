package keymap

import (
	"git-hint/internal/core"
	"git-hint/internal/engine"
	"git-hint/internal/state"
)

// groupHasItems reports whether the expanded screen for placeholder would
// show at least one selectable item (anything that is not the header itself
// or the "sair (Tab)" control).
func groupHasItems(matches []core.CommandMatch, placeholder string) bool {
	return groupItemsCount(matches, placeholder) > 0
}

// groupItemsCount counts selectable items inside a placeholder group.
func groupItemsCount(matches []core.CommandMatch, placeholder string) int {
	n := 0
	for _, m := range matches {
		if m.Placeholder == placeholder && m.Name != "" && !m.ExitControl {
			n++
		}
	}
	return n
}

func KeyHandler(key string) (string, int) {
	buffer := state.GetBuffer()
	selected := state.GetSelected()

	matches, currentToken, err := engine.Suggestions(buffer)
	if err != nil {
		return "", selected
	}
	listSize := len(matches)

	switch key {
	case "ESC":
		if group, _ := state.GetExpansion(); group != "" {
			return "", state.ExitExpansion()
		}
		return "send-break", selected
	case "arrowUP":
		if selected <= 0 {
			if group, _ := state.GetExpansion(); group != "" {
				return "", 0
			}
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
		// Quote bootstrap has top priority: TAB right after a message flag
		// (`git commit -m ` with nothing typed yet) ALWAYS inserts the opening
		// quote first — even when history suggestions are on the list. The
		// user then types inside the quotes and keeps filtering.
		if boot, ok := engine.BootstrapQuote(buffer); ok {
			state.SetBuffer(boot)
			return "", selected
		}
		if selected < 0 || listSize == 0 {
			return "expand-or-complete", selected
		}
		if selected >= 0 && selected < listSize {
			// Expanded screen check FIRST, before any single-item shortcut: the
			// list in expansion mode is [header?, items..., sair (Tab)], and a
			// one-item expanded group still has a real item to complete.
			if group, _ := state.GetExpansion(); group != "" {
				if matches[selected].ExitControl {
					return "", state.ExitExpansion()
				}
				buffer = engine.CompleteBuffer(buffer, selected, matches, currentToken)
				state.SetBuffer(buffer)
				return "", selected
			}
			if matches[selected].ExitControl {
				return "", state.ExitExpansion()
			}
			// Tab on a header enters its placeholder group. The header itself is
			// never inserted into the shell buffer. A group with zero items
			// (provider empty, e.g. <dir> in a directory with no subdirectories)
			// has nothing to expand into — fall through to normal completion
			// instead of trapping the user on a bare "sair (Tab)" screen.
			if matches[selected].Name == "" {
				if groupHasItems(matches, matches[selected].Placeholder) {
					// One-item group: expand straight into the item instead of
					// making the user select the header, press Tab again and
					// select the item. Two-item groups and up keep the screen.
					if groupItemsCount(matches, matches[selected].Placeholder) == 1 {
						for i, m := range matches {
							if m.Placeholder == matches[selected].Placeholder && m.Name != "" && !m.ExitControl {
								buffer = engine.CompleteBuffer(buffer, i, matches, currentToken)
								state.SetBuffer(buffer)
								return "", i
							}
						}
					}
					state.EnterExpansion(matches[selected].Placeholder, selected)
					state.SetExpansionBuffer(buffer)
					return "", 0
				}
				return "expand-or-complete", selected
			}
			buffer = engine.CompleteBuffer(buffer, selected, matches, currentToken)
			state.SetBuffer(buffer)
			return "", selected
		}
		return "expand-or-complete", selected

	default:
		return "", selected
	}
}
