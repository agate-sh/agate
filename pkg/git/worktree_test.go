package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testRepo represents a temporary Git repository for testing
type testRepo struct {
	path string
	t    *testing.T
}

// createTestRepo creates a temporary Git repository for testing
func createTestRepo(t *testing.T) *testRepo {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "agate-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize Git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()

	return &testRepo{
		path: tmpDir,
		t:    t,
	}
}

// cleanup removes the test repository
func (r *testRepo) cleanup() {
	if err := os.RemoveAll(r.path); err != nil {
		r.t.Logf("Warning: failed to cleanup test repo: %v", err)
	}
}

// writeFile writes a file to the test repository
func (r *testRepo) writeFile(relativePath, content string) {
	r.t.Helper()

	fullPath := filepath.Join(r.path, relativePath)

	// Create parent directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		r.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		r.t.Fatalf("Failed to write file %s: %v", relativePath, err)
	}
}

// commit creates a commit with the given message
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

// createBranch creates and checks out a new branch
func (r *testRepo) createBranch(branchName string) {
	r.t.Helper()

	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to create branch %s: %v", branchName, err)
	}
}

// checkout switches to a branch
func (r *testRepo) checkout(branchName string) {
	r.t.Helper()

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to checkout %s: %v", branchName, err)
	}
}

// getCurrentBranch returns the current branch name
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

// hasChanges returns true if there are uncommitted changes
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

// getCommitHash returns the hash of the latest commit
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

// getCommitCount returns the number of commits in current branch
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

// Example test to verify test infrastructure works
func TestTestInfrastructure(t *testing.T) {
	repo := createTestRepo(t)
	defer repo.cleanup()

	// Write initial file
	repo.writeFile("test.txt", "hello world")

	// Should have changes
	if !repo.hasChanges() {
		t.Error("Expected to have changes")
	}

	// Commit
	repo.commit("Initial commit")

	// Should not have changes
	if repo.hasChanges() {
		t.Error("Expected no changes after commit")
	}

	// Should be on main/master branch
	branch := repo.getCurrentBranch()
	if branch != "main" && branch != "master" {
		t.Errorf("Expected to be on main/master, got %s", branch)
	}
}

func TestCommitAll(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testRepo)
		message string
		wantErr bool
		errMsg  string
	}{
		{
			name: "commit tracked and untracked files",
			setup: func(repo *testRepo) {
				repo.writeFile("file1.txt", "initial content")
				repo.commit("Initial commit")
				repo.writeFile("file1.txt", "modified content")
				repo.writeFile("file2.txt", "new file")
			},
			message: "Commit all changes",
			wantErr: false,
		},
		{
			name: "commit with empty message",
			setup: func(repo *testRepo) {
				repo.writeFile("test.txt", "content")
			},
			message: "",
			wantErr: true,
			errMsg:  "commit message cannot be empty",
		},
		{
			name: "commit only tracked files",
			setup: func(repo *testRepo) {
				repo.writeFile("tracked.txt", "initial")
				repo.commit("Initial commit")
				repo.writeFile("tracked.txt", "modified")
			},
			message: "Update tracked file",
			wantErr: false,
		},
		{
			name: "commit only untracked files",
			setup: func(repo *testRepo) {
				repo.writeFile("existing.txt", "content")
				repo.commit("Initial commit")
				repo.writeFile("new.txt", "new content")
			},
			message: "Add new file",
			wantErr: false,
		},
		{
			name: "commit with no changes",
			setup: func(repo *testRepo) {
				repo.writeFile("test.txt", "content")
				repo.commit("Initial commit")
				// No new changes
			},
			message: "Empty commit",
			wantErr: true, // git commit will fail with no changes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := createTestRepo(t)
			defer repo.cleanup()

			// Setup test scenario
			tt.setup(repo)

			// Get commit count before
			countBefore := repo.getCommitCount()

			// Call CommitAll
			sha, err := CommitAll(repo.path, tt.message)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("CommitAll() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("CommitAll() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("CommitAll() unexpected error: %v", err)
				return
			}

			// Verify SHA was returned
			if sha == "" {
				t.Error("CommitAll() returned empty SHA")
			}
			if len(sha) != 40 { // Git SHA is 40 characters
				t.Errorf("CommitAll() returned invalid SHA length: %d, want 40", len(sha))
			}

			// Verify commit was created
			countAfter := repo.getCommitCount()
			if countAfter != countBefore+1 {
				t.Errorf("Expected %d commits, got %d", countBefore+1, countAfter)
			}

			// Verify no changes remain
			if repo.hasChanges() {
				t.Error("Expected no changes after CommitAll()")
			}
		})
	}
}

