package main

import (
	_ "embed"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed ascii-art.txt
var welcomeAsciiArt string

// WelcomeOverlay represents the first-time user welcome overlay
type WelcomeOverlay struct {
	width  int
	height int
}

// Styling for welcome overlay
var (
	welcomeOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2).
				MaxWidth(65) // Same as help dialog that works

	welcomeAsciiStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9d87ae")). // Same color as main ASCII art
				MarginBottom(1)

	welcomeSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")). // Light gray
				Align(lipgloss.Center).
				Bold(true).
				Width(55). // Smaller width to avoid breaking border
				MarginBottom(2)

	welcomeFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")). // Gray
				Italic(true).
				Align(lipgloss.Center).
				MarginTop(1)
)

// NewWelcomeOverlay creates a new welcome overlay
func NewWelcomeOverlay() *WelcomeOverlay {
	return &WelcomeOverlay{}
}

// Init implements tea.Model
func (w *WelcomeOverlay) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (w *WelcomeOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return w, nil
}

// SetSize updates the overlay dimensions
func (w *WelcomeOverlay) SetSize(width, height int) {
	w.width = width
	w.height = height
}

// View implements tea.Model and renders the welcome overlay content
func (w *WelcomeOverlay) View() string {
	// Build welcome content
	var content []string

	// ASCII Art
	content = append(content, welcomeAsciiStyle.Render(welcomeAsciiArt))
	content = append(content, "")

	// Subtitle
	content = append(content, welcomeSubtitleStyle.Render("Manage any agent, anywhere"))
	content = append(content, "")

	// Footer
	content = append(content, welcomeFooterStyle.Render("Press any key to close"))

	// Join all content
	welcomeContent := strings.Join(content, "\n")

	// Apply overlay styling to create the dialog box
	return welcomeOverlayStyle.Render(welcomeContent)
}

// WelcomeOverlayClosedMsg indicates the welcome overlay was closed
type WelcomeOverlayClosedMsg struct{}