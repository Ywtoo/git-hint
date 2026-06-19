package main

import (
	"fmt"
	"os"

	"git-hint/engine/engine"
)

func main() {
	if len(os.Args) < 2 {
		return
	}

	result, err := engine.Execute(os.Args[1])

	if err != nil {
		return
	}

	for _, item := range result {
		fmt.Println(item.Name)
	}
}
