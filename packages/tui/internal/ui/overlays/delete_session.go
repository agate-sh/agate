package overlays

import (
	"context"
	"strings"
	"time"

	agateclient "agate/sdk/gen"
	"agate/tui/internal/api"
	"agate/tui/internal/ui/components"
	"agate/tui/internal/ui/panes/agents"
	"agate/tui/internal/ui/theme"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DeleteSessionOverlay shows a confirmation dialog for session deletion.
type DeleteSessionOverlay struct {
	width    int
	height   int
	worktree *agents.WorktreeInfo
	client   *agateclient.APIClient
	focused  bool
	help     help.Model
	keys     deleteKeyMap
}

// deleteKeyMap defines the keybindings for the delete confirmation dialog.
type deleteKeyMap struct {
	Delete key.Binding
	Escape key.Binding
}

// ShortHelp returns keybindings to show in the mini help view.
func (k deleteKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Escape}
}

// FullHelp returns keybindings to show in the full help view.
func (k deleteKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Delete, k.Escape},
	}
}

// DeleteSessionConfirmedMsg is sent when the user confirms deletion.
type DeleteSessionConfirmedMsg struct {
	Worktree *agents.WorktreeInfo
}

// DeleteSessionCancelledMsg is sent when the user cancels deletion.
type DeleteSessionCancelledMsg struct{}

// DeleteSessionSuccessMsg is sent when deletion succeeds.
type DeleteSessionSuccessMsg struct {
	Worktree *agents.WorktreeInfo
}

// DeleteSessionErrorMsg is sent when deletion fails.
type DeleteSessionErrorMsg struct {
	Worktree *agents.WorktreeInfo
	Error    string
}

// NewDeleteSessionOverlay creates a new session deletion confirmation dialog.
func NewDeleteSessionOverlay(worktree *agents.WorktreeInfo, client *agateclient.APIClient) *DeleteSessionOverlay {
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

	return &DeleteSessionOverlay{
		worktree: worktree,
		client:   client,
		focused:  true,
		help:     h,
		keys:     keys,
	}
}

// SetSize sets the dialog dimensions.
func (d *DeleteSessionOverlay) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Update handles tea.Msg updates.
func (d *DeleteSessionOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Delete):
			// Confirm deletion
			return d, d.deleteSession()
		case key.Matches(msg, d.keys.Escape):
			// Cancel deletion
			return d, func() tea.Msg {
				return DeleteSessionCancelledMsg{}
			}
		}
	}
	return d, nil
}

// deleteSession performs the actual session deletion via the SDK.
func (d *DeleteSessionOverlay) deleteSession() tea.Cmd {
	return func() tea.Msg {
		if d.client == nil || d.worktree == nil || d.worktree.Session == nil {
			return DeleteSessionErrorMsg{
				Worktree: d.worktree,
				Error:    "Client or session is nil",
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		sessionID := d.worktree.Session.ID
		if err := api.DeleteSession(ctx, d.client, sessionID); err != nil {
			return DeleteSessionErrorMsg{
				Worktree: d.worktree,
				Error:    err.Error(),
			}
		}

		return DeleteSessionSuccessMsg{Worktree: d.worktree}
	}
}

// View renders the confirmation dialog.
func (d *DeleteSessionOverlay) View() string {
	if d.worktree == nil {
		return "Error: No worktree selected"
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
	if d.worktree != nil {
		repoName = d.worktree.RepoName
		branchName = d.worktree.Branch
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

	// Subtitle
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

	// Calculate content width
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

	// Create help text
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
		BorderForeground(lipgloss.Color(theme.BorderActive)).
		Padding(1, 2).
		Width(maxContentWidth)

	return dialogStyle.Render(dialogContent)
}

// Init initializes the dialog.
func (d *DeleteSessionOverlay) Init() tea.Cmd {
	return nil
}
