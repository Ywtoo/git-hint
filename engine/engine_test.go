package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"git-hint/core"
	"git-hint/registry"
)

func setupTestEnvironment(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexJSON := `{
		"git": {
			"name": "git",
			"description": "the stupid content tracker",
			"path": "/usr/bin/git"
		}
	}`
	if err := os.WriteFile(filepath.Join(dataDir, "index.json"), []byte(indexJSON), 0644); err != nil {
		t.Fatal(err)
	}

	gitJSON := `{
		"commit": {
			"name": "commit",
			"description": "Record changes to the repository",
			"subCommand": {
				"-m": {
					"name": "-m",
					"description": "Use the given message as the commit message",
					"subCommand": {
						"<msg>": {
							"name": "<msg>",
							"description": "commit message"
						}
					}
				},
				"-a": {
					"name": "-a",
					"description": "Automatically stage modified files"
				}
			}
		},
		"checkout": {
			"name": "checkout",
			"description": "Switch branches or restore working tree files",
			"subCommand": {
				"<branch>": {
					"name": "<branch>",
					"description": "branch to checkout"
				}
			}
		},
		"push": {
			"name": "push",
			"description": "Update remote refs along with associated objects",
			"subCommand": {
				"-u": {
					"name": "-u",
					"description": "Set upstream",
					"subCommand": {
						"<remote>": {
							"name": "<remote>",
							"description": "remote repository",
							"subCommand": {
								"<branch>": {
									"name": "<branch>",
									"description": "remote branch"
								}
							}
						}
					}
				},
				"origin": {
					"name": "origin",
					"description": "default remote"
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(dataDir, "git.json"), []byte(gitJSON), 0644); err != nil {
		t.Fatal(err)
	}

	fakeBinary := filepath.Join(tmpDir, "git-hint")
	if err := os.WriteFile(fakeBinary, []byte{}, 0755); err != nil {
		t.Fatal(err)
	}

	cleanupExe := core.SetExecutablePathForTest(func() (string, error) {
		return fakeBinary, nil
	})
	registry.ResetCacheForTest()

	t.Cleanup(func() {
		cleanupExe()
		registry.ResetCacheForTest()
	})
}

func TestIntegration(t *testing.T) {
	setupTestEnvironment(t)

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

			sortedMatches, currentToken, err := Suggestions(tt.input)
			if err != nil {
				t.Fatalf("Error: %s", err)
			}

			fmt.Printf("   Current Token: \"%s\"\n", currentToken)

			if len(sortedMatches) == 0 {
				fmt.Println("   ⚠️ No suggestions found")
			} else {
				for _, match := range sortedMatches {
					fmt.Printf("   👉 %s (Usage: %d): %s\n", match.Name, match.NUsed, match.Description)
				}
			}
		})
	}
}

func TestCompleteBuffer(t *testing.T) {
	tests := []struct {
		name        string
		buffer      string
		selIndex    int
		suggestions []core.CommandMatch
		token       string
		expected    string
	}{
		{
			name:     "Append suggestion",
			buffer:   "git ",
			selIndex: 0,
			suggestions: []core.CommandMatch{
				{Name: "commit"},
			},
			token:    "",
			expected: "git commit",
		},
		{
			name:     "Replace partial",
			buffer:   "git com",
			selIndex: 0,
			suggestions: []core.CommandMatch{
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