func TestGetCommitLog(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testRepo) (mainBranch, featureBranch string)
		wantCommit int
		wantErr    bool
	}{
		{
			name: "get commits between main and feature branch",
			setup: func(repo *testRepo) (string, string) {
				// Create initial commit on main
				repo.writeFile("main.txt", "main content")
				repo.commit("Initial commit on main")
				mainBranch := repo.getCurrentBranch()

				// Create feature branch with 2 commits
				repo.createBranch("feature")
				repo.writeFile("feature1.txt", "feature 1")
				repo.commit("Add feature 1")
				repo.writeFile("feature2.txt", "feature 2")
				repo.commit("Add feature 2")

				return mainBranch, "feature"
			},
			wantCommit: 2,
			wantErr:    false,
		},
		{
			name: "empty commit log when branches are equal",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("test.txt", "content")
				repo.commit("Initial commit")
				mainBranch := repo.getCurrentBranch()

				// Create branch but no new commits
				repo.createBranch("same")
				return mainBranch, "same"
			},
			wantCommit: 0,
			wantErr:    false,
		},
		{
			name: "get commits with various authors and dates",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("initial.txt", "content")
				repo.commit("Initial")
				mainBranch := repo.getCurrentBranch()

				repo.createBranch("multi")
				repo.writeFile("file1.txt", "1")
				repo.commit("First commit with detailed message")
				repo.writeFile("file2.txt", "2")
				repo.commit("Second commit")
				repo.writeFile("file3.txt", "3")
				repo.commit("Third commit")

				return mainBranch, "multi"
			},
			wantCommit: 3,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := createTestRepo(t)
			defer repo.cleanup()

			mainBranch, featureBranch := tt.setup(repo)

			// Call GetCommitLog
			commits, err := GetCommitLog(repo.path, mainBranch, featureBranch)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetCommitLog() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetCommitLog() unexpected error: %v", err)
				return
			}

			// Check number of commits
			if len(commits) != tt.wantCommit {
				t.Errorf("GetCommitLog() got %d commits, want %d", len(commits), tt.wantCommit)
			}

			// Verify commit info structure
			for i, commit := range commits {
				if commit.Hash == "" {
					t.Errorf("Commit %d has empty hash", i)
				}
				if commit.Message == "" {
					t.Errorf("Commit %d has empty message", i)
				}
				if commit.Author == "" {
					t.Errorf("Commit %d has empty author", i)
				}
				if commit.Date == "" {
					t.Errorf("Commit %d has empty date", i)
				}
			}
		})
	}
}

func TestGetDiffFiles(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testRepo) (targetBranch, sourceBranch string)
		wantFiles []string
		wantErr   bool
	}{
		{
			name: "get files changed between branches",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("base.txt", "base")
				repo.commit("Initial commit")
				targetBranch := repo.getCurrentBranch()

				repo.createBranch("feature")
				repo.writeFile("new1.txt", "new file 1")
				repo.writeFile("new2.txt", "new file 2")
				repo.commit("Add new files")

				return targetBranch, "feature"
			},
			wantFiles: []string{"new1.txt", "new2.txt"},
			wantErr:   false,
		},
		{
			name: "empty list when no differences",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("same.txt", "content")
				repo.commit("Initial")
				targetBranch := repo.getCurrentBranch()

				repo.createBranch("identical")
				return targetBranch, "identical"
			},
			wantFiles: []string{},
			wantErr:   false,
		},
		{
			name: "modified and new files",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("existing.txt", "old content")
				repo.commit("Initial")
				targetBranch := repo.getCurrentBranch()

				repo.createBranch("changes")
				repo.writeFile("existing.txt", "new content")
				repo.writeFile("added.txt", "added")
				repo.commit("Modify and add")

				return targetBranch, "changes"
			},
			wantFiles: []string{"added.txt", "existing.txt"},
			wantErr:   false,
		},
		{
			name: "files in subdirectories",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("root.txt", "root")
				repo.commit("Initial")
				targetBranch := repo.getCurrentBranch()

				repo.createBranch("nested")
				repo.writeFile("dir/subdir/file.txt", "nested file")
				repo.writeFile("dir/another.txt", "another")
				repo.commit("Add nested files")

				return targetBranch, "nested"
			},
			wantFiles: []string{"dir/another.txt", "dir/subdir/file.txt"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := createTestRepo(t)
			defer repo.cleanup()

			targetBranch, sourceBranch := tt.setup(repo)

			// Call GetDiffFiles
			files, err := GetDiffFiles(repo.path, sourceBranch, targetBranch)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetDiffFiles() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetDiffFiles() unexpected error: %v", err)
				return
			}

			// Check files (order may vary, so check length and membership)
			if len(files) != len(tt.wantFiles) {
				t.Errorf("GetDiffFiles() got %d files, want %d\nGot: %v\nWant: %v",
					len(files), len(tt.wantFiles), files, tt.wantFiles)
				return
			}

			// Convert to map for easier checking
			fileMap := make(map[string]bool)
			for _, f := range files {
				fileMap[f] = true
			}

			for _, wantFile := range tt.wantFiles {
				if !fileMap[wantFile] {
					t.Errorf("GetDiffFiles() missing file %q\nGot: %v\nWant: %v",
						wantFile, files, tt.wantFiles)
				}
			}
		})
	}
}

