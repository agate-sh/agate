package components

import (
	"strings"

	"agate/internal/version"
	"agate/pkg/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

// WelcomeHeader renders the header for the new session view
// Includes ASCII art with version to the right and shortcuts below
type WelcomeHeader struct {
	width int
}

// NewWelcomeHeader creates a new welcome header component
func NewWelcomeHeader() *WelcomeHeader {
	return &WelcomeHeader{
		width: 80,
	}
}

// SetWidth sets the width of the header
func (h *WelcomeHeader) SetWidth(width int) {
	h.width = width
}

// View renders the header with ASCII art, version, and shortcuts
func (h *WelcomeHeader) View() string {
	art := strings.TrimRight(LoadAgateASCII(), "\n")
	if art == "" {
		return ""
	}

	// Split art into lines to add version on the same line as the last visible character
	lines := strings.Split(art, "\n")

	// Determine widest line once so we can position the version label
	artWidth := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > artWidth {
			artWidth = w
		}
	}

	// Get version string
	ver := version.Short()
	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted))
	styledVersion := versionStyle.Render(ver)

	// Add version on its own line aligned to the bottom-right of the art block
	versionWidth := lipgloss.Width(styledVersion)
	versionPadding := artWidth - versionWidth
	if versionPadding < 0 {
		versionPadding = 0
	}
	lines = append(lines, strings.Repeat(" ", versionPadding)+styledVersion)
	if versionWidth+versionPadding > artWidth {
		artWidth = versionWidth + versionPadding
	}

	// Rejoin the art with version
	artWithVersion := strings.Join(lines, "\n")

	// Apply color to the art
	colorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.AgateColor))
	coloredArt := colorStyle.Render(artWithVersion)

	// Center the ASCII art
	containerWidth := h.width
	if containerWidth < artWidth {
		containerWidth = artWidth
	}
	artBlock := lipgloss.NewStyle().
		Width(artWidth).
		Render(coloredArt)
	centeredArt := lipgloss.NewStyle().
		Width(containerWidth).
		Align(lipgloss.Center).
		Render(artBlock)

	return centeredArt
}
