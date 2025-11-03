package panes

import (
	"agate/pkg/common"
	"agate/pkg/gui/components"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"agate/pkg/git"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// GitPane manages the display of Git file status information
type GitPane struct {
	*components.BasePane // Embedded BasePane for common functionality
	fileList             *components.GitFileList
	repoPath             string
}

// gitRefreshMsg is sent when the git pane needs to refresh
// GitRefreshMsg triggers a refresh of the git file status
type GitRefreshMsg struct{}

// NewGitPane creates a new GitPane instance
func NewGitPane() *GitPane {
	fileList := components.NewGitFileList("", true) // Show summary
	return &GitPane{
		BasePane: components.NewBasePane(2, "Changes"), // Pane index 2
		fileList: fileList,
	}
}

// SetSize updates the dimensions of the Git pane
func (g *GitPane) SetSize(width, height int) {
	g.BasePane.SetSize(width, height)
	if g.fileList != nil {
		g.fileList.SetSize(g.GetWidth())
		g.fileList.SetHeight(g.GetHeight())
	}
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
	if g.fileList != nil {
		g.fileList.SetActive(active)
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
	if !g.IsActive() || g.fileList == nil {
		return false, nil
	}

	switch key {
	case "up", "k":
		g.fileList.MoveUp()
		return true, nil
	case "down", "j":
		g.fileList.MoveDown()
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

// View renders the Git pane content
func (g *GitPane) View() string {
	if g.fileList == nil {
		return ""
	}
	return g.fileList.View()
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
