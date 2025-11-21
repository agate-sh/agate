package tui

import (
	"fmt"
	"time"

	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/session"
	"agate/pkg/tui/components"
	"agate/pkg/tui/overlays"
	"agate/pkg/tui/panes"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all messages and routes them to appropriate handlers
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global messages first (messages that apply to all modes)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tmuxSessionStartedMsg:
		// Store the session (msg.session is now a *session.Session)
		activeSession := msg.session

		// Set the current agent based on the session's agent
		app.SetCurrentAgent(activeSession.Agent())

		// Initialize session view pane with the session
		if m.sessionViewPane != nil {
			if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
				sessionViewPane.SetSession(activeSession)
			}
		}

		// Start loading timer for stopwatch
		m.loadingState.Start()

		// Update Changes pane now that the app is fully initialized
		// This ensures the worktree list has been sized and has a selection
		m.updateChangesPane()

		// Set initial tmux session size using layout
		if m.ready && activeSession.TmuxSession() != nil {
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				// For shared sessions, multiply width by agent count so each pane gets full width
				tmuxWidth, tmuxHeight := m.getDetachedTmuxSize(contentWidth, contentHeight)
				if err := activeSession.TmuxSession().SetDetachedSize(tmuxWidth, tmuxHeight); err != nil {
					debug.DebugLog("Failed to set tmux session initial size to %dx%d: %v", tmuxWidth, tmuxHeight, err)
					// Continue - tmux will use default size
				}
			}
		}

		// Start monitoring tmux output and set up loading timeout
		return m, tea.Batch(
			waitForTmuxOutput(activeSession.TmuxSession()),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return loadingTimeoutMsg{}
			}),
		)

	case tmuxOutputMsg:
		return m.handleTmuxOutput(msg)

	case autoAttachMsg:
		// Auto-attach to the tmux session after it's ready
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.state.Focus.IsTmuxFocus() {
			// Block directly in Update like Claude Squad
			detachCh, err := currentTmux.Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			// Block until detachment
			<-detachCh
			// Process detached message immediately
			return m.Update(tmuxDetachedMsg{})
		}
		return m, nil

	case initializationCompleteMsg:
		// TODO: This handler is part of the old session dialog system and should be removed
		// Auto-attach to the tmux session
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.state.Focus.IsTmuxFocus() {
			// Clear screen first
			fmt.Print("\033[2J\033[H")
			// Block directly in Update like Claude Squad
			detachCh, err := currentTmux.Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			// Block until detachment
			<-detachCh
			// Process detached message immediately
			return m.Update(tmuxDetachedMsg{})
		}
		return m, tea.ClearScreen

	case tmuxDetachedMsg:
		return m.handleTmuxDetached(msg)

	case errMsg:
		m.err = msg.error
		return m, nil

	case panes.DeleteSessionRequestMsg:
		return m.handleDeleteSessionRequest(msg)

	case panes.GitRefreshMsg:
		// Changes pane needs to refresh after discard or other operations
		if m.changesPane != nil {
			var cmd tea.Cmd
			m.changesPane, cmd = m.changesPane.Update(msg)
			return m, cmd
		}
		return m, nil

	case overlays.SessionDeletedMsg:
		return m.handleSessionDeleted(msg)

	case loadingTimeoutMsg:
		// After 3 seconds of loading, start periodic updates for stopwatch
		if m.loadingState.IsLoading() {
			return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
				return loadingTimeoutMsg{}
			})
		}
		return m, nil

	case spinner.TickMsg:
		var cmds []tea.Cmd
		if cmd := m.loadingState.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Update SessionViewPane's spinner if needed
		if m.sessionViewPane != nil {
			if _, cmd := m.sessionViewPane.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Update merge overlay's spinner (if in overlay mode)
		if m.state.HasOverlay(MergeOverlay) && m.mergeOverlay != nil {
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Update chat input spinner if generating
		if m.state.InNewSession() && m.chatInput != nil {
			if cmd := m.chatInput.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		return m, combineCmds(cmds...)

	case components.ToastTickMsg:
		// Handle toast timer updates
		if m.toast != nil {
			return m, m.toast.Update(msg)
		}
		return m, nil

	case branchNameGeneratedMsg:
		return m.handleBranchNameGenerated(msg)

	case sessionCreatedMsg:
		return m.handleSessionCreated(msg)

	case session.SessionDescriptionGeneratedMsg:
		// Description generation completed (async)
		m.generatingDescription = false

		if msg.Error != nil {
			debug.DebugLog("Description generation failed: %v", msg.Error)
			// Continue - session is functional without description
			return m, nil
		}

		// Description was already updated in the session manager by the generation function
		// Just refresh the session view
		if m.sessionViewPane != nil {
			activeSession := m.sessionManager.GetActiveSession()
			if activeSession != nil && activeSession.ID == msg.SessionID {
				if sessionView, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
					sessionView.SetSession(activeSession)
				}
			}
		}

		return m, nil
	}

	// Route to mode-specific handlers
	switch m.state.Mode {
	case ModeSession:
		return m.updateSession(msg)
	case ModeNewSession:
		return m.updateNewSession(msg)
	case ModeOverlay:
		return m.updateOverlay(msg)
	default:
		// Unknown mode - fall back to session mode
		m.SetMode(ModeSession)
		return m.updateSession(msg)
	}
}