func TestMergeBranch(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*testRepo) (targetBranch, sourceBranch string)
		commitMessage string
		wantErr       bool
		errContains   string
	}{
		{
			name: "successful merge",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("base.txt", "base")
				repo.commit("Initial commit")
				targetBranch := repo.getCurrentBranch()

				repo.createBranch("feature")
				repo.writeFile("feature.txt", "feature content")
				repo.commit("Add feature")

				// Go back to target branch for merge
				repo.checkout(targetBranch)
				return targetBranch, "feature"
			},
			commitMessage: "Merge feature branch",
			wantErr:       false,
		},
		{
			name: "merge with conflict",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("conflict.txt", "original")
				repo.commit("Initial")
				targetBranch := repo.getCurrentBranch()

				// Create feature branch with different content
				repo.createBranch("feature")
				repo.writeFile("conflict.txt", "feature version")
				repo.commit("Feature change")

				// Go back and make conflicting change
				repo.checkout(targetBranch)
				repo.writeFile("conflict.txt", "target version")
				repo.commit("Target change")

				return targetBranch, "feature"
			},
			commitMessage: "Merge with conflict",
			wantErr:       true,
			errContains:   "conflict",
		},
		{
			name: "merge non-existent branch",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("test.txt", "content")
				repo.commit("Initial")
				targetBranch := repo.getCurrentBranch()
				return targetBranch, "nonexistent"
			},
			commitMessage: "Merge nonexistent",
			wantErr:       true,
		},
		{
			name: "fast-forward merge",
			setup: func(repo *testRepo) (string, string) {
				repo.writeFile("base.txt", "base")
				repo.commit("Initial")
				targetBranch := repo.getCurrentBranch()

				repo.createBranch("ahead")
				repo.writeFile("new.txt", "new")
				repo.commit("Add new file")

				repo.checkout(targetBranch)
				return targetBranch, "ahead"
			},
			commitMessage: "Fast forward merge",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := createTestRepo(t)
			defer repo.cleanup()

			targetBranch, sourceBranch := tt.setup(repo)

			// Verify we're on target branch
			if repo.getCurrentBranch() != targetBranch {
				t.Fatalf("Not on target branch. Expected %s, got %s",
					targetBranch, repo.getCurrentBranch())
			}

			// Get commit count before merge
			commitsBefore := repo.getCommitCount()

			// Call MergeBranch
			err := MergeBranch(repo.path, sourceBranch, tt.commitMessage)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("MergeBranch() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errContains) {
					t.Errorf("MergeBranch() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("MergeBranch() unexpected error: %v", err)
				return
			}

			// Verify merge happened (commit count should increase or stay same for fast-forward)
			commitsAfter := repo.getCommitCount()
			if commitsAfter < commitsBefore {
				t.Errorf("Commit count decreased after merge: before=%d, after=%d",
					commitsBefore, commitsAfter)
			}

			// Verify we're still on target branch
			if repo.getCurrentBranch() != targetBranch {
				t.Errorf("Branch changed after merge. Expected %s, got %s",
					targetBranch, repo.getCurrentBranch())
			}
		})
	}
}

func TestStageFile(t *testing.T) {
	repo := createTestRepo(t)
	defer repo.cleanup()

	// Create initial commit
	repo.writeFile("file1.txt", "initial")
	repo.commit("Initial commit")

	// Modify file and create new file
	repo.writeFile("file1.txt", "modified")
	repo.writeFile("file2.txt", "new content")

	// Stage only file1.txt
	err := StageFile(repo.path, "file1.txt")
	if err != nil {
		t.Fatalf("StageFile() failed: %v", err)
	}

	// Check git status
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repo.path
	output, _ := cmd.Output()
	status := string(output)

	// file1.txt should be staged (M  or A )
	// file2.txt should be unstaged (??)
	if !strings.Contains(status, "M  file1.txt") {
		t.Errorf("Expected file1.txt to be staged, got status:\n%s", status)
	}
	if !strings.Contains(status, "?? file2.txt") {
		t.Errorf("Expected file2.txt to be untracked, got status:\n%s", status)
	}
}

