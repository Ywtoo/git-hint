package history

import (
	"os"
	"testing"
)

func TestFindHistoryCommands(t *testing.T) {
	content := ": 1712345678:0;git checkout main\n: 1712345679:0;git status\n: 1712345680:0;git checkout feature/test\n"
	tmpFile, err := os.CreateTemp("", "zsh_history_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Mock the history path to use our temp file
	originalFunc := historyPathFunc
	historyPathFunc = func() (string, error) {
		return tmpFile.Name(), nil
	}
	defer func() { historyPathFunc = originalFunc }()

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Filter checkout",
			query:    "git checkout",
			expected: []string{"git checkout feature/test", "git checkout main"},
		},
		{
			name:     "Filter status",
			query:    "git status",
			expected: []string{"git status"},
		},
		{
			name:     "No match",
			query:    "git commit",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindHistoryCommands(tt.query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(got) != len(tt.expected) {
				t.Errorf("Expected len %d, got %d (%v)", len(tt.expected), len(got), got)
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("At index %d: expected %s, got %s", i, tt.expected[i], got[i])
				}
			}
		})
	}
}
