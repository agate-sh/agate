package main

import (
	"fmt"
	"strings"

	"agate/git"
	"agate/icons"

	"github.com/charmbracelet/lipgloss"
)

// GitPane manages the display of Git file status information
type GitPane struct {
	fileStatus *git.RepoFileStatus
	width      int
	height     int
	repoPath   string
}

// NewGitPane creates a new GitPane instance
func NewGitPane() *GitPane {
	return &GitPane{}
}

// SetSize updates the dimensions of the Git pane
func (g *GitPane) SetSize(width, height int) {
	g.width = width
	g.height = height
}

// SetRepository updates the repository path and refreshes file status
func (g *GitPane) SetRepository(repoPath string) {
	if repoPath != g.repoPath {
		g.repoPath = repoPath
		g.Refresh()
	}
}

// Refresh updates the Git file status for the current repository
func (g *GitPane) Refresh() {
	if g.repoPath == "" {
		g.fileStatus = nil
		return
	}

	g.fileStatus = git.GetFileStatuses(g.repoPath)
}

// View renders the Git pane content
func (g *GitPane) View() string {
	if g.repoPath == "" || g.fileStatus == nil {
		// No repository selected
		return g.renderEmptyState("No repository selected")
	}

	if g.fileStatus.Error != nil {
		// Error getting status
		return g.renderEmptyState("Error getting git status")
	}

	if g.fileStatus.IsClean {
		// No changes
		return g.renderEmptyState("No changes")
	}

	// Render the file list with status
	var output strings.Builder

	// Summary line (centered)
	summaryStyle := lipgloss.NewStyle().
		Width(g.width).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(textDescription)).
		Bold(true)

	summary := g.fileStatus.FormatSummaryLine()
	output.WriteString(summaryStyle.Render(summary))
	output.WriteString("\n\n")

	// File rows
	for _, file := range g.fileStatus.Files {
		row := g.renderFileRow(file)
		output.WriteString(row)
		output.WriteString("\n")
	}

	return output.String()
}

// renderFileRow renders a single file row with icon, name, path, and change counts
func (g *GitPane) renderFileRow(file git.FileStatus) string {
	// Get the appropriate icon for the file status
	icon := icons.GetGitStatusIcon(file.Status)

	// Style for the icon based on status
	iconStyle := g.getIconStyle(file.Status)
	styledIcon := iconStyle.Render(icon)

	// File name style
	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(textPrimary))
	styledName := nameStyle.Render(file.FileName)

	// Directory path style (muted)
	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(textMuted))

	// Calculate available width for the path
	// Account for: icon(2) + space(1) + filename + space(2) + changes(~10) + margins
	usedWidth := 2 + 1 + len(file.FileName) + 2 + 10 + 4
	availableForPath := g.width - usedWidth
	if availableForPath < 0 {
		availableForPath = 0
	}

	// Format and truncate the directory path if needed
	dirPath := ""
	if file.DirPath != "" && file.DirPath != "." {
		dirPath = " " + truncatePathFromLeft(file.DirPath, availableForPath)
		dirPath = pathStyle.Render(dirPath)
	}

	// Format additions/deletions
	changesStr := ""
	if file.Additions > 0 || file.Deletions > 0 {
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(successStatus))
		delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(errorStatus))

		addStr := fmt.Sprintf("+%d", file.Additions)
		delStr := fmt.Sprintf("-%d", file.Deletions)

		changesStr = addStyle.Render(addStr) + " " + delStyle.Render(delStr)
	} else if file.IsUntracked {
		// For untracked files, show as new
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(successStatus))
		changesStr = addStyle.Render("new")
	}

	// Build the complete row
	// Left side: icon + filename + path
	leftSide := fmt.Sprintf("%s %s%s", styledIcon, styledName, dirPath)

	// Right side: changes
	if changesStr != "" {
		// Calculate padding to right-align the changes
		leftLen := lipgloss.Width(leftSide)
		rightLen := lipgloss.Width(changesStr)
		padding := g.width - leftLen - rightLen - 2 // 2 for margins
		if padding < 1 {
			padding = 1
		}

		return leftSide + strings.Repeat(" ", padding) + changesStr
	}

	return leftSide
}

// getIconStyle returns the appropriate style for a Git status icon
func (g *GitPane) getIconStyle(status string) lipgloss.Style {
	switch status {
	case "M", "MM", "AM", "RM": // Modified
		return lipgloss.NewStyle().Foreground(lipgloss.Color(warningStatus))
	case "A", "AD": // Added
		return lipgloss.NewStyle().Foreground(lipgloss.Color(successStatus))
	case "D", "DM": // Deleted
		return lipgloss.NewStyle().Foreground(lipgloss.Color(errorStatus))
	case "??": // Untracked
		return lipgloss.NewStyle().Foreground(lipgloss.Color(textMuted))
	case "R": // Renamed
		return lipgloss.NewStyle().Foreground(lipgloss.Color(infoStatus))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(textDescription))
	}
}

// renderEmptyState renders a centered message for empty/error states
func (g *GitPane) renderEmptyState(message string) string {
	style := lipgloss.NewStyle().
		Width(g.width).
		Height(g.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color(textMuted))

	return style.Render(message)
}