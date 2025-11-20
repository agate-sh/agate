// Package tmux provides loading state management for the tmux pane
// including spinner animation and stopwatch functionality.
package tmux

import (
	"agate/pkg/tui/components"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LoadingState manages the loading state for a tmux session
type LoadingState struct {
	startTime *time.Time
	loader    *components.LaunchAgentLoader
}

// NewLoadingState creates a new loading state manager
func NewLoadingState() *LoadingState {
	return &LoadingState{
		loader: components.NewLaunchAgentLoader(""),
	}
}

// Start begins tracking loading time
func (ls *LoadingState) Start() {
	now := time.Now()
	ls.startTime = &now
}

// Stop clears the loading state
func (ls *LoadingState) Stop() {
	ls.startTime = nil
}

// IsLoading returns true if currently in loading state
func (ls *LoadingState) IsLoading() bool {
	return ls.startTime != nil
}

// GetElapsed returns the elapsed time since loading started
func (ls *LoadingState) GetElapsed() time.Duration {
	if ls.startTime == nil {
		return 0
	}
	return time.Since(*ls.startTime)
}

// ShouldShowStopwatch returns true if loading has been going for >3 seconds
func (ls *LoadingState) ShouldShowStopwatch() bool {
	return ls.IsLoading() && ls.GetElapsed() >= 3*time.Second
}

// TickCmd starts the spinner animation.
func (ls *LoadingState) TickCmd() tea.Cmd {
	if ls.loader == nil {
		return nil
	}
	return ls.loader.TickCmd()
}

// Update handles spinner tick messages for the loader.
func (ls *LoadingState) Update(msg tea.Msg) tea.Cmd {
	if ls.loader == nil {
		return nil
	}
	return ls.loader.Update(msg)
}

// RenderLoadingView creates the complete loading view with spinner and optional stopwatch
func (ls *LoadingState) RenderLoadingView(agentName, agentColor string, width, height int, textMuted, textDescription string) string {
	if !ls.IsLoading() {
		return ""
	}

	// Create loading view with spinner and agent name
	loadingText := agentName + " is starting..."
	ls.loader.SetLabel(loadingText)

	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(agentColor)).
		Bold(true)
	loadingContent := loadingStyle.Render(ls.loader.View())

	// Always reserve space for stopwatch to prevent jank
	var stopwatchText string
	if ls.ShouldShowStopwatch() && ls.startTime != nil {
		stopwatchText = components.FormatElapsedTime(*ls.startTime, "q", "quit")
	} else {
		// Reserve empty space to prevent movement
		stopwatchText = " " // Empty line placeholder
	}

	finalContent := loadingContent + "\n\n" + stopwatchText

	// Center the loading content in the pane
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, finalContent)
}
