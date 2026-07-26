package app

import (
	"encoding/json"
	"fmt"
	"os"

	"git-hint/scraper"
)

func Rebuild(args []string) {

	if len(os.Args) < 3 {
		fmt.Println("uso: githint rebuild <binario>")
		return
	}
	binary := os.Args[2]

	fmt.Printf("rastreando comandos de '%s'...\n", binary)

	result, err := scraper.Rebuild(binary, scraper.Options{
		MaxDepth: 4,
		OnProgress: func(current, total int, cmd string) {
			pct := float64(current) / float64(total) * 100.0
			fmt.Printf("[%3d/%3d] (%5.1f%%) rastreando: %s\n", current, total, pct, cmd)
		},
	})
	if err != nil {
		fmt.Println("erro:", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Println("erro ao gerar json:", err)
		os.Exit(1)
	}

	path := "registry/data/" + binary + ".json"
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Println("erro ao salvar:", err)
		os.Exit(1)
	}

	fmt.Printf("%d comandos salvos em %s\n", len(result), path)
}
