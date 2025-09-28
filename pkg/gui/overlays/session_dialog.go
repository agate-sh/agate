package overlays

import (
	"agate/pkg/app"
	"fmt"
	"strings"
	"time"

	"agate/pkg/git"
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SessionDialog represents the dialog for creating new agent sessions
type SessionDialog struct {
	branchInput     textinput.Model
	agentInput      textinput.Model
	focusedField    int // 0 = branch, 1 = agent
	err             string
	repoName        string
	worktreeManager *git.WorktreeManager
	systemCaps      git.SystemCapabilities
	width           int
	height          int
	creating        bool
	initializing    bool
	selectedAgent   app.AgentConfig
	defaultAgent    string
	loader          *components.LaunchAgentLoader
	help            help.Model
	keys            sessionKeyMap
}

// sessionKeyMap defines the keybindings for the session dialog
type sessionKeyMap struct {
	Tab    key.Binding
	Escape key.Binding
}

// ShortHelp returns keybindings to show in the mini help view
func (k sessionKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Escape}
}

// FullHelp returns keybindings to show in the full help view
func (k sessionKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Escape},
	}
}

const worktreeDialogMinContentWidth = 60

// Styling for worktree dialog
var (
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.TextDescription)).
			Padding(1, 2)

	dialogTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				MarginBottom(1)

	dialogErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ErrorStatus)).
				MarginTop(1)

	dialogWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.WarningStatus)).
				MarginTop(1)

	dialogInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)). // Gray
			MarginTop(1)

	dialogButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.InfoStatus)).
				MarginTop(1)

	// Note: primaryDialogButtonStyle will use agent color dynamically
	primaryDialogButtonStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FFFFFF")).
					Padding(0, 2).
					Bold(true)
)

// NewSessionDialog creates a new agent creation dialog
func NewSessionDialog(worktreeManager *git.WorktreeManager, defaultAgent string) *SessionDialog {
	// Branch input - show random name as placeholder
	branchInput := textinput.New()
	branchInput.Placeholder = git.GenerateRandomBranchName()
	branchInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	branchInput.Focus()
	branchInput.CharLimit = 100
	branchInput.Width = 40
	branchInput.Prompt = ""

	// Agent input (normal text input, no autocomplete)
	agentInput := textinput.New()
	agentInput.Placeholder = "claude, codex, etc"
	agentInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	agentInput.CharLimit = 50
	agentInput.Width = 40
	agentInput.Prompt = ""

	// Set default value
	if defaultAgent != "" {
		agentInput.SetValue(defaultAgent)
	}

	var repoName string
	var systemCaps git.SystemCapabilities

	if worktreeManager != nil {
		repoName = worktreeManager.GetRepositoryName()
		systemCaps = worktreeManager.GetSystemCapabilities()
	}

	loader := components.NewLaunchAgentLoader("")

	// Get initial selected agent
	selectedAgent := app.GetAgentConfig(defaultAgent)

	// Initialize help
	h := help.New()
	h.ShowAll = false // Only show short help

	// Initialize keybindings
	keys := sessionKeyMap{
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "navigate fields"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}

	return &SessionDialog{
		branchInput:     branchInput,
		agentInput:      agentInput,
		focusedField:    0, // Start with branch focused
		repoName:        repoName,
		worktreeManager: worktreeManager,
		systemCaps:      systemCaps,
		creating:        false,
		initializing:    false,
		selectedAgent:   selectedAgent,
		defaultAgent:    defaultAgent,
		loader:          loader,
		help:            h,
		keys:            keys,
	}
}

// Init implements tea.Model
func (d *SessionDialog) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (d *SessionDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't process keys if we're in the middle of creating or initializing
		if d.creating || d.initializing {
			return d, nil
		}

		switch msg.String() {
		case "enter":
			// Only create if both fields are valid
			if d.isValid() {
				return d, d.createAndAttachWorktree()
			}
			return d, nil

		case "tab":
			// Switch to next field
			d.focusedField = (d.focusedField + 1) % 2
			d.updateFocus()
			return d, nil

		case "shift+tab":
			// Switch to previous field
			d.focusedField--
			if d.focusedField < 0 {
				d.focusedField = 1
			}
			d.updateFocus()
			return d, nil

		case "esc":
			// Cancel dialog
			return d, func() tea.Msg {
				return SessionDialogCancelledMsg{}
			}
		}

	case WorktreeCreatedMsg:
		// Worktree creation completed successfully, now start initializing
		d.creating = false
		d.initializing = true
		d.err = ""
		if d.loader != nil {
			d.loader.SetLabel(fmt.Sprintf("%s is starting...", d.selectedAgent.CompanyName))
		}

		var caseCmds []tea.Cmd
		if d.loader != nil {
			if cmd := d.loader.TickCmd(); cmd != nil {
				caseCmds = append(caseCmds, cmd)
			}
		}
		caseCmds = append(caseCmds, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return WorktreeInitializationCompleteMsg{Worktree: msg.Worktree}
		}))

		return d, tea.Batch(caseCmds...)

	case WorktreeCreationErrorMsg:
		// Worktree creation failed
		d.creating = false
		d.err = msg.Error
		return d, nil

	case WorktreeInitializationCompleteMsg:
		// Initialization complete - forward message to main app
		return d, func() tea.Msg {
			return msg
		}
	}

	// Update text inputs if not in creating/initializing state
	if !d.creating && !d.initializing {
		// Update the focused input
		var inputCmd tea.Cmd
		if d.focusedField == 0 {
			d.branchInput, inputCmd = d.branchInput.Update(msg)
		} else {
			d.agentInput, inputCmd = d.agentInput.Update(msg)
			// Update selected agent when agent input changes
			if app.IsValidAgent(d.agentInput.Value()) {
				d.selectedAgent = app.GetAgentConfig(d.agentInput.Value())
			}
		}
		cmds = append(cmds, inputCmd)
	}

	if d.initializing && d.loader != nil {
		if cmd := d.loader.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return d, tea.Batch(cmds...)
}

