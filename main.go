package main

import (
	"os"

	"git-hint/app"
)

func main() {
	if len(os.Args) < 2 {
		return
	}

	mode := os.Args[1]

	switch mode {
	case "list":
		app.List(os.Args)

	case "rebuild":
		app.Rebuild(os.Args)

	case "key":
		app.Key(os.Args)

	default:
	}
}
