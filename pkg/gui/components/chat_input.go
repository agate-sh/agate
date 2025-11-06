package components

import (
	"agate/pkg/app"
	"agate/pkg/gui/theme"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ChatInput represents a chat-style input component with agent mention support
type ChatInput struct {
	textarea       textarea.Model
	selectedAgents []app.AgentConfig
	defaultAgent   app.AgentConfig
	placeholder    string
	width          int
	height         int
}

// NewChatInput creates a new chat input with a default agent pre-selected
func NewChatInput(defaultAgent app.AgentConfig) *ChatInput {
	ta := textarea.New()
	ta.Placeholder = "// prompt"
	ta.Focus()
	ta.SetHeight(3)
	ta.CharLimit = 0 // No character limit
	ta.ShowLineNumbers = false

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderActive))
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderMuted))

	return &ChatInput{
		textarea:       ta,
		selectedAgents: []app.AgentConfig{defaultAgent},
		defaultAgent:   defaultAgent,
		placeholder:    "// prompt",
		width:          80,
		height:         3,
	}
}

// SetWidth sets the width of the input
func (c *ChatInput) SetWidth(width int) {
	c.width = width
	c.textarea.SetWidth(width - 4) // Account for borders/padding
}

// SetHeight sets the height of the input
func (c *ChatInput) SetHeight(height int) {
	c.height = height
	c.textarea.SetHeight(height)
}

// Focus sets focus on the textarea
func (c *ChatInput) Focus() {
	c.textarea.Focus()
}

// Blur removes focus from the textarea
func (c *ChatInput) Blur() {
	c.textarea.Blur()
}

// Update handles keyboard input and parses @ mentions in real-time
func (c *ChatInput) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	c.textarea, cmd = c.textarea.Update(msg)

	// Parse @ mentions after every update
	mentions := c.ParseAgentMentions(c.textarea.Value())
	c.updateSelectedAgents(mentions)

	return cmd
}

// ParseAgentMentions extracts all @agent mentions from the text
func (c *ChatInput) ParseAgentMentions(text string) []string {
	// Regex to match @word patterns
	re := regexp.MustCompile(`@(\w+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	var mentions []string
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, strings.ToLower(match[1]))
		}
	}

	return mentions
}

// updateSelectedAgents updates the selected agents based on mentions
// Always includes the default agent plus any valid mentioned agents
func (c *ChatInput) updateSelectedAgents(mentions []string) {
	// Start with default agent
	agents := []app.AgentConfig{c.defaultAgent}

	// Add mentioned agents if they're valid and not the default
	seenAgents := make(map[string]bool)
	seenAgents[strings.ToLower(c.defaultAgent.Name)] = true

	for _, mention := range mentions {
		// Check if this is a valid agent and not already added
		if !seenAgents[mention] && app.IsValidAgent(mention) {
			agent := app.GetAgentConfig(mention)
			if agent.Name != "default" { // Make sure we got a real agent
				agents = append(agents, agent)
				seenAgents[mention] = true
			}
		}
	}

	c.selectedAgents = agents
}

// GetValue returns the prompt text with @ mentions stripped out
func (c *ChatInput) GetValue() string {
	text := c.textarea.Value()
	// Remove all @ mentions
	re := regexp.MustCompile(`@\w+\s*`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}

// GetSelectedAgents returns the default agent plus any mentioned agents
func (c *ChatInput) GetSelectedAgents() []app.AgentConfig {
	return c.selectedAgents
}

// Reset clears the input and resets to default agent only
func (c *ChatInput) Reset() {
	c.textarea.Reset()
	c.selectedAgents = []app.AgentConfig{c.defaultAgent}
}

// View renders the input box with agent badges displayed below
func (c *ChatInput) View() string {
	var parts []string

	// Render the textarea
	parts = append(parts, c.textarea.View())

	// Render agent badges below
	if len(c.selectedAgents) > 0 {
		var badges []string
		for _, agent := range c.selectedAgents {
			badge := c.renderAgentBadge(agent)
			badges = append(badges, badge)
		}
		badgesLine := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextDescription)).
			Render(strings.Join(badges, " "))
		parts = append(parts, badgesLine)
	}

	return strings.Join(parts, "\n")
}

// renderAgentBadge renders a small badge for an agent
func (c *ChatInput) renderAgentBadge(agent app.AgentConfig) string {
	// Capitalize first letter
	displayName := strings.ToUpper(agent.Name[:1]) + agent.Name[1:]

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Background(lipgloss.Color(agent.BorderColor)).
		Padding(0, 1).
		Bold(true)

	return style.Render(displayName)
}
