package render

import (
	"git-hint/engine/parser"
	"strings"
	"testing"
)

func TestFormatList(t *testing.T) {
	matches := []parser.CommandMatch{
		{Name: "commit", Description: "desc1"},
		{Name: "checkout", Description: "desc2"},
		{Name: "status", Description: "desc3"},
		{Name: "push", Description: "desc4"},
		{Name: "pull", Description: "desc5"},
		{Name: "add", Description: "desc6"},
	}

	tests := []struct {
		name       string
		selected   int
		buffer     string
		token      string
		promptCol  int
		renderMode string
		contains   []string
		notContains []string
	}{
		{
			name:       "Basic mode - no indent",
			selected:   0,
			buffer:     "git ",
			token:      "",
			promptCol:  10,
			renderMode: "basic",
			contains:   []string{" >commit"},
			notContains: []string{"          "},
		},
		{
			name:       "OMZ mode - basic indent",
			selected:   0,
			buffer:     "git ",
			token:      "",
			promptCol:  10,
			renderMode: "ohmyzsh",
			contains:   []string{"           >commit"}, // 10 (prompt) + 4 (git ) + 1 = 15 spaces
		},
		{
			name:       "OMZ mode - partial token indent",
			selected:   0,
			buffer:     "git com",
			token:      "com",
			promptCol:  10,
			renderMode: "ohmyzsh",
			contains:   []string{"          >commit"}, // 10 (prompt) + 4 (git ) = 14 spaces
		},
		{
			name:       "Sliding window - middle",
			selected:   2,
			buffer:     "git ",
			token:      "",
			promptCol:  0,
			renderMode: "basic",
			contains:   []string{"↑", "↓", ">status"},
		},
		{
			name:       "Sliding window - end",
			selected:   5,
			buffer:     "git ",
			token:      "",
			promptCol:  0,
			renderMode: "basic",
			contains:   []string{"↑", ">add"},
			notContains: []string{"↓"},
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
				res := FormatList([]parser.CommandMatch{}, tt.selected, tt.buffer, tt.token, tt.promptCol, tt.renderMode)
				if res != "" {
					t.Errorf("Expected empty string, got %q", res)
				}
				return
			}


			res := FormatList(matches, tt.selected, tt.buffer, tt.token, tt.promptCol, tt.renderMode)
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
