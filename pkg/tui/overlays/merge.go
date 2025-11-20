package overlays

import (
	"agate/pkg/git"
	"agate/pkg/tui/components"
	"agate/pkg/tui/panes"
	"agate/pkg/tui/theme"
	"agate/pkg/session"
	"agate/pkg/telemetry"
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

// Styles for merge dialog
var (
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.BorderActive)).
			Padding(1, 2)

	dialogTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				MarginBottom(1)
)

// MergeOverlay represents the merge dialog
type MergeOverlay struct {
	mergeInput   *components.LabeledInput
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
	keys         mergeKeyMap
}

// mergeKeyMap defines the keybindings for the merge overlay
type mergeKeyMap struct {
	Tab    key.Binding
	Escape key.Binding
}

// ShortHelp returns keybindings to show in the mini help view
func (k mergeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Escape}
}

// FullHelp returns keybindings to show in the full help view
func (k mergeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Escape},
	}
}

// MergeSuccessMsg is sent when a merge is successfully created
type MergeSuccessMsg struct {
	SHA        string // First 6 characters of commit SHA
	BranchName string // Name of the branch that was merged
}

// MergeErrorMsg is sent when a merge fails
type MergeErrorMsg struct {
	Err error
}

// FileDiscardedMsg is sent when a file is successfully discarded
type FileDiscardedMsg struct{}

// MergeMessageGeneratedMsg is sent when the merge message generation completes
type MergeMessageGeneratedMsg struct {
	Message string
}


