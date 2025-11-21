package tui

import (
	"fmt"

	"agate/internal/debug"
	"agate/pkg/common"
	"agate/pkg/tui/layout"
	"agate/pkg/tui/overlays"
	"agate/pkg/tui/panes"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// updateSession handles messages in ModeSession (normal session view)
func (m *Model) updateSession(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		debug.DebugLog("[updateSession] Received KeyMsg: key=%q", msg.String())
		return m.handleSessionKeys(msg)

	case tea.MouseMsg:
		return m.handleSessionMouse(msg)

	case panes.SessionSwitchedMsg:
		// Session was auto-switched by navigation in agents pane
		// Switch to session mode and schedule a debounced UI update
		if msg.Session != nil {
			m.SetMode(ModeSession)
			return m, m.switchToSessionUI(msg.Session)
		}
		return m, nil

	case debouncedSessionSwitchMsg:
		// Debounce timer expired - perform the actual UI switch
		return m, m.performSessionSwitch(msg.session)

	case panes.AttachToSessionMsg:
		// User wants to attach to a tmux session from the agents pane
		if msg.Session != nil && msg.Session.TmuxSession() != nil {
			if m.sessionManager != nil {
				m.sessionManager.SwitchToSession(msg.Session.ID)
				m.sessionManager.GetActiveSession().Agent()
			}

			if m.sessionViewPane != nil {
				if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
					sessionViewPane.SetSession(msg.Session)
				}
			}
			m.state.Focus = layout.NewAgentsFocus(layout.SubPaneTmux)
			m.shortcutOverlay.SetFocus(m.state.Focus.String())
			m.shortcutOverlay.SetMode("attached")

			detachCh, err := msg.Session.TmuxSession().Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			<-detachCh
			model, cmd := m.Update(tmuxDetachedMsg{})
			return model.(*Model), cmd
		}
		return m, nil

	case panes.GitRefreshMsg:
		// Changes pane needs to refresh after discard or other operations
		if m.changesPane != nil {
			var cmd tea.Cmd
			m.changesPane, cmd = m.changesPane.Update(msg)
			return m, cmd
		}
		return m, nil

	default:
		// Pass all non-keyboard, non-mouse messages to active panes for cursor blinking, etc.
		// This includes cursor.BlinkMsg which is needed for textinput cursor animation
		msgType := fmt.Sprintf("%T", msg)
		if msgType == "cursor.BlinkMsg" || msgType == "cursor.blinkMsg" {
			debug.DebugLog("[updateSession] default case: received blink message, IsSessionsFocus=%v, repoPane=%v, IsActive=%v",
				m.state.Focus.IsSessionsFocus(), m.repoPane != nil, m.repoPane != nil && m.repoPane.IsActive())
		}
		if m.state.Focus.IsSessionsFocus() && m.repoPane != nil && m.repoPane.IsActive() {
			var cmd tea.Cmd
			m.repoPane, cmd = m.repoPane.Update(msg)
			if msgType == "cursor.BlinkMsg" || msgType == "cursor.blinkMsg" {
				debug.DebugLog("[updateSession] passed blink message to repoPane, got cmd back: %v", cmd != nil)
			}
			return m, cmd
		}
		if msgType == "cursor.BlinkMsg" || msgType == "cursor.blinkMsg" {
			debug.DebugLog("[updateSession] NOT passing blink message to repoPane")
		}
	}

	return m, nil
}

