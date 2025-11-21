package tui

import (
	"strings"

	"agate/pkg/overlay"
	"agate/pkg/tui/components"
	"agate/pkg/tui/layout"
	"agate/pkg/tui/theme"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// View renders the TUI
func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.sessionManager == nil {
		return "No session manager"
	}

	var panesWithPadding string
	var (
		newSessionPaneLeft   int
		newSessionPaneTop    int
		newSessionPaneWidth  int
		newSessionPaneHeight int
		hasNewSessionPane    bool
	)

	// Handle different UI states

	// State 1: Showing new session input (no sessions or user pressed 'n')
	if m.ShowNewSessionInput() {
		// Calculate layout dimensions
		chromeHeight := layout.TopPaddingRows + layout.BottomSpacerRows + layout.PaneTitleRows + layout.BottomMarginRows
		availableHeight := m.layout.GetHeight() - chromeHeight
		leftWidth, leftHeight := m.layout.GetLeftDimensions()
		frameHeight := components.PaneBaseStyle.GetVerticalFrameSize()
		contentPaddingWidth := components.PaneContentHorizontalPadding() * 2

		// Render agents pane on left (may be empty or show existing sessions)
		// When in new session input view, no panes should be highlighted/active
		agentsContent := m.repoPane.View()
		agentsStyle := components.PaneBaseStyle
		// Don't highlight agents pane in new session input view

		leftFullWidth := leftWidth + contentPaddingWidth
		if lipgloss.Width(agentsContent) < leftFullWidth {
			agentsContent = components.ApplyPaneContentPadding(agentsContent, leftWidth)
		}

		agentsWrapped := lipgloss.NewStyle().
			Width(leftFullWidth).
			MaxHeight(leftHeight).
			Render(agentsContent)
		agentsContentAligned := lipgloss.PlaceVertical(leftHeight, lipgloss.Top, agentsWrapped)
		agentsPane := agentsStyle.
			Height(availableHeight - 1).
			Render(agentsContentAligned)

		// Render center pane with header (ASCII art + version + shortcuts) + chat input
		totalHorizontalMargins := layout.HorizontalMargin*2 + layout.HorizontalGapWidth*2
		usableWidth := m.layout.GetWidth() - totalHorizontalMargins
		centerWidth := usableWidth - leftWidth - contentPaddingWidth - frameHeight

		// Render header with ASCII art, version, and shortcuts
		if m.welcomeHeader != nil {
			m.welcomeHeader.SetWidth(centerWidth - 4)
		}
		headerView := ""
		if m.welcomeHeader != nil {
			headerView = m.welcomeHeader.View()
		}

		// Render chat input centered in middle
		if m.chatInput != nil {
			m.chatInput.SetWidth(min(80, centerWidth-8))
		}
		chatInputView := ""
		if m.chatInput != nil {
			chatInputView = m.chatInput.View()
		}

		// Combine header + spacing + chat input
		var centerParts []string
		centerParts = append(centerParts, headerView)
		centerParts = append(centerParts, "")
		centerParts = append(centerParts, "")
		centerParts = append(centerParts, chatInputView)

		centerContent := strings.Join(centerParts, "\n")

		// Center the content vertically and horizontally
		placeholderContentHeight := availableHeight - frameHeight
		centeredContent := lipgloss.Place(
			centerWidth,
			placeholderContentHeight,
			lipgloss.Center,
			lipgloss.Center,
			centerContent,
		)

		// Render center pane with border (highlighted since it's the active/focused pane)
		centerPaneStyle := components.PaneBaseStyle.
			BorderForeground(lipgloss.Color(theme.BorderActive))
		centerPane := centerPaneStyle.
			Height(availableHeight - 1).
			Render(centeredContent)

		// Render pane titles with padding
		leftTitle := lipgloss.NewStyle().PaddingLeft(1).Render(m.renderPaneTitle(m.repoPane))
		// For new session input, show "New Session" as the title
		titleText := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Render("New Session")
		shortcut := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)).
			Render("(⌥n)")
		centerTitle := lipgloss.NewStyle().PaddingLeft(1).Render(titleText + " " + shortcut)

		// Join titles with panes vertically (title above pane)
		leftWithTitle := lipgloss.JoinVertical(lipgloss.Left, leftTitle, agentsPane)
		centerWithTitle := lipgloss.JoinVertical(lipgloss.Left, centerTitle, centerPane)

		// Track bounds of the new session pane so overlays can align to it later
		hasNewSessionPane = true
		leftBlockWidth := lipgloss.Width(leftWithTitle)
		centerBlockWidth := lipgloss.Width(centerWithTitle)
		newSessionPaneLeft = layout.HorizontalMargin + leftBlockWidth + layout.HorizontalGapWidth
		newSessionPaneTop = layout.TopPaddingRows + layout.PaneTitleRows
		newSessionPaneWidth = centerBlockWidth
		newSessionPaneHeight = lipgloss.Height(centerPane)

		// Join agents pane and center pane (right pane hidden)
		gap := lipgloss.NewStyle().Width(layout.HorizontalGapWidth).Render("")
		panes := lipgloss.JoinHorizontal(lipgloss.Top, leftWithTitle, gap, centerWithTitle)

		// Add padding
		panesWithPadding = lipgloss.NewStyle().
			PaddingTop(layout.TopPaddingRows).
			PaddingBottom(layout.BottomSpacerRows).
			PaddingLeft(layout.HorizontalMargin).
			PaddingRight(layout.HorizontalMargin).
			Render(panes)

	} else {
		// State 2: Active session - always render 3-pane layout: Agents | Session | Changes

		// Render pane titles with padding
		leftTitle := lipgloss.NewStyle().PaddingLeft(1).Render(m.renderPaneTitle(m.repoPane))
		centerTitle := lipgloss.NewStyle().PaddingLeft(1).Render(m.renderPaneTitle(m.sessionViewPane))
		rightTitle := lipgloss.NewStyle().PaddingLeft(1).Render(m.renderPaneTitle(m.changesPane))

		// Render pane content
		agentsContent := m.repoPane.View()
		sessionContent := m.sessionViewPane.View()
		changesContent := m.changesPane.View()

		defaultPadding := components.PaneContentVerticalPadding()

		leftPadTop, leftPadBottom := defaultPadding, defaultPadding
		if m.repoPane != nil {
			leftPadTop, leftPadBottom = m.repoPane.GetChromePadding()
		}

		centerPadTop, centerPadBottom := defaultPadding, defaultPadding
		if m.sessionViewPane != nil {
			centerPadTop, centerPadBottom = m.sessionViewPane.GetChromePadding()
		}

		rightPadTop, rightPadBottom := defaultPadding, defaultPadding
		if m.changesPane != nil {
			rightPadTop, rightPadBottom = m.changesPane.GetChromePadding()
		}

		// Use layout's RenderPanes to render the 3-column layout
		leftPane, tmuxPane, gitPane := m.layout.RenderPanes(
			layout.PaneRenderParams{
				Content:       agentsContent,
				PaddingTop:    leftPadTop,
				PaddingBottom: leftPadBottom,
			},
			layout.PaneRenderParams{
				Content:       sessionContent,
				PaddingTop:    centerPadTop,
				PaddingBottom: centerPadBottom,
			},
			layout.PaneRenderParams{
				Content:       changesContent,
				PaddingTop:    rightPadTop,
				PaddingBottom: rightPadBottom,
			},
			m.state.Focus,
			false, // isLoading - handled by SessionViewPane internally
			nil,   // loadingState - not needed since SessionViewPane handles it
		)

		// Join titles with panes vertically (title above pane)
		leftWithTitle := lipgloss.JoinVertical(lipgloss.Left, leftTitle, leftPane)
		tmuxWithTitle := lipgloss.JoinVertical(lipgloss.Left, centerTitle, tmuxPane)
		gitWithTitle := lipgloss.JoinVertical(lipgloss.Left, rightTitle, gitPane)

		// Join the three panes horizontally
		gap := lipgloss.NewStyle().Width(layout.HorizontalGapWidth).Render("")
		panes := lipgloss.JoinHorizontal(lipgloss.Top, leftWithTitle, gap, tmuxWithTitle, gap, gitWithTitle)

		// Add padding
		panesWithPadding = lipgloss.NewStyle().
			PaddingTop(layout.TopPaddingRows).
			PaddingBottom(layout.BottomSpacerRows).
			PaddingLeft(layout.HorizontalMargin).
			PaddingRight(layout.HorizontalMargin).
			Render(panes)
	}

	// Add bottom margin
	var bottomComponents []string
	bottomComponents = append(bottomComponents, panesWithPadding)
	for i := 0; i < layout.BottomMarginRows; i++ {
		bottomComponents = append(bottomComponents, "")
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left, bottomComponents...)

	// Overlay toast notifications if visible
	if m.toast != nil && m.toast.IsVisible() {
		mainView = m.toast.PlaceOverlay(mainView, m.layout.GetWidth(), m.layout.GetHeight())
	}

	// If help dialog is visible, overlay it
	if m.state.ActiveOverlay == HelpOverlay {
		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.helpDialog.View(), mainView, true, true))
	}

	// If debug overlay is visible, overlay it (high priority)
	if m.state.ActiveOverlay == DebugOverlay && m.debugOverlay != nil {
		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.debugOverlay.View(), mainView, true, true))
	}

	// If session deletion confirmation is visible, overlay it
	if m.state.ActiveOverlay == SessionDeleteOverlay && m.sessionConfirm != nil {
		// Update dialog size
		m.sessionConfirm.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.sessionConfirm.View(), mainView, true, true))
	}

	// If merge overlay is visible, overlay it
	if m.state.ActiveOverlay == MergeOverlay && m.mergeOverlay != nil {
		// Update dialog size
		m.mergeOverlay.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.mergeOverlay.View(), mainView, true, true))
	}

	// If agent selector is visible, overlay it
	if m.state.ActiveOverlay == AgentSelectorOverlay && m.agentSelector != nil {
		selectorWidth := m.layout.GetWidth()
		selectorHeight := m.layout.GetHeight()

		if m.ShowNewSessionInput() && hasNewSessionPane && newSessionPaneWidth > 0 && newSessionPaneHeight > 0 {
			selectorWidth = newSessionPaneWidth
			selectorHeight = newSessionPaneHeight
		}

		m.agentSelector.SetSize(selectorWidth, selectorHeight)
		selectorView := m.agentSelector.View()

		if m.ShowNewSessionInput() && hasNewSessionPane && newSessionPaneWidth > 0 && newSessionPaneHeight > 0 {
			overlayWidth := lipgloss.Width(selectorView)
			overlayHeight := lipgloss.Height(selectorView)

			overlayX := newSessionPaneLeft
			if overlayWidth < newSessionPaneWidth {
				overlayX += (newSessionPaneWidth - overlayWidth) / 2
			}

			overlayY := newSessionPaneTop
			if overlayHeight < newSessionPaneHeight {
				overlayY += (newSessionPaneHeight - overlayHeight) / 2
			}

			return zone.Scan(overlay.PlaceOverlay(overlayX, overlayY, selectorView, mainView, true, false))
		}

		// Default to centering on the entire screen
		return zone.Scan(overlay.PlaceOverlay(0, 0, selectorView, mainView, true, true))
	}

	// Render toast notifications (always rendered last, on top of everything)
	// Toasts do NOT dim the background and do NOT block interaction
	if m.toast != nil && m.toast.IsVisible() {
		return zone.Scan(m.toast.PlaceOverlay(mainView, m.layout.GetWidth(), m.layout.GetHeight()))
	}

	return zone.Scan(mainView)
}

