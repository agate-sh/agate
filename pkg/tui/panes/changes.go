package panes

import (
	"agate/pkg/common"
	"agate/pkg/git"
	"agate/pkg/tui/components"
	"agate/pkg/tui/icons"
	"agate/pkg/tui/theme"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// FocusPane represents which part of the changes pane is focused
type FocusPane int

const (
	FocusPaneFileList FocusPane = iota
	FocusPaneDiffViewer
)

const (
	filesTabZoneID   = "changes_tabs_files"
	diffTabZoneID    = "changes_tabs_diff"
	fileListZoneID   = "changes_content_files"
	diffViewerZoneID = "changes_content_diff"
)

// ChangesPane displays file changes from the active agent's worktree
type ChangesPane struct {
	*components.BasePane
	fileList         *components.GitFileList
	diffViewer       *components.GitDiffViewer
	worktreePath     string
	repository       *git.Repository
	focusedPane      FocusPane
	lastSelectedFile string // Track last selected file
}

// ChangesRefreshMsg triggers a refresh of the changes pane
type ChangesRefreshMsg struct{}

// GitRefreshMsg triggers a refresh of the git file status (for backwards compatibility)
type GitRefreshMsg struct{}

// NewChangesPane creates a new ChangesPane instance
func NewChangesPane() *ChangesPane {
	fileList := components.NewGitFileList("", nil, false) // Don't show summary, no repo yet
	diffViewer := components.NewGitDiffViewer()
	pane := &ChangesPane{
		BasePane:    components.NewBasePane(2, "Changes"), // Pane index 2
		fileList:    fileList,
		diffViewer:  diffViewer,
		focusedPane: FocusPaneFileList, // Start with file list focused
	}
	pane.SetChromePadding(0, components.PaneContentVerticalPadding())
	return pane
}

// SetRepositoryAndPath sets both the repository and the worktree path
func (p *ChangesPane) SetRepositoryAndPath(repo *git.Repository, worktreePath string) {
	p.repository = repo
	p.SetWorktreePath(worktreePath)
}

// SetWorktreePath sets the active worktree path to display changes from
func (p *ChangesPane) SetWorktreePath(worktreePath string) {
	if worktreePath != p.worktreePath {
		p.worktreePath = worktreePath
		if p.fileList != nil {
			p.fileList = components.NewGitFileList(worktreePath, p.repository, true)
			p.fileList.SetSize(p.GetWidth())
			p.fileList.SetHeight(p.GetHeight())
		}
		p.Refresh()
	}
}

// Refresh updates the file status for the current repository
func (p *ChangesPane) Refresh() {
	if p.fileList != nil {
		p.fileList.Refresh()
	}
}

// SetSize updates the dimensions of the changes pane
func (p *ChangesPane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)

	// Account for header (1 line) + divider (1 line) = 2 lines
	contentHeight := height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Both components use full width but reduced height
	if p.fileList != nil {
		p.fileList.SetSize(width)
		p.fileList.SetHeight(contentHeight)
	}

	if p.diffViewer != nil {
		p.diffViewer.SetSize(width, contentHeight)
	}
}

// SetActive sets whether this pane is currently focused
func (p *ChangesPane) SetActive(active bool) {
	p.BasePane.SetActive(active)
	// Only the focused sub-pane should be active
	if p.fileList != nil {
		p.fileList.SetActive(active && p.focusedPane == FocusPaneFileList)
	}
	if p.diffViewer != nil {
		p.diffViewer.SetActive(active && p.focusedPane == FocusPaneDiffViewer)
	}
}

// GetSelectedFile returns the currently selected file, or nil if none
func (p *ChangesPane) GetSelectedFile() *git.FileStatus {
	if p.fileList == nil {
		return nil
	}
	return p.fileList.GetSelectedFile()
}

// GetWorktreePath returns the current worktree path
func (p *ChangesPane) GetWorktreePath() string {
	return p.worktreePath
}

// MoveUp moves the selection up one item
func (p *ChangesPane) MoveUp() bool {
	if p.fileList == nil {
		return false
	}
	return p.fileList.MoveUp()
}

// MoveDown moves the selection down one item
func (p *ChangesPane) MoveDown() bool {
	if p.fileList == nil {
		return false
	}
	return p.fileList.MoveDown()
}

