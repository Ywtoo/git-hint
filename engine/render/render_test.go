package render

import (
	"strings"
	"testing"

	"git-hint/core"
)

func TestFormatList(t *testing.T) {
	matches := []core.CommandMatch{
		{Name: "--progress", Description: "show progress"},
		{Name: "--no-progress", Description: "don't show progress"},
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
			name:       "Group placeholder is rendered in green and non-selectable",
			selected:   2,
			buffer:     "git clone ",
			token:      "",
			promptCol:  0,
			renderMode: "basic",
			contains:   []string{"  \x01<dir>\x02", " >/jdjskhdf/klajs"},
		},
		{
			name:       "Placeholder header only when Name is empty",
			selected:   0,
			buffer:     "git commit -m ",
			token:      "",
			promptCol:  0,
			renderMode: "basic",
			contains:   []string{"  \x01<message>  commit message\x02"},
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
			if tt.name == "Placeholder header only when Name is empty" {
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
