package app

import (
	"fmt"

	"git-hint/engine/keymap"
	"git-hint/state"
)

// Key também vira pura: recebe tecla, selected e buffer prontos,
// devolve a string formatada em vez de imprimir.
func Key(key string, selected int, buffer string) string {
	state.SetSelected(selected)
	state.SetBuffer(buffer)
	widget, newSelected := keymap.KeyHandler(key)

	return fmt.Sprintf("%s|%d|%s", widget, newSelected, state.GetBuffer())
}
