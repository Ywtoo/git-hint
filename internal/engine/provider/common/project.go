package common

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"git-hint/internal/core"
	"git-hint/internal/state"
)

func MakeTargetProvider() []core.CommandMatch {
	file, err := os.Open(filepath.Join(projectDirectory(), "Makefile"))
	if err != nil {
		return nil
	}
	defer file.Close()
	seen := map[string]bool{}
	var out []core.CommandMatch
	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ".") {
			continue
		}
		if i := strings.Index(line, ":"); i > 0 && !strings.ContainsAny(line[:i], " \t") {
			for _, target := range strings.Fields(line[:i]) {
				if !seen[target] {
					seen[target] = true
					out = append(out, core.CommandMatch{Name: target})
				}
			}
		}
	}
	return out
}

func NPMScriptProvider() []core.CommandMatch {
	data, err := os.ReadFile(filepath.Join(projectDirectory(), "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	names := make([]string, 0, len(pkg.Scripts))
	for name := range pkg.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]core.CommandMatch, 0, len(names))
	for _, name := range names {
		description := pkg.Scripts[name]
		out = append(out, core.CommandMatch{Name: name, Description: description})
	}
	return out
}

func projectDirectory() string {
	if dir := state.GetWorkingDir(); dir != "" {
		return dir
	}
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "."
}
