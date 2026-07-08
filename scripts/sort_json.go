//go:build ignore

// sort_json.go – varre o git.json e reescreve tudo em ordem alfabética.
//
// Uso:
//
//	go run scripts/sort_json.go
//	go run scripts/sort_json.go -file engine/registry/data/git.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
)

// --------------------------------------------------------------------------
// Estrutura de dados
// --------------------------------------------------------------------------

// RawCommand é a representação fiel de cada entrada do JSON.
// Usamos map[string]json.RawMessage internamente para poder reconstruir
// a ordem em qualquer nível sem perder campos desconhecidos.
type RawCommand struct {
	Description         *string                     `json:"description,omitempty"`
	CompleteDescription *string                     `json:"completeDescription,omitempty"`
	NUsed               *int                        `json:"nUsed,omitempty"`
	SubCommand          map[string]*RawCommand      `json:"subCommand,omitempty"`
}

// --------------------------------------------------------------------------
// Ordenação recursiva
// --------------------------------------------------------------------------

// sortedKeys retorna as chaves de um map em ordem alfabética.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// marshalOrdered serializa um RawCommand garantindo que o campo subCommand
// esteja escrito em ordem alfabética (recursivamente).
func marshalOrdered(cmd *RawCommand, indent string) ([]byte, error) {
	// Construímos o JSON manualmente apenas para o campo subCommand,
	// que é a única parte que o encoder padrão não garante a ordem.
	// Para os campos escalares, deixamos o encoder cuidar.

	type cmdFlat struct {
		Description         *string `json:"description,omitempty"`
		CompleteDescription *string `json:"completeDescription,omitempty"`
		NUsed               *int    `json:"nUsed,omitempty"`
	}

	flat := cmdFlat{
		Description:         cmd.Description,
		CompleteDescription: cmd.CompleteDescription,
		NUsed:               cmd.NUsed,
	}

	flatBytes, err := json.MarshalIndent(flat, indent, "  ")
	if err != nil {
		return nil, err
	}

	if len(cmd.SubCommand) == 0 {
		return flatBytes, nil
	}

	// Remove o "}" final para injetar subCommand.
	// flatBytes termina com "}" ou "\n<indent>}"
	closing := []byte("\n" + indent + "}")
	flatBytes = flatBytes[:len(flatBytes)-len(closing)]

	// Adiciona vírgula após o último campo e abre subCommand.
	flatBytes = append(flatBytes, []byte(",\n"+indent+`  "subCommand": {`)...)

	keys := sortedKeys(cmd.SubCommand)
	innerIndent := indent + "    "
	for i, k := range keys {
		sub := cmd.SubCommand[k]
		subBytes, err := marshalOrdered(sub, innerIndent)
		if err != nil {
			return nil, fmt.Errorf("erro ao serializar subcomando %q: %w", k, err)
		}
		keyJSON, _ := json.Marshal(k)
		flatBytes = append(flatBytes, []byte("\n"+indent+`    `+string(keyJSON)+`: `)...)
		flatBytes = append(flatBytes, subBytes...)
		if i < len(keys)-1 {
			flatBytes = append(flatBytes, ',')
		}
	}

	flatBytes = append(flatBytes, []byte("\n"+indent+"  }")...)
	flatBytes = append(flatBytes, closing...)

	return flatBytes, nil
}

// --------------------------------------------------------------------------
// main
// --------------------------------------------------------------------------

func main() {
	filePath := flag.String("file", "engine/registry/data/git.json", "Caminho para o arquivo JSON a ordenar")
	flag.Parse()

	data, err := os.ReadFile(*filePath)
	if err != nil {
		log.Fatalf("Erro ao ler %s: %v", *filePath, err)
	}

	// Faz o unmarshal no mapa raiz (cada chave é um comando git).
	var root map[string]*RawCommand
	if err := json.Unmarshal(data, &root); err != nil {
		log.Fatalf("Erro ao parsear JSON: %v", err)
	}

	// Ordena as chaves de nível raiz.
	keys := sortedKeys(root)

	// Monta o JSON final manualmente para garantir a ordem em todos os níveis.
	out := []byte("{\n")
	for i, k := range keys {
		cmd := root[k]
		cmdBytes, err := marshalOrdered(cmd, "  ")
		if err != nil {
			log.Fatalf("Erro ao serializar comando %q: %v", k, err)
		}
		keyJSON, _ := json.Marshal(k)
		out = append(out, []byte("  "+string(keyJSON)+": ")...)
		out = append(out, cmdBytes...)
		if i < len(keys)-1 {
			out = append(out, ',')
		}
		out = append(out, '\n')
	}
	out = append(out, '}')
	out = append(out, '\n')

	if err := os.WriteFile(*filePath, out, 0644); err != nil {
		log.Fatalf("Erro ao escrever %s: %v", *filePath, err)
	}

	fmt.Printf("✅  %s reescrito com %d comandos em ordem alfabética.\n", *filePath, len(keys))
}
