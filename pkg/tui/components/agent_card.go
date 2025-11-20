package components

import (
	"agate/pkg/agents"
	"agate/pkg/tui/theme"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AgentCard represents a simple agent card component
type AgentCard struct {
	agent    agents.AgentConfig
	isActive bool
}

// NewAgentCard creates a new agent card
func NewAgentCard(agent agents.AgentConfig, isActive bool) *AgentCard {
	return &AgentCard{
		agent:    agent,
		isActive: isActive,
	}
}

// SetActive updates the active state of the card
func (c *AgentCard) SetActive(isActive bool) {
	c.isActive = isActive
}

// Render renders the agent card as a simple badge-style component
func (c *AgentCard) Render() string {
	// Capitalize first letter of agent name
	displayName := strings.ToUpper(c.agent.Name[:1]) + c.agent.Name[1:]

	var style lipgloss.Style

	if c.isActive {
		// Active state: border in agent's BorderColor, bold text
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(c.agent.BorderColor)).
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Bold(true).
			Padding(0, 1)
	} else {
		// Inactive state: muted gray border and text
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.TextMuted)).
			Foreground(lipgloss.Color(theme.TextMuted)).
			Padding(0, 1)
	}

	return style.Render(displayName)
}
