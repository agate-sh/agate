// Package git provides Git worktree management functionality including
// creation, deletion, and status tracking of Git worktrees.
package git

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DebugLog is a placeholder function - will be implemented based on build tags
var DebugLog = func(_ string, _ ...interface{}) {
	// No-op by default, overridden by main package when debug is enabled
}

// SystemCapabilities represents the copy-on-write capabilities of the system
type SystemCapabilities struct {
	SupportsCOW bool
	COWMethod   string
}

// GitStatus represents the Git status of a worktree
type GitStatus struct {
	Branch     string
	Ahead      int
	Behind     int
	Staged     int
	Modified   int
	Untracked  int
	Stashed    int
	IsClean    bool
	HasRemote  bool
	RemoteName string
}

// CommitInfo represents information about a Git commit
type CommitInfo struct {
	Hash    string
	Message string
	Author  string
	Date    string
}

// WorktreeInfo represents information about a Git worktree
type WorktreeInfo struct {
	Name      string
	Path      string
	RepoName  string
	Branch    string
	GitStatus *GitStatus
	CreatedAt time.Time
}

// WorktreeManager manages Git worktree operations
type WorktreeManager struct {
	repoPath     string
	worktreeBase string
	systemCaps   SystemCapabilities
	isGitRepo    bool
}

// NewWorktreeManager creates a new agentManager instance
func NewWorktreeManager() (*WorktreeManager, error) {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Get repository root
	repoPath, err := getRepositoryRoot(workDir)
	isGitRepo := err == nil
	if err != nil {
		// Allow non-Git directories but warn user
		repoPath = workDir
	}

	// Get worktree base directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	worktreeBase := filepath.Join(homeDir, ".agate", "worktrees")

	// Detect system capabilities
	systemCaps := detectCOWSupport()

	return &WorktreeManager{
		repoPath:     repoPath,
		worktreeBase: worktreeBase,
		systemCaps:   systemCaps,
		isGitRepo:    isGitRepo,
	}, nil
}

// IsGitRepo indicates whether the manager was initialized inside a Git repository.
func (wm *WorktreeManager) IsGitRepo() bool {
	return wm.isGitRepo
}

// GetSystemCapabilities returns the system's COW capabilities
func (wm *WorktreeManager) GetSystemCapabilities() SystemCapabilities {
	return wm.systemCaps
}

// detectCOWSupport detects if the system supports copy-on-write
func detectCOWSupport() SystemCapabilities {
	caps := SystemCapabilities{}

	switch runtime.GOOS {
	case "darwin":
		// Check if current directory is on APFS
		if isAPFS() {
			caps.SupportsCOW = true
			caps.COWMethod = "apfs"
		}
	case "linux":
		// Future: Check for Btrfs, ZFS, etc.
		// For now, assume no COW support on Linux
	}

	return caps
}

// isAPFS checks if the current filesystem supports APFS copy-on-write
func isAPFS() bool {
	// Method 1: Check current working directory filesystem type (most accurate)
	// This directly checks the filesystem where we'll be creating worktrees
	workDir, err := os.Getwd()
	if err == nil {
		cmd := exec.Command("diskutil", "info", workDir)
		output, err := cmd.Output()
		if err == nil {
			outputStr := strings.ToLower(string(output))
			// Look for APFS in the file system type field
			if strings.Contains(outputStr, "apfs") {
				return true
			}
		}
	}

	// Method 2: Check the root filesystem as fallback
	cmd := exec.Command("diskutil", "info", "/")
	output, err := cmd.Output()
	if err == nil {
		outputStr := strings.ToLower(string(output))
		// More specific check for APFS file system personality
		if strings.Contains(outputStr, "apfs") {
			return true
		}
	}

	// Method 3: Use stat with macOS-specific format (better than -c flag)
	cmd = exec.Command("stat", "-f", "%Sf", ".")
	output, err = cmd.Output()
	if err == nil {
		fsType := strings.TrimSpace(strings.ToLower(string(output)))
		if strings.Contains(fsType, "apfs") {
			return true
		}
	}

	// Method 4: Check mount table for current directory
	cmd = exec.Command("mount")
	output, err = cmd.Output()
	if err == nil {
		outputStr := strings.ToLower(string(output))
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			// Look for lines containing the current directory or root and APFS
			if strings.Contains(line, "apfs") && (strings.Contains(line, "/") || strings.Contains(line, workDir)) {
				return true
			}
		}
	}

	return false
}

