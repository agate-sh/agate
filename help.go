package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpDialog represents a help overlay showing all shortcuts
type HelpDialog struct {
	width  int
	height int
}

// Styling for help dialog
var (
	helpOverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			MaxWidth(65)  // Increase width to prevent border cutoff

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#9d87ae")). // Using ASCII art color
			MarginBottom(1)

	helpSectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")). // Cyan
			MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("220")) // Yellow

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	helpFooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			MarginTop(1)
)

// NewHelpDialog creates a new help dialog
func NewHelpDialog() *HelpDialog {
	return &HelpDialog{}
}

// Init implements tea.Model
func (h *HelpDialog) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (h *HelpDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return h, nil
}

// SetSize updates the dialog dimensions
func (h *HelpDialog) SetSize(width, height int) {
	h.width = width
	h.height = height
}


// View implements tea.Model and renders the help dialog content
func (h *HelpDialog) View() string {
	// Build help content
	var content []string

	// Title
	content = append(content, helpTitleStyle.Render("Agate - Terminal UI for AI Agents"))
	content = append(content, "")

	// Get all shortcuts grouped by category
	shortcuts := AllShortcuts()

	// Define the order of sections
	sectionOrder := []string{"Two-Mode System", "Navigation", "Mode Benefits", "Help"}

	for _, section := range sectionOrder {
		if items, ok := shortcuts[section]; ok {
			// Section header
			content = append(content, helpSectionStyle.Render(section))

			// Shortcuts in this section
			for _, shortcut := range items {
				line := "  " + helpKeyStyle.Render(padRight(shortcut.Key, 12)) +
					helpDescStyle.Render(shortcut.Description)
				content = append(content, line)
			}
		}
	}

	// Footer
	content = append(content, "")
	content = append(content, helpFooterStyle.Render("Press any key to close"))

	// Join all content
	helpContent := strings.Join(content, "\n")

	// Apply overlay styling to create the dialog box
	return helpOverlayStyle.Render(helpContent)
}

// padRight pads a string to the right with spaces
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}