package main

import (
	"fmt"
	"os"

	"agate/pkg/app"
	"agate/pkg/gui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	input  *components.ChatInput
	width  int
	height int
}

func newModel() model {
	return model{
		input: components.NewChatInput(app.ClaudeAgent),
	}
}

func (m model) Init() tea.Cmd {
	return m.input.InitCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width > 0 {
			m.input.SetWidth(m.width - 8)
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			return m, tea.Quit
		}
	}

	cmd := m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666A7A")).
		Render("Sandbox: alt+enter inserts newline • enter submits • esc/ctrl+c exits")

	value := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999")).
		Render(fmt.Sprintf("Current value: %q", m.input.GetValue()))

	internalValue := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777")).
		Render(fmt.Sprintf("Textarea value(): %q", m.input.DebugValue()))

	heightInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777")).
		Render(fmt.Sprintf("Height: %d lines", m.input.DebugHeight()))

	raw := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555")).
		Render(fmt.Sprintf("Raw view: %q", m.input.DebugRawView()))

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		instructions,
		"",
		m.input.View(),
		"",
		value,
		internalValue,
		heightInfo,
		raw,
	)

	if m.width == 0 || m.height == 0 {
		return content
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)
}

func main() {
	if err := tea.NewProgram(newModel(), tea.WithAltScreen()).Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
