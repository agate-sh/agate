package main

import (
	"fmt"
	"strings"
	"time"

	"agate/components"
	"agate/git"

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
	agentConfig     AgentConfig
	loader          *components.LaunchAgentLoader
}

// Styling for worktree dialog
var (
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderMuted)).
			Padding(1, 2).
			MaxWidth(50)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(agateColor)). // Using ASCII art color
				MarginBottom(1)

	dialogErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(errorStatus)).
				MarginTop(1)

	dialogWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(warningStatus)).
				MarginTop(1)

	dialogInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(textMuted)). // Gray
			MarginTop(1)

	dialogButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(infoStatus)).
				MarginTop(1)
)

// NewWorktreeDialog creates a new worktree creation dialog
func NewWorktreeDialog(worktreeManager *git.WorktreeManager, agentConfig AgentConfig) *WorktreeDialog {
	input := textinput.New()
	input.Placeholder = "Branch name [↵ for random name]"
	input.Focus()
	input.CharLimit = 100
	input.Width = 30

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
			// Create worktree
			return d, d.createWorktree()

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

// createWorktree creates a new worktree
func (d *WorktreeDialog) createWorktree() tea.Cmd {
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

	// Title
	content = append(content, dialogTitleStyle.Render("Create New Worktree"))
	content = append(content, "")

	// Repository info
	content = append(content, "Repository: "+d.repoName)
	content = append(content, "")

	// Input field
	if d.creating {
		content = append(content, "Branch name: Creating...")
	} else if d.initializing {
		content = append(content, "Branch name: Created!")
	} else {
		content = append(content, "Branch name: "+d.input.View())
	}
	content = append(content, "")

	// Buttons or progress
	if d.creating {
		content = append(content, dialogInfoStyle.Render("Creating worktree..."))
	} else if d.initializing {
		loadingTitle := fmt.Sprintf("Launching %s...", d.agentConfig.CompanyName)
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.agentConfig.BorderColor)).
			Bold(true)

		if d.loader != nil {
			d.loader.SetLabel(loadingTitle)
			content = append(content, loaderStyle.Render(d.loader.View()))
		} else {
			content = append(content, loaderStyle.Render(loadingTitle))
		}
	} else {
		content = append(content, dialogButtonStyle.Render("[ Create and attach ]  [ Cancel (ESC) ]"))
	}

	// Warning for non-COW systems
	if !d.systemCaps.SupportsCOW && !d.creating {
		content = append(content, "")
		content = append(content, dialogWarningStyle.Render("⚠️  Only version controlled files"))
		content = append(content, dialogWarningStyle.Render("   will be copied, which excludes"))
		content = append(content, dialogWarningStyle.Render("   things like your dependencies"))
		content = append(content, dialogWarningStyle.Render("   and .env files. This is because"))
		content = append(content, dialogWarningStyle.Render("   your OS does not support"))
		content = append(content, dialogWarningStyle.Render("   copy-on-write."))
	}

	// Error message
	if d.err != "" {
		content = append(content, "")
		content = append(content, dialogErrorStyle.Render("Error: "+d.err))
	}

	// Join all content and apply dialog styling
	dialogContent := strings.Join(content, "\n")
	return dialogStyle.Render(dialogContent)
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
