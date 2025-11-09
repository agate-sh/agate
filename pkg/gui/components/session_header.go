package components

import (
	"agate/pkg/app"
	"agate/pkg/gui/theme"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SessionHeader represents the header section of a session view
type SessionHeader struct {
	description           string
	branchName            string
	agentCards            []*AgentCard
	generatingDescription bool
	width                 int
	loaderFrame           int // For blinking animation
}

// Loader frames for spinning animation
var loaderFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSessionHeader creates a new session header
func NewSessionHeader(description, branchName string, agents []app.AgentConfig, activeIndex int) *SessionHeader {
	// Create agent cards
	cards := make([]*AgentCard, 0, len(agents))
	for i, agent := range agents {
		isActive := i == activeIndex
		card := NewAgentCard(agent, isActive)
		cards = append(cards, card)
	}

	return &SessionHeader{
		description:           description,
		branchName:            branchName,
		agentCards:            cards,
		generatingDescription: description == "", // If empty, we might be generating
		width:                 80,
		loaderFrame:           0,
	}
}

// UpdateDescription updates the description text
func (h *SessionHeader) UpdateDescription(description string) {
	h.description = description
	h.generatingDescription = description == ""
}

// UpdateBranchName updates the branch name
func (h *SessionHeader) UpdateBranchName(branchName string) {
	h.branchName = branchName
}

// UpdateActiveCard updates which agent card is highlighted
func (h *SessionHeader) UpdateActiveCard(activeIndex int) {
	for i, card := range h.agentCards {
		card.SetActive(i == activeIndex)
	}
}

// SetWidth sets the width of the header
func (h *SessionHeader) SetWidth(width int) {
	h.width = width
}

// SetGeneratingDescription controls the loader display
func (h *SessionHeader) SetGeneratingDescription(generating bool) {
	h.generatingDescription = generating
}

// TickLoader advances the loader animation frame
func (h *SessionHeader) TickLoader() {
	h.loaderFrame = (h.loaderFrame + 1) % len(loaderFrames)
}

// Render renders the complete header with all sections
func (h *SessionHeader) Render() string {
	var sections []string

	// 1. Description row
	descriptionSection := h.renderDescription()
	if descriptionSection != "" {
		sections = append(sections, descriptionSection)
	}

	// 2. Branch name row
	if h.branchName != "" {
		branchSection := h.renderBranchName()
		sections = append(sections, branchSection)
	}

	// 3. Agent cards row
	if len(h.agentCards) > 0 {
		// Add a blank spacer row before the cards
		sections = append(sections, "")

		cardsSection := h.renderAgentCards()
		sections = append(sections, cardsSection)
	}

	// 4. Horizontal divider
	divider := h.renderDivider()
	sections = append(sections, divider)

	return strings.Join(sections, "\n")
}

// renderDescription renders the description row with loader if generating
func (h *SessionHeader) renderDescription() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Width(h.width)

	if h.generatingDescription {
		// Show blinking loader
		loader := loaderFrames[h.loaderFrame]
		return style.Render(loader + " Generating description...")
	}

	if h.description == "" {
		return "" // Don't show anything if no description and not generating
	}

	return style.Render(h.description)
}

// renderBranchName renders the branch name row
func (h *SessionHeader) renderBranchName() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Width(h.width)

	if strings.TrimSpace(h.branchName) == "" {
		return ""
	}

	branchIcon := " " // Nerd Font branch icon
	return style.Render(branchIcon + h.branchName)
}

// renderAgentCards renders the horizontal layout of agent cards
func (h *SessionHeader) renderAgentCards() string {
	var cards []string
	for _, card := range h.agentCards {
		cards = append(cards, card.Render())
	}

	// Join multi-line cards horizontally so their borders line up
	cardsLine := lipgloss.JoinHorizontal(lipgloss.Top, cards...)

	// Ensure the row uses the full header width
	return lipgloss.NewStyle().
		Width(h.width).
		Render(cardsLine)
}

// renderDivider renders the horizontal separator line
func (h *SessionHeader) renderDivider() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.SeparatorColor)).
		Width(h.width)

	return style.Render(strings.Repeat("─", h.width))
}