// getRepositoryRoot finds the Git repository root
func getRepositoryRoot(workDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRepositoryName extracts the repository name from the current directory
func (wm *WorktreeManager) GetRepositoryName() string {
	repoName := filepath.Base(wm.repoPath)
	if repoName == "." || repoName == "/" {
		return "unknown"
	}
	return sanitizeRepoName(repoName)
}

// GetRepositoryPath returns the full path to the main repository
func (wm *WorktreeManager) GetRepositoryPath() string {
	return wm.repoPath
}

// GetMainWorktreeInfo returns metadata about the primary worktree (the main repository checkout)
func (wm *WorktreeManager) GetMainWorktreeInfo() (*WorktreeInfo, error) {
	if wm.repoPath == "" {
		return nil, fmt.Errorf("repository path is not set")
	}

	info, err := os.Stat(wm.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat repository path: %w", err)
	}

	gitStatus, err := wm.getWorktreeGitStatus(wm.repoPath)
	if err != nil {
		gitStatus = &GitStatus{Branch: "", IsClean: true}
	}

	name := gitStatus.Branch
	if name == "" {
		name = "main"
	}

	return &WorktreeInfo{
		Name:      name,
		Path:      wm.repoPath,
		RepoName:  wm.GetRepositoryName(),
		Branch:    gitStatus.Branch,
		GitStatus: gitStatus,
		CreatedAt: info.ModTime(),
	}, nil
}

// sanitizeRepoName cleans a repository name for filesystem use
func sanitizeRepoName(name string) string {
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ToLower(name)
	return name
}

// GenerateRandomBranchName creates a random branch name
func GenerateRandomBranchName() string {
	adjectives := []string{"quick", "bright", "swift", "clever", "bold", "neat", "clean", "smooth", "sharp", "cool"}
	nouns := []string{"fix", "update", "patch", "change", "work", "task", "feature", "test", "demo", "trial"}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	adj := adjectives[rng.Intn(len(adjectives))]
	noun := nouns[rng.Intn(len(nouns))]

	return fmt.Sprintf("%s-%s", adj, noun)
}

// ValidateBranchName validates a Git branch name
func ValidateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("invalid branch name format")
	}
	if strings.Contains(name, "..") || strings.Contains(name, " ") {
		return fmt.Errorf("branch name contains invalid characters")
	}
	if strings.Contains(name, "~") || strings.Contains(name, "^") || strings.Contains(name, ":") {
		return fmt.Errorf("branch name contains invalid characters")
	}
	return nil
}

// CreateWorktree creates a new Git worktree
func (wm *WorktreeManager) CreateWorktree(branchName string) (*WorktreeInfo, error) {
	// Validate branch name
	if err := ValidateBranchName(branchName); err != nil {
		return nil, err
	}

	// Check if branch already exists
	if err := wm.checkBranchExists(branchName); err != nil {
		return nil, err
	}

	// Get repository name
	repoName := wm.GetRepositoryName()

	// Create worktree path
	worktreePath := filepath.Join(wm.worktreeBase, repoName, branchName)

	// Ensure base directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Create Git worktree
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	cmd.Dir = wm.repoPath
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create Git worktree: %w", err)
	}

	// Copy files if COW is supported
	if wm.systemCaps.SupportsCOW {
		DebugLog("Starting COW copy to %s", worktreePath)
		if err := wm.copyFilesWithCOW(worktreePath); err != nil {
			// Log error but don't fail - Git worktree is still valid
			DebugLog("Warning: failed to copy files with COW: %v", err)
		} else {
			DebugLog("COW copy completed successfully")
		}
	}

	// Get Git status for the new agent
	gitStatus, err := wm.getWorktreeGitStatus(worktreePath)
	if err != nil {
		// Don't fail on status error, just use empty status
		gitStatus = &GitStatus{Branch: branchName, IsClean: true}
	}

	return &WorktreeInfo{
		Name:      branchName,
		Path:      worktreePath,
		RepoName:  repoName,
		Branch:    branchName,
		GitStatus: gitStatus,
		CreatedAt: time.Now(),
	}, nil
}

