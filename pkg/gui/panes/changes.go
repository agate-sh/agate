package panes

import (
	"agate/pkg/common"
	"agate/pkg/git"
	"agate/pkg/gui/components"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ChangesPane displays file changes from the active agent's worktree
type ChangesPane struct {
	*components.BasePane
	fileList         *components.GitFileList
	repoPath         string
	lastSelectedFile string // Track last selected file
}

// ChangesRefreshMsg triggers a refresh of the changes pane
type ChangesRefreshMsg struct{}

// NewChangesPane creates a new ChangesPane instance
func NewChangesPane() *ChangesPane {
	fileList := components.NewGitFileList("", false) // Don't show summary
	return &ChangesPane{
		BasePane: components.NewBasePane(2, "Changes"), // Pane index 2
		fileList: fileList,
	}
}

// SetRepository sets the active worktree path to display changes from
func (p *ChangesPane) SetRepository(repoPath string) {
	if repoPath != p.repoPath {
		p.repoPath = repoPath
		if p.fileList != nil {
			p.fileList = components.NewGitFileList(repoPath, true)
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
	if p.fileList != nil {
		p.fileList.SetSize(width)
		p.fileList.SetHeight(height)
	}
}

// SetActive sets whether this pane is currently focused
func (p *ChangesPane) SetActive(active bool) {
	p.BasePane.SetActive(active)
	if p.fileList != nil {
		p.fileList.SetActive(active)
	}
}

// GetSelectedFile returns the currently selected file, or nil if none
func (p *ChangesPane) GetSelectedFile() *git.FileStatus {
	if p.fileList == nil {
		return nil
	}
	return p.fileList.GetSelectedFile()
}

// GetRepoPath returns the current repository path
func (p *ChangesPane) GetRepoPath() string {
	return p.repoPath
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

// View renders the changes pane content (file list)
func (p *ChangesPane) View() string {
	if p.fileList == nil {
		return ""
	}
	return p.fileList.View()
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

// discardFile discards changes to the selected file
func (p *ChangesPane) discardFile() tea.Cmd {
	file := p.GetSelectedFile()
	if file == nil {
		return nil
	}

	return func() tea.Msg {
		err := git.DiscardFile(p.repoPath, file.FilePath)
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
	fullPath := filepath.Join(p.repoPath, file.DirPath, file.FileName)

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
		shortcuts = "↵ open • d discard"
	} else {
		shortcuts = "(2)"
	}

	label := "Changes"
	changeCount := p.changeCount()
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

// GetPaneSpecificKeybindings returns changes pane specific keybindings
func (p *ChangesPane) GetPaneSpecificKeybindings() []key.Binding {
	return []key.Binding{common.GlobalKeys.OpenInEditor}
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
