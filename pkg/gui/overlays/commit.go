package overlays

import (
	"agate/pkg/git"
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommitOverlay represents the commit dialog
type CommitOverlay struct {
	textInput    textinput.Model
	fileList     *components.GitFileList
	focusedField int // 0 = input, 1 = list
	repoPath     string
	width        int
	height       int
}

// CommitSuccessMsg is sent when a commit is successfully created
type CommitSuccessMsg struct {
	SHA string // First 6 characters of commit SHA
}

// CommitErrorMsg is sent when a commit fails
type CommitErrorMsg struct {
	Err error
}

// NewCommitOverlay creates a new commit overlay
func NewCommitOverlay(repoPath string) *CommitOverlay {
	// Create text input for commit message
	ti := textinput.New()
	ti.Placeholder = "Enter commit message"
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	// Create file list (same component as git pane!)
	fileList := components.NewGitFileList(repoPath, false) // Don't show summary line

	return &CommitOverlay{
		textInput:    ti,
		fileList:     fileList,
		focusedField: 0, // Start with input focused
		repoPath:     repoPath,
	}
}

// SetSize sets the dimensions for the commit overlay
func (c *CommitOverlay) SetSize(width, height int) {
	c.width = width
	c.height = height

	// Set file list width (will be constrained by dialog width)
	c.fileList.SetSize(50)
}

// Init initializes the commit overlay (required for tea.Model)
func (c *CommitOverlay) Init() tea.Cmd {
	return nil
}

// Update handles messages for the commit overlay
func (c *CommitOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel - handled by main model
			return c, nil

		case "tab":
			// Switch focus between input and file list
			if c.focusedField == 0 {
				c.focusedField = 1
				c.textInput.Blur()
				c.fileList.SetActive(true)
			} else {
				c.focusedField = 0
				c.textInput.Focus()
				c.fileList.SetActive(false)
			}
			return c, nil

		case "c":
			// Commit!
			return c, c.commit()

		default:
			// Delegate to focused component
			if c.focusedField == 0 {
				// Input focused - update text input
				c.textInput, cmd = c.textInput.Update(msg)
				return c, cmd
			} else {
				// File list focused - handle navigation and actions manually
				switch msg.String() {
				case "up", "k":
					c.fileList.MoveUp()
				case "down", "j":
					c.fileList.MoveDown()
				case "d":
					// Discard selected file
					return c, c.discardFile()
				case "enter":
					// Open selected file in editor
					return c, c.openSelectedFile()
				}
				return c, nil
			}
		}
	}

	return c, nil
}

// discardFile discards changes to the selected file
func (c *CommitOverlay) discardFile() tea.Cmd {
	file := c.fileList.GetSelectedFile()
	if file == nil {
		return nil
	}

	return func() tea.Msg {
		err := git.DiscardFile(c.repoPath, file.FilePath)
		if err == nil {
			// Refresh file list after discard
			return nil
		}
		return CommitErrorMsg{Err: err}
	}
}

// openSelectedFile opens the selected file in the user's editor
func (c *CommitOverlay) openSelectedFile() tea.Cmd {
	file := c.fileList.GetSelectedFile()
	if file == nil {
		return nil
	}

	// Build full file path (same as git pane)
	fullPath := filepath.Join(c.repoPath, file.DirPath, file.FileName)

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

// commit performs the commit operation
func (c *CommitOverlay) commit() tea.Cmd {
	message := c.textInput.Value()

	return func() tea.Msg {
		sha, err := git.CommitAll(c.repoPath, message)
		if err != nil {
			return CommitErrorMsg{Err: err}
		}

		// Return first 6 characters of SHA
		shortSHA := sha
		if len(sha) > 6 {
			shortSHA = sha[:6]
		}

		return CommitSuccessMsg{SHA: shortSHA}
	}
}

// View renders the commit overlay
func (c *CommitOverlay) View() string {
	// Refresh file list
	c.fileList.Refresh()

	// Title
	title := dialogTitleStyle.Render("Commit Changes")

	// Input label and field
	inputLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Render("Message:")

	inputField := c.textInput.View()
	inputLine := lipgloss.JoinHorizontal(lipgloss.Left, inputLabel, " ", inputField)

	// File list header
	fileStatus := c.fileList.GetFileStatus()
	fileCount := 0
	if fileStatus != nil {
		fileCount = fileStatus.TotalFiles
	}
	filesHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		MarginTop(1).
		Render(fmt.Sprintf("Files to commit (%d):", fileCount))

	// File list content
	filesContent := c.fileList.View()

	// Hints at bottom
	var hints string
	if c.focusedField == 0 {
		hints = "c commit • tab switch • esc cancel"
	} else {
		hints = "c commit • tab switch • esc cancel"
	}
	hintsLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		MarginTop(1).
		Render(hints)

	// Combine all parts
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		inputLine,
		filesHeader,
		filesContent,
		hintsLine,
	)

	// Apply dialog style (border, padding)
	dialog := dialogStyle.Render(content)

	return dialog
}
