package panes

import (
	"agate/pkg/app"
	"agate/pkg/common"
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"
	"agate/pkg/tmux"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// AgentTmuxPane manages the display of tmux terminal content
type AgentTmuxPane struct {
	*components.BasePane
	session      *tmux.TmuxSession
	content      string
	loadingState *tmux.LoadingState
	isLoading    bool
	mode         string // "preview" or "attached"
}

// NewAgentTmuxPane creates a new AgentTmuxPane instance
func NewAgentTmuxPane(loadingState *tmux.LoadingState) *AgentTmuxPane {
	// Get agent name from global state for the title
	agentName := app.GetCurrentAgentName()
	return &AgentTmuxPane{
		BasePane:     components.NewBasePane(1, agentName), // Pane index 1
		loadingState: loadingState,
		isLoading:    false,
		mode:         "preview", // Start in preview mode
	}
}

// SetSession sets the tmux session for this pane
func (t *AgentTmuxPane) SetSession(session *tmux.TmuxSession) {
	t.session = session
}

// SetContent updates the tmux content to display
func (t *AgentTmuxPane) SetContent(content string) {
	t.content = content
}

// SetLoading sets the loading state
func (t *AgentTmuxPane) SetLoading(loading bool) {
	t.isLoading = loading
}

// SetMode sets the interaction mode (preview/attached)
func (t *AgentTmuxPane) SetMode(mode string) {
	t.mode = mode
}

// GetTitleStyle returns the badge-style title for the tmux pane
func (t *AgentTmuxPane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	isActive := t.IsActive()

	if isActive {
		// When active, format shortcuts like the footer (without brackets)
		shortcuts = "↵ attach • ctrl+q detach"
	} else {
		// When not active, show pane number
		shortcuts = "(1)"
	}

	return components.TitleStyle{
		Type:      "badge",
		Color:     app.GetCurrentAgentColor(), // Use agent's brand color from global state
		Text:      app.GetCurrentAgentName(),
		Shortcuts: shortcuts,
	}
}

// View renders the tmux pane content
func (t *AgentTmuxPane) View() string {
	if t.isLoading && t.loadingState != nil {
		// Show loading view
		return t.loadingState.RenderLoadingView(
			app.GetCurrentAgentName(),
			app.GetCurrentAgentColor(),
			t.GetWidth(),
			t.GetHeight(),
			theme.TextMuted,
			theme.TextDescription,
		)
	}

	// Show tmux content
	return t.content
}

// Update handles tea.Msg updates for the tmux pane
func (t *AgentTmuxPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	// Handle spinner tick messages for loading state
	if t.loadingState != nil {
		if cmd := t.loadingState.Update(msg); cmd != nil {
			return t, cmd
		}
	}

	// Content updates are handled externally via SetContent
	return t, nil
}

// HandleKey processes keyboard input when the pane is active
func (t *AgentTmuxPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	// AgentTmuxPane key handling is managed at the main model level
	// (attach/detach logic, scrolling, etc.)
	return false, nil
}

// GetPaneSpecificKeybindings returns tmux pane specific keybindings
func (t *AgentTmuxPane) GetPaneSpecificKeybindings() []key.Binding {
	// Use the global keybindings to ensure consistency
	return []key.Binding{
		common.GlobalKeys.AttachTmux,
		common.GlobalKeys.DetachTmux,
	}
}