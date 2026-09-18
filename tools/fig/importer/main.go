// figimport — DEV-ONLY tool. Never ships to users.
//
// Converts Fig autocomplete specs (npm @withfig/autocomplete) into
// git-hint's data/<cmd>.json format. The generated files are committed
// to the repo in data/ and ship with the plugin, so end users never run
// this. The runtime crawler only kicks in for commands not covered by
// the generated bundle.
//
// Usage (from repo root):
//
//	cd tools/fig/node && npm install       # once, downloads the specs
//	go run ./tools/fig/importer           # writes data/fig/<cmd>.json
//
// A single node process loads every build/*.js bundle and prints one
// minified JSON per line ("<name>\t<json>"); Go parses and re-encodes
// each spec into core.CommandMatch trees identical to what the crawler's
// WriteCommand produces — same folder, same format, same consumer.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git-hint/internal/core"
)

// ── Fig schema types (subset we consume) ────────────────────────────

// figSpec mirrors Fig's v7 completion spec. Name is string or []string.
type figSpec struct {
	Name        interface{}   `json:"name"`
	Description string        `json:"description"`
	Subcommands []figSpec     `json:"subcommands"`
	Options     []figOption   `json:"options"`
	Args        *figArgOrList `json:"args"` // object or array
	Hidden      bool          `json:"hidden"`
}

type figOption struct {
	Name        interface{}   `json:"name"` // string or []string (aliases)
	Description string        `json:"description"`
	Args        *figArgOrList `json:"args"` // object or array
	Hidden      bool          `json:"hidden"`
}

// figArgOrList handles Fig's "args" being a single object or an array.
type figArgOrList []figArg

func (a *figArgOrList) UnmarshalJSON(data []byte) error { //nolint:unparam // deliberate: never fail the whole import on one malformed field
	var single figArg
	if err := json.Unmarshal(data, &single); err == nil && single.Name != "" {
		*a = figArgOrList{single}
		return nil
	}
	var list []figArg
	if err := json.Unmarshal(data, &list); err == nil {
		*a = list
		return nil
	}
	// Array entries without a name (generator-only) — keep the named ones.
	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		for _, r := range raw {
			var name string
			if err := json.Unmarshal(r["name"], &name); err == nil && name != "" {
				*a = append(*a, figArg{Name: name})
			}
		}
	}
	// A malformed args field must not reject the whole spec — swallowing
	// here keeps the import resilient, so the error result is always nil.
	return nil
}

type figArg struct {
	Name       string `json:"name"`
	IsOptional bool   `json:"isOptional"`
}

// orEmpty returns the parsed arg list, tolerating a nil pointer.
func (a *figArgOrList) orEmpty() []figArg {
	if a == nil {
		return nil
	}
	return *a
}

// ── Conversion to git-hint format ───────────────────────────────────

