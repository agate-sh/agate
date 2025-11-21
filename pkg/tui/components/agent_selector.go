package components

import (
	"agate/pkg/agents"
	"agate/pkg/tui/theme"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AgentItem represents an agent in the selection list
type AgentItem struct {
	agent      agents.AgentConfig
	selected   bool
	isHeader   bool
	headerText string
}

func (i AgentItem) FilterValue() string {
	if i.isHeader {
		return ""
	}
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

	// Render section headers
	if agentItem.isHeader {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextDescription)).
			Bold(true)
		fmt.Fprint(w, headerStyle.Render(agentItem.headerText))
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
	line := renderAgentRow(checkbox, agentItem.agent, rowWidth, isSelected)
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
func NewAgentSelector(initialAgents []agents.AgentConfig) *AgentSelector {
	// Build selection map from initial agents using agent IDs
	selectedMap := make(map[string]bool)
	for _, agent := range initialAgents {
		selectedMap[agent.ID()] = true
	}

	// Create list items with sections
	items := buildAgentListItems(selectedMap)

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

	selector := &AgentSelector{
		list:           l,
		filterInput:    ti,
		selectedAgents: selectedMap,
		width:          60,
		height:         20,
		filtering:      false, // Start in navigation mode; focus moves to the input on demand
		initCmd:        focusCmd,
	}

	// Ensure initial selection is on a selectable item
	selector.ensureSelectableItemSelected()

	return selector
}

// buildAgentListItems creates the list items with "Top" and "All" sections
func buildAgentListItems(selectedMap map[string]bool) []list.Item {
	var items []list.Item

	// Define top agents in alphabetical order: Claude, Codex, Gemini
	topAgents := []agents.AgentConfig{
		agents.ClaudeAgent,
		agents.CodexAgent,
		agents.GeminiAgent,
	}

	// Create a map of top agent IDs for filtering
	topAgentNames := make(map[string]bool)
	for _, agent := range topAgents {
		topAgentNames[agent.ID()] = true
	}

	// Add "Top" section header
	items = append(items, AgentItem{
		isHeader:   true,
		headerText: "Top",
	})

	// Add top agents
	for _, agent := range topAgents {
		items = append(items, AgentItem{
			agent:    agent,
			selected: selectedMap[agent.ID()],
		})
	}

	// Add empty row
	items = append(items, AgentItem{
		isHeader:   true,
		headerText: "",
	})

	// Add "All" section header
	items = append(items, AgentItem{
		isHeader:   true,
		headerText: "All",
	})

	// Add remaining agents (excluding top agents) in alphabetical order
	allAgents := agents.GetAllAgents()
	var remainingAgents []agents.AgentConfig
	for _, agent := range allAgents {
		if !topAgentNames[agent.ID()] {
			remainingAgents = append(remainingAgents, agent)
		}
	}

	// Sort remaining agents alphabetically by CompanyName
	sort.Slice(remainingAgents, func(i, j int) bool {
		nameI := remainingAgents[i].CompanyName
		if nameI == "" {
			nameI = remainingAgents[i].Name
		}
		nameJ := remainingAgents[j].CompanyName
		if nameJ == "" {
			nameJ = remainingAgents[j].Name
		}
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})

	for _, agent := range remainingAgents {
		items = append(items, AgentItem{
			agent:    agent,
			selected: selectedMap[agent.ID()],
		})
	}

	return items
}

// SetSize updates the dimensions of the selector
func (s *AgentSelector) SetSize(width, height int) {
	s.width = width
	s.height = height

	// Make modal take up a reasonable portion of the screen
	modalWidth := min(width-4, 60)
	modalHeight := min(height-4, 30)

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
	var cmds []tea.Cmd

	// Always pass non-key messages to filterInput for cursor blinking
	if _, isKey := msg.(tea.KeyMsg); !isKey {
		var inputCmd tea.Cmd
		s.filterInput, inputCmd = s.filterInput.Update(msg)
		if inputCmd != nil {
			cmds = append(cmds, inputCmd)
		}
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// If filtering, handle filter input
		if s.filtering {
			switch keyMsg.String() {
			case "esc":
				// Clear filter, refresh list, and exit filter mode
				s.clearFilter()
				s.filtering = false
				return s, tea.Batch(cmds...)
			case "down", "up":
				// Arrow keys exit filter mode and navigate list
				s.filtering = false
				s.navigateToSelectableItem(keyMsg.String())
				return s, tea.Batch(cmds...)
			case "enter":
				// Enter key toggles the currently selected item in the list
				// Keep the filter and stay in filter mode
				s.toggleCurrentAgent()
				s.clearFilter()
				return s, tea.Batch(cmds...)
			case "tab":
				// Ignore tab key completely
				return s, tea.Batch(cmds...)
			default:
				var cmd tea.Cmd
				s.filterInput, cmd = s.filterInput.Update(msg)
				s.filterList()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return s, tea.Batch(cmds...)
			}
		} else {
			// Not filtering - handle list navigation
			switch keyMsg.String() {
			case "enter":
				// Toggle the currently selected agent
				s.toggleCurrentAgent()
				return s, tea.Batch(cmds...)
			case "tab", "shift+tab":
				// Ignore tab keys when not filtering
				return s, tea.Batch(cmds...)
			case "up", "down", "j", "k":
				// Navigation keys - skip headers
				s.navigateToSelectableItem(keyMsg.String())
				return s, tea.Batch(cmds...)
			default:
				// Any other key starts filtering
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

	// Pass non-key messages to list
	if !s.filtering {
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return s, tea.Batch(cmds...)
}

// navigateToSelectableItem moves to the next/previous selectable item, skipping headers
func (s *AgentSelector) navigateToSelectableItem(direction string) {
	items := s.list.Items()
	if len(items) == 0 {
		return
	}

	current := s.list.Index()
	step := 1
	if direction == "up" || direction == "k" {
		step = -1
	}

	// Try to find next selectable item
	for i := 0; i < len(items); i++ {
		current += step

		// Wrap around
		if current < 0 {
			current = len(items) - 1
		} else if current >= len(items) {
			current = 0
		}

		if item, ok := items[current].(AgentItem); ok {
			if !item.isHeader {
				s.list.Select(current)
				return
			}
		}
	}
}

// filterList filters the list based on the filter input
func (s *AgentSelector) filterList() {
	filterText := strings.ToLower(s.filterInput.Value())

	// If no filter, show sectioned list
	if filterText == "" {
		items := buildAgentListItems(s.selectedAgents)
		s.list.SetItems(items)
		// Select first selectable item
		s.ensureSelectableItemSelected()
		return
	}

	// When filtering, show flat list without sections
	allAgents := agents.GetAllAgents()
	var items []list.Item

	for _, agent := range allAgents {
		// Search against human-readable name (CompanyName) if available, otherwise use Name
		searchName := agent.CompanyName
		if searchName == "" {
			searchName = agent.Name
		}
		if strings.Contains(strings.ToLower(searchName), filterText) {
			items = append(items, AgentItem{
				agent:    agent,
				selected: s.selectedAgents[agent.ID()],
			})
		}
	}

	s.list.SetItems(items)
	// Select first selectable item
	s.ensureSelectableItemSelected()
}

// ensureSelectableItemSelected ensures a selectable item is selected
func (s *AgentSelector) ensureSelectableItemSelected() {
	items := s.list.Items()
	if len(items) == 0 {
		return
	}

	current := s.list.Index()
	if current < len(items) {
		if item, ok := items[current].(AgentItem); ok && !item.isHeader {
			return // Current selection is valid
		}
	}

	// Find first selectable item
	for i, item := range items {
		if agentItem, ok := item.(AgentItem); ok && !agentItem.isHeader {
			s.list.Select(i)
			return
		}
	}
}

func renderAgentRow(checkbox string, agent agents.AgentConfig, rowWidth int, isSelected bool) string {
	displayName := agent.CompanyName
	if displayName == "" {
		displayName = agent.Name
	}

	left := fmt.Sprintf(" %s %s", checkbox, displayName)

	// Use agent color when selected, otherwise use muted color
	var agentNameColor string
	if isSelected {
		agentNameColor = agent.BorderColor
	} else {
		agentNameColor = theme.TextMuted
	}

	right := lipgloss.NewStyle().
		Foreground(lipgloss.Color(agentNameColor)).
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
	s.ensureSelectableItemSelected()
}

// toggleCurrentAgent toggles the selection state of the currently highlighted agent
func (s *AgentSelector) toggleCurrentAgent() {
	selectedItem := s.list.SelectedItem()
	if agentItem, ok := selectedItem.(AgentItem); ok {
		// Skip if it's a header
		if agentItem.isHeader {
			return
		}

		// Toggle in the map
		s.selectedAgents[agentItem.agent.ID()] = !s.selectedAgents[agentItem.agent.ID()]

		// Update the list item
		items := s.list.Items()
		for i, item := range items {
			if ai, ok := item.(AgentItem); ok && !ai.isHeader && ai.agent.ID() == agentItem.agent.ID() {
				items[i] = AgentItem{
					agent:    ai.agent,
					selected: s.selectedAgents[agentItem.agent.ID()],
				}
				break
			}
		}
		s.list.SetItems(items)
	}
}

// GetSelectedAgents returns the list of selected agents
func (s *AgentSelector) GetSelectedAgents() []agents.AgentConfig {
	var selectedList []agents.AgentConfig
	allAgents := agents.GetAllAgents()

	for _, agent := range allAgents {
		if s.selectedAgents[agent.ID()] {
			selectedList = append(selectedList, agent)
		}
	}

	// Ensure at least one agent is selected (return Claude as fallback)
	if len(selectedList) == 0 {
		selectedList = []agents.AgentConfig{agents.ClaudeAgent}
	}

	return selectedList
}

// View renders the agent selector modal
func (s *AgentSelector) View() string {
	var parts []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Bold(true)
	parts = append(parts, titleStyle.Render("Select Agents"))

	// Subtext
	subtextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	parts = append(parts, subtextStyle.Render("Each agent will run in an isolated worktree"))
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
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.AgateColor)).
		Bold(true)

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
