package engine

import (
	"fmt"
	"testing"

	"git-hint/engine/parser"
)

func TestIntegration(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Simple prefix", "git c"},
		{"Full command", "git commit"},
		{"Subcommand jump", "git push"},
		{"Dynamic placeholder", "git push -u origin "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("🚀 User Input Test: \"%s\"\n", tt.input)

			sortedMatches, currentToken, _, err := Suggestions(tt.input)
			if err != nil {
				t.Fatalf("Erro: %s", err)
			}

			fmt.Printf("   Token Atual: \"%s\"\n", currentToken)

			if len(sortedMatches) == 0 {
				fmt.Println("   ⚠️ No suggestions found")
			} else {
				for _, match := range sortedMatches {
					fmt.Printf("   👉 %s (Uso: %d): %s\n", match.Name, match.NUsed, match.Description)
				}
			}
		})
	}
}

func TestCompleteBuffer(t *testing.T) {
	tests := []struct {
		name       string
		buffer     string
		selIndex   int
		suggestions []parser.CommandMatch
		token      string
		expected   string
	}{
		{
			name:     "Append suggestion",
			buffer:   "git ",
			selIndex: 0,
			suggestions: []parser.CommandMatch{
				{Name: "commit"},
			},
			token:    "",
			expected: "git commit",
		},
		{
			name:     "Replace partial",
			buffer:   "git com",
			selIndex: 0,
			suggestions: []parser.CommandMatch{
				{Name: "commit"},
			},
			token:    "com",
			expected: "git commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompleteBuffer(tt.buffer, tt.selIndex, tt.suggestions, tt.token)
			fmt.Printf("Buffer: \"%s\" -> Result: \"%s\"\n", tt.buffer, got)
			if got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}
