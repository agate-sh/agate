package layout

import (
	"agate/pkg/app"
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"
	"agate/pkg/tmux"

	"github.com/charmbracelet/lipgloss"
)

type PaneType int

const (
	PaneTypeAgents PaneType = iota
	PaneTypeSession
)

type SubPane int

const (
	SubPaneTmux SubPane = iota
	SubPaneGit
)

type FocusState struct {
	PaneType       PaneType
	SessionSubPane SubPane
}

// String returns the string representation of the focus state
func (f FocusState) String() string {
	switch f.PaneType {
	case PaneTypeAgents:
		return "agents"
	case PaneTypeSession:
		switch f.SessionSubPane {
		case SubPaneTmux:
			return "tmux"
		case SubPaneGit:
			return "git"
		default:
			return "unknown"
		}
	default:
		return "unknown"
	}
}

// Helper functions to create focus states
func NewAgentsFocus() FocusState {
	return FocusState{PaneType: PaneTypeAgents, SessionSubPane: SubPaneTmux}
}

func NewSessionFocus(subPane SubPane) FocusState {
	return FocusState{PaneType: PaneTypeSession, SessionSubPane: subPane}
}

// IsAgentsFocus checks if focus is on the agents pane
func (f FocusState) IsAgentsFocus() bool {
	return f.PaneType == PaneTypeAgents
}

// IsTmuxFocus checks if focus is on a tmux sub-pane
func (f FocusState) IsTmuxFocus() bool {
	return f.PaneType == PaneTypeSession && f.SessionSubPane == SubPaneTmux
}

// IsGitFocus checks if focus is on a git sub-pane
func (f FocusState) IsGitFocus() bool {
	return f.PaneType == PaneTypeSession && f.SessionSubPane == SubPaneGit
}

const (
	TopPaddingRows     = 2
	BottomSpacerRows   = 0
	PaneTitleRows      = 1
	BottomMarginRows   = 1
	HorizontalMargin   = 2
	HorizontalGapWidth = 2
)

// Layout manages the pane layout and dimensions for the UI
type Layout struct {
	width  int
	height int

	// Content dimensions (without borders)
	leftContentWidth int
	tmuxContentWidth int
	gitContentWidth  int
	contentHeight    int

	// Full pane dimensions (with borders)
	leftWidth  int
	tmuxWidth  int
	gitWidth   int
	paneHeight int

	// Split pane heights for right section
	gitPaneHeight int
}

// PaneRenderParams describes how a pane should be rendered within the layout.
type PaneRenderParams struct {
	Content       string
	PaddingTop    int
	PaddingBottom int
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
	chromeHeight := TopPaddingRows + BottomSpacerRows + PaneTitleRows + BottomMarginRows
	availableHeight := l.height - chromeHeight

	totalHorizontalMargins := HorizontalMargin*2 + HorizontalGapWidth*2
	usableWidth := l.width - totalHorizontalMargins
	if usableWidth < 0 {
		usableWidth = 0
	}

	// Get frame dimensions from pane style
	frameWidth := components.PaneBaseStyle.GetHorizontalFrameSize()
	frameHeight := components.PaneBaseStyle.GetVerticalFrameSize()
	contentPaddingWidth := components.PaneContentHorizontalPadding() * 2
	minPaneHeight := frameHeight + 1 // At least one line of content inside the frame
	if availableHeight < minPaneHeight {
		availableHeight = minPaneHeight
	}

	// We have 3 main columns: left, tmux, and the stacked right column
	// Subtract the frame width and internal padding for each column to get available content width
	totalChromeWidth := (frameWidth + contentPaddingWidth) * 3

	// Calculate available content width
	availableContentWidth := usableWidth - totalChromeWidth
	if availableContentWidth < 0 {
		availableContentWidth = 0
	}

	// Split content: 25% left, 50% tmux, 25% git
	l.leftContentWidth = int(float64(availableContentWidth) * 0.25)
	l.tmuxContentWidth = int(float64(availableContentWidth) * 0.50)
	l.gitContentWidth = availableContentWidth - l.leftContentWidth - l.tmuxContentWidth

	// Calculate full pane widths (with borders)
	l.leftWidth = l.leftContentWidth + contentPaddingWidth + frameWidth
	l.tmuxWidth = l.tmuxContentWidth + contentPaddingWidth + frameWidth
	l.gitWidth = l.gitContentWidth + contentPaddingWidth + frameWidth

	// Calculate heights
	l.paneHeight = availableHeight
	l.contentHeight = availableHeight - frameHeight
	if l.contentHeight < 1 {
		l.contentHeight = 1
	}

	// Git pane uses full available height
	l.gitPaneHeight = availableHeight
}

