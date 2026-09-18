package scraper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-hint/internal/core"
)

// ---------------------------------------------------------------------
// HELPDIR plain-text extraction
// ---------------------------------------------------------------------

func TestFirstSynopsisLines(t *testing.T) {
	file := `cd [ -qsLP ] [ arg ]
cd [ -qsLP ] old new
cd [ -qsLP ] {+|-}n
       Change the current directory.  In the first form, change the
       current directory to arg.

chdir  Same as cd.
`
	want := "cd [ -qsLP ] [ arg ]\ncd [ -qsLP ] old new\ncd [ -qsLP ] {+|-}n"
	if got := firstSynopsisLines(file, "cd"); got != want {
		t.Errorf("firstSynopsisLines:\n got: %q\nwant: %q", got, want)
	}

	// "chdir" entry inside cd's file must not be picked up when asking cd.
	if got := firstSynopsisLines("chdir  Same as cd.\n", "cd"); got != "" {
		t.Errorf("chdir line must not match cd lookup, got %q", got)
	}
}

// ---------------------------------------------------------------------
// zsh synopsis normalization (grouped flags, brace alternatives)
// ---------------------------------------------------------------------

func TestNormalizeZshGroups(t *testing.T) {
	cases := []struct{ in, want string }{
		// Clustered short flags split into one optional flag each.
		{"cd [ -qsLP ] [ arg ]", "cd [ -q ] [ -s ] [ -L ] [ -P ] [ arg ]"},
		// The brace alternative {+|-}n keeps only its "-" form.
		{"cd [ -qsLP ] {+|-}n", "cd [ -q ] [ -s ] [ -L ] [ -P ] -n"},
		// name[=value] pairs lose the =value part.
		{"alias [ {+|-}gmrsL ] [ name[=value] ... ]", "alias [ -g ] [ -m ] [ -r ] [ -s ] [ -L ] [ name ... ]"},
		// Plain lines pass through untouched.
		{"printf [ -v param ] format [ arg ... ]", "printf [ -v param ] format [ arg ... ]"},
		// Mixed: only flag-only groups expand.
		{"print [ -R [ -en ]] [ arg ... ]", "print [ -R [ -e ] [ -n ]] [ arg ... ]"},
	}
	for _, tc := range cases {
		if got := normalizeZshGroups(tc.in); got != tc.want {
			t.Errorf("normalizeZshGroups(%q):\n got: %q\nwant: %q", tc.in, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------
// Bash help extraction
// ---------------------------------------------------------------------

const sampleBashHelp = `cd: cd [-L|[-P [-e]] [-@]] [dir]
    Change the shell working directory.

    Change the current directory to DIR.  The default DIR is the value of the
    HOME shell variable.

    Options:
      -L	force symbolic links to be followed
      -P	use the physical directory structure
      -e	if the -P option is supplied, and the current working
    		directory cannot be determined successfully, exit with
    		a non-zero status
      -@	on systems that support it, present a file with extended
    		attributes as a directory containing the file attributes
`

func TestExtractBashHelpSynopsis(t *testing.T) {
	got, ok := parseBashHelpOutput("cd", sampleBashHelp)
	if !ok {
		t.Fatal("expected bash help output to be parsed")
	}
	want := "cd [-L|[-P [-e]] [-@]] [dir]"
	if got != want {
		t.Errorf("bash help synopsis:\n got: %q\nwant: %q", got, want)
	}
}

func TestExtractBashHelpSynopsis_Simple(t *testing.T) {
	output := "alias: alias [-p] [name[=value] ... ]\n    Define or display aliases.\n"
	got, ok := parseBashHelpOutput("alias", output)
	if !ok {
		t.Fatal("expected alias bash help to be parsed")
	}
	want := "alias [-p] [name[=value] ... ]"
	if got != want {
		t.Errorf("alias synopsis:\n got: %q\nwant: %q", got, want)
	}
}

func TestExtractBashHelpSynopsis_NoPrefix(t *testing.T) {
	// Some builtins might not have the "name: " prefix
	output := "echo [-neE] [arg ...]\n    Write arguments to the standard output.\n"
	got, ok := parseBashHelpOutput("echo", output)
	if !ok {
		t.Fatal("expected echo bash help to be parsed")
	}
	want := "echo [-neE] [arg ...]"
	if got != want {
		t.Errorf("echo synopsis:\n got: %q\nwant: %q", got, want)
	}
}

func TestExtractBashHelpSynopsis_EmptyOutput(t *testing.T) {
	if _, ok := parseBashHelpOutput("cd", ""); ok {
		t.Error("empty output should not parse")
	}
	if _, ok := parseBashHelpOutput("cd", "some error message"); ok {
		t.Error("error message should not parse")
	}
}

func TestExtractBashHelpSynopsis_FromRealBash(t *testing.T) {
	output, err := execBashHelp("cd")
	if err != nil || len(output) == 0 {
		t.Skip("bash help not available in this environment")
	}

	got, ok := parseBashHelpOutput("cd", output)
	if !ok {
		t.Fatal("expected cd to be parsed from real bash help")
	}
	if !strings.Contains(got, "cd") {
		t.Errorf("synopsis should contain 'cd', got: %q", got)
	}
	if !strings.Contains(got, "[dir]") {
		t.Errorf("synopsis should contain '[dir]', got: %q", got)
	}
}

// ---------------------------------------------------------------------
// Rendered man-page extraction
// ---------------------------------------------------------------------

const sampleRenderedMan = `ZSHBUILTINS(1)              General Commands Manual

NAME
       zshbuiltins - zsh built-in commands

SHELL BUILTIN COMMANDS
       alias [ {+|-}gmrsL ] [ name[=value] ... ]
              For each name with a corresponding value, define an alias.

       cd [ -qsLP ] [ arg ]
       cd [ -qsLP ] old new
       cd [ -qsLP ] {+|-}n
              Change the current directory.  In the first form, change
              the current directory to arg.

       print [ -abcDilmnNoOpPrsSz ] [ -u n ] [ -f format ] [ -C cols ]
             [ -v name ] [ -xX tabstop ] [ -R [ -en ]] [ arg ... ]
              With the -f option the arguments are printed.


       printf [ -v param ] format [ arg ... ]
              Print formatted.
`

func TestExtractRenderedSynopsis(t *testing.T) {
	// cd has three consecutive forms.
	got, ok := extractRenderedSynopsis(sampleRenderedMan, "cd")
	if !ok {
		t.Fatal("expected cd synopsis to be found")
	}
	want := "cd [ -qsLP ] [ arg ]\ncd [ -qsLP ] old new\ncd [ -qsLP ] {+|-}n"
	if got != want {
		t.Errorf("cd synopsis:\n got: %q\nwant: %q", got, want)
	}

	// print's long synopsis wraps; the bracket continuation is absorbed.
	got, ok = extractRenderedSynopsis(sampleRenderedMan, "print")
	if !ok {
		t.Fatal("expected print synopsis to be found")
	}
	want = "print [ -abcDilmnNoOpPrsSz ] [ -u n ] [ -f format ] [ -C cols ] [ -v name ] [ -xX tabstop ] [ -R [ -en ]] [ arg ... ]"
	if got != want {
		t.Errorf("print synopsis:\n got: %q\nwant: %q", got, want)
	}

	// printf must be found independently of print.
	got, ok = extractRenderedSynopsis(sampleRenderedMan, "printf")
	if !ok {
		t.Fatal("expected printf synopsis to be found")
	}
	if got != "printf [ -v param ] format [ arg ... ]" {
		t.Errorf("printf synopsis = %q", got)
	}

	if _, ok := extractRenderedSynopsis(sampleRenderedMan, "nonexistent"); ok {
		t.Error("nonexistent builtin must not be found")
	}
}

// The extractor must work against real man output when available
// (skipped silently where there is no man page, e.g. Windows CI).
func TestExtractRenderedSynopsis_FromRealMan(t *testing.T) {
	out, err := execManZshbuiltins()
	if err != nil || len(out) == 0 {
		t.Skip("man zshbuiltins not available in this environment")
	}

	got, ok := extractRenderedSynopsis(string(out), "cd")
	if !ok {
		t.Fatal("expected cd synopsis from real man page")
	}
	if !strings.Contains(got, "cd [ -qsLP ] [ arg ]") {
		t.Errorf("unexpected cd synopsis: %q", got)
	}
}

// ---------------------------------------------------------------------
// End-to-end: CrawlBuiltin writes a data file parseable by the engine
// ---------------------------------------------------------------------

func TestCrawlBuiltin_EndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	cleanupExe := core.SetExecutablePathForTest(func() (string, error) {
		return filepath.Join(tmpDir, "githint"), nil
	})
	t.Cleanup(cleanupExe)

	// Use a name that bash doesn't have help for, so the test exercises
	// the HELPDIR fallback path (bash help is tried first and fails).
	helpDir := filepath.Join(tmpDir, "help")
	if err := os.MkdirAll(helpDir, 0755); err != nil {
		t.Fatal(err)
	}
	helpFile := `myzshcmd [ -qsLP ] [ arg ]
       Change the current directory.
`
	if err := os.WriteFile(filepath.Join(helpDir, "myzshcmd"), []byte(helpFile), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HELPDIR", helpDir)
	resetBuiltinFailuresForTest()

	if err := CrawlBuiltin("myzshcmd"); err != nil {
		t.Fatalf("CrawlBuiltin: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "data", "local", "myzshcmd.json"))
	if err != nil {
		t.Fatalf("expected myzshcmd.json to be written: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("myzshcmd.json is empty")
	}

	// The tree must contain the expected nodes and none of the raw zsh
	// syntax garbage. Go's JSON encoder HTML-escapes angle brackets
	// (\u003c), so compare loosely.
	text := string(data)
	for _, want := range []string{"arg", "argument"} {
		if !strings.Contains(text, want) {
			t.Errorf("myzshcmd.json should contain %q node, got: %s", want, text)
		}
	}
	for _, garbage := range []string{"-qsLP", "-}n"} {
		if strings.Contains(text, garbage) {
			t.Errorf("myzshcmd.json must not contain raw zsh group %q, got: %s", garbage, text)
		}
	}

	// Second call must succeed (already written file is fine to rewrite).
	if err := CrawlBuiltin("myzshcmd"); err != nil {
		t.Fatalf("second CrawlBuiltin: %v", err)
	}
}

// When the HELPDIR files are absent (e.g. macOS), the bash help source
// must produce a clean tree: no raw zsh group syntax in the keys. Skipped
// silently where bash is not available (Windows CI).
func TestCrawlBuiltin_FromBashHelp(t *testing.T) {
	if _, err := execBashHelp("cd"); err != nil {
		t.Skip("bash help not available in this environment")
	}

	tmpDir := t.TempDir()
	cleanupExe := core.SetExecutablePathForTest(func() (string, error) {
		return filepath.Join(tmpDir, "githint"), nil
	})
	t.Cleanup(cleanupExe)

	// No HELPDIR or man page candidates exist; the extractor must
	// fall through to bash help (last resort).
	t.Setenv("HELPDIR", filepath.Join(tmpDir, "missing-help"))
	resetBuiltinFailuresForTest()

	if err := CrawlBuiltin("cd"); err != nil {
		t.Fatalf("CrawlBuiltin from bash help: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "data", "local", "cd.json"))
	if err != nil {
		t.Fatalf("expected cd.json: %v", err)
	}
	text := string(data)
	for _, garbage := range []string{"-qsLP", "-}n", "+|-"} {
		if strings.Contains(text, garbage) {
			t.Errorf("cd.json must not contain raw zsh group %q, got: %s", garbage, text)
		}
	}
	// bash help uses [dir], zsh uses [arg] — either is acceptable
	if !strings.Contains(text, "arg") && !strings.Contains(text, "dir") {
		t.Errorf("cd.json should contain an <arg> or <dir> node, got: %s", text)
	}
}

// When the HELPDIR files are absent and bash help succeeds (the
// primary source), the tree must be clean: no raw zsh group syntax
// in the keys. Skipped silently where bash is not available.
func TestCrawlBuiltin_FromRealMan(t *testing.T) {
	if _, err := execManZshbuiltins(); err != nil {
		t.Skip("man zshbuiltins not available in this environment")
	}

	tmpDir := t.TempDir()
	cleanupExe := core.SetExecutablePathForTest(func() (string, error) {
		return filepath.Join(tmpDir, "githint"), nil
	})
	t.Cleanup(cleanupExe)

	// No HELPDIR candidates exist; bash help is the last resort.
	t.Setenv("HELPDIR", filepath.Join(tmpDir, "missing-help"))
	resetBuiltinFailuresForTest()

	if err := CrawlBuiltin("cd"); err != nil {
		t.Fatalf("CrawlBuiltin: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "data", "local", "cd.json"))
	if err != nil {
		t.Fatalf("expected cd.json: %v", err)
	}
	text := string(data)
	for _, garbage := range []string{"-qsLP", "-}n", "+|-"} {
		if strings.Contains(text, garbage) {
			t.Errorf("cd.json must not contain raw zsh group %q, got: %s", garbage, text)
		}
	}
	// bash help uses [dir], zsh uses [arg] — either is acceptable
	if !strings.Contains(text, "arg") && !strings.Contains(text, "dir") {
		t.Errorf("cd.json should contain an <arg> or <dir> node, got: %s", text)
	}
}

// A builtin with no documentation anywhere is marked failed and never
// retried (the failure cache prevents repeated `man` runs per keystroke).
func TestCrawlBuiltin_NoDocsMarksFailed(t *testing.T) {
	tmpDir := t.TempDir()
	cleanupExe := core.SetExecutablePathForTest(func() (string, error) {
		return filepath.Join(tmpDir, "githint"), nil
	})
	t.Cleanup(cleanupExe)

	t.Setenv("HELPDIR", filepath.Join(tmpDir, "missing-help"))
	resetBuiltinFailuresForTest()

	if err := CrawlBuiltin("nosuchbuiltin"); err == nil {
		t.Fatal("expected error for builtin without docs")
	}
	if !builtinIsFailed("nosuchbuiltin") {
		t.Error("failed builtin should be recorded in the failure cache")
	}
}

// resetBuiltinFailuresForTest clears the failure cache so tests don't
// leak marks into each other.
func resetBuiltinFailuresForTest() {
	builtinCrawlFailedMu.Lock()
	builtinCrawlFailed = make(map[string]bool)
	builtinCrawlFailedMu.Unlock()
}