// renderHeader renders the internal pane header with "Files | Diff"
func (p *ChangesPane) renderHeader() string {
	// Get change count badge
	changeCount := p.changeCount()
	badge := components.RenderChangeCountBadge(changeCount)

	// Build Files text with badge
	var filesLabel string
	if badge != "" {
		filesLabel = "Files " + badge
	} else {
		filesLabel = "Files"
	}

	width := p.GetWidth()
	if width <= 0 {
		return ""
	}

	// Split available space between tabs and leave room for the separator.
	leftWidth := width / 2
	rightWidth := width - leftWidth - 1
	if rightWidth < 0 {
		rightWidth = 0
	}

	// Create base styles
	baseStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary))
	boldStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Bold(true)

	// Render tabs centered in their width
	var filesTab string
	if leftWidth > 0 {
		filesStyle := baseStyle
		if p.focusedPane == FocusPaneFileList {
			filesStyle = boldStyle
		}
		filesTab = filesStyle.Width(leftWidth).Align(lipgloss.Center).Render(filesLabel)
		filesTab = zone.Mark(filesTabZoneID, filesTab)
	}

	var diffTab string
	if rightWidth > 0 {
		diffStyle := baseStyle
		if p.focusedPane == FocusPaneDiffViewer {
			diffStyle = boldStyle
		}
		diffTab = diffStyle.Width(rightWidth).Align(lipgloss.Center).Render("Diff")
		diffTab = zone.Mark(diffTabZoneID, diffTab)
	}

	// Create full-height pipe separator
	pipeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.BorderMuted))

	var pipe string
	if width > 1 {
		tabHeight := lipgloss.Height(filesTab)
		if h := lipgloss.Height(diffTab); h > tabHeight {
			tabHeight = h
		}
		if tabHeight < 1 {
			tabHeight = 1
		}

		pipeSegment := pipeStyle.Render("│")
		if tabHeight == 1 {
			pipe = pipeSegment
		} else {
			lines := make([]string, tabHeight)
			for i := range lines {
				lines[i] = pipeSegment
			}
			pipe = strings.Join(lines, "\n")
		}
	} else {
		pipe = ""
	}

	// Join tabs with pipe
	return lipgloss.JoinHorizontal(lipgloss.Top, filesTab, pipe, diffTab)
}

// View renders the changes pane content with header, divider, and active view
func (p *ChangesPane) View() string {
	// Render header
	header := p.renderHeader()

	// Render divider
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.BorderMuted))
	dividerWidth := p.GetWidth()
	if dividerWidth < 0 {
		dividerWidth = 0
	}
	divider := dividerStyle.Render(strings.Repeat("─", dividerWidth))

	// Get the content view
	var content string
	if p.focusedPane == FocusPaneFileList {
		if p.fileList == nil {
			content = ""
		} else {
			content = p.fileList.View()
			if content != "" {
				content = zone.Mark(fileListZoneID, content)
			}
		}
	} else {
		if p.diffViewer == nil {
			content = ""
		} else {
			// Ensure diff is loaded for selected file before showing
			p.updateDiffViewer()
			content = p.diffViewer.View()
			if content != "" {
				content = zone.Mark(diffViewerZoneID, content)
			}
		}
	}

	// Combine header + divider + content
	return lipgloss.JoinVertical(lipgloss.Left, header, divider, content)
}

// Update handles tea.Msg updates for the changes pane
func (p *ChangesPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	switch msg.(type) {
	case ChangesRefreshMsg, GitRefreshMsg:
		p.Refresh()
		return p, nil
	}
	return p, nil
}

// HandleKey processes keyboard input when the pane is active
func (p *ChangesPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	if !p.IsActive() {
		return false, nil
	}

	// Handle tab to toggle between sub-panes
	if key == "tab" {
		p.toggleFocus()
		return true, nil
	}

	// Route keys based on focused pane
	if p.focusedPane == FocusPaneFileList {
		return p.handleFileListKey(key)
	} else {
		return p.handleDiffViewerKey(key)
	}
}

// toggleFocus switches focus between file list and diff viewer
func (p *ChangesPane) toggleFocus() {
	if p.focusedPane == FocusPaneFileList {
		// Before switching to diff viewer, ensure diff is loaded
		p.updateDiffViewer()
		p.focusedPane = FocusPaneDiffViewer
	} else {
		p.focusedPane = FocusPaneFileList
	}
	// Update active state for both components
	p.SetActive(p.IsActive())
}

// handleFileListKey handles keys when file list is focused
func (p *ChangesPane) handleFileListKey(key string) (handled bool, cmd tea.Cmd) {
	if p.fileList == nil {
		return false, nil
	}

	switch key {
	case "up", "k":
		changed := p.MoveUp()
		return changed, nil
	case "down", "j":
		changed := p.MoveDown()
		return changed, nil
	case "d":
		// Discard selected file
		return true, p.discardFile()
	case "enter":
		// Open file in editor
		return true, p.openSelectedFile()
	default:
		return false, nil
	}
}

// handleDiffViewerKey handles keys when diff viewer is focused
func (p *ChangesPane) handleDiffViewerKey(key string) (handled bool, cmd tea.Cmd) {
	if p.diffViewer == nil {
		return false, nil
	}

	// Pass through key events to viewport for scrolling
	switch key {
	case "up", "k":
		cmd = p.diffViewer.Update(tea.KeyMsg{Type: tea.KeyUp})
		return true, cmd
	case "down", "j":
		cmd = p.diffViewer.Update(tea.KeyMsg{Type: tea.KeyDown})
		return true, cmd
	case "pgup":
		cmd = p.diffViewer.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		return true, cmd
	case "pgdown":
		cmd = p.diffViewer.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		return true, cmd
	default:
		return false, nil
	}
}

