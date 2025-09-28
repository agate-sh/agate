package overlays

import (
	"fmt"
	"strings"

	"agate/pkg/git"
	"agate/pkg/session"

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
	return &SessionDeleteConfirmDialog{
		session:        sess,
		sessionManager: sessionManager,
		focused:        true,
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("y", "enter"))):
			// Confirm deletion
			return d, d.deleteSession()
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "esc"))):
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

	// Dialog styling
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Margin(1).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252"))

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248"))

	buttonStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 2).
		Margin(0, 1)

	// Generate session name for display
	sessionName := d.session.Name
	if sessionName == "" && d.session.Worktree != nil {
		sessionName = fmt.Sprintf("%s:%s", d.session.Worktree.RepoName, d.session.Worktree.Branch)
	}

	// Build dialog content
	var content strings.Builder

	// Title
	content.WriteString(warningStyle.Render("Delete Session"))
	content.WriteString("\n\n")

	// Session details
	content.WriteString(fmt.Sprintf("Session: %s\n", sessionName))
	if d.session.Agent.Name != "" {
		content.WriteString(fmt.Sprintf("Agent: %s\n", d.session.Agent.Name))
	}
	if d.session.Worktree != nil {
		content.WriteString(fmt.Sprintf("Worktree: %s\n", d.session.Worktree.Path))
		if d.session.TmuxSession != nil {
			content.WriteString(fmt.Sprintf("Tmux Session: %s\n", d.session.TmuxSession.GetSessionName()))
		}
	}

	content.WriteString("\n")

	// Warning message
	content.WriteString(warningStyle.Render("This will delete both the git worktree and terminate the tmux session."))
	content.WriteString("\n")
	content.WriteString(infoStyle.Render("All unsaved work will be lost."))
	content.WriteString("\n\n")

	// Action buttons
	confirmButton := buttonStyle.Copy().Background(lipgloss.Color("196")).Render("[Y] Delete")
	cancelButton := buttonStyle.Render("[N] Cancel")
	content.WriteString(confirmButton + cancelButton)

	// Apply dialog styling
	dialog := dialogStyle.Render(content.String())

	// Center the dialog
	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)

	verticalPadding := (d.height - dialogHeight) / 2
	horizontalPadding := (d.width - dialogWidth) / 2

	if verticalPadding < 0 {
		verticalPadding = 0
	}
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	return lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// Init initializes the dialog
func (d *SessionDeleteConfirmDialog) Init() tea.Cmd {
	return nil
}
