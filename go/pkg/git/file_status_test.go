package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFileStatusUnstagedModification(t *testing.T) {
	// Create test repo
	repo := createSimpleTestRepo(t)
	defer os.RemoveAll(repo)

	// Create and commit a file
	testFile := filepath.Join(repo, "test.txt")
	if err := os.WriteFile(testFile, []byte("original content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "test.txt")
	runGit(t, repo, "commit", "-m", "initial commit")

	// Modify the file but don't stage it
	if err := os.WriteFile(testFile, []byte("modified content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Verify git sees it as unstaged
	cmd := exec.Command("git", "status", "--porcelain", "test.txt")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	gitStatus := string(out)
	t.Logf("Git status output: %q", gitStatus)

	// Should be " M" (space M) for unstaged modification
	if len(gitStatus) < 2 || gitStatus[0] != ' ' || gitStatus[1] != 'M' {
		t.Fatalf("Expected git status to be ' M', got %q", gitStatus[:2])
	}

	// Get our parsed status
	status := GetFileStatuses(repo)

	// Should have exactly 1 file
	if len(status.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(status.Files))
	}

	file := status.Files[0]
	t.Logf("Parsed file: IsStaged=%v, Status=%q, FilePath=%s", file.IsStaged, file.Status, file.FilePath)

	// File should NOT be staged (this is the key assertion)
	if file.IsStaged {
		t.Errorf("expected file to be unstaged (IsStaged=false), but got IsStaged=true. Status=%q", file.Status)
	}

	// Status should be " M" (unstaged modification)
	if file.Status != " M" {
		t.Errorf("expected status ' M' (unstaged modification), got %q", file.Status)
	}
}

func TestFileStatusStagedModification(t *testing.T) {
	// Create test repo
	repo := createSimpleTestRepo(t)
	defer os.RemoveAll(repo)

	// Create and commit a file
	testFile := filepath.Join(repo, "test.txt")
	if err := os.WriteFile(testFile, []byte("original content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "test.txt")
	runGit(t, repo, "commit", "-m", "initial commit")

	// Modify and stage the file
	if err := os.WriteFile(testFile, []byte("modified content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "test.txt")

	// Verify git sees it as staged
	cmd := exec.Command("git", "status", "--porcelain", "test.txt")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	gitStatus := string(out)
	t.Logf("Git status output: %q", gitStatus)

	// Should be "M " (M space) for staged modification
	if len(gitStatus) < 2 || gitStatus[0] != 'M' || gitStatus[1] != ' ' {
		t.Fatalf("Expected git status to be 'M ', got %q", gitStatus[:2])
	}

	// Get our parsed status
	status := GetFileStatuses(repo)

	// Should have exactly 1 file
	if len(status.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(status.Files))
	}

	file := status.Files[0]
	t.Logf("Parsed file: IsStaged=%v, Status=%q, FilePath=%s", file.IsStaged, file.Status, file.FilePath)

	// File should be staged
	if !file.IsStaged {
		t.Errorf("expected file to be staged (IsStaged=true), but got IsStaged=false. Status=%q", file.Status)
	}

	// Status should be "M " (staged modification)
	if file.Status != "M " {
		t.Errorf("expected status 'M ' (staged modification), got %q", file.Status)
	}
}

// Helper functions

func createSimpleTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.name", "Test User")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")

	return tmpDir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, output)
	}
}