// checkBranchExists checks if a branch already exists
func (wm *WorktreeManager) checkBranchExists(branchName string) error {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	cmd.Dir = wm.repoPath
	if cmd.Run() == nil {
		return fmt.Errorf("branch '%s' already exists", branchName)
	}
	return nil
}

// getUntrackedPaths returns a list of untracked files and directories in the repository.
// This includes files that are ignored by .gitignore (like node_modules, .env, dist/, etc.)
// but excludes files that are tracked by git.
func (wm *WorktreeManager) getUntrackedPaths() ([]string, error) {
	// Use git ls-files to get untracked paths
	// --others: show untracked files
	// --directory: show directory names (not contents) for untracked directories
	cmd := exec.Command("git", "ls-files", "--others", "--directory")
	cmd.Dir = wm.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list untracked files: %w", err)
	}

	// Parse output - each line is a path
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var paths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}

	return paths, nil
}

// copyUntrackedFilesWithCOW copies untracked files from source to destination using copy-on-write.
// This is designed to copy files like node_modules, .env, dist/, etc. that are not tracked by git.
// The function creates parent directories as needed and uses APFS COW on macOS.
func (wm *WorktreeManager) copyUntrackedFilesWithCOW(srcRepoPath, dstWorktreePath string, untrackedPaths []string) error {
	for _, relPath := range untrackedPaths {
		// Skip if it's the .git directory (shouldn't happen with --others flag, but be safe)
		if strings.HasPrefix(relPath, ".git/") || relPath == ".git" {
			continue
		}

		srcPath := filepath.Join(srcRepoPath, relPath)
		dstPath := filepath.Join(dstWorktreePath, relPath)

		// Check if source exists
		srcInfo, err := os.Stat(srcPath)
		if err != nil {
			// Source doesn't exist or can't be accessed - skip
			DebugLog("Skipping %s: %v", relPath, err)
			continue
		}

		// Ensure parent directory exists
		parentDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
		}

		// Copy based on COW method
		switch wm.systemCaps.COWMethod {
		case "apfs":
			var cmd *exec.Cmd
			if srcInfo.IsDir() {
				// For directories, use -Rc to recursively copy with COW
				cmd = exec.Command("cp", "-Rc", srcPath, dstPath)
			} else {
				// For files, use -c for COW
				cmd = exec.Command("cp", "-c", srcPath, dstPath)
			}

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to copy %s: %w", relPath, err)
			}

			DebugLog("Copied untracked: %s", relPath)
		default:
			return fmt.Errorf("COW method '%s' not implemented", wm.systemCaps.COWMethod)
		}
	}

	return nil
}

// copyFilesWithCOW copies untracked files using copy-on-write.
// This function only copies files that are not tracked by git (e.g., node_modules, .env, dist/).
// Git worktree has already copied all tracked files, so we only need to handle untracked ones.
func (wm *WorktreeManager) copyFilesWithCOW(worktreePath string) error {
	// Get list of untracked files/directories
	untrackedPaths, err := wm.getUntrackedPaths()
	if err != nil {
		return fmt.Errorf("failed to get untracked paths: %w", err)
	}

	// If no untracked files, nothing to copy
	if len(untrackedPaths) == 0 {
		DebugLog("No untracked files to copy")
		return nil
	}

	DebugLog("Found %d untracked paths to copy", len(untrackedPaths))

	// Copy untracked files with COW
	if err := wm.copyUntrackedFilesWithCOW(wm.repoPath, worktreePath, untrackedPaths); err != nil {
		return err
	}

	return nil
}