// NewMergeOverlay creates a new merge overlay
func NewMergeOverlay(sess *session.Session) *MergeOverlay {
	// Track merge overlay opened
	telemetry.TrackMergeOverlayOpened()

	repoPath := ""
	if sess != nil && sess.Worktree() != nil {
		repoPath = sess.Worktree().Path
	}

	// Create labeled input for merge message
	mergeInput := components.NewLabeledInput("Commit message", "Summary (required)")
	mergeInput.Focus()

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
	keys := mergeKeyMap{
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "navigate fields"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}

	return &MergeOverlay{
		mergeInput:  mergeInput,
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

// SetSize sets the dimensions for the merge overlay
func (m *MergeOverlay) SetSize(width, height int) {
	m.width = width
	m.height = height
	// File list width will be set dynamically in View()

	// Set input width to use most of the available dialog width
	// Dialog max width is 100, minus borders and padding
	if m.mergeInput != nil {
		m.mergeInput.SetWidth(90)
	}
}

// Init initializes the commit overlay and starts AI generation if supported
func (m *MergeOverlay) Init() tea.Cmd {
	// Check if agent has fast headless support
	if m.session == nil {
		m.generating = false
		m.mergeInput = components.NewLabeledInput("Commit message", "Summary (required)")
		m.mergeInput.Focus()
		return nil
	}

	testCmd := m.session.Agent().HeadlessCommand("")

	if testCmd == nil {
		// No fast headless support - skip generation, recreate input with proper placeholder
		m.generating = false
		m.mergeInput = components.NewLabeledInput("Commit message", "Summary (required)")
		m.mergeInput.Focus()
		return nil
	}

	// Start generation with loading animation and 30 second timeout
	var cmds []tea.Cmd

	// Track start time for elapsed display
	now := time.Now()
	m.startTime = &now

	m.loader.SetLabel(fmt.Sprintf("%s is generating a commit message", m.session.Agent().CompanyName))
	cmds = append(cmds, m.loader.TickCmd())
	cmds = append(cmds, m.generateCommitMessage())

	// Tick every second to update elapsed time
	cmds = append(cmds, m.tickEverySecond())

	return tea.Batch(cmds...)
}

// Update handles messages for the commit overlay
func (m *MergeOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case MergeMessageGeneratedMsg:
		// Generation complete (or timed out) - set value and stop generating
		m.generating = false
		m.startTime = nil
		m.mergeInput.SetValue(msg.Message)
		m.mergeInput.Focus()
		return m, nil

	case FileDiscardedMsg:
		// Refresh file list after discard
		m.fileList.Refresh()
		return m, nil

	case MergeSuccessMsg:
		// Refresh file list after successful commit and send message to refresh git pane
		m.fileList.Refresh()
		return m, func() tea.Msg {
			return panes.GitRefreshMsg{}
		}

	case MergeErrorMsg:
		// Handle discard errors (and commit errors)
		// For now, just refresh to show current state
		m.fileList.Refresh()
		return m, nil

	case tea.KeyMsg:
		// Handle 'c' to cancel generation
		if m.generating && msg.String() == "c" {
			m.generating = false
			m.startTime = nil
			m.mergeInput = components.NewLabeledInput("Commit message", "Summary (required)")
			m.mergeInput.Focus()
			return m, nil
		}

		// Don't process other keyboard input if we're generating
		if m.generating {
			return m, nil
		}

		switch msg.String() {
		case "esc":
			// Close overlay - handled by returning tea.Quit-like message
			// Actually, let main model handle it by not consuming the message
			// But we need to pass it through - return without handling
			return m, nil

		case "tab":
			// Switch focus between input and file list
			if m.focusedField == 0 {
				m.focusedField = 1
				m.mergeInput.Blur()
				m.fileList.SetActive(true)
			} else {
				m.focusedField = 0
				m.mergeInput.Focus()
				m.fileList.SetActive(false)
			}
			return m, nil

		case "enter":
			// Commit! Only if valid (and only when input is focused)
			if m.focusedField == 0 && m.isValid() {
				return m, m.commit()
			}
			return m, nil

		default:
			// Delegate to focused component
			if m.focusedField == 0 {
				// Input focused - update text input
				cmd := m.mergeInput.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				// File list focused - handle navigation and actions manually
				switch msg.String() {
				case "up", "k":
					m.fileList.MoveUp()
				case "down", "j":
					m.fileList.MoveDown()
				case "d":
					// Discard selected file
					return m, m.discardFile()
				case "enter":
					// Open selected file in editor
					return m, m.openSelectedFile()
				}
			}
		}

	case time.Time:
		// Tick every second for elapsed time updates
		if m.generating {
			cmds = append(cmds, m.tickEverySecond())
		}
	}

	// Update loader if generating
	if m.generating && m.loader != nil {
		if cmd := m.loader.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// tickEverySecond returns a command that ticks every second while generating
func (m *MergeOverlay) tickEverySecond() tea.Cmd {
	if !m.generating {
		return nil
	}
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return t // Return the time to trigger re-render
	})
}

// discardFile discards changes to the selected file
func (m *MergeOverlay) discardFile() tea.Cmd {
	file := m.fileList.GetSelectedFile()
	if file == nil {
		return nil
	}

	filePath := file.FilePath

	return func() tea.Msg {
		err := git.DiscardFile(m.repoPath, filePath)
		if err == nil {
			return FileDiscardedMsg{}
		}
		return MergeErrorMsg{Err: err}
	}
}

// openSelectedFile opens the selected file in the user's editor
func (m *MergeOverlay) openSelectedFile() tea.Cmd {
	file := m.fileList.GetSelectedFile()
	if file == nil {
		return nil
	}

	// Build full file path (same as git pane)
	fullPath := filepath.Join(m.repoPath, file.DirPath, file.FileName)

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
func (m *MergeOverlay) generateCommitMessage() tea.Cmd {
	return func() tea.Msg {
		agentConfig := m.session.Agent()
		message, err := git.GenerateCommitMessage(&agentConfig, m.repoPath, m.executor)
		if err != nil {
			// On error, return empty message (will show empty input)
			return MergeMessageGeneratedMsg{Message: ""}
		}
		return MergeMessageGeneratedMsg{Message: message}
	}
}

// commit performs the commit and merge operation
func (m *MergeOverlay) commit() tea.Cmd {
	message := m.mergeInput.Value()

	return func() tea.Msg {
		// Step 1: Commit all changes in the worktree
		sha, err := git.CommitAll(m.repoPath, message)
		if err != nil {
			return MergeErrorMsg{Err: err}
		}

		// Step 2: Get the branch name for the success message
		var branchName string
		if m.session != nil && m.session.Worktree() != nil {
			branchName = m.session.Worktree().Branch

			// TODO: Implement actual merge logic to merge changes back to main worktree
			// This will require:
			// 1. Getting the main repository path from session manager
			// 2. Switching to main branch
			// 3. Merging the worktree branch
			// 4. Handling merge conflicts if any
		}

		// Track merge completed
		if m.session != nil {
			telemetry.TrackMergeCompleted(m.session.Agent().Name)
		}

		// Return first 6 characters of SHA
		shortSHA := sha
		if len(sha) > 6 {
			shortSHA = sha[:6]
		}

		return MergeSuccessMsg{SHA: shortSHA, BranchName: branchName}
	}
}

// View renders the commit overlay
func (m *MergeOverlay) View() string {
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
	if m.session != nil && m.session.Worktree() != nil {
		repoName = m.session.Worktree().RepoName
		branchName = m.session.Worktree().Branch
	}

	repoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	titleStyle := dialogTitleStyle.Copy()

	repoText := repoStyle.Render(repoName)
	arrow1Text := titleStyle.Render(" > ")
	branchText := repoStyle.Render(branchName)
	arrow2Text := titleStyle.Render(" > ")
	actionText := titleStyle.Render("Commit & merge changes")

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, repoText, arrow1Text, branchText, arrow2Text, actionText)
	appendLine(headerLine)

	// Horizontal divider - will be sized later
	content = append(content, "DIVIDER_PLACEHOLDER")
	content = append(content, "")

	if m.generating {
		// Only show loader when generating
		loaderView := m.loader.View()
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.session.Agent().BorderColor)).
			Bold(true)
		appendLine(loaderStyle.Render(loaderView))
		content = append(content, "")

		// Show elapsed time and quit option after 3 seconds
		if m.startTime != nil && time.Since(*m.startTime) >= 3*time.Second {
			stopwatch := components.FormatElapsedTime(*m.startTime, "c", "cancel")
			// Add left padding to align with loader label text (spinner + space = 2 chars)
			appendLine("  " + stopwatch)
		}
	} else {
		// Show normal commit UI
		// Commit message input
		commitInputLines := strings.Split(m.mergeInput.View(), "\n")
		for _, line := range commitInputLines {
			appendLine(line)
		}
		content = append(content, "")

		// File list
		fileStatus := m.fileList.GetFileStatus()
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
	if m.width > 0 {
		maxAllowedContentWidth = m.width - frameWidth
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
	m.fileList.SetSize(actualContentWidth)

	// Replace divider placeholder (full width to match file list with padding)
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	divider := dividerStyle.Render(strings.Repeat("─", actualContentWidth))

	// Render file list now that width is set
	fileListView := m.fileList.View()

	// Create commit button
	mergeButton := components.NewButton("Commit & merge changes", "↵", components.ButtonVariantAgent)
	mergeButton.SetWidth(actualContentWidth)
	// Use the agent color from the session
	if m.session != nil && m.session.Agent().BorderColor != "" {
		mergeButton.SetAgentColor(m.session.Agent().BorderColor)
	}
	mergeButton.SetDisabled(!m.isValid())
	button := mergeButton.Render()

	// Create help text using shortcut component - using default variant (bubbles style)
	helpContent := components.RenderShortcutsFromBindings([]key.Binding{m.keys.Tab, m.keys.Escape}, components.ShortcutDefault, "")
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
func (m *MergeOverlay) isValid() bool {
	message := strings.TrimSpace(m.mergeInput.Value())
	return len(message) > 0
}
