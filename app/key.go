package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"git-hint/engine/keymap"
	"git-hint/state"
)

func Key(args []string) {

	if len(os.Args) < 5 {
		return
	}
	key := os.Args[2]

	valStr := strings.TrimSpace(os.Args[3])
	buffer := os.Args[4]

	selected, err := strconv.Atoi(valStr)
	if err != nil {
		return
	}

	state.Selected = selected
	state.Buffer = buffer
	widget, newSelected := keymap.KeyHandler(key)

	fmt.Printf("%s|%d|%s\n", widget, newSelected, state.Buffer)
}
