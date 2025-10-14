package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testRepo represents a temporary Git repository for testing.
type testRepo struct {
	path string
	t    *testing.T
}

// createTestRepo creates a temporary Git repository for testing scenarios that need a real Git environment.
func createTestRepo(t *testing.T) *testRepo {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "agate-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Ensure Git user configuration exists for commits in this repo
	configureGitUser(t, tmpDir)

	return &testRepo{path: tmpDir, t: t}
}

// configureGitUser configures a repo-local Git user to allow commits.
func configureGitUser(t *testing.T, repoPath string) {
	t.Helper()

	cmds := [][]string{
		{"config", "user.name", "Test User"},
		{"config", "user.email", "test@example.com"},
	}

	for _, args := range cmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to configure git %s: %v", strings.Join(args, " "), err)
		}
	}
}

// cleanup removes the test repository.
func (r *testRepo) cleanup() {
	if err := os.RemoveAll(r.path); err != nil {
		r.t.Logf("Warning: failed to cleanup test repo: %v", err)
	}
}

// writeFile writes a file to the test repository.
func (r *testRepo) writeFile(relativePath, content string) {
	r.t.Helper()

	fullPath := filepath.Join(r.path, relativePath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		r.t.Fatalf("Failed to write file %s: %v", relativePath, err)
	}
}

// commit creates a commit with the given message.
func (r *testRepo) commit(message string) {
	r.t.Helper()

	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to stage files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to commit: %v", err)
	}
}

// createBranch creates and checks out a new branch.
func (r *testRepo) createBranch(branchName string) {
	r.t.Helper()

	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to create branch %s: %v", branchName, err)
	}
}

// checkout switches to a branch.
func (r *testRepo) checkout(branchName string) {
	r.t.Helper()

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to checkout %s: %v", branchName, err)
	}
}

// getCurrentBranch returns the current branch name.
func (r *testRepo) getCurrentBranch() string {
	r.t.Helper()

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		r.t.Fatalf("Failed to get current branch: %v", err)
	}
	return strings.TrimSpace(string(output))
}

// hasChanges returns true if there are uncommitted changes.
func (r *testRepo) hasChanges() bool {
	r.t.Helper()

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		r.t.Fatalf("Failed to get status: %v", err)
	}
	return strings.TrimSpace(string(output)) != ""
}

// getCommitHash returns the hash of the latest commit.
func (r *testRepo) getCommitHash() string {
	r.t.Helper()

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		r.t.Fatalf("Failed to get commit hash: %v", err)
	}
	return strings.TrimSpace(string(output))
}

// getCommitCount returns the number of commits in the current branch.
func (r *testRepo) getCommitCount() int {
	r.t.Helper()

	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	count := 0
	_, _ = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count
}
