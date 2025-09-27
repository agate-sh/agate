package overlays

import (
	"agate/pkg/app"
	"fmt"
	"strings"
	"time"

	"agate/pkg/git"
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WorktreeDialog represents the dialog for creating new worktrees
type WorktreeDialog struct {
	input           textinput.Model
	err             string
	repoName        string
	worktreeManager *git.WorktreeManager
	systemCaps      git.SystemCapabilities
	width           int
	height          int
	creating        bool
	initializing    bool
	agentConfig     app.AgentConfig
	loader          *components.LaunchAgentLoader
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

	secondaryDialogButtonStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color(theme.TextMuted)).
					Padding(0, 1)

	dialogHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted))
)

// NewWorktreeDialog creates a new worktree creation dialog
func NewWorktreeDialog(worktreeManager *git.WorktreeManager, agentConfig app.AgentConfig) *WorktreeDialog {
	input := textinput.New()
	input.Placeholder = "  Branch name, ↵ for random name"
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	input.Focus()
	input.CharLimit = 100
	input.Width = 40

	var repoName string
	var systemCaps git.SystemCapabilities

	if worktreeManager != nil {
		repoName = worktreeManager.GetRepositoryName()
		systemCaps = worktreeManager.GetSystemCapabilities()
	}

	loader := components.NewLaunchAgentLoader("")

	return &WorktreeDialog{
		input:           input,
		repoName:        repoName,
		worktreeManager: worktreeManager,
		systemCaps:      systemCaps,
		creating:        false,
		initializing:    false,
		agentConfig:     agentConfig,
		loader:          loader,
	}
}

