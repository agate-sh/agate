package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FileStatus represents the Git status and change statistics for a single file
type FileStatus struct {
	FilePath    string // Relative path from repository root
	FileName    string // Just the filename
	DirPath     string // Directory path (for display truncation)
	Status      string // Git status code (M, A, D, ??, etc.)
	Additions   int    // Number of added lines
	Deletions   int    // Number of deleted lines
	IsUntracked bool   // Whether the file is untracked
}

// RepoFileStatus represents the Git status for an entire repository or worktree
type RepoFileStatus struct {
	Files          []FileStatus // List of changed files
	TotalFiles     int          // Total number of changed files
	TotalAdditions int          // Total lines added across all files
	TotalDeletions int          // Total lines deleted across all files
	IsClean        bool         // Whether the repo has no changes
	Error          error        // Any error that occurred during status retrieval
}

// GetFileStatuses returns the Git status for all changed files in a repository path
func GetFileStatuses(repoPath string) *RepoFileStatus {
	result := &RepoFileStatus{}

	// Get the list of changed files using git status --porcelain
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	statusOutput, err := cmd.Output()
	if err != nil {
		result.Error = fmt.Errorf("failed to get git status: %w", err)
		return result
	}

	statusLines := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
	if len(statusLines) == 1 && statusLines[0] == "" {
		// No changes
		result.IsClean = true
		return result
	}

	// Parse status output to get file list and statuses
	var files []FileStatus
	for _, line := range statusLines {
		if len(line) < 3 {
			continue
		}

		status := strings.TrimSpace(line[:2])
		filePath := strings.TrimSpace(line[3:])

		// Handle renamed files (format: "old -> new")
		if strings.Contains(filePath, " -> ") {
			parts := strings.Split(filePath, " -> ")
			if len(parts) == 2 {
				filePath = parts[1] // Use the new name
			}
		}

		fileName := filepath.Base(filePath)
		dirPath := filepath.Dir(filePath)
		if dirPath == "." {
			dirPath = ""
		}

		file := FileStatus{
			FilePath:    filePath,
			FileName:    fileName,
			DirPath:     dirPath,
			Status:      status,
			IsUntracked: status == "??",
		}

		files = append(files, file)
	}

	// Get addition/deletion counts for tracked files using git diff --numstat
	if len(files) > 0 {
		addDelCounts := getAdditionDeletionCounts(repoPath)

		// Match files with their add/del counts
		for i := range files {
			if !files[i].IsUntracked {
				if counts, exists := addDelCounts[files[i].FilePath]; exists {
					files[i].Additions = counts.additions
					files[i].Deletions = counts.deletions
				}
			}
		}
	}

	// Calculate totals
	totalAdditions := 0
	totalDeletions := 0
	for _, file := range files {
		totalAdditions += file.Additions
		totalDeletions += file.Deletions
	}

	result.Files = files
	result.TotalFiles = len(files)
	result.TotalAdditions = totalAdditions
	result.TotalDeletions = totalDeletions
	result.IsClean = len(files) == 0

	return result
}

type addDelCount struct {
	additions int
	deletions int
}

// getAdditionDeletionCounts gets the addition/deletion counts for changed files
func getAdditionDeletionCounts(repoPath string) map[string]addDelCount {
	counts := make(map[string]addDelCount)

	// Use git diff --numstat to get addition/deletion counts
	// This covers staged and unstaged changes
	cmd := exec.Command("git", "diff", "--numstat", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// Try without HEAD in case it's a new repo
		cmd = exec.Command("git", "diff", "--numstat", "--cached")
		cmd.Dir = repoPath
		output, err = cmd.Output()
		if err != nil {
			return counts
		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Format: "5	3	path/to/file"
		addStr := fields[0]
		delStr := fields[1]
		filePath := fields[2]

		// Handle binary files (marked with -)
		if addStr == "-" || delStr == "-" {
			continue
		}

		additions, err1 := strconv.Atoi(addStr)
		deletions, err2 := strconv.Atoi(delStr)

		if err1 == nil && err2 == nil {
			counts[filePath] = addDelCount{
				additions: additions,
				deletions: deletions,
			}
		}
	}

	return counts
}

// FormatSummaryLine returns a formatted summary line like "8 files changed"
func (r *RepoFileStatus) FormatSummaryLine() string {
	if r.Error != nil {
		return "Error getting git status"
	}

	if r.IsClean {
		return "No changes"
	}

	if r.TotalFiles == 1 {
		return "1 file changed"
	}

	return fmt.Sprintf("%d files changed", r.TotalFiles)
}
