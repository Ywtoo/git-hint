//go:build ignore

// sort_json.go – scans git.json and rewrites everything in alphabetical order.
//
// Usage:
//
//	go run ./tools/dev/sort_json.go
//	go run ./tools/dev/sort_json.go -file data/fig/git.json
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
// Data structure
// --------------------------------------------------------------------------

// RawCommand is the direct representation of each JSON entry.
// We use map[string]*RawCommand for subCommand to reconstruct order
// at any nesting level.
type RawCommand struct {
	Description         *string                `json:"description,omitempty"`
	CompleteDescription *string                `json:"completeDescription,omitempty"`
	NUsed               *int                   `json:"nUsed,omitempty"`
	SubCommand          map[string]*RawCommand `json:"subCommand,omitempty"`
}

// --------------------------------------------------------------------------
// Recursive sorting
// --------------------------------------------------------------------------

// sortedKeys returns map keys in alphabetical order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// marshalOrdered serializes a RawCommand ensuring that subCommand
// is written in alphabetical order (recursively).
func marshalOrdered(cmd *RawCommand, indent string) ([]byte, error) {
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

	// Strip closing "}" to inject subCommand.
	closing := []byte("\n" + indent + "}")
	flatBytes = flatBytes[:len(flatBytes)-len(closing)]

	flatBytes = append(flatBytes, []byte(",\n"+indent+`  "subCommand": {`)...)

	keys := sortedKeys(cmd.SubCommand)
	innerIndent := indent + "    "
	for i, k := range keys {
		sub := cmd.SubCommand[k]
		subBytes, err := marshalOrdered(sub, innerIndent)
		if err != nil {
			return nil, fmt.Errorf("error serializing subcommand %q: %w", k, err)
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
	filePath := flag.String("file", "engine/registry/data/git.json", "Path to JSON file to sort")
	flag.Parse()

	data, err := os.ReadFile(*filePath)
	if err != nil {
		log.Fatalf("Error reading %s: %v", *filePath, err)
	}

	var root map[string]*RawCommand
	if err := json.Unmarshal(data, &root); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	keys := sortedKeys(root)

	out := []byte("{\n")
	for i, k := range keys {
		cmd := root[k]
		cmdBytes, err := marshalOrdered(cmd, "  ")
		if err != nil {
			log.Fatalf("Error serializing command %q: %v", k, err)
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
		log.Fatalf("Error writing %s: %v", *filePath, err)
	}

	fmt.Printf("✅  %s rewritten with %d commands in alphabetical order.\n", *filePath, len(keys))
}
