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
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	generating   bool
	loader       *components.LaunchAgentLoader
	executor     git.CommandExecutor
	startTime    *time.Time
	help         help.Model
	keys         commitKeyMap
}

// commitKeyMap defines the keybindings for the commit overlay
type commitKeyMap struct {
	Tab    key.Binding
	Escape key.Binding
}

// ShortHelp returns keybindings to show in the mini help view
func (k commitKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Escape}
}

// FullHelp returns keybindings to show in the full help view
func (k commitKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Escape},
	}
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

// CommitMessageGeneratedMsg is sent when the commit message generation completes
type CommitMessageGeneratedMsg struct {
	Message string
}


// NewCommitOverlay creates a new commit overlay
func NewCommitOverlay(sess *session.Session) *CommitOverlay {
	repoPath := ""
	if sess != nil && sess.Worktree() != nil {
		repoPath = sess.Worktree().Path
	}

	// Create labeled input for commit message
	commitInput := components.NewLabeledInput("Commit message", "Summary (required)")
	commitInput.Focus()

	// Create file list (same component as git pane!)
	fileList := components.NewGitFileList(repoPath, false) // Don't show summary line
	fileList.SetPadding(0)                                 // No padding for dialog - full width rows
	fileList.Refresh()                                     // Load files once at creation

	// Create loader
	loader := components.NewLaunchAgentLoader("")

	// Initialize help
	h := help.New()
	h.ShowAll = false // Only show short help

	// Initialize keybindings
	keys := commitKeyMap{
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "navigate fields"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}

	return &CommitOverlay{
		commitInput:  commitInput,
		fileList:     fileList,
		focusedField: 0, // Start with input focused
		repoPath:     repoPath,
		session:      sess,
		generating:   true,
		loader:       loader,
		executor:     &git.DefaultCommandExecutor{},
		help:         h,
		keys:         keys,
	}
}

// SetSize sets the dimensions for the commit overlay
func (c *CommitOverlay) SetSize(width, height int) {
	c.width = width
	c.height = height
	// File list width will be set dynamically in View()

	// Set input width to use most of the available dialog width
	// Dialog max width is 100, minus borders and padding
	if c.commitInput != nil {
		c.commitInput.SetWidth(90)
	}
}

// Init initializes the commit overlay and starts AI generation if supported
func (c *CommitOverlay) Init() tea.Cmd {
	// Check if agent has fast headless support
	if c.session == nil {
		c.generating = false
		c.commitInput = components.NewLabeledInput("Commit message", "Summary (required)")
		c.commitInput.Focus()
		return nil
	}

	testCmd := c.session.Agent().HeadlessCommand("")

	if testCmd == nil {
		// No fast headless support - skip generation, recreate input with proper placeholder
		c.generating = false
		c.commitInput = components.NewLabeledInput("Commit message", "Summary (required)")
		c.commitInput.Focus()
		return nil
	}

	// Start generation with loading animation and 30 second timeout
	var cmds []tea.Cmd

	// Track start time for elapsed display
	now := time.Now()
	c.startTime = &now

	c.loader.SetLabel(fmt.Sprintf("%s is generating a commit message", c.session.Agent().CompanyName))
	cmds = append(cmds, c.loader.TickCmd())
	cmds = append(cmds, c.generateCommitMessage())

	// Tick every second to update elapsed time
	cmds = append(cmds, c.tickEverySecond())

	return tea.Batch(cmds...)
}

// Update handles messages for the commit overlay
func (c *CommitOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case CommitMessageGeneratedMsg:
		// Generation complete (or timed out) - set value and stop generating
		c.generating = false
		c.startTime = nil
		c.commitInput.SetValue(msg.Message)
		c.commitInput.Focus()
		return c, nil

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
		// Handle 'c' to cancel generation
		if c.generating && msg.String() == "c" {
			c.generating = false
			c.startTime = nil
			c.commitInput = components.NewLabeledInput("Commit message", "Summary (required)")
			c.commitInput.Focus()
			return c, nil
		}

		// Don't process other keyboard input if we're generating
		if c.generating {
			return c, nil
		}

		switch msg.String() {
		case "esc":
			// Close overlay - handled by returning tea.Quit-like message
			// Actually, let main model handle it by not consuming the message
			// But we need to pass it through - return without handling
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
				cmd := c.commitInput.Update(msg)
				cmds = append(cmds, cmd)
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
			}
		}

	case time.Time:
		// Tick every second for elapsed time updates
		if c.generating {
			cmds = append(cmds, c.tickEverySecond())
		}
	}

	// Update loader if generating
	if c.generating && c.loader != nil {
		if cmd := c.loader.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return c, tea.Batch(cmds...)
}

// tickEverySecond returns a command that ticks every second while generating
func (c *CommitOverlay) tickEverySecond() tea.Cmd {
	if !c.generating {
		return nil
	}
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return t // Return the time to trigger re-render
	})
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

// generateCommitMessage starts the commit message generation
func (c *CommitOverlay) generateCommitMessage() tea.Cmd {
	return func() tea.Msg {
		agentConfig := c.session.Agent()
		message, err := git.GenerateCommitMessage(&agentConfig, c.repoPath, c.executor)
		if err != nil {
			// On error, return empty message (will show empty input)
			return CommitMessageGeneratedMsg{Message: ""}
		}
		return CommitMessageGeneratedMsg{Message: message}
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
	if c.session != nil && c.session.Worktree() != nil {
		repoName = c.session.Worktree().RepoName
		branchName = c.session.Worktree().Branch
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

	if c.generating {
		// Only show loader when generating
		loaderView := c.loader.View()
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(c.session.Agent().BorderColor)).
			Bold(true)
		appendLine(loaderStyle.Render(loaderView))
		content = append(content, "")

		// Show elapsed time and quit option after 3 seconds
		if c.startTime != nil && time.Since(*c.startTime) >= 3*time.Second {
			stopwatch := components.FormatElapsedTime(*c.startTime, "c", "cancel")
			// Add left padding to align with loader label text (spinner + space = 2 chars)
			appendLine("  " + stopwatch)
		}
	} else {
		// Show normal commit UI
		// Commit message input
		commitInputLines := strings.Split(c.commitInput.View(), "\n")
		for _, line := range commitInputLines {
			appendLine(line)
		}
		content = append(content, "")

		// File list
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
		content = append(content, "FILELIST_PLACEHOLDER")
		content = append(content, "")
		content = append(content, "")

		// Button and help
		content = append(content, "BUTTON_PLACEHOLDER")
		content = append(content, "")
		content = append(content, "HELP_PLACEHOLDER")
	}

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
	if c.session != nil && c.session.Agent().BorderColor != "" {
		commitButton.SetAgentColor(c.session.Agent().BorderColor)
	}
	commitButton.SetDisabled(!c.isValid())
	button := commitButton.Render()

	// Create help text using shortcut component - using default variant (bubbles style)
	helpContent := components.RenderShortcutsFromBindings([]key.Binding{c.keys.Tab, c.keys.Escape}, components.ShortcutDefault, "")
	helpStyle := lipgloss.NewStyle().
		Width(actualContentWidth).
		Align(lipgloss.Center)
	helpText := helpStyle.Render(helpContent)

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
