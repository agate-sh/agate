package panes

import (
	"agate/pkg/agents"
	"agate/pkg/tui/components"
	"agate/pkg/tui/icons"
	"agate/pkg/session"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SessionViewPane displays session metadata and tmux output from the active agent
type SessionViewPane struct {
	*components.BasePane
	session       *session.Session
	sessionHeader *components.SessionHeader
	tmuxContent   string
	isCreating    bool // True when session is being created in background
}

// NewSessionViewPane creates a new SessionViewPane instance
func NewSessionViewPane() *SessionViewPane {
	return &SessionViewPane{
		BasePane:      components.NewBasePane(1, "Session"), // Pane index 1
		tmuxContent:   "",
		sessionHeader: nil,
	}
}

// SetSession updates the session reference and refreshes the header
func (p *SessionViewPane) SetSession(s *session.Session) {
	p.session = s
	if s == nil {
		p.sessionHeader = nil
		return
	}

	// Build agent configs and determine active index
	sessionAgents := s.GetOrderedAgents()
	agentConfigs := make([]agents.AgentConfig, 0, len(sessionAgents))
	for _, agent := range sessionAgents {
		agentConfigs = append(agentConfigs, agent.AgentConfig)
	}

	// Create/update session header
	p.sessionHeader = components.NewSessionHeader(
		s.Description,
		s.BranchBaseName,
		agentConfigs,
		s.ActiveAgentIndex,
	)
	if p.sessionHeader != nil {
		p.sessionHeader.SetWidth(p.GetWidth())
	}
}

// SetTmuxContent updates the tmux output display
func (p *SessionViewPane) SetTmuxContent(content string) {
	p.tmuxContent = content
}

// TickDescriptionLoader advances the description generation spinner animation
func (p *SessionViewPane) TickDescriptionLoader() {
	if p.sessionHeader != nil {
		p.sessionHeader.TickLoader()
	}
}

// SetCreating sets whether a session is currently being created
func (p *SessionViewPane) SetCreating(creating bool) {
	p.isCreating = creating
}

// GetSessionHeader returns the session header (for setting generation state)
func (p *SessionViewPane) GetSessionHeader() *components.SessionHeader {
	return p.sessionHeader
}

// SetSize updates the dimensions of the session view pane
func (p *SessionViewPane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)
	if p.sessionHeader != nil {
		p.sessionHeader.SetWidth(width)
	}
}

// View renders the pane with session header and tmux output
func (p *SessionViewPane) View() string {
	if p.session == nil {
		message := "No active session"
		if p.isCreating {
			message = "Creating session..."
		}
		return lipgloss.NewStyle().
			Width(p.GetWidth()).
			Height(p.GetHeight()).
			Render(message)
	}

	// Calculate heights: ~25% for header, ~75% for tmux
	headerHeight := p.GetHeight() / 4
	tmuxHeight := p.GetHeight() - headerHeight

	// Render header
	var headerContent string
	if p.sessionHeader != nil {
		headerContent = p.sessionHeader.Render()
	}

	// Limit header to its allocated height
	headerLines := strings.Split(headerContent, "\n")
	if len(headerLines) > headerHeight {
		headerLines = headerLines[:headerHeight]
	}
	headerContent = strings.Join(headerLines, "\n")

	// Pad header to full height
	for len(strings.Split(headerContent, "\n")) < headerHeight {
		headerContent += "\n"
	}

	// Render tmux content
	tmuxLines := strings.Split(p.tmuxContent, "\n")
	if len(tmuxLines) > tmuxHeight {
		// Take the last N lines to show most recent output
		tmuxLines = tmuxLines[len(tmuxLines)-tmuxHeight:]
	}
	tmuxContent := strings.Join(tmuxLines, "\n")

	// Pad tmux content to full height
	for len(strings.Split(tmuxContent, "\n")) < tmuxHeight {
		tmuxContent += "\n"
	}

	// Join header and tmux sections
	return headerContent + tmuxContent
}

// Update handles tea.Msg updates for the session view pane
func (p *SessionViewPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	// Loader animation ticking will be handled by the main application
	// when description generation is in progress
	return p, nil
}

// HandleKey processes keyboard input when the pane is active
func (p *SessionViewPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	return false, nil
}

// GetTitleStyle returns the title style based on the active agent
func (p *SessionViewPane) GetTitleStyle() components.TitleStyle {
	shortcuts := "(" + icons.GetOptionKey() + "a)"
	if p.IsActive() {
		shortcuts = "tab cycle agents"
	}

	return components.TitleStyle{
		Type:      "standard",
		Text:      "Agents",
		Shortcuts: shortcuts,
	}
}

// GetPaneSpecificKeybindings returns session view pane specific keybindings
func (p *SessionViewPane) GetPaneSpecificKeybindings() []key.Binding {
	return []key.Binding{} // Tab cycling is implicit
}
