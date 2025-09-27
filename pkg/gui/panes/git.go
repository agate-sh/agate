package panes

import (
	"agate/pkg/common"
	"agate/pkg/gui/components"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"agate/pkg/git"
	"agate/pkg/gui/icons"
	"agate/pkg/gui/theme"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GitPane manages the display of Git file status information
type GitPane struct {
	*components.BasePane // Embedded BasePane for common functionality
	fileStatus           *git.RepoFileStatus
	repoPath             string
	selectedIndex        int // Currently selected file index
	fullWidth            int // Cached width including pane padding
}

// NewGitPane creates a new GitPane instance
func NewGitPane() *GitPane {
	return &GitPane{
		BasePane: components.NewBasePane(2, "Git"), // Pane index 2
	}
}

// SetSize updates the dimensions of the Git pane
func (g *GitPane) SetSize(width, height int) {
	g.BasePane.SetSize(width, height)
	g.fullWidth = components.PaneFullWidth(width)
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
	g.BasePane.SetActive(active)
}

// MoveUp moves the selection up one file
func (g *GitPane) MoveUp() bool {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return false
	}
	if g.selectedIndex > 0 {
		g.selectedIndex--
		return true
	}
	return false
}

// MoveDown moves the selection down one file
func (g *GitPane) MoveDown() bool {
	if g.fileStatus == nil || len(g.fileStatus.Files) == 0 {
		return false
	}
	if g.selectedIndex < len(g.fileStatus.Files)-1 {
		g.selectedIndex++
		return true
	}
	return false
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
	if !g.IsActive() {
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
		// Log: Enter key pressed, opening selected file
		return true, g.openSelectedFile()
	default:
		return false, nil
	}
}

// openSelectedFile opens the selected file in the user's editor
func (g *GitPane) openSelectedFile() tea.Cmd {
	file := g.GetSelectedFile()
	if file == nil {
		// Log: No file selected
		return nil
	}

	// Build full file path
	fullPath := filepath.Join(g.repoPath, file.DirPath, file.FileName)
	// Log: Opening file

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Fallback to vi
	}
	// Log: Using editor

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
			// Log: Error opening file
		} else {
			// Log: File opened successfully in editor
		}
		return nil
	}
}

// GetTitle returns the dynamic title for the git pane
func (g *GitPane) GetTitle() string {
	return "Git"
}

// GetTitleStyle returns the title style for the git pane
func (g *GitPane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if g.IsActive() {
		// When active, show the shortcut hint
		help := common.GlobalKeys.OpenInEditor.Help()
		shortcuts = fmt.Sprintf("[%s: %s]", help.Key, help.Desc)
	} else {
		// When not active, show pane number
		shortcuts = "[2]"
	}

	return components.TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      "Git",
		Shortcuts: shortcuts,
	}
}

// Update handles tea.Msg updates for the git pane
func (g *GitPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	// GitPane doesn't handle any specific update messages currently
	// Navigation and key handling are done through the Pane interface methods
	return g, nil
}

