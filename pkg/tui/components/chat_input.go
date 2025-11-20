package components

import (
	"agate/pkg/agents"
	"agate/pkg/tui/icons"
	"agate/pkg/tui/theme"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// OpenAgentSelectorMsg is sent when the user wants to open the agent selector
type OpenAgentSelectorMsg struct{}

// AgentsSelectedMsg is sent when agents are selected from the selector
type AgentsSelectedMsg struct {
	Agents []agents.AgentConfig
}

// ChatInput represents a chat-style input component with agent selection support
type ChatInput struct {
	textarea       textarea.Model
	selectedAgents []agents.AgentConfig
	defaultAgent   agents.AgentConfig
	width          int
	height         int
}

// NewChatInput creates a new chat input with a default agent pre-selected
func NewChatInput(defaultAgent agents.AgentConfig) *ChatInput {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.SetHeight(3)
	ta.CharLimit = 0 // No character limit
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("enter", "ctrl+m", "alt+enter"),
		key.WithHelp("enter", "insert newline"),
	)

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary))
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))

	ci := &ChatInput{
		textarea:       ta,
		selectedAgents: []agents.AgentConfig{defaultAgent},
		defaultAgent:   defaultAgent,
		width:          80,
		height:         3,
	}

	ci.adjustHeight()
	return ci
}

// InitCmd returns a command to focus the textarea and start cursor blinking.
func (c *ChatInput) InitCmd() tea.Cmd {
	return c.Focus()
}

// SetWidth sets the width of the input
func (c *ChatInput) SetWidth(width int) {
	c.width = width
	innerWidth := width - 4
	if innerWidth < 4 {
		innerWidth = 4
	}
	c.textarea.SetWidth(innerWidth)
}

// SetHeight sets the height of the input
func (c *ChatInput) SetHeight(height int) {
	c.height = height
	c.textarea.SetHeight(height)
}

// Focus sets focus on the textarea
func (c *ChatInput) Focus() tea.Cmd {
	focusCmd := c.textarea.Focus()
	return tea.Batch(focusCmd, textarea.Blink)
}

// Blur removes focus from the textarea
func (c *ChatInput) Blur() {
	c.textarea.Blur()
}

// Update handles keyboard input
func (c *ChatInput) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Handle Tab key to open agent selector
		if keyMsg.Type == tea.KeyTab || keyMsg.String() == "tab" {
			return func() tea.Msg {
				return OpenAgentSelectorMsg{}
			}
		}

		if keyMsg.Type == tea.KeyEnter && keyMsg.Alt {
			c.textarea.InsertRune('\n')
			c.adjustHeight()
			return nil
		}
	}

	// Handle agent selection update
	if msg, ok := msg.(AgentsSelectedMsg); ok {
		c.selectedAgents = msg.Agents
		return nil
	}

	// Normal textarea input
	var cmd tea.Cmd
	c.textarea, cmd = c.textarea.Update(msg)
	c.adjustHeight()
	return cmd
}

// GetValue returns the prompt text
func (c *ChatInput) GetValue() string {
	return strings.TrimSpace(c.textarea.Value())
}

// GetSelectedAgents returns the selected agents
func (c *ChatInput) GetSelectedAgents() []agents.AgentConfig {
	return c.selectedAgents
}

// SetSelectedAgents updates the selected agents
func (c *ChatInput) SetSelectedAgents(selectedAgents []agents.AgentConfig) {
	if len(selectedAgents) == 0 {
		selectedAgents = []agents.AgentConfig{c.defaultAgent}
	}
	c.selectedAgents = selectedAgents
}

// Reset clears the input and resets to default agent only
func (c *ChatInput) Reset() {
	c.textarea.Reset()
	c.selectedAgents = []agents.AgentConfig{c.defaultAgent}
	c.adjustHeight()
}

func (c *ChatInput) adjustHeight() {
	lines := c.textarea.LineCount()
	if lines < 3 {
		lines = 3
	}
	if lines != c.height {
		c.height = lines
		c.textarea.SetHeight(lines)
	}
}

// DebugRawView exposes the underlying textarea view for sandbox inspection.
func (c *ChatInput) DebugRawView() string {
	return c.textarea.View()
}

// DebugValue exposes the raw textarea value without trimming.
func (c *ChatInput) DebugValue() string {
	return c.textarea.Value()
}

// DebugHeight exposes the current textarea height.
func (c *ChatInput) DebugHeight() int {
	return c.height
}

// View renders the input box
func (c *ChatInput) View() string {
	innerWidth := c.width - 4
	if innerWidth < 0 {
		innerWidth = 0
	}

	body := c.textarea.View()
	agentLine := c.renderAgentLine(innerWidth)

	borderColor := theme.BorderMuted
	if c.textarea.Focused() {
		borderColor = theme.BorderActive
	}

	containerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1)

	container := containerStyle.Render(body)
	if agentLine == "" {
		return container
	}

	return lipgloss.JoinVertical(lipgloss.Left, container, agentLine)
}

func (c *ChatInput) renderAgentLine(width int) string {
	if width <= 0 || len(c.selectedAgents) == 0 {
		return ""
	}

	var agentNames []string
	for _, agent := range c.selectedAgents {
		if agent.Name == "" {
			continue
		}
		agentNames = append(agentNames, strings.ToUpper(agent.Name[:1])+agent.Name[1:])
	}

	if len(agentNames) == 0 {
		return ""
	}

	agentText := strings.Join(agentNames, ", ") + " " + icons.GetChevronDown()
	line := fitToWidth(agentText, width)
	if line == "" {
		return ""
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription)).
		PaddingLeft(1).
		Render(line)
}

func fitToWidth(text string, width int) string {
	if width <= 0 {
		return ""
	}

	var builder strings.Builder
	currentWidth := 0

	for _, r := range text {
		runeWidth := runewidth.RuneWidth(r)
		if runeWidth == 0 {
			continue
		}
		if currentWidth+runeWidth > width {
			break
		}
		builder.WriteRune(r)
		currentWidth += runeWidth
	}

	if currentWidth < width {
		builder.WriteString(strings.Repeat(" ", width-currentWidth))
	}

	return builder.String()
}
