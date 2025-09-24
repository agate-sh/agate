package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestLayoutPanesMatchConfiguredHeights(t *testing.T) {
	layout := NewLayout(160, 60)
	sample := strings.Repeat("item\n", 30)

	left, tmux, git, shell := layout.RenderPanes(sample, sample, sample, sample, focusTmux, "213", false, nil, "")

	if got := lipgloss.Height(left); got != layout.paneHeight {
		t.Fatalf("left pane height = %d want %d", got, layout.paneHeight)
	}
	if got := lipgloss.Height(tmux); got != layout.paneHeight {
		t.Fatalf("tmux pane height = %d want %d", got, layout.paneHeight)
	}
	if got := lipgloss.Height(git); got != layout.gitPaneHeight {
		t.Fatalf("git pane height = %d want %d", got, layout.gitPaneHeight)
	}
	if got := lipgloss.Height(shell); got != layout.shellPaneHeight {
		t.Fatalf("shell pane height = %d want %d", got, layout.shellPaneHeight)
	}

	leftTitle := lipgloss.NewStyle().PaddingLeft(1).Render("Left")
	tmuxTitle := lipgloss.NewStyle().PaddingLeft(1).Render("Tmux")
	gitTitle := lipgloss.NewStyle().PaddingLeft(1).Render("Git")
	shellTitle := lipgloss.NewStyle().PaddingLeft(1).Render("Shell")

	leftWithTitle := lipgloss.JoinVertical(lipgloss.Left, leftTitle, left)
	tmuxWithTitle := lipgloss.JoinVertical(lipgloss.Left, tmuxTitle, tmux)
	gitWithTitle := lipgloss.JoinVertical(lipgloss.Left, gitTitle, git)
	shellWithTitle := lipgloss.JoinVertical(lipgloss.Left, shellTitle, shell)

	gap := strings.Repeat(" ", horizontalGapWidth)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftWithTitle, gap, tmuxWithTitle, gap, lipgloss.JoinVertical(lipgloss.Top, gitWithTitle, shellWithTitle))
	panesWithPadding := lipgloss.NewStyle().
		PaddingTop(topPaddingRows).
		PaddingBottom(bottomSpacerRows).
		PaddingLeft(horizontalMargin).
		PaddingRight(horizontalMargin).
		Render(panes)

	if w := lipgloss.Width(panesWithPadding); w != layout.width {
		t.Fatalf("pane block width = %d want %d", w, layout.width)
	}

	var bottomComponents []string
	bottomComponents = append(bottomComponents, panesWithPadding)
	bottomComponents = append(bottomComponents, strings.Repeat("-", layout.width))
	for i := 0; i < bottomMarginRows; i++ {
		bottomComponents = append(bottomComponents, "")
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left, bottomComponents...)
	if got := lipgloss.Height(mainView); got != layout.height {
		t.Fatalf("main view height = %d want %d", got, layout.height)
	}
	if got := lipgloss.Width(mainView); got != layout.width {
		t.Fatalf("main view width = %d want %d", got, layout.width)
	}
}
