package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"git-hint/engine"
	"git-hint/engine/keymap"
	"git-hint/engine/render"
)

func main() {
	if len(os.Args) < 2 {
		return
	}

	mode := os.Args[1]

	switch mode {
	case "list":
		if len(os.Args) < 4 {
			return
		}
		buffer := os.Args[2]

		selected := -1
		if len(os.Args) >= 4 {
			valStr := strings.TrimSpace(os.Args[3])
			if val, err := strconv.Atoi(valStr); err == nil {
				selected = val
			}
		}

		promptCol := 0
		if len(os.Args) >= 5 {
			valStr := strings.TrimSpace(os.Args[4])
			if val, err := strconv.Atoi(valStr); err == nil {
				promptCol = val
			}
		}

		renderMode := "ohmyzsh"
		if len(os.Args) >= 6 {
			renderMode = strings.TrimSpace(os.Args[5])
		}

		matches, err := engine.Suggestions(buffer)
		if err != nil {
			return
		}

		fmt.Print(render.FormatList(matches, selected, buffer, promptCol, renderMode))

	case "key":
		if len(os.Args) < 5 {
			return
		}
		key := os.Args[2]

		valStr := strings.TrimSpace(os.Args[3])
		buffer := strings.TrimSpace(os.Args[4])

		selected, err := strconv.Atoi(valStr)
		if err != nil {
			return
		}

		keymap.Selected = selected
		keymap.Buffer = buffer
		widget, newSelected := keymap.KeyHandler(key)

		// SAÍDA PURA: "widget|selected|buffer"
		fmt.Printf("%s|%d|%s\n", widget, newSelected, keymap.Buffer)

	default:
	}
}
