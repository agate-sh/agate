package components

import (
	"agate/pkg/git"
	"agate/pkg/gui/icons"
	"agate/pkg/gui/theme"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// GitFileList is a reusable component for displaying and interacting with Git file status
type GitFileList struct {
	fileStatus    *git.RepoFileStatus
	repoPath      string
	selectedIndex int    // Currently selected file index (across all files)
	width         int    // Available width for rendering
	fullWidth     int    // Full width including padding
	active        bool   // Whether this list is currently active/focused
	showSummary   bool   // Whether to show the summary line at top
}

// NewGitFileList creates a new Git file list component
func NewGitFileList(repoPath string, showSummary bool) *GitFileList {
	return &GitFileList{
		repoPath:      repoPath,
		selectedIndex: 0,
		showSummary:   showSummary,
	}
}

// SetSize sets the dimensions for rendering
func (g *GitFileList) SetSize(width int) {
	g.width = width
	g.fullWidth = PaneFullWidth(width)
}

// SetActive sets whether this list is currently focused
func (g *GitFileList) SetActive(active bool) {
	g.active = active
}

// Refresh updates the file status from git
func (g *GitFileList) Refresh() {
	if g.repoPath == "" {
		g.fileStatus = nil
		return
	}
	g.fileStatus = git.GetFileStatuses(g.repoPath)
	g.selectedIndex = 0
}

// GetFileStatus returns the current file status
func (g *GitFileList) GetFileStatus() *git.RepoFileStatus {
	return g.fileStatus
}

// MoveUp moves selection up
func (g *GitFileList) MoveUp() bool {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return false
	}
	if g.selectedIndex > 0 {
		g.selectedIndex--
		return true
	}
	return false
}

// MoveDown moves selection down
func (g *GitFileList) MoveDown() bool {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return false
	}
	if g.selectedIndex < len(g.fileStatus.Files)-1 {
		g.selectedIndex++
		return true
	}
	return false
}

// GetSelectedFile returns the currently selected file
func (g *GitFileList) GetSelectedFile() *git.FileStatus {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return nil
	}
	if g.selectedIndex >= 0 && g.selectedIndex < len(g.fileStatus.Files) {
		return &g.fileStatus.Files[g.selectedIndex]
	}
	return nil
}

// IsInStagedSection returns true if the selected file is in the staged section
func (g *GitFileList) IsInStagedSection() bool {
	file := g.GetSelectedFile()
	return file != nil && file.IsStaged
}

// View renders the file list
func (g *GitFileList) View() string {
	if g.repoPath == "" || g.fileStatus == nil {
		return g.renderEmptyState("No repository")
	}

	if g.fileStatus.Error != nil {
		return g.renderEmptyState("Error getting git status")
	}

	if g.fileStatus.IsClean {
		return g.renderEmptyState("No changes")
	}

	var output strings.Builder

	// Summary line (if enabled)
	if g.showSummary {
		summaryStyle := lipgloss.NewStyle().
			Width(g.width).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Bold(true)

		summary := g.fileStatus.FormatSummaryLine()
		summaryLine := summaryStyle.Render(summary)
		output.WriteString(ApplyPaneContentPadding(summaryLine, g.width))
		output.WriteString("\n")
	}

	// Render all files in a single flat list (no sections)
	// This matches GitHub Desktop's approach where all changes are shown together
	for i, file := range g.fileStatus.Files {
		output.WriteString("\n")
		row := g.renderFileRow(file, i)
		output.WriteString(row)
	}

	return output.String()
}