// GetPaneSpecificKeybindings returns git pane specific keybindings
func (g *GitPane) GetPaneSpecificKeybindings() []key.Binding {
	// Use the global keybindings to ensure consistency
	return []key.Binding{common.GlobalKeys.OpenInEditor}
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
	innerWidth := g.GetWidth()
	if g.fullWidth == 0 {
		g.fullWidth = components.PaneFullWidth(innerWidth)
	}

	// Summary line (centered)
	summaryStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(theme.TextDescription)).
		Bold(true)

	summary := g.fileStatus.FormatSummaryLine()
	summaryLine := summaryStyle.Render(summary)
	output.WriteString(components.ApplyPaneContentPadding(summaryLine, innerWidth))
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
	innerWidth := g.GetWidth()

	// Style for the icon based on status
	iconStyle := g.getIconStyle(file.Status)
	styledIcon := iconStyle.Render(icon)

	// File name style
	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary))
	styledName := nameStyle.Render(file.FileName)

	// Directory path style (muted)
	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted))

	// Calculate available width for the path
	// Account for: icon(2) + space(1) + filename + space(2) + changes(~10) + margins
	usedWidth := 2 + 1 + len(file.FileName) + 2 + 10 + 4
	availableForPath := innerWidth - usedWidth
	if availableForPath < 0 {
		availableForPath = 0
	}

	// Format and truncate the directory path if needed
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
		// For untracked files, show line count in green
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
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
		padding := innerWidth - leftLen - rightLen - 2 // 2 for margins
		if padding < 1 {
			padding = 1
		}

		fullRow = leftSide + strings.Repeat(" ", padding) + changesStr
	} else {
		fullRow = leftSide
	}

	// Apply selection highlighting if this row is selected
	if index == g.selectedIndex && g.IsActive() {
		// We need to rebuild the row with background applied to each part
		// Create styles with background
		bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(theme.RowHighlight))

		// Reapply all styles with background
		iconWithBg := g.getIconStyle(file.Status).Background(lipgloss.Color(theme.RowHighlight)).Render(icon)
		nameWithBg := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Background(lipgloss.Color(theme.RowHighlight)).
			Render(file.FileName)

		// Directory path with background
		dirPathWithBg := ""
		if file.DirPath != "" && file.DirPath != "." {
			truncated := truncatePath(file.DirPath, availableForPath)
			dirPathWithBg = bgStyle.Render(" ") + lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.TextMuted)).
				Background(lipgloss.Color(theme.RowHighlight)).
				Render(truncated)
		}

		// Changes with background
		changesWithBg := ""
		if file.Additions > 0 || file.Deletions > 0 {
			addStr := fmt.Sprintf("+%d", file.Additions)
			delStr := fmt.Sprintf("-%d", file.Deletions)

			addWithBg := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.SuccessStatus)).
				Background(lipgloss.Color(theme.RowHighlight)).
				Render(addStr)
			delWithBg := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ErrorStatus)).
				Background(lipgloss.Color(theme.RowHighlight)).
				Render(delStr)

			spacerWithBg := bgStyle.Render(" ")
			changesWithBg = addWithBg + spacerWithBg + delWithBg
		} else if file.IsUntracked {
			if file.Additions > 0 {
				changesWithBg = lipgloss.NewStyle().
					Foreground(lipgloss.Color(theme.SuccessStatus)).
					Background(lipgloss.Color(theme.RowHighlight)).
					Render(fmt.Sprintf("+%d", file.Additions))
			} else {
				changesWithBg = lipgloss.NewStyle().
					Foreground(lipgloss.Color(theme.SuccessStatus)).
					Background(lipgloss.Color(theme.RowHighlight)).
					Render("new")
			}
		}

		// Build the row with backgrounds
		leftSide := iconWithBg + bgStyle.Render(" ") + nameWithBg + dirPathWithBg

		// Calculate padding
		var fullRow string
		if changesWithBg != "" {
			leftLen := lipgloss.Width(leftSide)
			rightLen := lipgloss.Width(changesWithBg)
			padding := innerWidth - leftLen - rightLen - 2
			if padding < 1 {
				padding = 1
			}
			paddingStr := bgStyle.Render(strings.Repeat(" ", padding))
			fullRow = leftSide + paddingStr + changesWithBg
		} else {
			fullRow = leftSide
		}

		// Pad to full width with background
		currentWidth := lipgloss.Width(fullRow)
		if currentWidth < innerWidth {
			fullRow = fullRow + bgStyle.Render(strings.Repeat(" ", innerWidth-currentWidth))
		}

		padCount := components.PaneContentHorizontalPadding()
		if padCount > 0 {
			pad := strings.Repeat(" ", padCount)
			leftPad := bgStyle.Render(pad)
			rightPad := bgStyle.Render(pad)
			fullRow = leftPad + fullRow + rightPad
		}

		finalWidth := lipgloss.Width(fullRow)
		if g.fullWidth > 0 && finalWidth < g.fullWidth {
			fullRow += bgStyle.Render(strings.Repeat(" ", g.fullWidth-finalWidth))
		}

		return fullRow
	}

	return components.ApplyPaneContentPadding(fullRow, innerWidth)
}

// getIconStyle returns the appropriate style for a Git status icon
func (g *GitPane) getIconStyle(status string) lipgloss.Style {
	switch status {
	case "M", "MM", "AM", "RM": // Modified
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.WarningStatus))
	case "A", "AD": // Added
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
	case "D", "DM": // Deleted
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ErrorStatus))
	case "??": // Untracked (new files - green like added)
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
	case "R": // Renamed
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.InfoStatus))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	}
}

// renderEmptyState renders a centered message for empty/error states
func (g *GitPane) renderEmptyState(message string) string {
	style := lipgloss.NewStyle().
		Width(components.PaneFullWidth(g.GetWidth())).
		Height(g.GetHeight()).
		Align(lipgloss.Center, lipgloss.Center).
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
