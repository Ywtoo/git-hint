package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-hint/internal/core"
	"git-hint/internal/registry"
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
		"rebase": {
			"name": "rebase",
			"description": "Reapply commits on top of another base tip",
			"options": {
				"-i": {
					"name": "-i",
					"kind": "option",
					"description": "Make a list of the commits which are about to be rebased"
				},
				"--onto": {
					"name": "--onto",
					"kind": "option",
					"description": "Rebase onto given upstream",
					"subCommand": {
						"<commit>": {
							"name": "<commit>",
							"kind": "argument",
							"description": "commit to rebase onto"
						}
					},
					"requires": ["<commit>"]
				}
			},
			"subCommand": {
				"-i": {
					"name": "-i",
					"kind": "option",
					"description": "Make a list of the commits which are about to be rebased"
				},
				"--onto": {
					"name": "--onto",
					"kind": "option",
					"description": "Rebase onto given upstream",
					"subCommand": {
						"<commit>": {
							"name": "<commit>",
							"kind": "argument",
							"description": "commit to rebase onto"
						}
					},
					"requires": ["<commit>"]
				},
				"<commit>": {
					"name": "<commit>",
					"kind": "argument",
					"description": "upstream commit or ref"
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
		{"Required-now flag argument", "git commit -m "},
		{"Flags combined before required-now", "git commit -a -m "},
		{"Rebase with interactive flag first", "git rebase -i "},
		{"Rebase typing flag prefix", "git rebase -"},
		{"Rebase required-now for --onto", "git rebase --onto "},
		{"Commit with completed message", `git commit -m "foo" `},
		{"Commit single completed flag no argument", "git commit -a"},
		{"Commit single completed flag requires argument", "git commit -m"},
		{"Commit completed message without trailing space", `git commit -m "foo"`},
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

			switch tt.name {
			case "Commit single completed flag no argument":
				if len(sortedMatches) == 0 {
					t.Fatalf("Expected next flags for %s, got none", tt.name)
				}
				for _, m := range sortedMatches {
					if m.Name == "-a" {
						t.Errorf("Should have jumped past completed -a, but still suggested -a")
					}
				}
			case "Commit single completed flag requires argument":
				if len(sortedMatches) == 0 {
					t.Fatalf("Expected <msg> for %s, got none", tt.name)
				}
				for _, m := range sortedMatches {
					if m.Name == "-m" {
						t.Errorf("Should have jumped past completed -m to its argument, but still suggested -m")
					}
				}
			case "Commit completed message without trailing space", "Commit with completed message":
				for _, m := range sortedMatches {
					if m.Placeholder != "" && strings.Contains(m.Placeholder, "<msg>") {
						t.Errorf("Should NOT suggest <msg> after message is completed, got placeholder %s", m.Placeholder)
					}
				}
			case "Required-now flag argument", "Flags combined before required-now":
				if len(sortedMatches) == 0 {
					t.Fatalf("Expected <msg> suggestion for %s, got none", tt.name)
				}
				for _, m := range sortedMatches {
					if strings.HasPrefix(m.Name, "-") {
						t.Errorf("Required-now state should NOT suggest flag %s", m.Name)
					}
				}
			case "Rebase required-now for --onto":
				// A required-now flag with no provider (like --onto's <arg>)
				// emits nothing: the shell's own completion takes over. The
				// assertions guard against flag leakage and literal insertion.
				for _, m := range sortedMatches {
					if strings.HasPrefix(m.Name, "-") {
						t.Errorf("--onto required-now state should NOT suggest flag %s", m.Name)
					}
					if m.Name == "<arg>" {
						t.Errorf("literal placeholder <arg> must not be suggested")
					}
				}
			case "Rebase typing flag prefix":
				for _, m := range sortedMatches {
					if !strings.HasPrefix(m.Name, "-") && m.Placeholder == "" {
						t.Errorf("Typing '-' prefix should only match flags, got %s", m.Name)
					}
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
