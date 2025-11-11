package components

import (
	"agate/pkg/gui/theme"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GitDiffViewer displays a scrollable diff for a single file
type GitDiffViewer struct {
	viewport    viewport.Model
	diffContent string // Pre-rendered diff content from delta
	width       int
	height      int
	active      bool
}

// NewGitDiffViewer creates a new diff viewer
func NewGitDiffViewer() *GitDiffViewer {
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true
	return &GitDiffViewer{
		viewport: vp,
	}
}

// SetSize updates the viewport dimensions
func (d *GitDiffViewer) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.Width = width
	d.viewport.Height = height
}

// SetActive sets whether the viewer is focused
func (d *GitDiffViewer) SetActive(active bool) {
	d.active = active
}

// Width returns the current width
func (d *GitDiffViewer) Width() int {
	return d.width
}

// SetDiffContent updates the diff content being displayed
func (d *GitDiffViewer) SetDiffContent(content string) {
	d.diffContent = content
	d.viewport.GotoTop()
	d.viewport.SetContent(content)
}

// Update handles viewport updates
func (d *GitDiffViewer) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return cmd
}

// View renders the diff viewer
func (d *GitDiffViewer) View() string {
	if d.diffContent == "" {
		// Show empty state
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)).
			Width(d.width).
			Height(d.height).
			Align(lipgloss.Center, lipgloss.Center)
		return emptyStyle.Render("Select a file to view diff")
	}

	return d.viewport.View()
}
