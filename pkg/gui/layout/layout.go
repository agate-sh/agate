package layout

import (
	"agate/pkg/app"
	"agate/pkg/gui/components"
	"agate/pkg/gui/icons"
	"agate/pkg/gui/theme"
	"agate/pkg/tmux"
	"runtime"
	"strings"

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
	SessionIndex   int // 0-3 for pinned sessions
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
	return FocusState{PaneType: PaneTypeAgents, SessionIndex: 0, SessionSubPane: SubPaneTmux}
}

func NewSessionFocus(sessionIndex int, subPane SubPane) FocusState {
	return FocusState{PaneType: PaneTypeSession, SessionIndex: sessionIndex, SessionSubPane: subPane}
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
	BottomSpacerRows   = 1
	PaneTitleRows      = 1
	FooterRows         = 1
	BottomMarginRows   = 1
	HorizontalMargin   = 2
	HorizontalGapWidth = 2
)

// GridCell represents the dimensions and position of a session pane in the grid
type GridCell struct {
	Width  int
	Height int
	Row    int // 0 or 1
	Col    int // 0 or 1
}

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

	// Grid layout for pinned sessions (up to 4)
	gridCells       []GridCell
	numGridSessions int
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
func (l *Layout) RenderPanes(leftContent, tmuxContent, gitContent string, focus FocusState, isLoading bool, loadingState *tmux.LoadingState) (string, string, string) {
	// Determine which panes are focused
	leftStyle := components.PaneBaseStyle
	tmuxStyle := components.PaneBaseStyle
	gitStyle := components.PaneBaseStyle

	// Apply focus styling - only the actively focused pane gets colored border
	if focus.IsAgentsFocus() {
		leftStyle = leftStyle.BorderForeground(lipgloss.Color(app.GetCurrentAgentColor()))
	} else if focus.IsTmuxFocus() {
		tmuxStyle = tmuxStyle.BorderForeground(lipgloss.Color(app.GetCurrentAgentColor()))
	} else if focus.IsGitFocus() {
		gitStyle = gitStyle.BorderForeground(lipgloss.Color(app.GetCurrentAgentColor()))
	}

	// Correct approach: Apply Width() first, then PlaceVertical
	// Calculate the content height (excluding borders and padding)
	frameHeight := components.PaneBaseStyle.GetVerticalFrameSize()
	contentHeight := l.paneHeight - frameHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	horizontalPadding := components.PaneContentHorizontalPadding() * 2
	leftFullWidth := l.leftContentWidth + horizontalPadding
	tmuxFullWidth := l.tmuxContentWidth + horizontalPadding
	gitFullWidth := l.gitContentWidth + horizontalPadding

	// Inner widths without padding
	tmuxContentWidth := l.tmuxContentWidth

	// Ensure content includes horizontal padding unless the pane already accounted for it
	if lipgloss.Width(leftContent) < leftFullWidth {
		leftContent = components.ApplyPaneContentPadding(leftContent, l.leftContentWidth)
	}
	if lipgloss.Width(tmuxContent) < tmuxFullWidth {
		tmuxContent = components.ApplyPaneContentPadding(tmuxContent, l.tmuxContentWidth)
	}
	if lipgloss.Width(gitContent) < gitFullWidth {
		gitContent = components.ApplyPaneContentPadding(gitContent, l.gitContentWidth)
	}

	leftWrapped := lipgloss.NewStyle().
		Width(leftFullWidth).
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
			app.GetCurrentAgentName(), app.GetCurrentAgentColor(), tmuxContentWidth, contentHeight, theme.TextMuted, theme.TextDescription,
		)
	} else {
		// Use normal tmux content
		tmuxWrapped := lipgloss.NewStyle().
			Width(tmuxFullWidth).
			MaxHeight(contentHeight).
			Render(tmuxContent)
		tmuxContentToRender = lipgloss.PlaceVertical(contentHeight, lipgloss.Top, tmuxWrapped)
	}
	if lipgloss.Width(tmuxContentToRender) < tmuxFullWidth {
		tmuxContentToRender = components.ApplyPaneContentPadding(tmuxContentToRender, l.tmuxContentWidth)
	}

	tmuxPane := tmuxStyle.
		Height(l.paneHeight - 2).
		Render(tmuxContentToRender)

	gitContentHeight := l.gitPaneHeight - frameHeight
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

// GetWidth returns the layout width
func (l *Layout) GetWidth() int {
	return l.width
}

// GetHeight returns the layout height
func (l *Layout) GetHeight() int {
	return l.height
}

