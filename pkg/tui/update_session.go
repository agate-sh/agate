package tui

import (
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
		return m.handleSessionKeys(msg)

	case tea.MouseMsg:
		return m.handleSessionMouse(msg)

	case panes.SessionSwitchedMsg:
		// Session was auto-switched by navigation in agents pane
		// Hide new session input and show the 3-pane layout
		if msg.Session != nil {
			m.SetMode(ModeSession)

			// Ensure session pane reflects the new active session
			if m.sessionViewPane != nil {
				if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
					sessionViewPane.SetSession(msg.Session)
				}
			}

			// Force a tmux content refresh for the newly selected agent
			if msg.Session.TmuxSession() != nil {
				return m, waitForTmuxOutput(msg.Session.TmuxSession())
			}
		}
		return m, nil

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
			m.state.Focus = layout.NewSessionFocus(layout.SubPaneTmux)
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
	}

	return m, nil
}

// handleSessionKeys handles keyboard input in session mode
func (m *Model) handleSessionKeys(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch {
	case msg.String() == "enter":
		// Enter key handling - delegate to the active pane
		switch {
		case m.state.Focus.IsAgentsFocus():
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
		// 'alt+d' key handling - delegate to the agents pane for session deletion
		if m.state.Focus.IsAgentsFocus() && m.repoPane != nil {
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
		// Show help dialog
		m.SetOverlay(HelpOverlay)
		return m, nil

	case key.Matches(msg, common.GlobalKeys.DebugOverlay):
		// Show debug overlay
		m.SetOverlay(DebugOverlay)
		return m, nil

	case key.Matches(msg, common.GlobalKeys.NewSession):
		// Step 5.5: 'n' key shows new chat input interface
		m.SetMode(ModeNewSession)
		var focusCmd tea.Cmd
		if m.chatInput != nil {
			m.chatInput.Reset()
			focusCmd = m.chatInput.Focus()
		}
		// Mark all panes as inactive when entering new session view
		if m.repoPane != nil {
			m.repoPane.SetActive(false)
		}
		if m.sessionViewPane != nil {
			m.sessionViewPane.SetActive(false)
		}
		if m.changesPane != nil {
			m.changesPane.SetActive(false)
		}
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
		switch {
		case m.state.Focus.IsAgentsFocus():
			if m.repoPane != nil {
				// Use HandleKey to ensure search exit logic runs
				handled, cmd := m.repoPane.HandleKey("up")
				if handled {
					// Update changes pane after navigation
					return m, tea.Batch(cmd, m.updateChangesPane())
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
		switch {
		case m.state.Focus.IsAgentsFocus():
			if m.repoPane != nil {
				// Use HandleKey to ensure search exit logic runs
				handled, cmd := m.repoPane.HandleKey("down")
				if handled {
					// Update changes pane after navigation
					return m, tea.Batch(cmd, m.updateChangesPane())
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
		return m.switchToPane(layout.NewAgentsFocus())

	case key.Matches(msg, common.GlobalKeys.SessionPane):
		// Alt+A jumps to agents pane (if session exists)
		if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
			m.SetMode(ModeSession)
			return m.switchToPane(layout.NewSessionFocus(layout.SubPaneTmux))
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.ChangesPane):
		// Alt+C jumps to changes pane (if session exists)
		if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
			m.SetMode(ModeSession)
			return m.switchToPane(layout.NewSessionFocus(layout.SubPaneGit))
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.NextAgent):
		// Tab toggles sub-panes in changes pane, or cycles agents in session pane
		// Check git focus first since it's also PaneTypeSession
		if m.state.Focus.IsGitFocus() && m.changesPane != nil {
			handled, cmd := m.changesPane.HandleKey("tab")
			if handled {
				return m, cmd
			}
		} else if m.state.Focus.PaneType == layout.PaneTypeSession {
			// Cycle to next agent in session pane
			return m, m.cycleNextAgent()
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.PrevAgent):
		// Shift+Tab cycles to previous agent (only on session pane)
		if m.state.Focus.PaneType == layout.PaneTypeSession {
			// TODO: Implement backwards agent cycling
			// For now, do nothing
		}
		return m, nil

	default:
		// If agents pane is focused, try HandleKey first for navigation/special keys
		// then fall back to Update for search input
		if m.state.Focus.IsAgentsFocus() && m.repoPane != nil && m.repoPane.IsActive() {
			// Try HandleKey first (handles navigation and search exit logic)
			keyStr := msg.String()
			debug.DebugLog("[MAIN UPDATE] default case: keyStr=%q, Focus.IsAgentsFocus()=true, calling HandleKey", keyStr)
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
		debug.DebugLog("[MAIN UPDATE] default case: keyStr=%q, Focus.IsAgentsFocus()=%v, repoPane=%v, IsActive=%v - NOT handling",
			msg.String(), m.state.Focus.IsAgentsFocus(), m.repoPane != nil, m.repoPane != nil && m.repoPane.IsActive())
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