// ListWorktrees returns all worktrees grouped by repository
func (wm *WorktreeManager) ListWorktrees() (map[string][]WorktreeInfo, error) {
	groups := make(map[string][]WorktreeInfo)

	// Ensure worktree base directory exists
	if err := os.MkdirAll(wm.worktreeBase, 0755); err != nil {
		return groups, fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	// Read repository directories
	repos, err := os.ReadDir(wm.worktreeBase)
	if err != nil {
		return groups, fmt.Errorf("failed to read worktree directory: %w", err)
	}

	for _, repo := range repos {
		if !repo.IsDir() {
			continue
		}

		repoName := repo.Name()
		repoPath := filepath.Join(wm.worktreeBase, repoName)

		// Read worktrees in this repository
		worktrees, err := os.ReadDir(repoPath)
		if err != nil {
			continue
		}

		var repoWorktrees []WorktreeInfo
		for _, worktree := range worktrees {
			if !worktree.IsDir() {
				continue
			}

			worktreeName := worktree.Name()
			worktreePath := filepath.Join(repoPath, worktreeName)

			// Get worktree info
			info, err := os.Stat(worktreePath)
			if err != nil {
				continue
			}

			// Get Git status
			gitStatus, err := wm.getWorktreeGitStatus(worktreePath)
			if err != nil {
				gitStatus = &GitStatus{Branch: worktreeName, IsClean: true}
			}

			repoWorktrees = append(repoWorktrees, WorktreeInfo{
				Name:      worktreeName,
				Path:      worktreePath,
				RepoName:  repoName,
				Branch:    gitStatus.Branch,
				GitStatus: gitStatus,
				CreatedAt: info.ModTime(),
			})
		}

		if len(repoWorktrees) > 0 {
			groups[repoName] = repoWorktrees
		}
	}

	return groups, nil
}

// getWorktreeGitStatus gets the Git status for a worktree
func (wm *WorktreeManager) getWorktreeGitStatus(worktreePath string) (*GitStatus, error) {
	status := &GitStatus{}

	// Get current branch
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err == nil {
		status.Branch = strings.TrimSpace(string(output))
	}

	// Get detailed status
	cmd = exec.Command("git", "status", "--porcelain=v1", "--branch")
	cmd.Dir = worktreePath
	output, err = cmd.Output()
	if err == nil {
		wm.parseGitStatusOutput(string(output), status)
	}

	// Get stash count
	cmd = exec.Command("git", "stash", "list")
	cmd.Dir = worktreePath
	output, err = cmd.Output()
	if err == nil {
		stashOutput := strings.TrimSpace(string(output))
		if stashOutput != "" {
			status.Stashed = len(strings.Split(stashOutput, "\n"))
		}
	}

	status.IsClean = status.Staged == 0 && status.Modified == 0 && status.Untracked == 0

	return status, nil
}

// parseGitStatusOutput parses git status --porcelain output
func (wm *WorktreeManager) parseGitStatusOutput(output string, status *GitStatus) {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i == 0 && strings.HasPrefix(line, "##") {
			// Parse branch tracking info
			if strings.Contains(line, "...") {
				status.HasRemote = true
				parts := strings.Split(line, "...")
				if len(parts) > 1 {
					remotePart := strings.Fields(parts[1])[0]
					if strings.Contains(remotePart, "/") {
						status.RemoteName = strings.Split(remotePart, "/")[0]
					}
				}
			}
			// Parse ahead/behind info
			if strings.Contains(line, "[ahead") || strings.Contains(line, "[behind") {
				wm.parseAheadBehind(line, status)
			}
			continue
		}

		if len(line) >= 2 {
			staged := line[0]
			modified := line[1]

			if staged != ' ' && staged != '?' {
				status.Staged++
			}
			if modified != ' ' && modified != '?' {
				status.Modified++
			}
			if staged == '?' && modified == '?' {
				status.Untracked++
			}
		}
	}
}

// parseAheadBehind parses ahead/behind information from git status
func (wm *WorktreeManager) parseAheadBehind(line string, status *GitStatus) {
	// Simple parsing - could be enhanced for exact counts
	if strings.Contains(line, "ahead") {
		status.Ahead = 1 // Simplified for now
	}
	if strings.Contains(line, "behind") {
		status.Behind = 1 // Simplified for now
	}
}

