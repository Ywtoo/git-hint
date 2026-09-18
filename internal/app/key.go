package app

import (
	"fmt"

	"git-hint/internal/engine/keymap"
	"git-hint/internal/state"
)

// Key handles a keystroke: receives key, selected index, and buffer,
// and returns the formatted response string.
func Key(key string, selected int, buffer string) string {
	state.SetSelected(selected)
	state.SetBuffer(buffer)
	widget, newSelected := keymap.KeyHandler(key)

	return fmt.Sprintf("%s|%d|%s", widget, newSelected, state.GetBuffer())
}
