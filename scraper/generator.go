package scraper

import (
	"fmt"

	"git-hint/core"
)

// Rebuild é a única função pública que importa quem for usar esse
// pacote (main.go). Roda "<binary> help -a", desce em cada comando
// recursivamente e devolve a árvore pronta pra virar JSON.
func Rebuild(binary string, opts Options) (map[string]core.CommandMatch, error) {
	visited := make(map[string]bool)
	result := make(map[string]core.CommandMatch)

	rootOutput, err := executeRootHelp(binary)
	if err != nil {
		return nil, fmt.Errorf("erro ao pegar lista raiz de %s: %w", binary, err)
	}

	rootParsed := ParseCommand(rootOutput, []string{binary})

	totalCount := len(rootParsed.Nodes)
	if totalCount > 0 {
		opts.Progress = &ProgressState{
			Total:   totalCount,
			Current: 0,
		}
	}

	for name, pNode := range rootParsed.Nodes {
		child := crawlCommand([]string{binary, name}, opts, visited)
		child.Description = pNode.Description
		result[name] = child
	}

	return result, nil
}

