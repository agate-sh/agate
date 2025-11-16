package metrics

import (
	"fmt"
	"time"

	"agate/pkg/common"
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"
	"agate/pkg/session"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

type model struct {
	session   *session.Session
	width     int
	height    int
	startTime time.Time
}

type tickMsg time.Time

func NewModel(sess *session.Session) model {
	return model{
		session:   sess,
		startTime: time.Now(),
	}
}

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tickEvery()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tickEvery()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Create session info
	var sessionInfo string
	if m.session.Description != "" {
		sessionInfo += fmt.Sprintf("Description: %s\n", m.session.Description)
	}
	if m.session.BranchBaseName != "" {
		sessionInfo += fmt.Sprintf("Branch: %s\n", m.session.BranchBaseName)
	}

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		PaddingLeft(2).
		PaddingTop(2)
	styledInfo := infoStyle.Render(sessionInfo)

	// Calculate elapsed time
	elapsed := time.Since(m.startTime)
	minutes := int(elapsed.Minutes())
	seconds := int(elapsed.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d", minutes, seconds)

	// Render each character separately in fixed-width columns to prevent jank
	font := "doom"
	var timerParts []string

	// First pass: find the maximum height across all characters
	maxHeight := 0
	for _, char := range timeStr {
		charFig := figure.NewFigure(string(char), font, false)
		charStr := charFig.String()
		height := lipgloss.Height(charStr)
		if height > maxHeight {
			maxHeight = height
		}
	}

	// Second pass: render each character in fixed-width and fixed-height columns
	for _, char := range timeStr {
		charFig := figure.NewFigure(string(char), font, false)
		charStr := charFig.String()

		// Get the max width for this character (use widest digit "8" for numbers, actual width for ":")
		var maxCharWidth int
		if char >= '0' && char <= '9' {
			// For digits, use width of "8" (typically widest)
			maxFig := figure.NewFigure("8", font, false)
			maxCharWidth = lipgloss.Width(maxFig.String())
		} else {
			// For colon, use actual width
			maxCharWidth = lipgloss.Width(charStr)
		}

		// Render in fixed-width and fixed-height column with vertical centering
		charStyled := lipgloss.NewStyle().
			Width(maxCharWidth).
			Height(maxHeight).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(charStr)

		timerParts = append(timerParts, charStyled)
	}

	// Join all characters horizontally
	timerStr := lipgloss.JoinHorizontal(lipgloss.Left, timerParts...)

	// Style and center the complete timer
	timerStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Render(timerStr)

	// Center the timer in the screen
	timerCentered := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(timerStyled)

	// Create left footer with detach shortcut
	leftFooter := components.RenderShortcutFromBinding(common.GlobalKeys.DetachTmux, components.ShortcutDefault, "")
	leftFooterStyled := lipgloss.NewStyle().
		PaddingLeft(2).
		Render(leftFooter)

	// Create right footer with branding (reversed order: agate.sh | Run... | Agate)
	// Use muted color to match the left side shortcut style
	link := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Underline(true).
		Render("agate.sh")

	pipe := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Render(" | ")

	tagline := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)).
		Render("Run and evaluate any CLI coding agent")

	agateName := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.AgateColor)).
		Bold(true).
		Render("Agate")

	rightFooter := lipgloss.JoinHorizontal(lipgloss.Left, link, pipe, tagline, pipe, agateName)
	rightFooterStyled := lipgloss.NewStyle().
		PaddingRight(2).
		Render(rightFooter)

	// Join left and right footers with spacer
	spacerWidth := m.width - lipgloss.Width(leftFooterStyled) - lipgloss.Width(rightFooterStyled)
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	brandFooterStyled := lipgloss.NewStyle().
		PaddingBottom(1).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, leftFooterStyled, spacer, rightFooterStyled))

	// Layout: session info at top, timer in center, branding at bottom
	topSection := styledInfo
	middleSection := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(topSection) - lipgloss.Height(brandFooterStyled) - lipgloss.Height(timerCentered)).
		Render("")

	return lipgloss.JoinVertical(lipgloss.Left, topSection, middleSection, timerCentered, brandFooterStyled)
}

// Run starts the metrics TUI
func Run(sess *session.Session) error {
	p := tea.NewProgram(NewModel(sess), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
