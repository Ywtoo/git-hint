package app

import (
	"errors"

	"git-hint/internal/engine"
	"git-hint/internal/engine/render"
	"git-hint/internal/engine/tokenizer"
	"git-hint/internal/state"
)

// crawling tracks commands currently being indexed on demand, so a burst
// of keystrokes for the same command doesn't launch CrawlOne repeatedly
// while the first one is still running.
func List(buffer string, selected int, promptCol int, renderMode string, workingDir string) string {
	state.SetBuffer(buffer)
	state.SetWorkingDir(workingDir)

	// Expansion is scoped to one buffer: if the user left the group screen
	// (typed a different command, deleted past the entry point), the stale
	// expansion must not leak into the new suggestion list. Buffer-change
	// detection lives in the plugin; comparing with the stored buffer here
	// covers the same case for direct requests.
	if expanded, _ := state.GetExpansion(); expanded != "" {
		if state.GetBuffer() != state.GetExpansionBuffer() {
			state.ClearExpansion()
		}
	}

	parts := tokenizer.TokenizeBuffer(buffer)
	if len(parts) > 0 && parts[0] != "" {
		maybeCrawlInBackground(parts[0])
	}

	matches, currentToken, err := engine.Suggestions(buffer)
	if errors.Is(err, engine.ErrNotIndexed) {
		return handleNotIndexed(buffer)
	}
	if err != nil {
		return ""
	}

	// Headers are labels, never selectable: land the cursor on the first
	// real item (e.g. `source ` with the <file-path> header on row 0 must
	// highlight ~/.zshrc, not the label).
	selected = engine.NormalizeSelected(matches, selected)

	return render.FormatList(matches, selected, buffer, currentToken, promptCol, renderMode)
}
