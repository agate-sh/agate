package overlays

import (
	"agate/pkg/git"
	"agate/pkg/gui/components"
	"agate/pkg/gui/panes"
	"agate/pkg/gui/theme"
	"agate/pkg/session"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommitOverlay represents the commit dialog
type CommitOverlay struct {
	commitInput  *components.LabeledInput
	fileList     *components.GitFileList
	focusedField int // 0 = input, 1 = list
	repoPath     string
	session      *session.Session
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

// FileDiscardedMsg is sent when a file is successfully discarded
type FileDiscardedMsg struct{}


// NewCommitOverlay creates a new commit overlay
func NewCommitOverlay(sess *session.Session) *CommitOverlay {
	repoPath := ""
	if sess != nil && sess.Worktree != nil {
		repoPath = sess.Worktree.Path
	}

	// Create labeled input for commit message
	commitInput := components.NewLabeledInput("Commit message", "Summary (required)")
	commitInput.Focus()

	// Create file list (same component as git pane!)
	fileList := components.NewGitFileList(repoPath, false) // Don't show summary line
	fileList.SetPadding(0)                                 // No padding for dialog - full width rows
	fileList.Refresh()                                     // Load files once at creation

	return &CommitOverlay{
		commitInput:  commitInput,
		fileList:     fileList,
		focusedField: 0, // Start with input focused
		repoPath:     repoPath,
		session:      sess,
	}
}

// SetSize sets the dimensions for the commit overlay
func (c *CommitOverlay) SetSize(width, height int) {
	c.width = width
	c.height = height
	// File list width will be set dynamically in View()
}

// Init initializes the commit overlay (required for tea.Model)
func (c *CommitOverlay) Init() tea.Cmd {
	return nil
}

// Update handles messages for the commit overlay
func (c *CommitOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case FileDiscardedMsg:
		// Refresh file list after discard
		c.fileList.Refresh()
		return c, nil

	case CommitSuccessMsg:
		// Refresh file list after successful commit and send message to refresh git pane
		c.fileList.Refresh()
		return c, func() tea.Msg {
			return panes.GitRefreshMsg{}
		}

	case CommitErrorMsg:
		// Handle discard errors (and commit errors)
		// For now, just refresh to show current state
		c.fileList.Refresh()
		return c, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel - handled by main model
			return c, nil

		case "tab":
			// Switch focus between input and file list
			if c.focusedField == 0 {
				c.focusedField = 1
				c.commitInput.Blur()
				c.fileList.SetActive(true)
			} else {
				c.focusedField = 0
				c.commitInput.Focus()
				c.fileList.SetActive(false)
			}
			return c, nil

		case "enter":
			// Commit! Only if valid (and only when input is focused)
			if c.focusedField == 0 && c.isValid() {
				return c, c.commit()
			}
			return c, nil

		default:
			// Delegate to focused component
			if c.focusedField == 0 {
				// Input focused - update text input
				cmd = c.commitInput.Update(msg)
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

	filePath := file.FilePath

	return func() tea.Msg {
		err := git.DiscardFile(c.repoPath, filePath)
		if err == nil {
			return FileDiscardedMsg{}
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
	message := c.commitInput.Value()

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
	// Don't refresh here - it resets selection! Refresh when overlay is created instead

	var content []string
	maxContentWidth := 0

	appendLine := func(line string) {
		content = append(content, line)
		if w := lipgloss.Width(line); w > maxContentWidth {
			maxContentWidth = w
		}
	}

	// Header: Repo > Branch > Commit changes (same style as session dialog)
	repoName := "unknown"
	branchName := "unknown"
	if c.session != nil && c.session.Worktree != nil {
		repoName = c.session.Worktree.RepoName
		branchName = c.session.Worktree.Branch
	}

	repoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	titleStyle := dialogTitleStyle.Copy()

	repoText := repoStyle.Render(repoName)
	arrow1Text := titleStyle.Render(" > ")
	branchText := repoStyle.Render(branchName)
	arrow2Text := titleStyle.Render(" > ")
	actionText := titleStyle.Render("Commit changes")

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, repoText, arrow1Text, branchText, arrow2Text, actionText)
	appendLine(headerLine)

	// Horizontal divider - will be sized later
	content = append(content, "DIVIDER_PLACEHOLDER")
	content = append(content, "")

	// Render commit input (includes label)
	commitInputLines := strings.Split(c.commitInput.View(), "\n")
	for _, line := range commitInputLines {
		appendLine(line)
	}
	content = append(content, "")

	// File list header - bold white like other labels
	fileStatus := c.fileList.GetFileStatus()
	fileCount := 0
	if fileStatus != nil {
		fileCount = fileStatus.TotalFiles
	}
	fileListLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Render(fmt.Sprintf("Files to commit (%d)", fileCount))
	appendLine(fileListLabel)
	// File list placeholder - will be rendered after width calculation
	content = append(content, "FILELIST_PLACEHOLDER")
	content = append(content, "")

	// Button placeholder
	content = append(content, "BUTTON_PLACEHOLDER")
	content = append(content, "")

	// Help text placeholder
	content = append(content, "HELP_PLACEHOLDER")

	// Calculate widths
	frameWidth := dialogStyle.GetHorizontalFrameSize()
	maxAllowedContentWidth := 0
	if c.width > 0 {
		maxAllowedContentWidth = c.width - frameWidth
		if maxAllowedContentWidth < 0 {
			maxAllowedContentWidth = 0
		}
	}

	minContentWidth := 60
	if maxAllowedContentWidth > 0 && maxAllowedContentWidth < minContentWidth {
		minContentWidth = maxAllowedContentWidth
	}

	if maxContentWidth < minContentWidth {
		maxContentWidth = minContentWidth
	}

	actualContentWidth := maxContentWidth
	if actualContentWidth < minContentWidth {
		actualContentWidth = minContentWidth
	}
	if maxAllowedContentWidth > 0 && actualContentWidth > maxAllowedContentWidth {
		actualContentWidth = maxAllowedContentWidth
	}

	// Account for dialog padding
	if actualContentWidth > 4 {
		actualContentWidth -= 4
	}

	// Set file list width to match dialog content width
	// File list has padding=0 for dialogs, so it renders to full actualContentWidth
	c.fileList.SetSize(actualContentWidth)

	// Replace divider placeholder (full width to match file list with padding)
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	divider := dividerStyle.Render(strings.Repeat("─", actualContentWidth))

	// Render file list now that width is set
	fileListView := c.fileList.View()

	// Create commit button
	commitButton := components.NewButton("Commit", "↵", components.ButtonVariantAgent)
	commitButton.SetWidth(actualContentWidth)
	// Use the agent color from the session
	if c.session != nil && c.session.Agent.BorderColor != "" {
		commitButton.SetAgentColor(c.session.Agent.BorderColor)
	}
	commitButton.SetDisabled(!c.isValid())
	button := commitButton.Render()

	// Create help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Width(actualContentWidth).
		Align(lipgloss.Center)
	helpText := helpStyle.Render("tab navigate fields • esc cancel")

	// Replace placeholders
	for i, line := range content {
		if line == "DIVIDER_PLACEHOLDER" {
			content[i] = divider
		} else if line == "FILELIST_PLACEHOLDER" {
			content[i] = fileListView
		} else if line == "BUTTON_PLACEHOLDER" {
			content[i] = button
		} else if line == "HELP_PLACEHOLDER" {
			content[i] = helpText
		}
	}

	// Join and apply dialog style
	dialog := dialogStyle.Render(strings.Join(content, "\n"))

	return dialog
}

// isValid checks if the commit message is valid (at least 1 character)
func (c *CommitOverlay) isValid() bool {
	message := strings.TrimSpace(c.commitInput.Value())
	return len(message) > 0
}
