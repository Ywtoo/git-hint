package render

import (
	"strings"
	"testing"

	"git-hint/internal/core"
)

func TestFormatList(t *testing.T) {
	matches := []core.CommandMatch{
		{Name: "--progress", Description: "show progress"},
		{Name: "--no-progress", Description: "don't show progress"},
		{Name: "", Placeholder: "<dir>"},
		{Name: "/jdjskhdf/klajs", Placeholder: "<dir>"},
		{Name: "/tmp/foo", Placeholder: "<dir>"},
	}

	tests := []struct {
		name        string
		selected    int
		buffer      string
		token       string
		promptCol   int
		renderMode  string
		contains    []string
		notContains []string
	}{
		{
			name:        "Basic mode - no indent",
			selected:    0,
			buffer:      "git clone ",
			token:       "",
			promptCol:   10,
			renderMode:  "basic",
			contains:    []string{" >--progress", "  \x01<dir>\x02"},
			notContains: []string{"          "},
		},
		{
			name:       "Group placeholder is rendered in green",
			selected:   3,
			buffer:     "git clone ",
			token:      "",
			promptCol:  0,
			renderMode: "basic",
			contains:   []string{"  \x01<dir>\x02", " >/jdjskhdf/klajs"},
		},
		{
			name:       "Selected placeholder header shows cursor",
			selected:   0,
			buffer:     "git commit -m ",
			token:      "",
			promptCol:  0,
			renderMode: "basic",
			contains:   []string{"\x05 ><message>  commit message\x06"},
		},
		{
			name:       "Empty list",
			selected:   0,
			buffer:     "git ",
			token:      "",
			promptCol:  0,
			renderMode: "basic",
			contains:   []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Empty list" {
				res := FormatList([]core.CommandMatch{}, tt.selected, tt.buffer, tt.token, tt.promptCol, tt.renderMode)
				if res != "" {
					t.Errorf("Expected empty string, got %q", res)
				}
				return
			}

			matchesToUse := matches
			if tt.name == "Selected placeholder header shows cursor" {
				matchesToUse = []core.CommandMatch{
					{Name: "", Placeholder: "<message>  commit message"},
				}
			}

			res := FormatList(matchesToUse, tt.selected, tt.buffer, tt.token, tt.promptCol, tt.renderMode)
			for _, s := range tt.contains {
				if !strings.Contains(res, s) {
					t.Errorf("Expected result to contain %q, got:\n%s", s, res)
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(res, s) {
					t.Errorf("Expected result NOT to contain %q, got:\n%s", s, res)
				}
			}
		})
	}
}

func TestFormatListKeepsSelectionOnGroupHeader(t *testing.T) {
	got := FormatList([]core.CommandMatch{
		{Name: "", Placeholder: "<branch>"},
		{Name: "main", Placeholder: "<branch>"},
	}, 0, "git checkout ", "", 0, "basic")

	if !strings.Contains(got, "\x05 ><branch>\x06") {
		t.Fatalf("selected header did not render a cursor: %q", got)
	}
}

func TestFormatListFlattensMultilineDescription(t *testing.T) {
	got := FormatList([]core.CommandMatch{{Name: "--example", Description: "first line\nsecond line"}}, 0, "cmd ", "", 0, "basic")
	if strings.Contains(got, "first line\nsecond line") {
		t.Fatalf("description created an extra terminal row: %q", got)
	}
	if !strings.Contains(got, "first line second line") {
		t.Fatalf("description was not flattened: %q", got)
	}
}

func TestFormatListTruncatesLongDescriptionWithoutPanicking(t *testing.T) {
	got := FormatList([]core.CommandMatch{{
		Name:        "--example",
		Description: "Uma descrição muito longa que não deve ocupar diversas linhas no terminal.",
	}}, 0, "cmd ", "", 0, "basic")

	if !strings.Contains(got, "…") {
		t.Fatalf("long description was not truncated: %q", got)
	}
	if strings.Contains(got, "diversas linhas no terminal") {
		t.Fatalf("long description was not shortened: %q", got)
	}
}
