package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderRowWithHint_HintNeverWraps(t *testing.T) {
	// Test that hint text never wraps to next line
	content := "This is a very long piece of content that should be truncated"
	hint := "↵ open • d delete"
	width := 40
	paddingCount := 2

	result := RenderRowWithHint(content, hint, width, paddingCount, "#282c34", "#ffffff", "#6c7a96")

	// The result width should not exceed fullWidth
	// (ANSI codes add newlines internally but that's not actual wrapping)
	fullWidth := width + 2*paddingCount
	resultWidth := lipgloss.Width(result)

	if resultWidth > fullWidth {
		t.Errorf("Content wrapped. Expected max width %d, got %d", fullWidth, resultWidth)
	}

	// Verify both content and hint are present
	if !strings.Contains(result, "This is") {
		t.Error("Content missing from result")
	}
	if !strings.Contains(result, "open") {
		t.Error("Hint missing from result")
	}
}

func TestRenderRowWithHint_ContentTruncated(t *testing.T) {
	// Test that content is truncated when it's too long
	content := "Very long content that exceeds available width after hint"
	hint := "↵ open • d delete"
	width := 30
	paddingCount := 2

	result := RenderRowWithHint(content, hint, width, paddingCount, "#282c34", "#ffffff", "#6c7a96")

	// Result should not exceed fullWidth (width + 2*paddingCount)
	fullWidth := width + 2*paddingCount
	actualWidth := lipgloss.Width(result)

	if actualWidth > fullWidth {
		t.Errorf("Row exceeded full width. Expected max %d, got %d", fullWidth, actualWidth)
	}
}

func TestRenderRowWithHint_UnicodeCharacters(t *testing.T) {
	// Test that Unicode characters in hint are measured correctly
	content := "File.go"
	hint := "↵ • ✓ ×" // Various Unicode chars
	width := 20
	paddingCount := 2

	result := RenderRowWithHint(content, hint, width, paddingCount, "#282c34", "#ffffff", "#6c7a96")

	// Should not exceed full width
	fullWidth := width + 2*paddingCount
	resultWidth := lipgloss.Width(result)

	if resultWidth > fullWidth {
		t.Errorf("Unicode content wrapped. Expected max width %d, got %d", fullWidth, resultWidth)
	}
}

func TestRenderRowWithHint_MinimumSpacing(t *testing.T) {
	// Test that there's at least some space between content and hint
	content := "Short"
	hint := "hint"
	width := 20
	paddingCount := 2

	result := RenderRowWithHint(content, hint, width, paddingCount, "#282c34", "#ffffff", "#6c7a96")

	// Result should contain both content and hint
	if !strings.Contains(result, "Short") {
		t.Error("Result missing content")
	}
	// Note: hint might be styled, so we check for the presence of hint chars
	if !strings.Contains(result, "hint") {
		t.Error("Result missing hint")
	}
}

func TestRenderRowWithHint_EmptyHint(t *testing.T) {
	// Test with no hint
	content := "Regular content"
	hint := ""
	width := 30
	paddingCount := 2

	result := RenderRowWithHint(content, hint, width, paddingCount, "#282c34", "#ffffff", "#6c7a96")

	// Should still render content
	if !strings.Contains(result, "Regular content") {
		t.Error("Result missing content when hint is empty")
	}
}

func TestRenderRowWithHint_PaddingApplied(t *testing.T) {
	// Test that padding is correctly applied
	content := "Content"
	hint := "h"
	width := 20
	paddingCount := 3 // 3 chars padding on each side

	result := RenderRowWithHint(content, hint, width, paddingCount, "#282c34", "#ffffff", "#6c7a96")

	// Full width should be width + 2*paddingCount
	expectedFullWidth := width + 2*paddingCount // 26
	actualWidth := lipgloss.Width(result)

	if actualWidth != expectedFullWidth {
		t.Errorf("Padding not applied correctly. Expected width %d, got %d", expectedFullWidth, actualWidth)
	}
}
