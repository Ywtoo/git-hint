package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Render       string `json:"render"`
	Items        int    `json:"items"`
	CommentLimit int    `json:"comment_limit"`
}

func Defaults() Config {
	return Config{Render: "ohmyzsh", Items: 4, CommentLimit: 48}
}

func Path() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", ".config", "githint", "config.json")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "githint", "config.json")
}

func Load() Config {
	cfg := Defaults()
	data, err := os.ReadFile(Path())
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	if cfg.Render != "basic" && cfg.Render != "ohmyzsh" {
		cfg.Render = Defaults().Render
	}
	if cfg.Items < 1 || cfg.Items > 20 {
		cfg.Items = Defaults().Items
	}
	if cfg.CommentLimit < 10 || cfg.CommentLimit > 200 {
		cfg.CommentLimit = Defaults().CommentLimit
	}
	return cfg
}

func Save(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(Path()), 0755); err != nil {
		return err
	}
	return os.WriteFile(Path(), append(data, '\n'), 0644)
}

func Set(key, value string) error {
	cfg := Load()
	switch key {
	case "render":
		if value != "basic" && value != "ohmyzsh" {
			return fmt.Errorf("render deve ser basic ou ohmyzsh")
		}
		cfg.Render = value
	case "items":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 20 {
			return fmt.Errorf("items deve ser um número entre 1 e 20")
		}
		cfg.Items = n
	case "comment_limit":
		n, err := strconv.Atoi(value)
		if err != nil || n < 10 || n > 200 {
			return fmt.Errorf("comment_limit deve ser um número entre 10 e 200")
		}
		cfg.CommentLimit = n
	default:
		return fmt.Errorf("configuração desconhecida: %s", key)
	}
	return Save(cfg)
}
