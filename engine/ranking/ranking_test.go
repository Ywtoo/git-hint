package ranking

import (
	"git-hint/engine/parser"
	"strings"
	"testing"
)

func TestRankSuggestionsWithReader(t *testing.T) {
	tests := []struct {
		name            string
		commandName     string
		history         string
		suggestions     []parser.CommandMatch
		expectedOrder   []string
	}{
		{
			name:        "Prioritize frequency over alphabet",
			commandName: "git",
			history:     ": 1718540000:0;git checkout\n: 1718540001:0;git checkout\n: 1718540002:0;git status\n",
			suggestions: []parser.CommandMatch{
				{Name: "checkout", Description: "desc1"},
				{Name: "status", Description: "desc2"},
				{Name: "add", Description: "desc3"},
			},
			expectedOrder: []string{"checkout", "status", "add"},
		},
		{
			name:        "Alphabetical tie-break",
			commandName: "git",
			history:     ": 1718540000:0;git add\n: 1718540001:0;git commit\n",
			suggestions: []parser.CommandMatch{
				{Name: "commit", Description: "desc1"},
				{Name: "add", Description: "desc2"},
			},
			expectedOrder: []string{"add", "commit"},
		},
		{
			name:        "Flag priority over alphabet",
			commandName: "git commit",
			history:     ": 1718540000:0;git commit -m 'msg'\n: 1718540001:0;git commit -m 'msg2'\n: 1718540002:0;git commit -a\n",
			suggestions: []parser.CommandMatch{
				{Name: "-a", Description: "desc1"},
				{Name: "-m", Description: "desc2"},
				{Name: "--amend", Description: "desc3"},
			},
			expectedOrder: []string{"-m", "-a", "--amend"},
		},
		{
			name:        "Command name with arguments",
			commandName: "git remote",
			history:     ": 1718540000:0;git remote add\n: 1718540001:0;git remote add\n: 1718540002:0;git remote set-url\n",
			suggestions: []parser.CommandMatch{
				{Name: "add", Description: "desc1"},
				{Name: "set-url", Description: "desc2"},
				{Name: "remove", Description: "desc3"},
			},
			expectedOrder: []string{"add", "set-url", "remove"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.history)
			got, err := RankSuggestionsWithReader(tt.commandName, tt.suggestions, reader)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(got) != len(tt.expectedOrder) {
				t.Errorf("Expected length %d, got %d", len(tt.expectedOrder), len(got))
			}

			for i, name := range tt.expectedOrder {
				if i < len(got) && got[i].Name != name {
					t.Errorf("At index %d, expected %s, got %s", i, name, got[i].Name)
				}
			}
		})
	}
}
