package layout

import (
	"agate/pkg/gui/theme"
	"agate/tmux"

	"github.com/charmbracelet/lipgloss"
)

var (
	PaneBaseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderMuted)).
		Padding(1, 2)
)

type FocusState int

const (
	FocusReposAndWorktrees FocusState = iota
	FocusTmux
	FocusGit
	FocusShell
)

// String returns the string representation of the focus state
func (f FocusState) String() string {
	switch f {
	case FocusReposAndWorktrees:
		return "reposAndWorktrees"
	case FocusTmux:
		return "tmux"
	case FocusGit:
		return "git"
	case FocusShell:
		return "shell"
	default:
		return "unknown"
	}
}

const (
	TopPaddingRows     = 1
	BottomSpacerRows   = 1
	PaneTitleRows      = 1
	FooterRows         = 1
	BottomMarginRows   = 1
	HorizontalMargin   = 2
	HorizontalGapWidth = 2
)

// Layout manages the pane layout and dimensions for the UI
type Layout struct {
	width  int
	height int

	// Content dimensions (without borders)
	leftContentWidth  int
	tmuxContentWidth  int
	gitContentWidth   int
	shellContentWidth int
	contentHeight     int

	// Full pane dimensions (with borders)
	leftWidth  int
	tmuxWidth  int
	gitWidth   int
	shellWidth int
	paneHeight int

	// Split pane heights for right section
	gitPaneHeight   int
	shellPaneHeight int
}

// NewLayout creates a new layout with the given terminal dimensions
func NewLayout(width, height int) *Layout {
	l := &Layout{
		width:  width,
		height: height,
	}
	l.calculate()
	return l
}

// Update recalculates the layout for new terminal dimensions
func (l *Layout) Update(width, height int) {
	l.width = width
	l.height = height
	l.calculate()
}

// calculate computes all pane dimensions based on terminal size
func (l *Layout) calculate() {
	// Reserve space for non-pane rows (top padding, titles, footer spacing)
	chromeHeight := TopPaddingRows + BottomSpacerRows + PaneTitleRows + FooterRows + BottomMarginRows
	availableHeight := l.height - chromeHeight

	totalHorizontalMargins := HorizontalMargin*2 + HorizontalGapWidth*2
	usableWidth := l.width - totalHorizontalMargins
	if usableWidth < 0 {
		usableWidth = 0
	}

	// Get frame dimensions from pane style
	frameWidth := PaneBaseStyle.GetHorizontalFrameSize()
	frameHeight := PaneBaseStyle.GetVerticalFrameSize()
	minPaneHeight := frameHeight + 1 // At least one line of content inside the frame
	if availableHeight < minPaneHeight {
		availableHeight = minPaneHeight
	}

	// We have 3 main columns: left, tmux, and the stacked right column
	// Subtract the frame width for each column to get available content width
	totalFrameWidth := frameWidth * 3

	// Calculate available content width
	availableContentWidth := usableWidth - totalFrameWidth
	if availableContentWidth < 0 {
		availableContentWidth = 0
	}

	// Split content: 25% left, 50% tmux, 25% right
	l.leftContentWidth = int(float64(availableContentWidth) * 0.25)
	l.tmuxContentWidth = int(float64(availableContentWidth) * 0.50)
	rightSectionWidth := availableContentWidth - l.leftContentWidth - l.tmuxContentWidth

	// Git and Shell share the right section width
	l.gitContentWidth = rightSectionWidth
	l.shellContentWidth = rightSectionWidth

	// Calculate full pane widths (with borders)
	l.leftWidth = l.leftContentWidth + frameWidth
	l.tmuxWidth = l.tmuxContentWidth + frameWidth
	l.gitWidth = l.gitContentWidth + frameWidth
	l.shellWidth = l.shellContentWidth + frameWidth

	// Calculate heights
	l.paneHeight = availableHeight
	l.contentHeight = availableHeight - frameHeight
	if l.contentHeight < 1 {
		l.contentHeight = 1
	}

	// RIGHT SECTION CALCULATION - Different from left/tmux because it has 2 titles + 2 panes
	// The right section total height (git + shell + their titles) must equal left/tmux total height
	//
	// Left/Tmux structure: 1 title + 1 pane = availableHeight + title
	// Right structure: 2 titles + 2 panes = availableHeight + title (to match left/tmux)
	//
	// So: rightContentHeight = availableHeight - 1 title line (since we need to account for the extra title)
	rightContentHeight := availableHeight - PaneTitleRows

	// Split the remaining content height between git and shell panes
	halfHeight := rightContentHeight / 2
	l.gitPaneHeight = halfHeight
	l.shellPaneHeight = rightContentHeight - halfHeight

	// Ensure minimum heights for right column panes
	const minRightPaneHeight = 3
	if l.gitPaneHeight < minRightPaneHeight {
		l.gitPaneHeight = minRightPaneHeight
		l.shellPaneHeight = rightContentHeight - minRightPaneHeight
	}
	if l.shellPaneHeight < minRightPaneHeight {
		l.shellPaneHeight = minRightPaneHeight
		l.gitPaneHeight = rightContentHeight - minRightPaneHeight
	}
}

