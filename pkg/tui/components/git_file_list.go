package components

import (
	"agate/pkg/git"
	"agate/pkg/tui/icons"
	"agate/pkg/tui/theme"
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
	height        int    // Available height for rendering
	fullWidth     int    // Full width including padding
	active        bool   // Whether this list is currently active/focused
	showSummary   bool   // Whether to show the summary line at top
	padding       int    // Horizontal padding for rows (0 for dialogs, 1 for panes)
	summaryGap    bool   // Whether to add a gap after the summary line (true for panes, false for dialogs)
}

// NewGitFileList creates a new Git file list component
func NewGitFileList(repoPath string, showSummary bool) *GitFileList {
	return &GitFileList{
		repoPath:      repoPath,
		selectedIndex: 0,
		showSummary:   showSummary,
		padding:       PaneContentHorizontalPadding(), // Default to pane padding
		summaryGap:    showSummary,                    // Default: gap if showing summary (pane), no gap if not (dialog)
	}
}

// SetPadding sets the horizontal padding for rows (0 for dialogs, 1 for panes)
func (g *GitFileList) SetPadding(padding int) {
	g.padding = padding
}

// SetSize sets the dimensions for rendering
func (g *GitFileList) SetSize(width int) {
	g.width = width
	g.fullWidth = PaneFullWidth(width)
}

// SetHeight sets the height for rendering (used for centering empty state)
func (g *GitFileList) SetHeight(height int) {
	g.height = height
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

// MoveDown moves selection down (wraps to top)
func (g *GitFileList) MoveDown() bool {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return false
	}
	if g.selectedIndex < len(g.fileStatus.Files)-1 {
		g.selectedIndex++
	} else {
		g.selectedIndex = 0 // Wrap to top
	}
	return true
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

	// Render all files in a single flat list (no sections)
	// This matches GitHub Desktop's approach where all changes are shown together
	for i, file := range g.fileStatus.Files {
		if i > 0 {
			output.WriteString("\n")
		}
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

	// Regular rendering (no selection) - use RenderRowWithHint for consistent right-aligned additions/deletions
	iconStyle := g.getIconStyle(file.Status)
	styledIcon := iconStyle.Render(icon)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary))
	styledName := nameStyle.Render(" " + file.FileName)

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted))

	dirPath := ""
	if file.DirPath != "" && file.DirPath != "." {
		dirPath = " " + file.DirPath
		dirPath = pathStyle.Render(dirPath)
	}

	// Format additions/deletions as "hint" to right-align them
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

	// Build content (icon + name + path)
	content := styledIcon + styledName + dirPath

	// Use RenderRowWithHint to right-align the additions/deletions
	return RenderRowWithHint(
		content,
		changesStr, // Additions/deletions as "hint" to right-align them
		g.width,
		g.padding,
		"", // No background color for non-selected rows
		theme.TextPrimary,
		theme.TextMuted,
	)
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
		g.padding,
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

// renderEmptyState renders a centered message with icon for empty/error states
func (g *GitFileList) renderEmptyState(message string) string {
	// Icon style - larger and centered
	iconStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Width(g.width).
		Align(lipgloss.Center)

	// Text style - muted and centered
	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Width(g.width).
		Align(lipgloss.Center)

	// Only show icon for "No changes" message
	var content string
	if message == "No changes" {
		icon := iconStyle.Render(icons.GitRepo.NerdFont)
		text := textStyle.Render(message)
		content = icon + "\n" + text
	} else {
		content = textStyle.Render(message)
	}

	// Center vertically if we have height
	if g.height > 0 {
		contentHeight := lipgloss.Height(content)
		topPadding := (g.height - contentHeight) / 2
		if topPadding > 0 {
			content = strings.Repeat("\n", topPadding) + content
		}
	}

	return content
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