// Init implements tea.Model
func (d *WorktreeDialog) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (d *WorktreeDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't process keys if we're in the middle of creating or initializing
		if d.creating || d.initializing {
			return d, nil
		}

		switch msg.String() {
		case "enter":
			// Create and attach worktree
			return d, d.createAndAttachWorktree()

		case "esc":
			// Cancel dialog
			return d, func() tea.Msg {
				return WorktreeDialogCancelledMsg{}
			}
		}

	case WorktreeCreatedMsg:
		// Worktree creation completed successfully, now start initializing
		d.creating = false
		d.initializing = true
		d.err = ""
		if d.loader != nil {
			d.loader.SetLabel(fmt.Sprintf("Launching %s...", d.agentConfig.CompanyName))
		}

		var caseCmds []tea.Cmd
		if d.loader != nil {
			if cmd := d.loader.TickCmd(); cmd != nil {
				caseCmds = append(caseCmds, cmd)
			}
		}
		caseCmds = append(caseCmds, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return WorktreeInitializationCompleteMsg(msg)
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

	// Update text input if not in creating/initializing state
	if !d.creating && !d.initializing {
		var inputCmd tea.Cmd
		d.input, inputCmd = d.input.Update(msg)
		cmds = append(cmds, inputCmd)
	}

	if d.initializing && d.loader != nil {
		if cmd := d.loader.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return d, tea.Batch(cmds...)
}

// createAndAttachWorktree creates a new worktree and attaches to it
func (d *WorktreeDialog) createAndAttachWorktree() tea.Cmd {
	if d.worktreeManager == nil {
		return func() tea.Msg {
			return WorktreeCreationErrorMsg{Error: "Worktree manager not available"}
		}
	}

	// Get branch name from input or generate random name
	branchName := strings.TrimSpace(d.input.Value())
	if branchName == "" {
		branchName = git.GenerateRandomBranchName()
	}

	// Validate branch name
	if err := git.ValidateBranchName(branchName); err != nil {
		return func() tea.Msg {
			return WorktreeCreationErrorMsg{Error: err.Error()}
		}
	}

	// Set creating state
	d.creating = true
	d.err = ""

	// Create worktree in background
	return func() tea.Msg {
		worktree, err := d.worktreeManager.CreateWorktree(branchName)
		if err != nil {
			return WorktreeCreationErrorMsg{Error: err.Error()}
		}
		return WorktreeCreatedMsg{Worktree: worktree}
	}
}

// SetSize updates the dialog dimensions
func (d *WorktreeDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View implements tea.Model and renders the dialog
func (d *WorktreeDialog) View() string {
	var content []string
	maxContentWidth := 0

	appendLine := func(line string) {
		content = append(content, line)
		if w := lipgloss.Width(line); w > maxContentWidth {
			maxContentWidth = w
		}
	}

	// Header: Repository name > New worktree
	repoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	titleStyle := dialogTitleStyle.Copy()

	repoText := repoStyle.Render(d.repoName)
	arrowText := titleStyle.Render(" > ")
	worktreeText := titleStyle.Render("New worktree")

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, repoText, arrowText, worktreeText)
	appendLine(headerLine)

	// Horizontal divider - will be sized later after we know content width
	content = append(content, "DIVIDER_PLACEHOLDER")
	content = append(content, "")

	// Input field - just the branch name input
	if !d.creating && !d.initializing {
		appendLine(d.input.View())
		content = append(content, "")
	}

	// Progress or button
	if d.creating {
		appendLine(dialogInfoStyle.Render("Creating worktree..."))
	} else if d.initializing {
		loadingTitle := fmt.Sprintf("Launching %s...", d.agentConfig.CompanyName)
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.agentConfig.BorderColor)).
			Bold(true)

		if d.loader != nil {
			d.loader.SetLabel(loadingTitle)
			appendLine(loaderStyle.Render(d.loader.View()))
		} else {
			appendLine(loaderStyle.Render(loadingTitle))
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

	// Add Create and attach button if not creating or initializing
	if !d.creating && !d.initializing {
		// Add some spacing before the button
		content = append(content, "")

		// Add button placeholder - will be replaced later with proper width
		content = append(content, "BUTTON_PLACEHOLDER")

		// Add error message directly under button (centered)
		if d.err != "" {
			content = append(content, "ERROR_PLACEHOLDER")
		} else {
			content = append(content, "")
		}
	}

	// Warning for non-COW systems
	if !d.systemCaps.SupportsCOW && !d.creating {
		appendLine(dialogWarningStyle.Render("⚠️  Only version controlled files"))
		appendLine(dialogWarningStyle.Render("   will be copied, which excludes"))
		appendLine(dialogWarningStyle.Render("   things like your dependencies"))
		appendLine(dialogWarningStyle.Render("   and .env files. This is because"))
		appendLine(dialogWarningStyle.Render("   your OS does not support"))
		appendLine(dialogWarningStyle.Render("   copy-on-write."))
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

	// Create full-width button
	buttonStyle := primaryDialogButtonStyle.Copy().
		Background(lipgloss.Color(d.agentConfig.BorderColor)).
		Width(actualContentWidth).
		Align(lipgloss.Center)
	button := buttonStyle.Render("Create and attach (↵)")

	// Create centered error message
	errorMsg := ""
	if d.err != "" {
		errorStyle := dialogErrorStyle.Copy().
			Width(actualContentWidth).
			Align(lipgloss.Center)
		errorMsg = errorStyle.Render("Error: " + d.err)
	}

	// Replace placeholders with actual content
	for i, line := range content {
		if line == "DIVIDER_PLACEHOLDER" {
			content[i] = divider
		} else if line == "BUTTON_PLACEHOLDER" {
			content[i] = button
		} else if line == "ERROR_PLACEHOLDER" {
			content[i] = errorMsg
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
	Worktree *git.WorktreeInfo
}

// WorktreeCreationErrorMsg indicates worktree creation failed
type WorktreeCreationErrorMsg struct {
	Error string
}

// WorktreeDialogCancelledMsg indicates the dialog was cancelled
type WorktreeDialogCancelledMsg struct{}

// WorktreeInitializationCompleteMsg indicates the worktree and session are ready
type WorktreeInitializationCompleteMsg struct {
	Worktree *git.WorktreeInfo
}