// renderFileRow renders a single file row
func (g *GitFileList) renderFileRow(file git.FileStatus, index int) string {
	icon := icons.GetGitStatusIcon(file.Status)
	isSelected := index == g.selectedIndex && g.active

	// If selected, show hint text
	var hint string
	if isSelected {
		hint = "↵ open in editor • d discard"
	}

	// Apply selection highlighting
	if isSelected {
		return g.renderRowWithBackgroundAndHint(file, icon, hint)
	}

	// Regular rendering (no selection)
	iconStyle := g.getIconStyle(file.Status)
	styledIcon := iconStyle.Render(icon)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary))
	styledName := nameStyle.Render(file.FileName)

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted))

	// Calculate available width for path
	usedWidth := 2 + 1 + len(file.FileName) + 2 + 10 + 4
	availableForPath := g.width - usedWidth
	if availableForPath < 0 {
		availableForPath = 0
	}

	dirPath := ""
	if file.DirPath != "" && file.DirPath != "." {
		dirPath = " " + truncatePath(file.DirPath, availableForPath)
		dirPath = pathStyle.Render(dirPath)
	}

	// Format additions/deletions
	changesStr := ""
	if file.Additions > 0 || file.Deletions > 0 {
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
		delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ErrorStatus))

		addStr := fmt.Sprintf("+%d", file.Additions)
		delStr := fmt.Sprintf("-%d", file.Deletions)

		changesStr = addStyle.Render(addStr) + " " + delStyle.Render(delStr)
	} else if file.IsUntracked {
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
		if file.Additions > 0 {
			changesStr = addStyle.Render(fmt.Sprintf("+%d", file.Additions))
		} else {
			changesStr = addStyle.Render("new")
		}
	}

	// Build row
	leftSide := fmt.Sprintf("%s %s%s", styledIcon, styledName, dirPath)

	var fullRow string
	if changesStr != "" {
		leftLen := lipgloss.Width(leftSide)
		rightLen := lipgloss.Width(changesStr)
		padding := g.width - leftLen - rightLen - 2
		if padding < 1 {
			padding = 1
		}
		fullRow = leftSide + strings.Repeat(" ", padding) + changesStr
	} else {
		fullRow = leftSide
	}

	return ApplyPaneContentPadding(fullRow, g.width)
}

// renderRowWithBackgroundAndHint renders a row with background highlighting and hint text
// Uses the shared RenderRowWithHint utility
func (g *GitFileList) renderRowWithBackgroundAndHint(file git.FileStatus, icon string, hint string) string {
	// Style the icon with its proper color and background
	iconStyle := g.getIconStyle(file.Status)
	styledIcon := iconStyle.Background(lipgloss.Color(theme.RowHighlight)).Render(icon)

	// Style the filename with background
	filenameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Background(lipgloss.Color(theme.RowHighlight))
	styledFilename := filenameStyle.Render(" " + file.FileName)

	// Build content: styled icon + styled filename (NO path on selected row)
	content := styledIcon + styledFilename

	// Use shared utility
	return RenderRowWithHint(
		content,
		hint,
		g.width,
		PaneContentHorizontalPadding(),
		theme.RowHighlight,
		theme.TextPrimary,
		theme.TextMuted,
	)
}

// getIconStyle returns the style for a git status icon
func (g *GitFileList) getIconStyle(status string) lipgloss.Style {
	// Handle space-padded statuses from git porcelain format
	// Format: XY where X=staged, Y=unstaged, space=no change in that area
	switch status {
	// Modified (yellow) - covers all modification cases
	case " M", "M ", "MM", "AM":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.WarningStatus))
	// Added (green) - new files in index
	case "A ", "AD":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
	// Deleted (red) - removed files
	case " D", "D ", "DD", "DM":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ErrorStatus))
	// Untracked (green) - new files not in index
	case "??":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
	// Renamed (blue/info) - includes renamed + modified
	case "R ", "RM":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.InfoStatus))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	}
}

// renderEmptyState renders a message for empty/error states
func (g *GitFileList) renderEmptyState(message string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted))
	return style.Render(message)
}

// truncatePath truncates a path from the left if it's longer than maxWidth
func truncatePath(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}
	if maxWidth <= 3 {
		return "..."
	}
	return "..." + path[len(path)-(maxWidth-3):]
}
