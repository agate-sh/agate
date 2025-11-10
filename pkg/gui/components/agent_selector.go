package components

import (
	"agate/pkg/app"
	"agate/pkg/gui/theme"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AgentItem represents an agent in the selection list
type AgentItem struct {
	agent    app.AgentConfig
	selected bool
}

func (i AgentItem) FilterValue() string {
	return i.agent.Name
}

// agentDelegate renders agent items in the selection list
type agentDelegate struct{}

func (d agentDelegate) Height() int  { return 1 }
func (d agentDelegate) Spacing() int { return 0 }
func (d agentDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d agentDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	agentItem, ok := item.(AgentItem)
	if !ok {
		return
	}

	const uncheckedIcon = "\U000F0131"
	const checkedIcon = "\U000F0132"

	// Show checkbox indicator
	checkbox := uncheckedIcon
	if agentItem.selected {
		checkbox = checkedIcon
	}

	// Highlight if selected in list
	isSelected := index == m.Index()

	var style lipgloss.Style
	if isSelected {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Background(lipgloss.Color(theme.RowHighlight))
	} else {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextDescription))
	}

	rowWidth := m.Width()
	line := renderAgentRow(checkbox, agentItem.agent, rowWidth)
	fmt.Fprint(w, style.Render(line))
}

// AgentSelector is a modal for selecting multiple agents
type AgentSelector struct {
	list           list.Model
	filterInput    textinput.Model
	selectedAgents map[string]bool
	width          int
	height         int
	filtering      bool
	initCmd        tea.Cmd
}

// NewAgentSelector creates a new agent selection modal
func NewAgentSelector(initialAgents []app.AgentConfig) *AgentSelector {
	// Build selection map from initial agents
	selectedMap := make(map[string]bool)
	for _, agent := range initialAgents {
		selectedMap[agent.Name] = true
	}

	// Create list items for all available agents
	allAgents := app.GetAllAgents()
	items := make([]list.Item, len(allAgents))
	for i, agent := range allAgents {
		items[i] = AgentItem{
			agent:    agent,
			selected: selectedMap[agent.Name],
		}
	}

	delegate := agentDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false) // Don't show list title
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	// Create filter input (focused to show cursor)
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Agent name"
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	ti.CharLimit = 50
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary))
	focusCmd := ti.Focus()

	return &AgentSelector{
		list:           l,
		filterInput:    ti,
		selectedAgents: selectedMap,
		width:          60,
		height:         20,
		filtering:      false, // Start in navigation mode; focus moves to the input on demand
		initCmd:        focusCmd,
	}
}

// SetSize updates the dimensions of the selector
func (s *AgentSelector) SetSize(width, height int) {
	s.width = width
	s.height = height

	// Make modal take up a reasonable portion of the screen
	modalWidth := min(width-4, 60)
	modalHeight := min(height-4, 20)

	// Account for border(2), padding(2), title(1), filter input(1), shortcuts(3)
	maxListHeight := modalHeight - 9
	if maxListHeight < 5 {
		maxListHeight = 5
	}

	// Limit list height to actual number of items
	numItems := len(s.list.Items())
	listHeight := min(numItems, maxListHeight)
	if listHeight < 1 {
		listHeight = 1
	}

	s.list.SetSize(modalWidth-6, listHeight)

	inputWidth := modalWidth - 6
	if inputWidth < 10 {
		inputWidth = 10
	}
	s.filterInput.Width = inputWidth
}

// InitCmd returns a command that should be run once to start the cursor blinking.
func (s *AgentSelector) InitCmd() tea.Cmd {
	cmd := s.initCmd
	s.initCmd = nil
	return cmd
}

