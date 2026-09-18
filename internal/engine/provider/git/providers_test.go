package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"git-hint/internal/engine/provider/git"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)

	runCmd := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null"}, args...)...)
		cmd.Dir = dir
		cmd.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + dir,
			"GIT_AUTHOR_NAME=Test User",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test User",
			"GIT_COMMITTER_EMAIL=test@example.com",
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(out))
		}
	}

	runCmd("init", "-b", "main")
	runCmd("config", "user.name", "Test User")
	runCmd("config", "user.email", "test@example.com")

	// Create tracked files
	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}
	runCmd("add", "file1.txt", "file2.txt")
	runCmd("commit", "--no-verify", "-m", "feat: initial commit")

	// Create branches
	runCmd("branch", "feature/awesome")

	// Create tags
	runCmd("tag", "v1.0.0")

	// Create remotes
	runCmd("remote", "add", "origin", "https://github.com/example/repo.git")
	runCmd("remote", "add", "upstream", "https://github.com/upstream/repo.git")

	// Create stash
	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("stash content"), 0644); err != nil {
		t.Fatal(err)
	}
	runCmd("stash", "push", "-m", "work in progress")

	return dir
}

func TestBranchProvider(t *testing.T) {
	setupGitRepo(t)

	branches := git.BranchProvider()
	if len(branches) == 0 {
		t.Fatal("BranchProvider() returned empty slice")
	}

	names := make(map[string]bool)
	for _, b := range branches {
		names[b.Name] = true
	}

	if !names["main"] {
		t.Errorf("BranchProvider() missing 'main', got %v", branches)
	}
	if !names["feature/awesome"] {
		t.Errorf("BranchProvider() missing 'feature/awesome', got %v", branches)
	}
}

func TestRemoteProvider(t *testing.T) {
	setupGitRepo(t)

	remotes := git.RemoteProvider()
	if len(remotes) != 2 {
		t.Fatalf("RemoteProvider() len = %d, want 2", len(remotes))
	}

	names := make(map[string]bool)
	for _, r := range remotes {
		names[r.Name] = true
	}

	if !names["origin"] || !names["upstream"] {
		t.Errorf("RemoteProvider() missing remotes, got %v", remotes)
	}
}

func TestTagProvider(t *testing.T) {
	setupGitRepo(t)

	tags := git.TagProvider()
	if len(tags) != 1 {
		t.Fatalf("TagProvider() len = %d, want 1", len(tags))
	}

	if tags[0].Name != "v1.0.0" {
		t.Errorf("TagProvider()[0].Name = %q, want %q", tags[0].Name, "v1.0.0")
	}
}

func TestTrackedFileProvider(t *testing.T) {
	setupGitRepo(t)

	files := git.TrackedFileProvider()
	if len(files) != 2 {
		t.Fatalf("TrackedFileProvider() len = %d, want 2", len(files))
	}

	names := make(map[string]bool)
	for _, f := range files {
		names[f.Name] = true
	}

	if !names["file1.txt"] || !names["file2.txt"] {
		t.Errorf("TrackedFileProvider() missing files, got %v", files)
	}
}

func TestStashProvider(t *testing.T) {
	setupGitRepo(t)

	stashes := git.StashProvider()
	if len(stashes) == 0 {
		t.Fatal("StashProvider() returned empty slice")
	}

	if !strings.Contains(stashes[0].Name, "stash@{0}") {
		t.Errorf("StashProvider() expected stash entry, got %q", stashes[0].Name)
	}
}

func TestRefProvider(t *testing.T) {
	setupGitRepo(t)

	refs := git.RefProvider()
	if len(refs) == 0 {
		t.Fatal("RefProvider() returned empty slice")
	}

	names := make(map[string]bool)
	for _, r := range refs {
		names[r.Name] = true
	}

	if !names["main"] || !names["v1.0.0"] {
		t.Errorf("RefProvider() missing expected refs, got %v", refs)
	}
}

func TestUpstreamProvider(t *testing.T) {
	setupGitRepo(t)

	upstreams := git.UpstreamProvider()
	t.Logf("UpstreamProvider() returned %d items", len(upstreams))
}

func TestCommitProvider(t *testing.T) {
	setupGitRepo(t)

	commits := git.CommitProvider()
	if len(commits) == 0 {
		t.Fatal("CommitProvider() returned empty slice")
	}

	if commits[0].Name != "HEAD" {
		t.Errorf("CommitProvider()[0].Name = %q, want 'HEAD'", commits[0].Name)
	}
	if !strings.Contains(commits[0].Description, "feat: initial commit") {
		t.Errorf("CommitProvider()[0].Description missing commit message: %q", commits[0].Description)
	}
	if commits[0].MatchKey == "" {
		t.Error("CommitProvider()[0].MatchKey should contain commit hash")
	}
}

func TestGitProviders_OutsideRepo(t *testing.T) {
	nonRepoDir := t.TempDir()
	t.Chdir(nonRepoDir)

	if res := git.BranchProvider(); res != nil {
		t.Errorf("BranchProvider() outside repo should return nil, got %v", res)
	}
	if res := git.RemoteProvider(); res != nil {
		t.Errorf("RemoteProvider() outside repo should return nil, got %v", res)
	}
	if res := git.StashProvider(); res != nil {
		t.Errorf("StashProvider() outside repo should return nil, got %v", res)
	}
	if res := git.TagProvider(); res != nil {
		t.Errorf("TagProvider() outside repo should return nil, got %v", res)
	}
	if res := git.TrackedFileProvider(); res != nil {
		t.Errorf("TrackedFileProvider() outside repo should return nil, got %v", res)
	}
	if res := git.RefProvider(); res != nil {
		t.Errorf("RefProvider() outside repo should return nil, got %v", res)
	}
	if res := git.UpstreamProvider(); res != nil {
		t.Errorf("UpstreamProvider() outside repo should return nil, got %v", res)
	}
	if res := git.CommitProvider(); res != nil {
		t.Errorf("CommitProvider() outside repo should return nil, got %v", res)
	}
}
