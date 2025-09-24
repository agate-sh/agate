package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"agate/git"
	"agate/icons"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GitPane manages the display of Git file status information
type GitPane struct {
	fileStatus    *git.RepoFileStatus
	width         int
	height        int
	repoPath      string
	selectedIndex int  // Currently selected file index
	isActive      bool // Whether this pane is currently focused
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
	// Reset selection when refreshing
	g.selectedIndex = 0
}

// SetActive sets whether this pane is currently focused
func (g *GitPane) SetActive(active bool) {
	g.isActive = active
}

// MoveUp moves the selection up one file
func (g *GitPane) MoveUp() {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return
	}
	if g.selectedIndex > 0 {
		g.selectedIndex--
	}
}

// MoveDown moves the selection down one file
func (g *GitPane) MoveDown() {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return
	}
	if g.selectedIndex < len(g.fileStatus.Files)-1 {
		g.selectedIndex++
	}
}

// GetSelectedFile returns the currently selected file, or nil if none
func (g *GitPane) GetSelectedFile() *git.FileStatus {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return nil
	}
	if g.selectedIndex >= 0 && g.selectedIndex < len(g.fileStatus.Files) {
		return &g.fileStatus.Files[g.selectedIndex]
	}
	return nil
}

// HandleKey processes keyboard input when the pane is active
func (g *GitPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	if !g.isActive {
		return false, nil
	}

	switch key {
	case "up", "k":
		g.MoveUp()
		return true, nil
	case "down", "j":
		g.MoveDown()
		return true, nil
	case "enter":
		DebugLog("GitPane: Enter key pressed, opening selected file")
		return true, g.openSelectedFile()
	default:
		return false, nil
	}
}

// openSelectedFile opens the selected file in the user's editor
func (g *GitPane) openSelectedFile() tea.Cmd {
	file := g.GetSelectedFile()
	if file == nil {
		DebugLog("GitPane: No file selected")
		return nil
	}

	// Build full file path
	fullPath := filepath.Join(g.repoPath, file.DirPath, file.FileName)
	DebugLog("GitPane: Opening file: %s", fullPath)

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Fallback to vi
	}
	DebugLog("GitPane: Using editor: %s", editor)

	// Parse editor command (handle cases like "code --wait")
	editorParts := strings.Fields(editor)
	var cmd *exec.Cmd
	if len(editorParts) > 1 {
		// Editor has arguments (e.g., "code --wait")
		cmd = exec.Command(editorParts[0], append(editorParts[1:], fullPath)...)
	} else {
		// Simple editor command (e.g., "vi")
		cmd = exec.Command(editor, fullPath)
	}

	// Launch editor in background without blocking the terminal
	return func() tea.Msg {
		err := cmd.Start()
		if err != nil {
			DebugLog("GitPane: Error opening file: %v", err)
		} else {
			DebugLog("GitPane: File opened successfully in editor")
		}
		return nil
	}
}

// GetTitle returns the dynamic title for the git pane
func (g *GitPane) GetTitle() string {
	if g.isActive {
		// When active, show the shortcut hint in square brackets like other panes
		return "Git " + lipgloss.NewStyle().
			Foreground(lipgloss.Color(textMuted)).
			Render("[↵ open]")
	}
	// When not active, show the pane number
	return "Git " + lipgloss.NewStyle().
		Foreground(lipgloss.Color(textMuted)).
		Render("[2]")
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
	// No extra padding - files start immediately after summary

	// File rows
	for i, file := range g.fileStatus.Files {
		output.WriteString("\n") // Line break before each file
		row := g.renderFileRow(file, i)
		output.WriteString(row)
	}

	return output.String()
}

// renderFileRow renders a single file row with icon, name, path, and change counts
func (g *GitPane) renderFileRow(file git.FileStatus, index int) string {
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
		// For untracked files, show line count in green
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(successStatus))
		// For new files, show the total lines as additions
		if file.Additions > 0 {
			changesStr = addStyle.Render(fmt.Sprintf("+%d", file.Additions))
		} else {
			changesStr = addStyle.Render("new")
		}
	}

	// Build the complete row
	// Left side: icon + filename + path
	leftSide := fmt.Sprintf("%s %s%s", styledIcon, styledName, dirPath)

	// Right side: changes
	var fullRow string
	if changesStr != "" {
		// Calculate padding to right-align the changes
		leftLen := lipgloss.Width(leftSide)
		rightLen := lipgloss.Width(changesStr)
		padding := g.width - leftLen - rightLen - 2 // 2 for margins
		if padding < 1 {
			padding = 1
		}

		fullRow = leftSide + strings.Repeat(" ", padding) + changesStr
	} else {
		fullRow = leftSide
	}

	// Apply selection highlighting if this row is selected
	if index == g.selectedIndex {
		// Pad the row to full width first
		paddedRow := fullRow
		currentWidth := lipgloss.Width(fullRow)
		if currentWidth < g.width {
			paddedRow = fullRow + strings.Repeat(" ", g.width-currentWidth)
		}

		selectionStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(textMuted))
		return selectionStyle.Render(paddedRow)
	}

	return fullRow
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
	case "??": // Untracked (new files - green like added)
		return lipgloss.NewStyle().Foreground(lipgloss.Color(successStatus))
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