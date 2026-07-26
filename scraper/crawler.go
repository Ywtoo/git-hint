package scraper

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"git-hint/core"
)

const helpTimeout = 5 * time.Second

// executeHelp runs `<path...> -h` and returns its combined output.
// We ignore the exec error itself (many CLIs exit non-zero on -h) and
// only treat a timeout as a real failure.
func executeHelp(path []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), helpTimeout)
	defer cancel()

	args := append(append([]string{}, path[1:]...), "-h")
	cmd := exec.CommandContext(ctx, path[0], args...)
	out, _ := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timeout running %s", strings.Join(path, " "))
	}
	return string(out), nil
}

// executeRootHelp tries `<binary> help -a` first (richer root listing on
// some CLIs), falling back to plain `-h` if that produced nothing.
func executeRootHelp(binary string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), helpTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "help", "-a")
	out, err := cmd.CombinedOutput()
	if err == nil || len(out) > 0 {
		return string(out), nil
	}

	return executeHelp([]string{binary})
}

type ProgressState struct {
	Total   int
	Current int
}

type Options struct {
	MaxDepth int
	Verbose  bool
	OnProgress func(current, total int, cmd string)
	Progress *ProgressState
}

// isTerminalToken tells apart a real subcommand from a flag or a
// placeholder, purely by how the parser named it: flags start with "-"
// (e.g. "--branch") and placeholders start with "<" (e.g. "<refname>").
// Neither has a "-h" of its own to run, so they're always leaves.
func isTerminalToken(name string) bool {
	return strings.HasPrefix(name, "-") || strings.HasPrefix(name, "<")
}

// crawlCommand walks the CLI's help tree depth-first, starting at path.
// For each real subcommand it runs "-h", parses the output, and — if the
// parser found more subcommands (i.e. it's not a leaf) — recurses into
// each one. Recursion stops when a node is a leaf, when the last path
// element is a flag/placeholder, when MaxDepth is reached, or when the
// same path has already been visited (cycle guard).
func crawlCommand(path []string, opts Options, visited map[string]bool) core.CommandMatch {
	node := core.NewCommandMatch("", "")

	key := strings.Join(path, " ")
	if visited[key] {
		return node
	}
	visited[key] = true

	if isTerminalToken(path[len(path)-1]) {
		return node // flag or placeholder: nothing to run "-h" against
	}

	depth := len(path) - 1 // path[0] is the root binary, so depth 0 = root command
	if depth > opts.MaxDepth {
		return node
	}

	if depth == 1 && opts.Progress != nil {
		opts.Progress.Current++
		if opts.OnProgress != nil {
			opts.OnProgress(opts.Progress.Current, opts.Progress.Total, key)
		}
	} else if opts.Verbose {
		fmt.Println("crawling:", key)
	}


	output, err := executeHelp(path)
	if err != nil {
		if opts.Verbose {
			fmt.Println("  error:", err)
		}
		return node
	}

	parsed := ParseCommand(output, path)
	if parsed.IsLeaf() {
		return node
	}

	for subName, pNode := range parsed.Nodes {
		childPath := append(append([]string{}, path...), subName)

		var child core.CommandMatch
		if isTerminalToken(subName) {
			// Flags and placeholders don't have their own -h to run.
			// Build their node directly from the parsed info.
			child = core.NewCommandMatch("", "")
			child.Description = pNode.Description
			// If this flag has placeholder children, add them.
			for chName, chNode := range pNode.Children {
				ph := core.NewCommandMatch("", "")
				ph.Description = chNode.Description
				child.SubCommand[chName] = ph
			}
		} else {
			child = crawlCommand(childPath, opts, visited)
			child.Description = pNode.Description
		}

		node.SubCommand[subName] = child
	}

	return node
}