// RenderPanes renders all panes with the given content
func (l *Layout) RenderPanes(left PaneRenderParams, tmux PaneRenderParams, git PaneRenderParams, focus FocusState, isLoading bool, loadingState *tmux.LoadingState) (string, string, string) {
	// Determine which panes are focused
	leftStyle := components.PaneBaseStyle.
		PaddingTop(left.PaddingTop).
		PaddingBottom(left.PaddingBottom)
	tmuxStyle := components.PaneBaseStyle.
		PaddingTop(tmux.PaddingTop).
		PaddingBottom(tmux.PaddingBottom)
	gitStyle := components.PaneBaseStyle.
		PaddingTop(git.PaddingTop).
		PaddingBottom(git.PaddingBottom)

	// Apply focus styling - only the actively focused pane gets active border
	if focus.IsAgentsFocus() {
		leftStyle = leftStyle.BorderForeground(lipgloss.Color(theme.BorderActive))
	} else if focus.IsTmuxFocus() {
		tmuxStyle = tmuxStyle.BorderForeground(lipgloss.Color(theme.BorderActive))
	} else if focus.IsGitFocus() {
		gitStyle = gitStyle.BorderForeground(lipgloss.Color(theme.BorderActive))
	}

	// Calculate content heights (excluding borders and padding) per pane
	leftContentHeight := l.paneHeight - leftStyle.GetVerticalFrameSize()
	if leftContentHeight < 1 {
		leftContentHeight = 1
	}

	tmuxContentHeight := l.paneHeight - tmuxStyle.GetVerticalFrameSize()
	if tmuxContentHeight < 1 {
		tmuxContentHeight = 1
	}

	horizontalPadding := components.PaneContentHorizontalPadding() * 2
	leftFullWidth := l.leftContentWidth + horizontalPadding
	tmuxFullWidth := l.tmuxContentWidth + horizontalPadding
	gitFullWidth := l.gitContentWidth + horizontalPadding

	// Inner widths without padding
	tmuxContentWidth := l.tmuxContentWidth

	// Ensure content includes horizontal padding unless the pane already accounted for it
	leftContent := left.Content
	if lipgloss.Width(leftContent) < leftFullWidth {
		leftContent = components.ApplyPaneContentPadding(leftContent, l.leftContentWidth)
	}

	tmuxContent := tmux.Content
	if lipgloss.Width(tmuxContent) < tmuxFullWidth {
		tmuxContent = components.ApplyPaneContentPadding(tmuxContent, l.tmuxContentWidth)
	}

	gitContent := git.Content
	if lipgloss.Width(gitContent) < gitFullWidth {
		gitContent = components.ApplyPaneContentPadding(gitContent, l.gitContentWidth)
	}

	leftWrapped := lipgloss.NewStyle().
		Width(leftFullWidth).
		MaxHeight(leftContentHeight).
		Render(leftContent)
	leftContentAligned := lipgloss.PlaceVertical(leftContentHeight, lipgloss.Top, leftWrapped)
	leftPane := leftStyle.
		Height(l.paneHeight - 2).
		Render(leftContentAligned)

	// Handle loading state for tmux pane
	var tmuxContentToRender string
	if isLoading && loadingState != nil {
		// Use the loading state to render the complete loading view
		tmuxContentToRender = loadingState.RenderLoadingView(
			app.GetCurrentAgentName(), app.GetCurrentAgentColor(), tmuxContentWidth, tmuxContentHeight, theme.TextMuted, theme.TextDescription,
		)
	} else {
		// Use normal tmux content
		tmuxWrapped := lipgloss.NewStyle().
			Width(tmuxFullWidth).
			MaxHeight(tmuxContentHeight).
			Render(tmuxContent)
		tmuxContentToRender = lipgloss.PlaceVertical(tmuxContentHeight, lipgloss.Top, tmuxWrapped)
	}
	if lipgloss.Width(tmuxContentToRender) < tmuxFullWidth {
		tmuxContentToRender = components.ApplyPaneContentPadding(tmuxContentToRender, l.tmuxContentWidth)
	}

	tmuxPane := tmuxStyle.
		Height(l.paneHeight - 2).
		Render(tmuxContentToRender)

	gitContentHeight := l.gitPaneHeight - gitStyle.GetVerticalFrameSize()
	if gitContentHeight < 1 {
		gitContentHeight = 1
	}
	gitWrapped := lipgloss.NewStyle().
		Width(gitFullWidth).
		MaxHeight(gitContentHeight).
		Render(gitContent)
	gitContentAligned := lipgloss.PlaceVertical(gitContentHeight, lipgloss.Top, gitWrapped)
	gitPane := gitStyle.
		Height(l.gitPaneHeight - 2).
		Render(gitContentAligned)

	return leftPane, tmuxPane, gitPane
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
	frameHeight := components.PaneBaseStyle.GetVerticalFrameSize()
	gitContentHeight := l.gitPaneHeight - frameHeight
	if gitContentHeight < 1 {
		gitContentHeight = 1
	}
	return l.gitContentWidth, gitContentHeight
}

// GetCenterDimensions returns the content dimensions for the center pane (session view)
func (l *Layout) GetCenterDimensions() (width, height int) {
	return l.tmuxContentWidth, l.contentHeight
}

// GetRightDimensions returns the content dimensions for the right pane (changes)
func (l *Layout) GetRightDimensions() (width, height int) {
	return l.gitContentWidth, l.contentHeight
}

// CalculateThreePaneLayout calculates dimensions for the 3-pane system
// Left: ~25% (Agents), Center: ~50% (SessionView), Right: ~25% (Changes)
func (l *Layout) CalculateThreePaneLayout(totalWidth, totalHeight int) {
	l.width = totalWidth
	l.height = totalHeight
	l.calculate()
}

// GetWidth returns the layout width
func (l *Layout) GetWidth() int {
	return l.width
}

// GetHeight returns the layout height
func (l *Layout) GetHeight() int {
	return l.height
}