// CalculateGridLayout calculates grid cell dimensions for n pinned sessions (1-4)
func (l *Layout) CalculateGridLayout(numSessions int) {
	if numSessions < 1 || numSessions > 4 {
		l.gridCells = nil
		l.numGridSessions = 0
		return
	}

	l.numGridSessions = numSessions

	// Calculate available space for the right section (sessions grid)
	chromeHeight := TopPaddingRows + BottomSpacerRows + PaneTitleRows + FooterRows + BottomMarginRows
	availableHeight := l.height - chromeHeight

	totalHorizontalMargins := HorizontalMargin*2 + HorizontalGapWidth
	usableWidth := l.width - totalHorizontalMargins
	if usableWidth < 0 {
		usableWidth = 0
	}

	// Calculate left pane width (25% of usable width)
	frameWidth := components.PaneBaseStyle.GetHorizontalFrameSize()
	contentPaddingWidth := components.PaneContentHorizontalPadding() * 2

	totalChromeWidth := (frameWidth + contentPaddingWidth) * 2 // left pane + right section
	availableContentWidth := usableWidth - totalChromeWidth
	if availableContentWidth < 0 {
		availableContentWidth = 0
	}

	leftContentWidth := int(float64(availableContentWidth) * 0.25)
	rightContentWidth := availableContentWidth - leftContentWidth
	if rightContentWidth < 0 {
		rightContentWidth = 0
	}

	// Update left pane dimensions
	l.leftContentWidth = leftContentWidth
	l.leftWidth = l.leftContentWidth + contentPaddingWidth + frameWidth

	// Calculate the full width (including chrome) available for the grid section
	rightFullWidth := usableWidth - l.leftWidth
	if rightFullWidth < 0 {
		rightFullWidth = 0
	}

	// Calculate grid cell dimensions based on number of sessions
	l.gridCells = make([]GridCell, numSessions)

	switch numSessions {
	case 1:
		// Full right side
		l.gridCells[0] = GridCell{
			Width:  rightFullWidth,
			Height: availableHeight,
			Row:    0,
			Col:    0,
		}

	case 2:
		// Horizontal split (left/right columns)
		halfWidth := rightFullWidth / 2
		l.gridCells[0] = GridCell{
			Width:  halfWidth,
			Height: availableHeight,
			Row:    0,
			Col:    0,
		}
		l.gridCells[1] = GridCell{
			Width:  rightFullWidth - halfWidth,
			Height: availableHeight,
			Row:    0,
			Col:    1,
		}

	case 3:
		// Top row: 2 columns, Bottom row: spans full width
		halfWidth := rightFullWidth / 2
		halfHeight := availableHeight / 2

		l.gridCells[0] = GridCell{
			Width:  halfWidth,
			Height: halfHeight,
			Row:    0,
			Col:    0,
		}
		l.gridCells[1] = GridCell{
			Width:  rightFullWidth - halfWidth,
			Height: halfHeight,
			Row:    0,
			Col:    1,
		}
		l.gridCells[2] = GridCell{
			Width:  rightFullWidth,
			Height: availableHeight - halfHeight,
			Row:    1,
			Col:    0,
		}

	case 4:
		// 2x2 grid
		halfWidth := rightFullWidth / 2
		halfHeight := availableHeight / 2

		l.gridCells[0] = GridCell{
			Width:  halfWidth,
			Height: halfHeight,
			Row:    0,
			Col:    0,
		}
		l.gridCells[1] = GridCell{
			Width:  rightFullWidth - halfWidth,
			Height: halfHeight,
			Row:    0,
			Col:    1,
		}
		l.gridCells[2] = GridCell{
			Width:  halfWidth,
			Height: availableHeight - halfHeight,
			Row:    1,
			Col:    0,
		}
		l.gridCells[3] = GridCell{
			Width:  rightFullWidth - halfWidth,
			Height: availableHeight - halfHeight,
			Row:    1,
			Col:    1,
		}
	}
}

// GetGridCell returns the grid cell dimensions for a specific session index
func (l *Layout) GetGridCell(index int) *GridCell {
	if index < 0 || index >= len(l.gridCells) {
		return nil
	}
	return &l.gridCells[index]
}

// GetNumGridSessions returns the number of grid sessions currently laid out
func (l *Layout) GetNumGridSessions() int {
	return l.numGridSessions
}

// SessionPaneContent contains the content for a single session pane (tabbed tmux/git)
type SessionPaneContent struct {
	TmuxContent  string
	GitContent   string
	BranchName   string
	AgentColor   string
	IsFocused    bool
	ActiveTab    int // 0 = tmux, 1 = git
	ChangesCount int
}

