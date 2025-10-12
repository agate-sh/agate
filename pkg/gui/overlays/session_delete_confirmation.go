package overlays

import (
	"strings"

	"agate/pkg/git"
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"
	"agate/pkg/session"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SessionDeleteConfirmDialog shows a confirmation dialog for session deletion
type SessionDeleteConfirmDialog struct {
	width           int
	height          int
	session         *session.Session
	sessionManager  *session.Manager
	worktree        *git.WorktreeInfo    // For worktree-only deletion
	worktreeManager *git.WorktreeManager // For worktree-only deletion
	focused         bool
	help            help.Model
	keys            deleteKeyMap
}

// deleteKeyMap defines the keybindings for the delete confirmation dialog
type deleteKeyMap struct {
	Delete key.Binding
	Escape key.Binding
}

// ShortHelp returns keybindings to show in the mini help view
func (k deleteKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Escape}
}

// FullHelp returns keybindings to show in the full help view
func (k deleteKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Delete, k.Escape},
	}
}

// SessionDeletedMsg is sent when a session is successfully deleted
type SessionDeletedMsg struct {
	Session *session.Session
}

// SessionDeletionErrorMsg is sent when session deletion fails
type SessionDeletionErrorMsg struct {
	Session *session.Session
	Error   string
}

// SessionDeleteCancelledMsg is sent when session deletion is cancelled
type SessionDeleteCancelledMsg struct{}

// NewSessionDeleteConfirmDialog creates a new session deletion confirmation dialog
func NewSessionDeleteConfirmDialog(sess *session.Session, sessionManager *session.Manager) *SessionDeleteConfirmDialog {
	// Initialize help
	h := help.New()
	h.ShowAll = false // Only show short help

	// Initialize keybindings
	keys := deleteKeyMap{
		Delete: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "delete"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}

	return &SessionDeleteConfirmDialog{
		session:        sess,
		sessionManager: sessionManager,
		focused:        true,
		help:           h,
		keys:           keys,
	}
}

// SetSize sets the dialog dimensions
func (d *SessionDeleteConfirmDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetWorktreeInfo sets worktree info for worktree-only deletion
func (d *SessionDeleteConfirmDialog) SetWorktreeInfo(worktree *git.WorktreeInfo, worktreeManager *git.WorktreeManager) {
	d.worktree = worktree
	d.worktreeManager = worktreeManager
}

// Update handles tea.Msg updates
func (d *SessionDeleteConfirmDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Delete):
			// Confirm deletion
			return d, d.deleteSession()
		case key.Matches(msg, d.keys.Escape):
			// Cancel deletion
			return d, func() tea.Msg {
				return SessionDeleteCancelledMsg{}
			}
		}
	}
	return d, nil
}

// deleteSession performs the actual session deletion
func (d *SessionDeleteConfirmDialog) deleteSession() tea.Cmd {
	return func() tea.Msg {
		if d.sessionManager == nil || d.session == nil {
			return SessionDeletionErrorMsg{
				Session: d.session,
				Error:   "Session manager or session is nil",
			}
		}

		// Delete the session using the session manager
		err := d.sessionManager.DeleteSession(d.session.WorktreeKey)
		if err != nil {
			return SessionDeletionErrorMsg{
				Session: d.session,
				Error:   err.Error(),
			}
		}

		return SessionDeletedMsg{Session: d.session}
	}
}

// View renders the confirmation dialog
func (d *SessionDeleteConfirmDialog) View() string {
	if d.session == nil {
		return "Error: No session selected"
	}

	var content []string
	maxContentWidth := 0

	appendLine := func(line string) {
		content = append(content, line)
		if w := lipgloss.Width(line); w > maxContentWidth {
			maxContentWidth = w
		}
	}

	// Header: Repo > Branch > Delete session (same style as session dialog)
	repoName := "unknown"
	branchName := "unknown"
	if d.session != nil && d.session.Worktree != nil {
		repoName = d.session.Worktree.RepoName
		branchName = d.session.Worktree.Branch
	}

	repoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Bold(true)

	repoText := repoStyle.Render(repoName)
	arrow1Text := titleStyle.Render(" > ")
	branchText := repoStyle.Render(branchName)
	arrow2Text := titleStyle.Render(" > ")
	actionText := titleStyle.Render("Delete session")

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, repoText, arrow1Text, branchText, arrow2Text, actionText)
	appendLine(headerLine)
	content = append(content, "")

	// Horizontal divider
	content = append(content, "DIVIDER_PLACEHOLDER")
	content = append(content, "")

	// Subtitle - don't constrain width to avoid early wrapping
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary))
	appendLine(subtitleStyle.Render("This will delete the worktree, along with all uncommitted changes"))
	content = append(content, "")
	content = append(content, "")

	// Button placeholder
	content = append(content, "BUTTON_PLACEHOLDER")
	content = append(content, "")

	// Help text placeholder
	content = append(content, "HELP_PLACEHOLDER")

	// Calculate content width - increase to prevent wrapping
	const minContentWidth = 60
	if maxContentWidth < minContentWidth {
		maxContentWidth = minContentWidth
	}

	// Calculate actual content width accounting for dialog padding (2 on each side = 4 total)
	actualContentWidth := maxContentWidth
	if actualContentWidth > 4 {
		actualContentWidth -= 4
	}

	// Create delete button
	deleteButton := components.NewButton("Delete", "↵", components.ButtonVariantDanger)
	deleteButton.SetWidth(actualContentWidth)
	button := deleteButton.Render()

	// Create help text - using TextMuted like session dialog
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Width(actualContentWidth).
		Align(lipgloss.Center)
	helpText := helpStyle.Render(d.help.View(d.keys))

	// Create divider
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	divider := dividerStyle.Render(strings.Repeat("─", actualContentWidth))

	// Replace placeholders
	for i, line := range content {
		if line == "DIVIDER_PLACEHOLDER" {
			content[i] = divider
		} else if line == "BUTTON_PLACEHOLDER" {
			content[i] = button
		} else if line == "HELP_PLACEHOLDER" {
			content[i] = helpText
		}
	}

	// Join all content
	dialogContent := strings.Join(content, "\n")

	// Apply dialog border and padding
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.TextDescription)).
		Padding(1, 2).
		Width(maxContentWidth)

	return dialogStyle.Render(dialogContent)
}

// Init initializes the dialog
func (d *SessionDeleteConfirmDialog) Init() tea.Cmd {
	return nil
}