// renderPaneTitle renders the title for a pane with appropriate styling
func (m *Model) renderPaneTitle(pane components.Pane) string {
	if pane == nil {
		return ""
	}

	titleStyle := pane.GetTitleStyle()

	// Style the text based on the title type
	var styledText string
	if titleStyle.Type == "badge" {
		// Badge style (like tmux pane) with colored background
		var backgroundColor string
		if pane.IsActive() {
			// When active, use the agent's brand color
			backgroundColor = titleStyle.Color
		} else {
			// When inactive, use very muted color to blend into background
			backgroundColor = theme.SeparatorColor
		}

		badgeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(backgroundColor)).
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Padding(0, 1).
			Bold(true)
		styledText = badgeStyle.Render(titleStyle.Text)
	} else {
		// Plain style
		var textStyle lipgloss.Style
		if pane.IsActive() {
			textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary)).Bold(true)
		} else {
			textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
		}
		styledText = textStyle.Render(titleStyle.Text)
	}

	// Add shortcuts with appropriate styling
	if titleStyle.Shortcuts != "" {
		if pane.IsActive() {
			// When active, put formatted shortcuts in parentheses
			formattedShortcuts := m.parseAndStyleShortcuts(titleStyle.Shortcuts)

			// Style the parentheses consistently
			parenStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.TextMuted))

			leftParen := parenStyle.Render("(")
			rightParen := parenStyle.Render(")")
			return styledText + " " + leftParen + formattedShortcuts + rightParen
		} else {
			// When inactive, show pane number in parentheses
			shortcutStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.TextMuted))
			return styledText + " " + shortcutStyle.Render(titleStyle.Shortcuts)
		}
	}

	return styledText
}

// parseAndStyleShortcuts parses shortcut strings and applies default bubbles styling
func (m *Model) parseAndStyleShortcuts(shortcuts string) string {
	// Use the shortcut component's ParseAndRenderShortcuts function with default variant
	return components.ParseAndRenderShortcuts(shortcuts, components.ShortcutDefault, "")
}
