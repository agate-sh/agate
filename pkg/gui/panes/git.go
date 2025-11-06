package panes

import (
	"agate/pkg/common"
	"agate/pkg/gui/components"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"agate/internal/debug"
	"agate/pkg/git"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// FocusPane represents which part of the git pane is focused
type FocusPane int

const (
	FocusPaneFileList FocusPane = iota
	FocusPaneDiffViewer
)

// GitPane manages the display of Git file status information
type GitPane struct {
	*components.BasePane // Embedded BasePane for common functionality
	fileList             *components.GitFileList
	diffViewer           *components.GitDiffViewer
	repoPath             string
	focusedPane          FocusPane
	lastSelectedFile     string // Track last selected file to avoid re-loading diff
}

// gitRefreshMsg is sent when the git pane needs to refresh
// GitRefreshMsg triggers a refresh of the git file status
type GitRefreshMsg struct{}

// NewGitPane creates a new GitPane instance
func NewGitPane() *GitPane {
	fileList := components.NewGitFileList("", false) // Don't show summary
	diffViewer := components.NewGitDiffViewer()
	return &GitPane{
		BasePane:    components.NewBasePane(2, "Changes"), // Pane index 2
		fileList:    fileList,
		diffViewer:  diffViewer,
		focusedPane: FocusPaneFileList, // Start with file list focused
	}
}

// SetSize updates the dimensions of the Git pane
func (g *GitPane) SetSize(width, height int) {
	g.BasePane.SetSize(width, height)

	// Account for border (1 char wide)
	borderWidth := 1

	// Split: 25% for file list, remainder for diff viewer (accounting for border)
	fileListWidth := int(float64(width) * 0.25)
	diffViewerWidth := width - fileListWidth - borderWidth

	if g.fileList != nil {
		g.fileList.SetSize(fileListWidth)
		g.fileList.SetHeight(height)
	}

	prevDiffWidth := 0
	if g.diffViewer != nil {
		prevDiffWidth = g.diffViewer.Width()
	}

	if diffViewerWidth < 0 {
		diffViewerWidth = 0
	}

	if g.diffViewer != nil {
		g.diffViewer.SetSize(diffViewerWidth, height)
	}

	if diffViewerWidth != prevDiffWidth {
		// Force re-render of the current diff with the new width
		g.lastSelectedFile = ""
	}

	debug.DebugLog("[GitPane] SetSize width=%d height=%d fileListWidth=%d diffWidth=%d", width, height, fileListWidth, diffViewerWidth)
}

// SetRepository updates the repository path and refreshes file status
func (g *GitPane) SetRepository(repoPath string) {
	if repoPath != g.repoPath {
		g.repoPath = repoPath
		if g.fileList != nil {
			g.fileList = components.NewGitFileList(repoPath, true)
			g.fileList.SetSize(g.GetWidth())
			g.fileList.SetHeight(g.GetHeight())
		}
		g.Refresh()
	}
}

// Refresh updates the Git file status for the current repository
func (g *GitPane) Refresh() {
	if g.fileList != nil {
		g.fileList.Refresh()
	}
}

// SetActive sets whether this pane is currently focused
func (g *GitPane) SetActive(active bool) {
	g.BasePane.SetActive(active)
	// Only the focused sub-pane should be active
	if g.fileList != nil {
		g.fileList.SetActive(active && g.focusedPane == FocusPaneFileList)
	}
	if g.diffViewer != nil {
		g.diffViewer.SetActive(active && g.focusedPane == FocusPaneDiffViewer)
	}
}

// GetSelectedFile returns the currently selected file, or nil if none
func (g *GitPane) GetSelectedFile() *git.FileStatus {
	if g.fileList == nil {
		return nil
	}
	return g.fileList.GetSelectedFile()
}

// GetRepoPath returns the current repository path
func (g *GitPane) GetRepoPath() string {
	return g.repoPath
}

// MoveUp moves the selection up one item
func (g *GitPane) MoveUp() bool {
	if g.fileList == nil {
		return false
	}
	return g.fileList.MoveUp()
}

// MoveDown moves the selection down one item
func (g *GitPane) MoveDown() bool {
	if g.fileList == nil {
		return false
	}
	return g.fileList.MoveDown()
}

// HandleKey processes keyboard input when the pane is active
func (g *GitPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	if !g.IsActive() {
		return false, nil
	}

	// Handle left/right arrow keys to toggle between file list and diff viewer
	if key == "right" && g.focusedPane == FocusPaneFileList {
		g.toggleFocus()
		return true, nil
	}
	if key == "left" && g.focusedPane == FocusPaneDiffViewer {
		g.toggleFocus()
		return true, nil
	}

	// Route keys based on focused pane
	if g.focusedPane == FocusPaneFileList {
		return g.handleFileListKey(key)
	} else {
		return g.handleDiffViewerKey(key)
	}
}

// toggleFocus switches focus between file list and diff viewer
func (g *GitPane) toggleFocus() {
	if g.focusedPane == FocusPaneFileList {
		g.focusedPane = FocusPaneDiffViewer
	} else {
		g.focusedPane = FocusPaneFileList
	}
	// Update active state for both components
	g.SetActive(g.IsActive())
}

// handleFileListKey handles keys when file list is focused
func (g *GitPane) handleFileListKey(key string) (handled bool, cmd tea.Cmd) {
	if g.fileList == nil {
		return false, nil
	}

	switch key {
	case "up", "k":
		changed := g.fileList.MoveUp()
		if changed {
			g.updateDiffViewer()
		}
		return true, nil
	case "down", "j":
		changed := g.fileList.MoveDown()
		if changed {
			g.updateDiffViewer()
		}
		return true, nil
	case "d":
		// Discard selected file
		return true, g.discardFile()
	case "enter":
		return true, g.openSelectedFile()
	default:
		return false, nil
	}
}

// handleDiffViewerKey handles keys when diff viewer is focused
func (g *GitPane) handleDiffViewerKey(key string) (handled bool, cmd tea.Cmd) {
	if g.diffViewer == nil {
		return false, nil
	}

	// Pass through key events to viewport for scrolling
	switch key {
	case "up", "k":
		cmd = g.diffViewer.Update(tea.KeyMsg{Type: tea.KeyUp})
		return true, cmd
	case "down", "j":
		cmd = g.diffViewer.Update(tea.KeyMsg{Type: tea.KeyDown})
		return true, cmd
	case "pgup":
		cmd = g.diffViewer.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		return true, cmd
	case "pgdown":
		cmd = g.diffViewer.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		return true, cmd
	default:
		return false, nil
	}
}

// discardFile discards changes to the selected file
func (g *GitPane) discardFile() tea.Cmd {
	file := g.GetSelectedFile()
	if file == nil {
		return nil
	}

	return func() tea.Msg {
		err := git.DiscardFile(g.repoPath, file.FilePath)
		if err == nil {
			return GitRefreshMsg{}
		}
		return nil
	}
}

// openSelectedFile opens the selected file in the user's editor
func (g *GitPane) openSelectedFile() tea.Cmd {
	file := g.GetSelectedFile()
	if file == nil {
		return nil
	}

	// Build full file path
	fullPath := filepath.Join(g.repoPath, file.DirPath, file.FileName)

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

// GetTitle returns the dynamic title for the git pane
func (g *GitPane) GetTitle() string {
	return "Changes"
}

// GetTitleStyle returns the title style for the git pane
func (g *GitPane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if g.IsActive() {
		shortcuts = "c commit"
	} else {
		shortcuts = "(2)"
	}

	label := "Changes"
	changeCount := g.changeCount()
	badge := components.RenderChangeCountBadge(changeCount)
	if badge != "" {
		label = label + " " + badge
	}

	return components.TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      label,
		Shortcuts: shortcuts,
	}
}

// Update handles tea.Msg updates for the git pane
func (g *GitPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	switch msg.(type) {
	case GitRefreshMsg:
		g.Refresh()
		return g, nil
	}
	return g, nil
}

// GetPaneSpecificKeybindings returns git pane specific keybindings
func (g *GitPane) GetPaneSpecificKeybindings() []key.Binding {
	// Use the global keybindings to ensure consistency
	return []key.Binding{common.GlobalKeys.OpenInEditor}
}

// updateDiffViewer updates the diff viewer with the currently selected file
func (g *GitPane) updateDiffViewer() {
	if g.diffViewer == nil {
		return
	}

	if g.diffViewer.Width() < 1 {
		g.diffViewer.SetDiffContent("")
		return
	}

	diffWidth := g.diffViewer.Width()
	file := g.GetSelectedFile()
	filePath := "<nil>"
	if file != nil {
		filePath = file.FilePath
	}
	debug.DebugLog("[GitPane] updateDiffViewer width=%d file=%s", diffWidth, filePath)
	if file == nil {
		g.diffViewer.SetDiffContent("")
		g.lastSelectedFile = ""
		return
	}

	// Avoid re-loading if same file
	if file.FilePath == g.lastSelectedFile {
		return
	}

	g.lastSelectedFile = file.FilePath

	// Get rendered diff from delta
	diffContent, err := git.GetFileDiffRendered(g.repoPath, file.FilePath, g.diffViewer.Width())
	if err != nil {
		// If error, clear the diff viewer
		g.diffViewer.SetDiffContent("")
		return
	}

	g.diffViewer.SetDiffContent(diffContent)
}

// View renders the Git pane content with split layout
func (g *GitPane) View() string {
	if g.fileList == nil || g.diffViewer == nil {
		return ""
	}

	// Ensure diff is loaded for selected file
	g.updateDiffViewer()

	// Render both sides
	fileListView := g.fileList.View()
	diffViewerView := g.diffViewer.View()

	// Add a vertical border between the panes (full height)
	borderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3c3c3c")).
		Height(g.GetHeight())

	// Create a border string with the full height
	var borderBuilder strings.Builder
	for i := 0; i < g.GetHeight(); i++ {
		if i > 0 {
			borderBuilder.WriteString("\n")
		}
		borderBuilder.WriteString("│")
	}
	border := borderStyle.Render(borderBuilder.String())

	return joinGitSections(fileListView, border, diffViewerView, g.GetWidth(), g.GetHeight())
}

func (g *GitPane) changeCount() int {
	if g.fileList == nil {
		return 0
	}
	status := g.fileList.GetFileStatus()
	if status == nil || status.IsClean {
		return 0
	}
	return len(status.Files)
}

// joinGitSections composes the Git pane sections without padding trailing spaces on the diff content.
func joinGitSections(fileListView, borderView, diffView string, targetWidth, targetHeight int) string {
	left := lipgloss.JoinHorizontal(lipgloss.Top, fileListView, borderView)
	leftLines := strings.Split(left, "\n")
	diffLines := strings.Split(diffView, "\n")

	leftWidth := 0
	for _, line := range leftLines {
		if w := ansi.StringWidth(line); w > leftWidth {
			leftWidth = w
		}
	}

	maxLines := max(len(leftLines), len(diffLines))
	if targetHeight > maxLines {
		maxLines = targetHeight
	}

	var b strings.Builder
	for i := 0; i < maxLines; i++ {
		var leftLine string
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		padding := leftWidth - ansi.StringWidth(leftLine)
		if padding > 0 {
			leftLine += strings.Repeat(" ", padding)
		}

		leftLineWidth := ansi.StringWidth(leftLine)
		diffTargetWidth := targetWidth - leftLineWidth
		if diffTargetWidth < 0 {
			diffTargetWidth = 0
		}

		var diffLine string
		if i < len(diffLines) {
			diffLine = diffLines[i]
		}

		diffLine = components.FitANSIString(diffLine, diffTargetWidth)

		b.WriteString(leftLine)
		b.WriteString(diffLine)

		if i < maxLines-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
