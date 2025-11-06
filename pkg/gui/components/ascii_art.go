package components

import (
	_ "embed"
	"strings"

	"agate/pkg/gui/theme"

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

	// Style with muted purple color
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.AgateColor)).
		Width(width).
		Align(lipgloss.Center)

	// Split into lines and render each line centered
	lines := strings.Split(strings.TrimSpace(art), "\n")
	var styledLines []string
	for _, line := range lines {
		styledLines = append(styledLines, style.Render(line))
	}

	return strings.Join(styledLines, "\n")
}
