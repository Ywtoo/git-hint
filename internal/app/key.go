package app

import (
	"fmt"

	"git-hint/internal/engine"
	"git-hint/internal/engine/keymap"
	"git-hint/internal/state"
)

// Key handles a keystroke: receives key, selected index, and buffer,
// and returns the formatted response string.
func Key(key string, selected int, buffer string) string {
	state.SetSelected(selected)
	state.SetBuffer(buffer)
	widget, newSelected := keymap.KeyHandler(key)

	// Headers are labels, never selectable: whatever the key handler did,
	// the selection must not rest on a group header row.
	if widget == "" {
		if matches, token, err := engine.Suggestions(buffer); err == nil && matches != nil {
			newSelected = engine.NormalizeSelected(matches, newSelected)
			_ = token
		}
	}

	return fmt.Sprintf("%s|%d|%s", widget, newSelected, state.GetBuffer())
}
