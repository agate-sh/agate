package components

import (
	"agate/pkg/gui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LabeledInput represents a text input with a label above it
type LabeledInput struct {
	label       string
	placeholder string
	textInput   textinput.Model
}

// NewLabeledInput creates a new labeled input
func NewLabeledInput(label, placeholder string) *LabeledInput {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	ti.CharLimit = 200
	ti.Width = 40
	ti.Prompt = "" // No prompt, label is above

	return &LabeledInput{
		label:       label,
		placeholder: placeholder,
		textInput:   ti,
	}
}

// Focus focuses the input
func (l *LabeledInput) Focus() {
	l.textInput.Focus()
}

// Blur removes focus from the input
func (l *LabeledInput) Blur() {
	l.textInput.Blur()
}

// SetValue sets the input value
func (l *LabeledInput) SetValue(value string) {
	l.textInput.SetValue(value)
}

// Value returns the current input value
func (l *LabeledInput) Value() string {
	return l.textInput.Value()
}

// Update handles tea.Msg updates
func (l *LabeledInput) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	l.textInput, cmd = l.textInput.Update(msg)
	return cmd
}

// View renders the labeled input
func (l *LabeledInput) View() string {
	// Label style - bold white like session dialog
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	label := labelStyle.Render(l.label)
	input := l.textInput.View()

	return label + "\n" + input
}
