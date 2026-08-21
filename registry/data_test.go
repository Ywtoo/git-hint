package registry_test

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"git-hint/core"
	"git-hint/engine/provider"
)

// --------------------------------------------------------------------------
// Data structure mirroring git.json
// --------------------------------------------------------------------------

type CommandEntry struct {
	Description         string                  `json:"description"`
	CompleteDescription string                  `json:"completeDescription"`
	NUsed               int                     `json:"nUsed"`
	SubCommand          map[string]CommandEntry `json:"subCommand"`
}

// rePlaceholder detects <flag> (dynamic wildcard placeholder).
var rePlaceholder = regexp.MustCompile(`^<(.+)>$`)

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// loadJSON reads git.json from the real data/ directory (next to test binary),
// via core.CommandDataPath — the same resolution used in production.
// If the file has not been generated yet, the test is skipped instead of failing.
func loadJSON(t *testing.T) map[string]CommandEntry {
	t.Helper()

	path := gitJSONPath(t)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("git.json not found in %s — run 'githint rebuild' before running this test", path)
	}

	var root map[string]CommandEntry
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	return root
}

func gitJSONPath(t *testing.T) string {
	t.Helper()

	path, err := core.CommandDataPath("git")
	if err != nil {
		t.Fatalf("failed to resolve git.json path: %v", err)
	}
	return path
}

// collectPlaceholders recursively walks commands and collects all placeholder
// names (without angle brackets) along with their path for clear error messages.
func collectPlaceholders(commands map[string]CommandEntry, path string, out map[string][]string) {
	for name, cmd := range commands {
		currentPath := path + " > " + name
		if m := rePlaceholder.FindStringSubmatch(name); len(m) == 2 {
			flag := m[1]
			out[flag] = append(out[flag], currentPath)
		}
		if cmd.SubCommand != nil {
			collectPlaceholders(cmd.SubCommand, currentPath, out)
		}
	}
}

// --------------------------------------------------------------------------
// Tests
// --------------------------------------------------------------------------

// TestJSONParsable ensures the file is valid JSON and not empty.
func TestJSONParsable(t *testing.T) {
	root := loadJSON(t)
	if len(root) == 0 {
		t.Fatal("git.json is empty or has no root level commands")
	}
	t.Logf("✅  git.json loaded with %d root level commands", len(root))
}

// TestCommandsHaveDescription ensures no root command is missing a description.
func TestCommandsHaveDescription(t *testing.T) {
	root := loadJSON(t)
	for name, cmd := range root {
		if strings.TrimSpace(cmd.Description) == "" {
			t.Errorf("command %q has no 'description'", name)
		}
	}
}

// TestSubCommandsHaveDescription ensures all subcommands (except placeholders)
// have a description.
func TestSubCommandsHaveDescription(t *testing.T) {
	root := loadJSON(t)

	var check func(map[string]CommandEntry, string)
	check = func(commands map[string]CommandEntry, path string) {
		for name, cmd := range commands {
			fullPath := path + " > " + name
			isPlaceholder := rePlaceholder.MatchString(name)
			if !isPlaceholder && strings.TrimSpace(cmd.Description) == "" {
				t.Errorf("subcommand missing description: %s", fullPath)
			}
			if cmd.SubCommand != nil {
				check(cmd.SubCommand, fullPath)
			}
		}
	}
	check(root, "git")
}

// TestPlaceholdersProvider scans all <flag> placeholders found in JSON
// and compares them against provider.Providers.
func TestPlaceholdersProvider(t *testing.T) {
	root := loadJSON(t)

	found := make(map[string][]string)
	collectPlaceholders(root, "git", found)

	var withProvider []string
	var withoutProvider []string

	for flag := range found {
		if _, ok := provider.Providers[flag]; ok {
			withProvider = append(withProvider, "<"+flag+">")
		} else {
			withoutProvider = append(withoutProvider, "<"+flag+">")
		}
	}

	sort.Strings(withProvider)
	sort.Strings(withoutProvider)

	var orphans []string
	for flag := range provider.Providers {
		if _, ok := found[flag]; !ok {
			orphans = append(orphans, flag)
		}
	}
	sort.Strings(orphans)

	var sb strings.Builder
	sb.WriteString("\n=======================================================\n")
	sb.WriteString("            PLACEHOLDERS & PROVIDERS REPORT            \n")
	sb.WriteString("=======================================================\n\n")

	fmt.Fprintf(&sb, "🟢 SUPPORTED PROVIDERS IN USE (%d):\n", len(withProvider))
	for _, p := range withProvider {
		fmt.Fprintf(&sb, "   - %s\n", p)
	}
	sb.WriteString("\n")

	fmt.Fprintf(&sb, "⚠️  ORPHAN PROVIDERS (In Go, but missing from git.json) (%d):\n", len(orphans))
	if len(orphans) == 0 {
		sb.WriteString("   (No orphan providers found!)\n")
	} else {
		for _, o := range orphans {
			fmt.Fprintf(&sb, "   - <%s>  --> Provider exists in engine/provider, but <%s> is not used in JSON!\n", o, o)
		}
	}
	sb.WriteString("\n")

	fmt.Fprintf(&sb, "📝 MANUAL / UNIMPLEMENTED PLACEHOLDERS (%d):\n", len(withoutProvider))
	for _, flagWithBrackets := range withoutProvider {
		flag := strings.Trim(flagWithBrackets, "<>")
		paths := found[flag]
		fmt.Fprintf(&sb, "   • %s (%d occurrence(s))\n", flagWithBrackets, len(paths))
		for _, p := range paths {
			fmt.Fprintf(&sb, "       └── %s\n", p)
		}
	}
	sb.WriteString("=======================================================\n")

	t.Log(sb.String())

	for _, o := range orphans {
		t.Errorf("❌ Orphan provider detected: %q is in provider.Providers but never appears as <%s> in JSON", o, o)
	}
}

// TestRootAlphabeticalOrder verifies that root level commands are in alphabetical order.
func TestRootAlphabeticalOrder(t *testing.T) {
	path, err := core.CommandDataPath("git")
	if err != nil {
		t.Fatalf("failed to resolve git.json path: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("git.json not found in %s — run 'githint rebuild' before running this test", path)
	}

	jsonKeys := extractTopLevelKeysOrdered(data)
	for i := 1; i < len(jsonKeys); i++ {
		if jsonKeys[i] < jsonKeys[i-1] {
			t.Errorf("root keys out of order: %q comes after %q (run scripts/sort_json.go)",
				jsonKeys[i], jsonKeys[i-1])
		}
	}
	if !t.Failed() {
		t.Logf("✅  %d root keys in alphabetical order", len(jsonKeys))
	}
}

// extractTopLevelKeysOrdered extracts the root level keys of a JSON object
// in the order they appear in the raw text, without using encoding/json (which doesn't preserve order).
func extractTopLevelKeysOrdered(data []byte) []string {
	var keys []string
	depth := 0
	i := 0
	for i < len(data) {
		c := data[i]
		switch c {
		case '{':
			depth++
			i++
		case '}':
			depth--
			i++
		case '"':
			j := i + 1
			for j < len(data) && data[j] != '"' {
				if data[j] == '\\' {
					j++
				}
				j++
			}
			key := string(data[i+1 : j])
			i = j + 1
			for i < len(data) && (data[i] == ' ' || data[i] == '\n' || data[i] == '\r' || data[i] == '\t') {
				i++
			}
			if i < len(data) && data[i] == ':' && depth == 1 {
				keys = append(keys, key)
			}
		default:
			i++
		}
	}
	return keys
}
