package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// LaunchAgentLoader renders a blinking cursor spinner with a launch label.
type LaunchAgentLoader struct {
	spinner spinner.Model
	label   string
}

// NewLaunchAgentLoader returns a loader configured with the blinking cursor spinner.
func NewLaunchAgentLoader(label string) *LaunchAgentLoader {
	s := spinner.New(spinner.WithSpinner(BlinkingCursor))
	return &LaunchAgentLoader{
		spinner: s,
		label:   label,
	}
}

// SetLabel updates the loader label.
func (l *LaunchAgentLoader) SetLabel(label string) {
	if l == nil {
		return
	}
	l.label = label
}

// TickCmd starts the spinner animation.
func (l *LaunchAgentLoader) TickCmd() tea.Cmd {
	if l == nil {
		return nil
	}
	return l.spinner.Tick
}

// Update advances the spinner animation when receiving tick messages.
func (l *LaunchAgentLoader) Update(msg tea.Msg) tea.Cmd {
	if l == nil {
		return nil
	}

	switch tick := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		l.spinner, cmd = l.spinner.Update(tick)
		return cmd
	}

	return nil
}

// View renders the spinner and label.
func (l *LaunchAgentLoader) View() string {
	if l == nil {
		return ""
	}

	spinnerView := l.spinner.View()
	if l.label == "" {
		return spinnerView
	}
	return spinnerView + " " + l.label
}
