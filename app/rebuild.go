package app

import (
	"fmt"
	"os"

	"git-hint/scraper"
)

// Rebuild discovers every command available on the system ($PATH binaries
// + zsh builtins), crawls each one's help tree, and writes the full
// registry to data/ next to the binary. No arguments needed — scraper
// owns the entire discover -> crawl -> write cycle.
func Rebuild(args []string) {
	fmt.Println("descobrindo comandos disponíveis no sistema...")

	err := scraper.Rebuild(scraper.Options{
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

	fmt.Println("registro reconstruído com sucesso.")
}
