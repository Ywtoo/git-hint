package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"git-hint/app"
	"git-hint/daemon"
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
		selected := -1
		if val, err := strconv.Atoi(strings.TrimSpace(os.Args[3])); err == nil {
			selected = val
		}
		promptCol := 0
		if len(os.Args) >= 5 {
			if val, err := strconv.Atoi(strings.TrimSpace(os.Args[4])); err == nil {
				promptCol = val
			}
		}
		renderMode := "ohmyzsh"
		if len(os.Args) >= 6 {
			renderMode = strings.TrimSpace(os.Args[5])
		}
		fmt.Print(app.List(os.Args[2], selected, promptCol, renderMode))

	case "rebuild":
		app.Rebuild(os.Args)

	case "key":
		if len(os.Args) < 5 {
			return
		}
		selected, err := strconv.Atoi(strings.TrimSpace(os.Args[3]))
		if err != nil {
			return
		}
		fmt.Println(app.Key(os.Args[2], selected, os.Args[4]))

	case "daemon":
		daemon.Run()

	default:
	}
}