// HandleMouse processes mouse interactions for tabs and content zones.
func (p *ChangesPane) HandleMouse(msg tea.MouseMsg) (handled bool, cmd tea.Cmd) {
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease {
		if zone.Get(filesTabZoneID).InBounds(msg) {
			if p.focusedPane != FocusPaneFileList {
				p.focusedPane = FocusPaneFileList
				p.SetActive(p.IsActive())
			}
			return true, nil
		}
		if zone.Get(diffTabZoneID).InBounds(msg) {
			if p.focusedPane != FocusPaneDiffViewer {
				p.focusedPane = FocusPaneDiffViewer
				p.updateDiffViewer()
				p.SetActive(p.IsActive())
			}
			return true, nil
		}
	}

	if zone.Get(fileListZoneID).InBounds(msg) && p.fileList != nil {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.fileList.MoveUp()
			return true, nil
		case tea.MouseButtonWheelDown:
			p.fileList.MoveDown()
			return true, nil
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress && p.focusedPane != FocusPaneFileList {
				p.focusedPane = FocusPaneFileList
				p.SetActive(p.IsActive())
				return true, nil
			}
		}
	}

	if zone.Get(diffViewerZoneID).InBounds(msg) && p.diffViewer != nil {
		switch msg.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			cmd = p.diffViewer.Update(msg)
			return true, cmd
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress && p.focusedPane != FocusPaneDiffViewer {
				p.focusedPane = FocusPaneDiffViewer
				p.updateDiffViewer()
				p.SetActive(p.IsActive())
				return true, nil
			}
		}
	}

	return false, nil
}

// discardFile discards changes to the selected file
func (p *ChangesPane) discardFile() tea.Cmd {
	file := p.GetSelectedFile()
	if file == nil {
		return nil
	}

	return func() tea.Msg {
		if p.repository == nil {
			return nil
		}
		err := p.repository.DiscardFile(p.worktreePath, file.FilePath)
		if err == nil {
			return ChangesRefreshMsg{}
		}
		return nil
	}
}

// openSelectedFile opens the selected file in the user's editor
func (p *ChangesPane) openSelectedFile() tea.Cmd {
	file := p.GetSelectedFile()
	if file == nil {
		return nil
	}

	// Build full file path
	fullPath := filepath.Join(p.worktreePath, file.DirPath, file.FileName)

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Fallback to vi
	}

	// Parse editor command (handle cases like "code --wait")
	editorParts := strings.Fields(editor)
	var cmd *exec.Cmd
	if len(editorParts) > 1 {
		cmd = exec.Command(editorParts[0], append(editorParts[1:], fullPath)...)
	} else {
		cmd = exec.Command(editor, fullPath)
	}

	// Launch editor in background without blocking the terminal
	return func() tea.Msg {
		_ = cmd.Start()
		return nil
	}
}

// GetTitleStyle returns the title style for the changes pane
func (p *ChangesPane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if p.IsActive() {
		shortcuts = "tab cycle"
	} else {
		shortcuts = "(" + icons.GetOptionKey() + "c)"
	}

	return components.TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      "Changes",
		Shortcuts: shortcuts,
	}
}

// GetPaneSpecificKeybindings returns changes pane specific keybindings
func (p *ChangesPane) GetPaneSpecificKeybindings() []key.Binding {
	return []key.Binding{common.GlobalKeys.OpenInEditor}
}

// updateDiffViewer updates the diff viewer with the currently selected file
func (p *ChangesPane) updateDiffViewer() {
	if p.diffViewer == nil {
		return
	}

	if p.diffViewer.Width() < 1 {
		p.diffViewer.SetDiffContent("")
		return
	}

	file := p.GetSelectedFile()
	if file == nil {
		p.diffViewer.SetDiffContent("")
		p.lastSelectedFile = ""
		return
	}

	// Avoid re-loading if same file
	if file.FilePath == p.lastSelectedFile {
		return
	}

	p.lastSelectedFile = file.FilePath

	// Get rendered diff from delta
	if p.repository == nil {
		return
	}
	diffContent, err := p.repository.GetFileDiffRendered(p.worktreePath, file.FilePath, p.diffViewer.Width())
	if err != nil {
		// If error, clear the diff viewer
		p.diffViewer.SetDiffContent("")
		return
	}

	p.diffViewer.SetDiffContent(diffContent)
}

// changeCount returns the number of changed files
func (p *ChangesPane) changeCount() int {
	if p.fileList == nil {
		return 0
	}
	status := p.fileList.GetFileStatus()
	if status == nil || status.IsClean {
		return 0
	}
	return len(status.Files)
}