// Update handles keyboard input
func (s *AgentSelector) Update(msg tea.Msg) (*AgentSelector, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// If filtering, handle filter input
		if s.filtering {
			switch keyMsg.String() {
			case "esc":
				// Clear filter, refresh list, and exit filter mode
				s.clearFilter()
				s.filtering = false
				return s, nil
			case "down", "up":
				// Arrow keys exit filter mode and navigate list
				s.filtering = false
				var cmd tea.Cmd
				s.list, cmd = s.list.Update(msg)
				return s, cmd
			case "enter":
				// Enter key toggles the currently selected item in the list
				// Keep the filter and stay in filter mode
				s.toggleCurrentAgent()
				s.clearFilter()
				return s, nil
			case "tab":
				// Ignore tab key completely
				return s, nil
			default:
				var cmd tea.Cmd
				s.filterInput, cmd = s.filterInput.Update(msg)
				s.filterList()
				return s, cmd
			}
		} else {
			// Not filtering - handle list navigation
			switch keyMsg.String() {
			case "enter":
				// Toggle the currently selected agent
				s.toggleCurrentAgent()
				return s, nil
			case "tab", "shift+tab":
				// Ignore tab keys when not filtering
				return s, nil
			case "up", "down", "j", "k":
				// Navigation keys - let list handle them
				var cmd tea.Cmd
				s.list, cmd = s.list.Update(msg)
				return s, cmd
			default:
				// Any other key starts filtering
				var cmds []tea.Cmd
				if !s.filterInput.Focused() {
					if focusCmd := s.filterInput.Focus(); focusCmd != nil {
						cmds = append(cmds, focusCmd)
					}
				}
				s.filtering = true
				var cmd tea.Cmd
				s.filterInput, cmd = s.filterInput.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				s.filterList()
				return s, tea.Batch(cmds...)
			}
		}
	}

	var cmds []tea.Cmd
	if !s.filtering {
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	var inputCmd tea.Cmd
	s.filterInput, inputCmd = s.filterInput.Update(msg)
	if inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}

	return s, tea.Batch(cmds...)
}

// filterList filters the list based on the filter input
func (s *AgentSelector) filterList() {
	filterText := strings.ToLower(s.filterInput.Value())

	allAgents := app.GetAllAgents()
	var items []list.Item

	for _, agent := range allAgents {
		if filterText == "" || strings.Contains(strings.ToLower(agent.Name), filterText) {
			items = append(items, AgentItem{
				agent:    agent,
				selected: s.selectedAgents[agent.Name],
			})
		}
	}

	s.list.SetItems(items)
}

func renderAgentRow(checkbox string, agent app.AgentConfig, rowWidth int) string {
	displayName := agent.CompanyName
	if displayName == "" {
		displayName = agent.Name
	}

	left := fmt.Sprintf(" %s %s", checkbox, displayName)
	right := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Render(agent.Name)

	if rowWidth <= 0 {
		return left + " " + right
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := rowWidth - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

// clearFilter resets the filter input and refreshes the list.
func (s *AgentSelector) clearFilter() {
	s.filterInput.Reset()
	s.filterList()
}

// toggleCurrentAgent toggles the selection state of the currently highlighted agent
func (s *AgentSelector) toggleCurrentAgent() {
	selectedItem := s.list.SelectedItem()
	if agentItem, ok := selectedItem.(AgentItem); ok {
		// Toggle in the map
		s.selectedAgents[agentItem.agent.Name] = !s.selectedAgents[agentItem.agent.Name]

		// Update the list item
		items := s.list.Items()
		for i, item := range items {
			if ai, ok := item.(AgentItem); ok && ai.agent.Name == agentItem.agent.Name {
				items[i] = AgentItem{
					agent:    ai.agent,
					selected: s.selectedAgents[agentItem.agent.Name],
				}
				break
			}
		}
		s.list.SetItems(items)
	}
}

// GetSelectedAgents returns the list of selected agents
func (s *AgentSelector) GetSelectedAgents() []app.AgentConfig {
	var agents []app.AgentConfig
	allAgents := app.GetAllAgents()

	for _, agent := range allAgents {
		if s.selectedAgents[agent.Name] {
			agents = append(agents, agent)
		}
	}

	// Ensure at least one agent is selected (return Claude as fallback)
	if len(agents) == 0 {
		agents = []app.AgentConfig{app.ClaudeAgent}
	}

	return agents
}

// View renders the agent selector modal
func (s *AgentSelector) View() string {
	var parts []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Bold(true)
	parts = append(parts, titleStyle.Render("Select Agents"))
	parts = append(parts, "")

	// Filter input
	parts = append(parts, s.filterInput.View())
	parts = append(parts, "")

	// List
	parts = append(parts, s.list.View())
	parts = append(parts, "")

	// Shortcuts
	shortcutBase := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	keyStyle := shortcutBase.Copy().Bold(true)

	shortcuts := []string{
		keyStyle.Render("↑/↓") + " " + shortcutBase.Render("navigate"),
		keyStyle.Render("enter") + " " + shortcutBase.Render("toggle"),
		keyStyle.Render("esc") + " " + shortcutBase.Render("done"),
	}
	separator := shortcutBase.Render(" • ")
	parts = append(parts, strings.Join(shortcuts, separator))

	content := strings.Join(parts, "\n")

	// Create modal border
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderActive)).
		Padding(1, 2)

	return modalStyle.Render(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