// handleSessionKeys handles keyboard input in session mode
func (m *Model) handleSessionKeys(msg tea.KeyMsg) (*Model, tea.Cmd) {
	debug.DebugLog("[handleSessionKeys] ENTRY: key=%q, checking if matches NewSession binding", msg.String())
	if key.Matches(msg, common.GlobalKeys.NewSession) {
		debug.DebugLog("[handleSessionKeys] YES - key matches NewSession!")
	} else {
		debug.DebugLog("[handleSessionKeys] NO - key does NOT match NewSession")
	}

	if m.consumePendingAltRune(msg) {
		debug.DebugLog("[handleSessionKeys] Consumed stray key %q after alt shortcut", msg.String())
		return m, nil
	}

	switch {
	case msg.String() == "enter":
		// Enter key handling - delegate to the active pane
		switch {
		case m.state.Focus.IsSessionsFocus():
			// Let the repo pane handle enter key for toggling repo expansion only
			if m.repoPane != nil {
				handled, cmd := m.repoPane.HandleKey("enter")
				if handled {
					return m, cmd
				}
			}
		case m.state.Focus.IsGitFocus():
			// Let the changes pane handle enter key for opening files
			if m.changesPane != nil {
				handled, cmd := m.changesPane.HandleKey("enter")
				if handled {
					return m, cmd
				}
			}
		case m.state.Focus.IsTmuxFocus():
			// Enter key attaches to agent tmux session when tmux pane is focused
			if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
				// Update UI to show attached mode
				m.shortcutOverlay.SetMode("attached")
				// Block directly in Update like Claude Squad - don't return to event loop during attachment
				detachCh, err := currentTmux.Attach()
				if err != nil {
					return m, func() tea.Msg { return errMsg{err} }
				}
				// Block until detachment like Claude Squad does
				<-detachCh
				// Process detached message immediately
				model, cmd := m.Update(tmuxDetachedMsg{})
				return model.(*Model), cmd
			}
		}

	case msg.String() == "alt+d":
		m.startSuppressingAltRune(msg)
		// 'alt+d' key handling - delegate to the sessions pane for session deletion
		if m.state.Focus.IsSessionsFocus() && m.repoPane != nil {
			handled, cmd := m.repoPane.HandleKey("alt+d")
			if handled {
				return m, cmd
			}
		}
		// Also handle 'alt+d' in changes pane for discarding files
		if m.state.Focus.IsGitFocus() && m.changesPane != nil {
			handled, cmd := m.changesPane.HandleKey("alt+d")
			if handled {
				// Refresh changes pane after discard
				m.updateChangesPane()
				return m, cmd
			}
		}

	case key.Matches(msg, common.GlobalKeys.Quit):
		// Persist sessions before quitting
		if m.sessionManager != nil {
			debug.DebugLog("Quit: Persisting sessions before exit")
			if err := m.sessionManager.PersistSessions(); err != nil {
				debug.DebugLog("Quit: Failed to persist sessions: %v", err)
			} else {
				debug.DebugLog("Quit: Successfully persisted sessions before exit")
			}
		}

		// Close debug panel and log file
		if m.debugLogger != nil {
			m.debugLogger.Close()
		}
		return m, tea.Quit

	case key.Matches(msg, common.GlobalKeys.Keybindings):
		m.startSuppressingAltRune(msg)
		// Show help dialog
		m.SetOverlay(HelpOverlay)
		return m, nil

	case key.Matches(msg, common.GlobalKeys.DebugOverlay):
		// Show debug overlay
		m.SetOverlay(DebugOverlay)
		return m, nil

	case key.Matches(msg, common.GlobalKeys.NewSession):
		// Step 5.5: 'n' key shows new chat input interface
		debug.DebugLog("[handleSessionKeys] NewSession key matched! Switching to new session mode")
		m.startSuppressingAltRune(msg)
		focusCmd := m.enterNewSessionMode(true)
		debug.DebugLog("[handleSessionKeys] Returning focus command")
		return m, focusCmd

	case key.Matches(msg, common.GlobalKeys.AttachAgent):
		// Attach to agent tmux session (global shortcut)
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
			m.shortcutOverlay.SetMode("attached")
			detachCh, err := currentTmux.Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			<-detachCh
			model, cmd := m.Update(tmuxDetachedMsg{})
			return model.(*Model), cmd
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.MergeChanges):
		m.startSuppressingAltRune(msg)
		// Show merge overlay (global shortcut)
		activeSession := m.sessionManager.GetActiveSession()
		if activeSession != nil && activeSession.Worktree() != nil {
			// Check if there are any changes to merge
			repo, err := m.sessionManager.GetRepository()
			if err != nil {
				toastCmd := m.toast.Show("Failed to get repository", 0)
				return m, toastCmd
			}
			fileStatus := repo.GetFileStatuses(activeSession.Worktree().Path)
			if fileStatus == nil || fileStatus.IsClean {
				// No changes to merge - show toast
				toastCmd := m.toast.Show("No changes to merge", 0)
				return m, toastCmd
			}

			// There are changes - show merge overlay
			m.mergeOverlay = overlays.NewMergeOverlay(activeSession, repo)
			m.mergeOverlay.SetSize(m.layout.GetWidth(), m.layout.GetHeight())
			m.SetOverlay(MergeOverlay)
			initCmd := m.mergeOverlay.Init()
			return m, initCmd
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.Up):
		// Navigate up in focused pane
		debug.DebugLog("[MAIN UPDATE] Up key matched! Focus.IsSessionsFocus()=%v, repoPane=%v",
			m.state.Focus.IsSessionsFocus(), m.repoPane != nil)
		switch {
		case m.state.Focus.IsSessionsFocus():
			if m.repoPane != nil {
				// HandleKey calls MoveUpWithCmd which triggers onSelectionChanged
				debug.DebugLog("[MAIN UPDATE] Calling repoPane.HandleKey for Up")
				handled, cmd := m.repoPane.HandleKey("up")
				debug.DebugLog("[MAIN UPDATE] HandleKey returned handled=%v, cmd=%v", handled, cmd != nil)
				if handled {
					return m, cmd
				}
			}
		case m.state.Focus.IsGitFocus():
			if m.changesPane != nil {
				handled, cmd := m.changesPane.HandleKey(msg.String())
				if handled {
					return m, cmd
				}
				return m, nil
			}
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.Down):
		// Navigate down in focused pane
		debug.DebugLog("[MAIN UPDATE] Down key matched! Focus.IsSessionsFocus()=%v, repoPane=%v",
			m.state.Focus.IsSessionsFocus(), m.repoPane != nil)
		switch {
		case m.state.Focus.IsSessionsFocus():
			if m.repoPane != nil {
				// HandleKey calls MoveDownWithCmd which triggers onSelectionChanged
				debug.DebugLog("[MAIN UPDATE] Calling repoPane.HandleKey for Down")
				handled, cmd := m.repoPane.HandleKey("down")
				debug.DebugLog("[MAIN UPDATE] HandleKey returned handled=%v, cmd=%v", handled, cmd != nil)
				if handled {
					return m, cmd
				}
			}
		case m.state.Focus.IsGitFocus():
			if m.changesPane != nil {
				handled, cmd := m.changesPane.HandleKey(msg.String())
				if handled {
					return m, cmd
				}
				return m, nil
			}
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.AgentsPane):
		// Alt+S jumps to sessions pane
		m.startSuppressingAltRune(msg)
		return m.switchToPane(layout.NewSessionsFocus())

	case key.Matches(msg, common.GlobalKeys.SessionPane):
		// Alt+A jumps to agents pane (if session exists)
		m.startSuppressingAltRune(msg)
		if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
			m.SetMode(ModeSession)
			return m.switchToPane(layout.NewTmuxFocus())
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.ChangesPane):
		// Alt+C jumps to changes pane (if session exists)
		m.startSuppressingAltRune(msg)
		if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
			m.SetMode(ModeSession)
			return m.switchToPane(layout.NewGitFocus())
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.NextAgent):
		// Tab toggles sub-panes in changes pane, or cycles agents in session pane
		// Check git focus first since it's also PaneTypeAgents
		if m.state.Focus.IsGitFocus() && m.changesPane != nil {
			handled, cmd := m.changesPane.HandleKey("tab")
			if handled {
				return m, cmd
			}
		} else if m.state.Focus.PaneType == layout.PaneTypeAgents {
			// Cycle to next agent in agents pane
			return m, m.cycleNextAgent()
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.PrevAgent):
		// Shift+Tab cycles to previous agent (only on agents pane)
		if m.state.Focus.PaneType == layout.PaneTypeAgents {
			// TODO: Implement backwards agent cycling
			// For now, do nothing
		}
		return m, nil

	default:
		// If sessions pane is focused, try HandleKey first for navigation/special keys
		// then fall back to Update for search input
		if m.state.Focus.IsSessionsFocus() && m.repoPane != nil && m.repoPane.IsActive() {
			// Try HandleKey first (handles navigation and search exit logic)
			keyStr := msg.String()
			debug.DebugLog("[MAIN UPDATE] default case: keyStr=%q, Focus.IsSessionsFocus()=true, calling HandleKey", keyStr)
			handled, cmd := m.repoPane.HandleKey(keyStr)
			debug.DebugLog("[MAIN UPDATE] HandleKey returned handled=%v, cmd=%v", handled, cmd != nil)
			if handled {
				debug.DebugLog("[MAIN UPDATE] Key was handled by HandleKey, returning")
				return m, cmd
			}
			// If not handled by HandleKey, pass to Update for search input
			debug.DebugLog("[MAIN UPDATE] Key not handled by HandleKey, calling Update")
			m.repoPane, cmd = m.repoPane.Update(msg)
			debug.DebugLog("[MAIN UPDATE] Update returned cmd=%v", cmd != nil)
			return m, cmd
		}
		debug.DebugLog("[MAIN UPDATE] default case: keyStr=%q, Focus.IsSessionsFocus()=%v, repoPane=%v, IsActive=%v - NOT handling",
			msg.String(), m.state.Focus.IsSessionsFocus(), m.repoPane != nil, m.repoPane != nil && m.repoPane.IsActive())
	}

	return m, nil
}

// handleSessionMouse handles mouse events in session mode
func (m *Model) handleSessionMouse(msg tea.MouseMsg) (*Model, tea.Cmd) {
	// Allow changes pane to react to mouse events regardless of focus
	if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
		if handled, cmd := changesPane.HandleMouse(msg); handled {
			return m, cmd
		}
	}

	// Handle mouse events when right pane is focused
	if currentTmux := m.getCurrentTmuxSession(); m.state.Focus.IsTmuxFocus() && currentTmux != nil {
		switch msg.Action {
		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonWheelUp {
				// Enter copy mode and scroll up
				if err := currentTmux.SendScrollUp(); err != nil {
					debug.DebugLog("Failed to send scroll up to tmux session: %v", err)
				}
			} else if msg.Button == tea.MouseButtonWheelDown {
				// Scroll down (or exit copy mode if at bottom)
				if err := currentTmux.SendScrollDown(); err != nil {
					debug.DebugLog("Failed to send scroll down to tmux session: %v", err)
				}
			}
		}
		// Trigger content refresh after scroll
		return m, waitForTmuxOutput(currentTmux)
	}

	return m, nil
}
