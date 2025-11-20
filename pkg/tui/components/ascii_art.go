package components

import (
	_ "embed"

	"agate/pkg/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

//go:embed ascii-art.txt
var asciiArtContent string

// LoadAgateASCII returns the Agate ASCII art as a string
func LoadAgateASCII() string {
	return asciiArtContent
}

// RenderAgateASCII centers the ASCII art and applies styling
func RenderAgateASCII(width int) string {
	art := LoadAgateASCII()

	// Just apply color - no width or centering manipulation
	// The ASCII art file has its own internal alignment that must be preserved
	colorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.AgateColor))

	coloredArt := colorStyle.Render(art)

	// Center the entire block
	centerStyle := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center)

	return centerStyle.Render(coloredArt)
}