// createAndAttachWorktree creates a new agent and attaches to it
func (d *SessionDialog) createAndAttachWorktree() tea.Cmd {
	if d.worktreeManager == nil {
		return func() tea.Msg {
			return WorktreeCreationErrorMsg{Error: "Worktree manager not available"}
		}
	}

	// Get branch name from input or generate random name
	branchName := strings.TrimSpace(d.branchInput.Value())
	if branchName == "" {
		branchName = git.GenerateRandomBranchName()
	}

	// Validate branch name
	if err := git.ValidateBranchName(branchName); err != nil {
		return func() tea.Msg {
			return WorktreeCreationErrorMsg{Error: err.Error()}
		}
	}

	// Set creating state and prepare for agent loader
	d.creating = true
	d.initializing = false
	d.err = ""
	if d.loader != nil {
		d.loader.SetLabel(fmt.Sprintf("%s is starting...", d.selectedAgent.CompanyName))
	}

	// Create worktree in background and start loader ticking
	var cmds []tea.Cmd

	// Add the worktree creation command
	cmds = append(cmds, func() tea.Msg {
		worktree, err := d.worktreeManager.CreateWorktree(branchName)
		if err != nil {
			return WorktreeCreationErrorMsg{Error: err.Error()}
		}
		return WorktreeCreatedMsg{
			Worktree:  worktree,
			AgentName: d.agentInput.Value(), // Pass the selected agent command
		}
	})

	// Start the loader spinner animation
	if d.loader != nil {
		if tickCmd := d.loader.TickCmd(); tickCmd != nil {
			cmds = append(cmds, tickCmd)
		}
	}

	return tea.Batch(cmds...)
}