func figName(n interface{}) string {
	switch v := n.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

func figAllNames(n interface{}) []string {
	switch v := n.(type) {
	case string:
		return []string{v}
	case []interface{}:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func placeholder(name string) string {
	if name == "" || strings.HasPrefix(name, "<") {
		return name
	}
	return "<" + name + ">"
}

// convertSubcommand converts a Fig spec node into a git-hint CommandMatch,
// mirroring the shape crawlCommand + attachChildNode produce at runtime:
// every child in SubCommand, flags/arguments also mirrored into Options.
func convertSubcommand(spec figSpec) core.CommandMatch {
	cmd := core.CommandMatch{
		Name:        figName(spec.Name),
		Kind:        core.KindSubcommand,
		Description: spec.Description,
		SubCommand:  make(map[string]core.CommandMatch),
		Options:     make(map[string]core.CommandMatch),
	}

	for _, sub := range spec.Subcommands {
		name := figName(sub.Name)
		if name == "" || sub.Hidden {
			continue
		}
		cmd.SubCommand[name] = convertSubcommand(sub)
	}

	for _, opt := range spec.Options {
		names := figAllNames(opt.Name)
		if len(names) == 0 || opt.Hidden {
			continue
		}
		primary := names[0]
		child := convertOption(opt, primary)
		cmd.SubCommand[primary] = child
		if strings.HasPrefix(primary, "-") {
			cmd.Options[primary] = child
		}
		for _, alias := range names[1:] {
			aliasChild := child
			aliasChild.Name = alias
			cmd.SubCommand[alias] = aliasChild
			if strings.HasPrefix(alias, "-") {
				cmd.Options[alias] = aliasChild
			}
		}
	}

	for _, arg := range spec.Args.orEmpty() {
		if arg.Name == "" {
			continue
		}
		ph := placeholder(arg.Name)
		argChild := core.CommandMatch{
			Name:        ph,
			Kind:        core.KindArgument,
			Description: arg.Name,
		}
		cmd.SubCommand[ph] = argChild
		cmd.Options[ph] = argChild
		if !arg.IsOptional {
			cmd.Requires = append(cmd.Requires, ph)
		}
	}

	return cmd
}

func convertOption(opt figOption, primary string) core.CommandMatch {
	cmd := core.CommandMatch{
		Name:        primary,
		Kind:        core.KindOption,
		Description: opt.Description,
		SubCommand:  make(map[string]core.CommandMatch),
		Options:     make(map[string]core.CommandMatch),
	}

	if opt.Args != nil {
		for _, arg := range *opt.Args {
			if arg.Name == "" {
				continue
			}
			ph := placeholder(arg.Name)
			argChild := core.CommandMatch{
				Name:        ph,
				Kind:        core.KindArgument,
				Description: arg.Name,
			}
			cmd.SubCommand[ph] = argChild
			cmd.Options[ph] = argChild
			if !arg.IsOptional {
				cmd.Requires = append(cmd.Requires, ph)
			}
		}
	}
	return cmd
}

// ── Node bridge ─────────────────────────────────────────────────────

// nodeLoaderScript prints one "<name>\t<minified json>" line per spec.
// Minified JSON never contains raw newlines (they're escaped), so line
// splitting is safe.
const nodeLoaderScript = `
const fs = require("fs");
const path = require("path");
const dir = path.resolve(process.argv[2]);
for (const f of fs.readdirSync(dir)) {
  if (!f.endsWith(".js") || f === "index.js") continue;
  try {
    const mod = require(path.join(dir, f));
    const spec = mod.default || mod;
    if (!spec || !spec.name) continue;
    process.stdout.write(f.slice(0, -3) + "\t" + JSON.stringify(spec) + "\n");
  } catch (e) {
    process.stderr.write("ERR " + f + " " + e.message + "\n");
  }
}
`

func loadAllSpecsWithNode(buildDir string) (map[string][]byte, error) {
	tmp, err := os.CreateTemp("", "figload-*.js")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(nodeLoaderScript); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()

	cmd := exec.Command("node", tmpName, buildDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("node: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	out := make(map[string][]byte)
	scanner := bufio.NewScanner(&stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 64*1024*1024) // aws.js is huge
	for scanner.Scan() {
		line := scanner.Bytes()
		tab := bytes.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		out[string(line[:tab])] = append([]byte(nil), line[tab+1:]...)
	}
	return out, scanner.Err()
}

// ── Main ────────────────────────────────────────────────────────────

func main() {
	buildDir := "tools/fig/node/node_modules/@withfig/autocomplete/build"
	if env := os.Getenv("FIG_BUILD_DIR"); env != "" {
		buildDir = env
	}
	outDir := "data/fig"
	if env := os.Getenv("GITHINT_DATA_DIR"); env != "" {
		outDir = env
	}

	if _, err := os.Stat(buildDir); err != nil {
		fmt.Fprintf(os.Stderr, "Fig specs not found at %s\n", buildDir)
		fmt.Fprintf(os.Stderr, "run: cd tools/fig/node && npm install\n")
		os.Exit(1)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "cannot create %s: %v\n", outDir, err)
		os.Exit(1)
	}

	specs, err := loadAllSpecsWithNode(buildDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	ok, skipped := 0, 0
	for name, raw := range specs {
		var spec figSpec
		if err := json.Unmarshal(raw, &spec); err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", name, err)
			skipped++
			continue
		}

		tree := convertSubcommand(spec).SubCommand
		if len(tree) == 0 {
			skipped++
			continue
		}

		out, err := json.MarshalIndent(tree, "", "  ")
		if err != nil {
			skipped++
			continue
		}
		target := filepath.Join(outDir, name+".json")
		if err := os.WriteFile(target, append(out, '\n'), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "  write %s: %v\n", target, err)
			skipped++
			continue
		}
		ok++
	}

	fmt.Printf("figimport: wrote %d files to %s (skipped %d)\n", ok, outDir, skipped)
}