// RenderGridPanes renders agents pane + grid of session panes
func (l *Layout) RenderGridPanes(agentsContent string, sessions []SessionPaneContent, focus FocusState) string {
	if l.numGridSessions == 0 || len(sessions) == 0 {
		// No sessions - just render empty state or single session fallback
		return ""
	}

	// Calculate dimensions
	chromeHeight := TopPaddingRows + BottomSpacerRows + PaneTitleRows + FooterRows + BottomMarginRows
	availableHeight := l.height - chromeHeight
	frameHeight := components.PaneBaseStyle.GetVerticalFrameSize()
	contentPaddingWidth := components.PaneContentHorizontalPadding() * 2

	// Render agents pane (left)
	agentsStyle := components.PaneBaseStyle
	if focus.IsAgentsFocus() {
		agentsStyle = agentsStyle.BorderForeground(lipgloss.Color(theme.BorderActive))
	}

	leftContentHeight := availableHeight - frameHeight
	if leftContentHeight < 1 {
		leftContentHeight = 1
	}
	leftFullWidth := l.leftContentWidth + contentPaddingWidth

	if lipgloss.Width(agentsContent) < leftFullWidth {
		agentsContent = components.ApplyPaneContentPadding(agentsContent, l.leftContentWidth)
	}

	agentsWrapped := lipgloss.NewStyle().
		Width(leftFullWidth).
		MaxHeight(leftContentHeight).
		Render(agentsContent)
	agentsContentAligned := lipgloss.PlaceVertical(leftContentHeight, lipgloss.Top, agentsWrapped)
	agentPaneHeight := availableHeight - components.PaneContentVerticalPadding()*2
	if agentPaneHeight < 1 {
		agentPaneHeight = availableHeight
	}
	agentsPane := agentsStyle.
		Height(agentPaneHeight).
		Render(agentsContentAligned)

	// Render each session pane with tabbed interface
	renderedCells := make([]string, len(sessions))
	for i, session := range sessions {
		if i >= len(l.gridCells) {
			break
		}

		cell := l.gridCells[i]

		tabbedPane := components.NewTabbedPane(buildSessionTabs(session))
		// Set the tabbed pane size to cell dimensions
		tabbedPane.SetSize(cell.Width, cell.Height)
		tabbedPane.SetActiveTab(session.ActiveTab)

		// Apply agent color to border if focused
		borderColor := theme.BorderMuted
		if session.IsFocused {
			borderColor = session.AgentColor
		}
		tabbedPane.SetBorderColor(borderColor)

		renderedCells[i] = tabbedPane.View()
	}

	// Arrange cells in grid based on layout pattern
	var rightSection string
	switch l.numGridSessions {
	case 1:
		// Full right side
		rightSection = renderedCells[0]

	case 2:
		// Horizontal split (left/right columns)
		rightSection = lipgloss.JoinHorizontal(lipgloss.Top, renderedCells[0], renderedCells[1])

	case 3:
		// Top row: 2 columns, Bottom row: full width
		topRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedCells[0], renderedCells[1])
		rightSection = lipgloss.JoinVertical(lipgloss.Top, topRow, renderedCells[2])

	case 4:
		// 2x2 grid
		topRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedCells[0], renderedCells[1])
		bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedCells[2], renderedCells[3])
		rightSection = lipgloss.JoinVertical(lipgloss.Top, topRow, bottomRow)
	}

	// Join agents pane and grid
	gap := lipgloss.NewStyle().Width(HorizontalGapWidth).Render("")
	panes := lipgloss.JoinHorizontal(lipgloss.Top, agentsPane, gap, rightSection)

	// Add padding
	return lipgloss.NewStyle().
		PaddingTop(TopPaddingRows).
		PaddingBottom(BottomSpacerRows).
		PaddingLeft(HorizontalMargin).
		PaddingRight(HorizontalMargin).
		Render(panes)
}

func buildSessionTabs(session SessionPaneContent) []components.Tab {
	branchLabel := icons.GetGitRepo() + " " + defaultBranchLabel(session.BranchName)
	changesLabel := "Changes"

	badge := components.RenderChangeCountBadge(session.ChangesCount)
	if badge != "" {
		changesLabel = changesLabel + " " + badge
	}

	shortcut := renderTabShortcut()
	branchTrailing := ""
	changesTrailing := ""
	if shortcut != "" {
		if session.ActiveTab == 0 {
			changesTrailing = shortcut
		} else {
			branchTrailing = shortcut
		}
	}

	return []components.Tab{
		{Name: branchLabel, Content: session.TmuxContent, Trailing: branchTrailing},
		{Name: changesLabel, Content: session.GitContent, Trailing: changesTrailing},
	}
}

func defaultBranchLabel(name string) string {
	if strings.TrimSpace(name) == "" {
		return "Session"
	}
	return name
}

func renderTabShortcut() string {
	modifier := optionKeyIcon()
	if modifier == "" {
		return ""
	}

	shortcut := modifier + " + tab"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.InfoStatus)).
		Bold(true).
		Render(shortcut)
}

func optionKeyIcon() string {
	if runtime.GOOS == "darwin" {
		return "⌥"
	}
	// Use Alternate key symbol for non-mac platforms
	return "⎇"
}

// RightGapHeight returns the vertical gap between git and shell panes