// DeleteWorktree removes a worktree and its associated branch
func (wm *WorktreeManager) DeleteWorktree(worktreeInfo WorktreeInfo) error {
	// Remove Git worktree
	cmd := exec.Command("git", "worktree", "remove", "-f", worktreeInfo.Path)
	cmd.Dir = wm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove Git worktree: %w", err)
	}

	// Delete the branch
	cmd = exec.Command("git", "branch", "-D", worktreeInfo.Branch)
	cmd.Dir = wm.repoPath
	if err := cmd.Run(); err != nil {
		// Log warning but don't fail - worktree is already removed
		fmt.Printf("Warning: failed to delete branch '%s': %v\n", worktreeInfo.Branch, err)
	}

	// Remove directory if it still exists
	if err := os.RemoveAll(worktreeInfo.Path); err != nil {
		fmt.Printf("Warning: failed to remove directory '%s': %v\n", worktreeInfo.Path, err)
	}

	// Clean up empty parent directory
	parentDir := filepath.Dir(worktreeInfo.Path)
	if isEmpty, _ := isDirEmpty(parentDir); isEmpty {
		_ = os.Remove(parentDir) // Ignore error as this is cleanup
	}

	return nil
}

// isDirEmpty checks if a directory is empty
func isDirEmpty(dirname string) (bool, error) {
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// CommitAll stages all changes (tracked and untracked) and commits with the provided message
// Returns the commit SHA on success
func CommitAll(repoPath, message string) (string, error) {
	if message == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}

	// Stage all changes
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit changes
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a "nothing to commit" error
		if strings.Contains(string(output), "nothing to commit") {
			return "", fmt.Errorf("no changes to commit")
		}
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	// Get the commit SHA
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	shaOutput, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	sha := strings.TrimSpace(string(shaOutput))
	return sha, nil
}

// StageFile stages a specific file
func StageFile(repoPath, filePath string) error {
	cmd := exec.Command("git", "add", filePath)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file %s: %w", filePath, err)
	}
	return nil
}

// UnstageFile unstages a specific file
func UnstageFile(repoPath, filePath string) error {
	cmd := exec.Command("git", "restore", "--staged", filePath)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unstage file %s: %w", filePath, err)
	}
	return nil
}

// DiscardFile discards changes to a specific file
func DiscardFile(repoPath, filePath string) error {
	cmd := exec.Command("git", "restore", filePath)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to discard changes for file %s: %w", filePath, err)
	}
	return nil
}

// StageAll stages all changes in the repository
func StageAll(repoPath string) error {
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage all changes: %w", err)
	}
	return nil
}

// GetCommitLog gets list of commits on sourceBranch that aren't on targetBranch
func GetCommitLog(repoPath, targetBranch, sourceBranch string) ([]CommitInfo, error) {
	// Use git log with custom format: hash|message|author|date
	// Format: %h = short hash, %s = subject, %an = author name, %ad = author date
	cmd := exec.Command("git", "log", fmt.Sprintf("%s..%s", targetBranch, sourceBranch),
		"--pretty=format:%h|%s|%an|%ad", "--date=short")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	// Parse output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var commits []CommitInfo

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}

		commits = append(commits, CommitInfo{
			Hash:    parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}

	return commits, nil
}

// GetDiffFiles gets list of files that differ between source and target branches
func GetDiffFiles(repoPath, sourceBranch, targetBranch string) ([]string, error) {
	// Use git diff --name-only to get list of changed files
	// Using three-dot syntax (targetBranch...sourceBranch) to show changes in sourceBranch
	// that are not in targetBranch
	cmd := exec.Command("git", "diff", "--name-only", fmt.Sprintf("%s...%s", targetBranch, sourceBranch))
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff files: %w", err)
	}

	// Parse output - each line is a file path
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// MergeBranch merges source branch into current branch of target repo (main worktree)
func MergeBranch(targetRepoPath, sourceBranch, commitMessage string) error {
	// Verify we're on a valid branch (not detached HEAD)
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = targetRepoPath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	currentBranch := strings.TrimSpace(string(output))
	if currentBranch == "" {
		return fmt.Errorf("not on a valid branch (detached HEAD)")
	}

	// Verify source branch exists
	cmd = exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+sourceBranch)
	cmd.Dir = targetRepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("source branch '%s' does not exist", sourceBranch)
	}

	// Perform merge
	cmd = exec.Command("git", "merge", sourceBranch, "-m", commitMessage)
	cmd.Dir = targetRepoPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// Check for merge conflicts
		if strings.Contains(outputStr, "CONFLICT") || strings.Contains(outputStr, "conflict") {
			return fmt.Errorf("merge conflict occurred: %w", err)
		}
		return fmt.Errorf("merge failed: %w\nOutput: %s", err, outputStr)
	}

	return nil
}
