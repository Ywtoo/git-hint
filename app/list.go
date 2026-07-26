package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"git-hint/engine"
	"git-hint/engine/render"
	"git-hint/state"
)

func List(args []string) {
	if len(os.Args) < 4 {
		return
	}
	buffer := os.Args[2]
	state.Buffer = buffer

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

	matches, currentToken, err := engine.Suggestions(buffer)
	if err != nil {
		return
	}

	fmt.Print(render.FormatList(matches, selected, buffer, currentToken, promptCol, renderMode))
}
