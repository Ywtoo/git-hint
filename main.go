package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"

	"git-hint/internal/app"
	"git-hint/internal/config"
	"git-hint/internal/daemon"
	"git-hint/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		return
	}

	mode := os.Args[1]

	switch mode {
	case "config":
		handleConfig(os.Args[2:])
	case "help", "--help", "-h":
		printHelp()
	case "version", "--version", "-v":
		fmt.Println("git-hint " + version.Value)
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
		workingDir, _ := os.Getwd()
		if len(os.Args) >= 7 {
			workingDir = os.Args[6]
		}
		fmt.Print(app.List(os.Args[2], selected, promptCol, renderMode, workingDir))

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
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "--kill":
				if err := daemon.Kill(); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				fmt.Println("githint: daemon killed")
			case "--restart":
				_ = daemon.Kill()
				time.Sleep(200 * time.Millisecond)
				daemon.Run()
			default:
				fmt.Fprintln(os.Stderr, "githint daemon: unknown flag:", os.Args[2])
				os.Exit(1)
			}
			return
		}
		daemon.Run()

	default:
	}
}

func handleConfig(args []string) {
	if len(args) == 0 {
		interactiveConfig()
		return
	}
	if args[0] == "get" && len(args) == 2 {
		cfg := config.Load()
		switch args[1] {
		case "render":
			fmt.Println(cfg.Render)
		case "items":
			fmt.Println(cfg.Items)
		case "comment_limit":
			fmt.Println(cfg.CommentLimit)
		default:
			fmt.Fprintln(os.Stderr, "githint config: configuração desconhecida:", args[1])
			os.Exit(2)
		}
		return
	}
	if args[0] == "set" && len(args) == 3 {
		if err := config.Set(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "githint config:", err)
			os.Exit(2)
		}
		fmt.Println("configuração salva em", config.Path())
		return
	}
	fmt.Fprintln(os.Stderr, "uso: githint config | githint config get <chave> | githint config set <render|items|comment_limit> <valor>")
	os.Exit(2)
}

func interactiveConfig() {
	for {
		cfg := config.Load()
		fields := []string{
			fmt.Sprintf("render: %s", cfg.Render),
			fmt.Sprintf("items: %d", cfg.Items),
			fmt.Sprintf("comment_limit: %d", cfg.CommentLimit),
			"salvar e sair",
		}
		selector := promptui.Select{Label: "githint config", Items: fields, Size: len(fields)}
		choice, _, err := selector.Run()
		if err != nil || choice == 3 {
			return
		}

		switch choice {
		case 0:
			renderSelector := promptui.Select{Label: "Perfil de render", Items: []string{"ohmyzsh", "basic"}, Size: 2}
			_, value, err := renderSelector.Run()
			if err == nil {
				_ = config.Set("render", value)
			}
		case 1, 2:
			key := "items"
			current := strconv.Itoa(cfg.Items)
			if choice == 2 {
				key = "comment_limit"
				current = strconv.Itoa(cfg.CommentLimit)
			}
			prompt := promptui.Prompt{Label: key, Default: current}
			value, err := prompt.Run()
			if err == nil {
				if err := config.Set(key, value); err != nil {
					fmt.Println("Erro:", err)
				}
			}
		}
	}
}

func printHelp() {
	fmt.Print(`git-hint — sugestões rápidas para comandos do shell

Uso:
  githint <comando> [opções]

Comandos:
  help, -h, --help       Mostra esta ajuda
  version, --version     Mostra a versão instalada
  config                 Mostra a configuração atual
  rebuild                Recria o índice de comandos disponíveis
  daemon                 Inicia o daemon
  daemon --restart       Reinicia o daemon
  daemon --kill          Para o daemon

Desenvolvimento local:
  make build
  make test
  make install-local VERSION=0.1.0
  make update-local VERSION=0.1.1
  make uninstall-local

Teclas no Zsh:
  Tab                    Completa a sugestão selecionada
  ↑ / ↓                  Navega pelas sugestões
  Tab no cabeçalho       Abre o grupo de placeholders
  Esc                    Sai do grupo expandido
  Enter                  Aceita a linha do shell

Para ativar o plugin Zsh manualmente:
  source ~/.config/githint/zsh/githint.zsh

`)
}
