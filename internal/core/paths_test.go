package core

import (
	"os"
	"path/filepath"
	"testing"
)

// withFakeBinary points executablePath at a fake binary inside a fresh
// temp dir for the duration of the test, restoring the original after.
func withFakeBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	fakeBinary := filepath.Join(tmpDir, "git-hint")
	if err := os.WriteFile(fakeBinary, []byte{}, 0755); err != nil {
		t.Fatal(err)
	}

	original := executablePath
	executablePath = func() (string, error) { return fakeBinary, nil }
	t.Cleanup(func() { executablePath = original })

	return tmpDir
}

func TestDataDir_CreatesDirNextToBinary(t *testing.T) {
	tmpDir := withFakeBinary(t)

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() error = %v", err)
	}

	want := filepath.Join(tmpDir, "data")
	if got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}

	info, err := os.Stat(got)
	if err != nil || !info.IsDir() {
		t.Errorf("DataDir() did not create the directory at %q", got)
	}
}

func TestDataDir_ResolvesSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	realDir := filepath.Join(tmpDir, "real")
	if err := os.Mkdir(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	realBinary := filepath.Join(realDir, "git-hint")
	if err := os.WriteFile(realBinary, []byte{}, 0755); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(tmpDir, "git-hint-link")
	if err := os.Symlink(realBinary, symlinkPath); err != nil {
		t.Skipf("symlinks not supported on this system: %v", err)
	}

	original := executablePath
	executablePath = func() (string, error) { return symlinkPath, nil }
	t.Cleanup(func() { executablePath = original })

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() error = %v", err)
	}

	want := filepath.Join(realDir, "data")
	if got != want {
		t.Errorf("DataDir() = %q, want %q (should follow symlink to real binary location, not link location)", got, want)
	}
}

func TestCommandDataPath(t *testing.T) {
	tmpDir := withFakeBinary(t)

	got, err := CommandDataPath("git")
	if err != nil {
		t.Fatalf("CommandDataPath() error = %v", err)
	}

	want := filepath.Join(tmpDir, "data", "local", "git.json")
	if got != want {
		t.Errorf("CommandDataPath() = %q, want %q", got, want)
	}
}

func TestIndexPath(t *testing.T) {
	tmpDir := withFakeBinary(t)

	got, err := IndexPath()
	if err != nil {
		t.Fatalf("IndexPath() error = %v", err)
	}

	want := filepath.Join(tmpDir, "data", "index.json")
	if got != want {
		t.Errorf("IndexPath() = %q, want %q", got, want)
	}
}
