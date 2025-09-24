package panes

import (
	"agate/theme"
	"agate/tmux"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// TmuxPane manages the display of tmux terminal content
type TmuxPane struct {
	*BasePane
	session      *tmux.TmuxSession
	content      string
	agentConfig  AgentConfig
	loadingState *tmux.LoadingState
	isLoading    bool
}

// AgentConfig represents agent-specific configuration
type AgentConfig struct {
	CompanyName string
	BorderColor string
}

// NewTmuxPane creates a new TmuxPane instance
func NewTmuxPane(agentConfig AgentConfig, loadingState *tmux.LoadingState) *TmuxPane {
	return &TmuxPane{
		BasePane:     NewBasePane(1, agentConfig.CompanyName), // Pane index 1
		agentConfig:  agentConfig,
		loadingState: loadingState,
		isLoading:    false,
	}
}

// SetSession sets the tmux session for this pane
func (t *TmuxPane) SetSession(session *tmux.TmuxSession) {
	t.session = session
}

// SetContent updates the tmux content to display
func (t *TmuxPane) SetContent(content string) {
	t.content = content
}

// SetLoading sets the loading state
func (t *TmuxPane) SetLoading(loading bool) {
	t.isLoading = loading
}

// GetTitleStyle returns the badge-style title for the tmux pane
func (t *TmuxPane) GetTitleStyle() TitleStyle {
	shortcuts := ""
	if t.IsActive() {
		// When active, show attach hint with agent color
		shortcuts = "[⏎ attach]"
	} else {
		// When not active, show pane number
		shortcuts = "[1]"
	}

	return TitleStyle{
		Type:      "badge",
		Color:     t.agentConfig.BorderColor, // Use agent's brand color for badge
		Text:      t.agentConfig.CompanyName,
		Shortcuts: shortcuts,
	}
}

// View renders the tmux pane content
func (t *TmuxPane) View() string {
	if t.isLoading && t.loadingState != nil {
		// Show loading view
		return t.loadingState.RenderLoadingView(
			t.agentConfig.CompanyName,
			t.agentConfig.BorderColor,
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
func (t *TmuxPane) Update(msg tea.Msg) (Pane, tea.Cmd) {
	// TmuxPane doesn't handle specific update messages currently
	// Content updates are handled externally via SetContent
	return t, nil
}

// HandleKey processes keyboard input when the pane is active
func (t *TmuxPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	// TmuxPane key handling is managed at the main model level
	// (attach/detach logic, scrolling, etc.)
	return false, nil
}

// GetPaneSpecificKeybindings returns tmux pane specific keybindings
func (t *TmuxPane) GetPaneSpecificKeybindings() []key.Binding {
	// Tmux pane keybindings - attach and detach
	attachTmux := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "attach to tmux"),
	)

	detachTmux := key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "detach from tmux"),
	)

	return []key.Binding{attachTmux, detachTmux}
}