func TestUnstageFile(t *testing.T) {
	repo := createTestRepo(t)
	defer repo.cleanup()

	// Create initial commit
	repo.writeFile("file1.txt", "initial")
	repo.commit("Initial commit")

	// Modify and stage file
	repo.writeFile("file1.txt", "modified")
	cmd := exec.Command("git", "add", "file1.txt")
	cmd.Dir = repo.path
	cmd.Run()

	// Verify it's staged
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = repo.path
	output, _ := cmd.Output()
	if !strings.Contains(string(output), "M  file1.txt") {
		t.Fatal("Setup failed: file should be staged")
	}

	// Unstage the file
	err := UnstageFile(repo.path, "file1.txt")
	if err != nil {
		t.Fatalf("UnstageFile() failed: %v", err)
	}

	// Verify it's unstaged
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = repo.path
	output, _ = cmd.Output()
	if !strings.Contains(string(output), " M file1.txt") {
		t.Errorf("Expected file1.txt to be unstaged, got status:\n%s", string(output))
	}
}

func TestDiscardFile(t *testing.T) {
	repo := createTestRepo(t)
	defer repo.cleanup()

	// Create initial commit
	repo.writeFile("file1.txt", "initial content")
	repo.commit("Initial commit")

	// Modify file
	repo.writeFile("file1.txt", "modified content")

	// Verify file is modified
	content, _ := os.ReadFile(filepath.Join(repo.path, "file1.txt"))
	if string(content) != "modified content" {
		t.Fatal("Setup failed: file should be modified")
	}

	// Discard changes
	err := DiscardFile(repo.path, "file1.txt")
	if err != nil {
		t.Fatalf("DiscardFile() failed: %v", err)
	}

	// Verify file is restored to original content
	content, _ = os.ReadFile(filepath.Join(repo.path, "file1.txt"))
	if string(content) != "initial content" {
		t.Errorf("Expected file to be restored to 'initial content', got: %s", string(content))
	}

	// Verify no changes in status
	if repo.hasChanges() {
		t.Error("Expected no changes after DiscardFile()")
	}
}

func TestStageAll(t *testing.T) {
	repo := createTestRepo(t)
	defer repo.cleanup()

	// Create initial commit
	repo.writeFile("file1.txt", "initial")
	repo.commit("Initial commit")

	// Modify existing file and create new files
	repo.writeFile("file1.txt", "modified")
	repo.writeFile("file2.txt", "new file 2")
	repo.writeFile("dir/file3.txt", "new file 3")

	// Verify we have changes
	if !repo.hasChanges() {
		t.Fatal("Setup failed: should have changes")
	}

	// Stage all changes
	err := StageAll(repo.path)
	if err != nil {
		t.Fatalf("StageAll() failed: %v", err)
	}

	// Check git status - all files should be staged
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repo.path
	output, _ := cmd.Output()
	status := string(output)

	// All files should have staged status (first character not space or ?)
	lines := strings.Split(strings.TrimSpace(status), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		firstChar := line[0]
		if firstChar == ' ' || firstChar == '?' {
			t.Errorf("Expected all files to be staged, but found unstaged file in: %s", line)
		}
	}

	// Verify we can commit (all changes are staged)
	sha, err := CommitAll(repo.path, "Commit all staged files")
	if err != nil {
		t.Errorf("Failed to commit after StageAll(): %v", err)
	}
	if sha == "" {
		t.Error("Expected commit SHA after staging all files")
	}
}

func TestNewWorktreeManagerForPath(t *testing.T) {
	repo := createTestRepo(t)
	defer repo.cleanup()

	manager, err := NewWorktreeManagerForPath(repo.path)
	if err != nil {
		t.Fatalf("NewWorktreeManagerForPath returned error: %v", err)
	}

	if manager == nil {
		t.Fatalf("NewWorktreeManagerForPath returned nil manager")
	}

	gotPath := filepath.Clean(manager.GetRepositoryPath())
	gotEval, err := filepath.EvalSymlinks(gotPath)
	if err == nil {
		gotPath = gotEval
	}

	wantedPath := filepath.Clean(repo.path)
	wantEval, err := filepath.EvalSymlinks(wantedPath)
	if err == nil {
		wantedPath = wantEval
	}

	if gotPath != wantedPath {
		t.Fatalf("GetRepositoryPath = %s, want %s", gotPath, wantedPath)
	}

	expectedName := sanitizeRepoName(filepath.Base(repo.path))
	if got := manager.GetRepositoryName(); got != expectedName {
		t.Fatalf("GetRepositoryName = %s, want %s", got, expectedName)
	}
}