// SetSize updates the dialog dimensions
func (d *SessionDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View implements tea.Model and renders the dialog
func (d *SessionDialog) View() string {
	var content []string
	maxContentWidth := 0

	appendLine := func(line string) {
		content = append(content, line)
		if w := lipgloss.Width(line); w > maxContentWidth {
			maxContentWidth = w
		}
	}

	// Header: Repository name > new agent
	repoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	titleStyle := dialogTitleStyle.Copy()

	repoText := repoStyle.Render(d.repoName)
	arrowText := titleStyle.Render(" > ")
	sessionText := titleStyle.Render("New agent")

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, repoText, arrowText, sessionText)
	appendLine(headerLine)

	// Horizontal divider - will be sized later after we know content width
	content = append(content, "DIVIDER_PLACEHOLDER")
	content = append(content, "")

	// Show form or loading state
	if d.creating || d.initializing {
		// Loading state
		loadingTitle := fmt.Sprintf("%s is starting...", d.selectedAgent.CompanyName)
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.selectedAgent.BorderColor)).
			Bold(true)

		if d.loader != nil {
			d.loader.SetLabel(loadingTitle)
			appendLine(loaderStyle.Render(d.loader.View()))
		} else {
			appendLine(loaderStyle.Render(loadingTitle))
		}
	} else {
		// Form state - credit card style
		labelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

		// Branch name field
		appendLine(labelStyle.Render("Branch name"))
		appendLine(d.branchInput.View())
		content = append(content, "")

		// Agent command field
		appendLine(labelStyle.Render("Agent command"))
		appendLine(d.agentInput.View())
		content = append(content, "")

		// Add button placeholder - will be replaced later with proper width
		content = append(content, "BUTTON_PLACEHOLDER")

		// Add error message directly under button (centered)
		if d.err != "" {
			content = append(content, "ERROR_PLACEHOLDER")
		} else {
			content = append(content, "")
		}

		// Add help text (will be centered later with actual width)
		content = append(content, "HELP_PLACEHOLDER")

		// Warning for non-COW systems
		if !d.systemCaps.SupportsCOW {
			content = append(content, "")
			appendLine(dialogWarningStyle.Render("⚠️  Only version controlled files"))
			appendLine(dialogWarningStyle.Render("   will be copied, which excludes"))
			appendLine(dialogWarningStyle.Render("   things like your dependencies"))
			appendLine(dialogWarningStyle.Render("   and .env files. This is because"))
			appendLine(dialogWarningStyle.Render("   your OS does not support"))
			appendLine(dialogWarningStyle.Render("   copy-on-write."))
		}
	}

	frameWidth := dialogStyle.GetHorizontalFrameSize()
	maxAllowedContentWidth := 0
	if d.width > 0 {
		maxAllowedContentWidth = d.width - frameWidth
		if maxAllowedContentWidth < 0 {
			maxAllowedContentWidth = 0
		}
	}

	minContentWidth := worktreeDialogMinContentWidth
	if maxAllowedContentWidth > 0 && maxAllowedContentWidth < minContentWidth {
		minContentWidth = maxAllowedContentWidth
	}

	// Ensure we have at least minimum width
	if maxContentWidth < minContentWidth {
		maxContentWidth = minContentWidth
	}

	// Calculate actual content width inside dialog (accounting for padding)
	actualContentWidth := maxContentWidth
	if actualContentWidth < minContentWidth {
		actualContentWidth = minContentWidth
	}
	if maxAllowedContentWidth > 0 && actualContentWidth > maxAllowedContentWidth {
		actualContentWidth = maxAllowedContentWidth
	}

	// Account for dialog padding (2 horizontal padding on each side = 4 total)
	if actualContentWidth > 4 {
		actualContentWidth -= 4
	}

	// Replace divider placeholder with actual divider
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	divider := dividerStyle.Render(strings.Repeat("─", actualContentWidth))

	// Create button - disabled or enabled based on validation
	var button string
	if d.isValid() {
		// Enabled button with agent color
		buttonStyle := primaryDialogButtonStyle.Copy().
			Background(lipgloss.Color(d.selectedAgent.BorderColor)).
			Width(actualContentWidth).
			Align(lipgloss.Center)
		button = buttonStyle.Render("Create and attach (↵)")
	} else {
		// Disabled button
		disabledButtonStyle := primaryDialogButtonStyle.Copy().
			Background(lipgloss.Color(theme.TextDescription)).
			Foreground(lipgloss.Color(theme.TextMuted)).
			Width(actualContentWidth).
			Align(lipgloss.Center)
		button = disabledButtonStyle.Render("Create and attach (↵)")
	}

	// Create centered error message
	errorMsg := ""
	if d.err != "" {
		errorStyle := dialogErrorStyle.Copy().
			Width(actualContentWidth).
			Align(lipgloss.Center)
		errorMsg = errorStyle.Render("Error: " + d.err)
	}

	// Create centered help text
	helpText := ""
	if !d.creating && !d.initializing {
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)).
			Width(actualContentWidth).
			Align(lipgloss.Center)
		helpText = helpStyle.Render(d.help.View(d.keys))
	}

	// Replace placeholders with actual content
	for i, line := range content {
		if line == "DIVIDER_PLACEHOLDER" {
			content[i] = divider
		} else if line == "BUTTON_PLACEHOLDER" {
			content[i] = button
		} else if line == "ERROR_PLACEHOLDER" {
			content[i] = errorMsg
		} else if line == "HELP_PLACEHOLDER" {
			content[i] = helpText
		}
	}

	// Join all content lines
	dialogContent := strings.Join(content, "\n")

	// Set the dialog width
	style := dialogStyle.Copy()
	if maxContentWidth > 0 {
		style = style.Width(maxContentWidth)
	}
	return style.Render(dialogContent)
}

// WorktreeCreatedMsg indicates a worktree was successfully created
type WorktreeCreatedMsg struct {
	Worktree  *git.WorktreeInfo
	AgentName string // The agent command selected by the user
}

// WorktreeCreationErrorMsg indicates worktree creation failed
type WorktreeCreationErrorMsg struct {
	Error string
}

// SessionDialogCancelledMsg indicates the dialog was cancelled
type SessionDialogCancelledMsg struct{}

// WorktreeInitializationCompleteMsg indicates the worktree and session are ready
type WorktreeInitializationCompleteMsg struct {
	Worktree *git.WorktreeInfo
}

// isValid checks if the agent command is valid (branch name is optional)
func (d *SessionDialog) isValid() bool {
	agentCommand := strings.TrimSpace(d.agentInput.Value())
	return app.IsValidAgent(agentCommand)
}

// updateFocus updates which input field is focused
func (d *SessionDialog) updateFocus() {
	d.branchInput.Blur()
	d.agentInput.Blur()

	if d.focusedField == 0 {
		d.branchInput.Focus()
	} else {
		d.agentInput.Focus()
	}
}
