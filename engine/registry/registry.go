package registry

import (
	"fmt"
	"os"
	"path/filepath"
)

// TODO: Array de comandos
func ResolveCommandPath(commandName string) (commandPath string, err error) {
	pathRoot, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("não foi possível obter o diretório atual: %w", err)
	}

	for {
		_, err = os.Stat(filepath.Join(pathRoot, "go.mod"))

		if err != nil {
			pathRootTry := filepath.Dir(pathRoot)

			if pathRoot == pathRootTry {
				return "", fmt.Errorf("raiz do projeto não encontrada (limite do sistema atingido): %s", pathRootTry)
			}
			pathRoot = pathRootTry
		} else {
			break
		}
	}

	commandPath = filepath.Join(pathRoot, "data", commandName+".json")

	_, err = os.Stat(commandPath)
	if err != nil {
		return "", fmt.Errorf("comando não encontrado: %s", commandName)
	}

	return commandPath, nil
}