// RenderPanes renders all panes with the given content
func (l *Layout) RenderPanes(leftContent, tmuxContent, gitContent, shellContent string, focused FocusState, agentColor string, isLoading bool, loadingState *tmux.LoadingState, agentName string) (string, string, string, string) {
	// Determine which panes are focused
	leftStyle := PaneBaseStyle
	tmuxStyle := PaneBaseStyle
	gitStyle := PaneBaseStyle
	shellStyle := PaneBaseStyle

	// Apply focus styling
	switch focused {
	case FocusReposAndWorktrees:
		leftStyle = leftStyle.BorderForeground(lipgloss.Color(theme.BorderActive))
	case FocusTmux:
		// Use the agent's specific color when tmux is focused
		tmuxStyle = tmuxStyle.BorderForeground(lipgloss.Color(agentColor))
	case FocusGit:
		gitStyle = gitStyle.BorderForeground(lipgloss.Color(theme.BorderActive))
	case FocusShell:
		shellStyle = shellStyle.BorderForeground(lipgloss.Color(theme.BorderActive))
	}

	// Correct approach: Apply Width() first, then PlaceVertical
	// Calculate the content height (excluding borders and padding)
	frameHeight := PaneBaseStyle.GetVerticalFrameSize()
	contentHeight := l.paneHeight - frameHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	frameWidth := PaneBaseStyle.GetHorizontalFrameSize()
	leftContentWidth := l.leftWidth - frameWidth
	tmuxContentWidth := l.tmuxWidth - frameWidth

	leftWrapped := lipgloss.NewStyle().
		Width(leftContentWidth).
		MaxHeight(contentHeight).
		Render(leftContent)
	leftContentAligned := lipgloss.PlaceVertical(contentHeight, lipgloss.Top, leftWrapped)
	leftPane := leftStyle.
		Height(l.paneHeight - 2).
		Render(leftContentAligned)

	// Handle loading state for tmux pane
	var tmuxContentToRender string
	if isLoading && loadingState != nil {
		// Use the loading state to render the complete loading view
		tmuxContentToRender = loadingState.RenderLoadingView(
			agentName, agentColor, tmuxContentWidth, contentHeight, theme.TextMuted, theme.TextDescription,
		)
	} else {
		// Use normal tmux content
		tmuxWrapped := lipgloss.NewStyle().
			Width(tmuxContentWidth).
			MaxHeight(contentHeight).
			Render(tmuxContent)
		tmuxContentToRender = lipgloss.PlaceVertical(contentHeight, lipgloss.Top, tmuxWrapped)
	}

	tmuxPane := tmuxStyle.
		Height(l.paneHeight - 2).
		Render(tmuxContentToRender)

	gitContentWidth := l.gitWidth - frameWidth
	gitContentHeight := l.gitPaneHeight - frameHeight
	if gitContentHeight < 1 {
		gitContentHeight = 1
	}
	gitWrapped := lipgloss.NewStyle().
		Width(gitContentWidth).
		MaxHeight(gitContentHeight).
		Render(gitContent)
	gitContentAligned := lipgloss.PlaceVertical(gitContentHeight, lipgloss.Top, gitWrapped)
	gitPane := gitStyle.
		Height(l.gitPaneHeight - 2).
		Render(gitContentAligned)

	shellContentWidth := l.shellWidth - frameWidth
	shellContentHeight := l.shellPaneHeight - frameHeight
	if shellContentHeight < 1 {
		shellContentHeight = 1
	}
	shellWrapped := lipgloss.NewStyle().
		Width(shellContentWidth).
		MaxHeight(shellContentHeight).
		Render(shellContent)
	shellContentAligned := lipgloss.PlaceVertical(shellContentHeight, lipgloss.Top, shellWrapped)
	shellPane := shellStyle.
		Height(l.shellPaneHeight - 2).
		Render(shellContentAligned)

	return leftPane, tmuxPane, gitPane, shellPane
}

// GetTmuxDimensions returns the content dimensions for the tmux pane
func (l *Layout) GetTmuxDimensions() (width, height int) {
	return l.tmuxContentWidth, l.contentHeight
}

// GetLeftDimensions returns the content dimensions for the left pane
func (l *Layout) GetLeftDimensions() (width, height int) {
	return l.leftContentWidth, l.contentHeight
}

// GetGitDimensions returns the content dimensions for the git pane
func (l *Layout) GetGitDimensions() (width, height int) {
	frameHeight := PaneBaseStyle.GetVerticalFrameSize()
	gitContentHeight := l.gitPaneHeight - frameHeight
	if gitContentHeight < 1 {
		gitContentHeight = 1
	}
	return l.gitContentWidth, gitContentHeight
}

// GetShellDimensions returns the content dimensions for the shell pane
func (l *Layout) GetShellDimensions() (width, height int) {
	frameHeight := PaneBaseStyle.GetVerticalFrameSize()
	shellContentHeight := l.shellPaneHeight - frameHeight
	if shellContentHeight < 1 {
		shellContentHeight = 1
	}
	return l.shellContentWidth, shellContentHeight
}

// GetWidth returns the layout width
func (l *Layout) GetWidth() int {
	return l.width
}

// GetHeight returns the layout height
func (l *Layout) GetHeight() int {
	return l.height
}

// RightGapHeight returns the vertical gap between git and shell panes
