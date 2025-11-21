package tui

import (
	"fmt"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/session"
	"agate/pkg/tui/layout"
	"agate/pkg/tui/overlays"
	"agate/pkg/tui/panes"
	"agate/pkg/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message types (moved from main.go)
type tmuxSessionStartedMsg struct {
	session *session.Session
}

type tmuxOutputMsg struct {
	content string
}

type tmuxDetachedMsg struct{}

type autoAttachMsg struct{}

type initializationCompleteMsg struct{}

type errMsg struct {
	error
}

type loadingTimeoutMsg struct{}

type branchNameGeneratedMsg struct {
	branchName string
	prompt     string // Optional: if empty, get from chatInput
}

type sessionCreatedMsg struct {
	session *session.Session
	err     error
}

// handleWindowSize handles tea.WindowSizeMsg
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) (*Model, tea.Cmd) {
	// Update layout with new dimensions
	m.layout.Update(msg.Width, msg.Height)
	m.ready = true

	// Set the active pane based on current focus
	// UNLESS we're showing new session input (then all panes should be inactive)
	if m.repoPane != nil {
		if !m.ShowNewSessionInput() {
			m.repoPane.SetActive(m.state.Focus.IsAgentsFocus())
		}
		// Update repo pane size (always uses left dimensions)
		leftWidth, leftHeight := m.layout.GetLeftDimensions()
		m.repoPane.SetSize(leftWidth, leftHeight)
	}

	if m.sessionViewPane != nil {
		if !m.ShowNewSessionInput() {
			m.sessionViewPane.SetActive(m.state.Focus.IsTmuxFocus())
		}
		centerWidth, centerHeight := m.layout.GetCenterDimensions()
		m.sessionViewPane.SetSize(centerWidth, centerHeight)
	}

	if m.changesPane != nil {
		if !m.ShowNewSessionInput() {
			m.changesPane.SetActive(m.state.Focus.IsGitFocus())
		}
		rightWidth, rightHeight := m.layout.GetRightDimensions()
		m.changesPane.SetSize(rightWidth, rightHeight)
	}

	// Update component sizes
	m.helpDialog.SetSize(msg.Width, msg.Height)

	// Update debug overlay size
	if m.debugOverlay != nil {
		m.debugOverlay.SetSize(msg.Width, msg.Height)
	}

	// Update Changes pane content after all components are sized
	// This ensures the worktree list has proper dimensions and selection
	m.updateChangesPane()

	return m, nil
}

// handleSessionCreated handles sessionCreatedMsg
func (m *Model) handleSessionCreated(msg sessionCreatedMsg) (*Model, tea.Cmd) {
	// Session creation completed
	m.creatingSession = false

	if msg.err != nil {
		// Session creation failed - show error
		toastCmd := m.toast.Show(fmt.Sprintf("Failed to create session: %v", msg.err), 0)
		return m, toastCmd
	}

	// Session created successfully!
	m.SetMode(ModeSession)
	m.chatInput.Reset()

	// Make the new session the active session immediately so the rest of the UI
	// (changes pane, tmux monitor, agent badges, etc.) points at the right data.
	if m.sessionManager != nil && msg.session != nil {
		if _, err := m.sessionManager.SwitchToSession(msg.session.ID); err != nil {
			debug.DebugLog("Failed to switch to newly created session %s: %v", msg.session.ID, err)
		}
	}
	app.SetCurrentAgent(msg.session.Agent())

	// Update session view pane
	if m.sessionViewPane != nil {
		if sessionView, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
			sessionView.SetSession(msg.session)
		}
	}

	var cmds []tea.Cmd

	// Set tmux content from shared session
	activeAgent := msg.session.GetActiveAgent()
	if activeAgent != nil && msg.session.SharedTmux != nil {
		if sessionView, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
			content, _ := msg.session.SharedTmux.CapturePaneContent()
			debug.DebugLog("[ISSUE1] Initial capture in sessionCreatedMsg: len=%d, content=%q", len(content), content)
			sessionView.SetTmuxContent(content)
		}

		// Ensure the agents pane selects this worktree so focus/changes pane stay in sync
		if m.repoPane != nil && activeAgent.Worktree != nil {
			if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
				if err := repoPane.Refresh(); err != nil {
					debug.DebugLog("Failed to refresh agents pane after session creation: %v", err)
				}
				if !repoPane.SelectWorktreeByPath(activeAgent.Worktree.Path) {
					debug.DebugLog("Agents pane could not find worktree %s right after session creation", activeAgent.Worktree.Path)
				}
			}
		}

		// Update changes pane
		if m.changesPane != nil && activeAgent.Worktree != nil {
			if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
				if repo, err := m.sessionManager.GetRepository(); err == nil {
					changesPane.SetRepositoryAndPath(repo, activeAgent.Worktree.Path)
				}
			}
		}

		// Force an immediate tmux capture to populate the pane, then start monitoring
		cmds = append(cmds, func() tea.Msg {
			if msg.session.SharedTmux == nil {
				return tmuxOutputMsg{content: ""}
			}
			content, err := msg.session.SharedTmux.CapturePaneContent()
			if err != nil {
				debug.DebugLog("ERROR capturing pane content for new session %s: %v", msg.session.ID, err)
				return tmuxOutputMsg{content: ""}
			}
			debug.DebugLog("[ISSUE1] Forced capture in sessionCreatedMsg: len=%d, content=%q", len(content), content)
			return tmuxOutputMsg{content: content}
		})

		cmds = append(cmds, waitForTmuxOutput(msg.session.SharedTmux))
	}

	// Persist the selected agents to config for next session
	if m.chatInput != nil && m.stateManager != nil {
		selectedAgents := m.chatInput.GetSelectedAgents()
		agentNames := make([]string, 0, len(selectedAgents))
		for _, agent := range selectedAgents {
			agentNames = append(agentNames, agent.Name)
		}
		if err := m.stateManager.SetSelectedAgents(agentNames); err != nil {
			debug.DebugLog("Failed to persist selected agents: %v", err)
			// Continue - not critical
		} else {
			debug.DebugLog("Persisted selected agents: %v", agentNames)
		}
	}

	// Start async description generation
	m.generatingDescription = true
	prompt := msg.session.Prompt
	defaultAgent := m.sessionManager.GetActiveSession().Agent()
	descCmd := session.GenerateSessionDescription(prompt, defaultAgent, msg.session, m.sessionManager)

	cmds = append(cmds, descCmd)

	return m, tea.Batch(cmds...)
}

// handleTmuxOutput handles tmuxOutputMsg
func (m *Model) handleTmuxOutput(msg tmuxOutputMsg) (*Model, tea.Cmd) {
	// Update session view pane tmux content
	if msg.content != "" {
		debug.DebugLog("[tmuxOutputMsg] Received update: len=%d, preview=%q", len(msg.content), truncateString(msg.content, 100))
		if m.sessionViewPane != nil {
			if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
				sessionViewPane.SetTmuxContent(msg.content)
				debug.DebugLog("[tmuxOutputMsg] Set tmux content on session view pane")
			}
		}
		// Clear loading timer when we have meaningful content
		if m.loadingState.IsLoading() && strings.TrimSpace(msg.content) != "" {
			m.loadingState.Stop()
		}

		// On first real output, ensure Changes pane is initialized
		m.updateChangesPane()
	}

	// Continue monitoring (increased frequency for better responsiveness)
	return m, tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
			return waitForTmuxOutput(currentTmux)()
		}
		return nil
	})
}

// handleTmuxDetached handles tmuxDetachedMsg
func (m *Model) handleTmuxDetached(msg tmuxDetachedMsg) (*Model, tea.Cmd) {
	debug.DebugLog("[tmuxDetachedMsg] ===== DETACHED FROM TMUX =====")
	if m.sessionManager != nil {
		activeSession := m.sessionManager.GetActiveSession()
		if activeSession != nil {
			if activeSession.Worktree() != nil {
				debug.DebugLog("[tmuxDetachedMsg] Active session before switchToPane: path=%s, branch=%s",
					activeSession.Worktree().Path, activeSession.Worktree().Branch)
			}
		} else {
			debug.DebugLog("[tmuxDetachedMsg] No active session or worktree")
		}
	}

	// Update shortcut overlay back to preview mode
	m.shortcutOverlay.SetMode("preview")

	// Log which agent we're detaching from
	var agentInfo string
	if activeSession := m.sessionManager.GetActiveSession(); activeSession != nil {
		activeAgent := activeSession.GetActiveAgent()
		if activeAgent != nil {
			agentInfo = fmt.Sprintf("agent=%s, index=%d, pane=%d",
				activeAgent.AgentConfig.Name, activeSession.ActiveAgentIndex, activeAgent.PaneIndex)
		} else {
			agentInfo = fmt.Sprintf("no active agent (index=%d)", activeSession.ActiveAgentIndex)
		}
	} else {
		agentInfo = "no active session"
	}
	debug.DebugLog("[ISSUE3] Detaching from tmux: %s", agentInfo)

	debug.DebugLog("[tmuxDetachedMsg] About to call switchToPane(FocusAgents)")
	// Return focus to the agents pane (which will automatically jump to the active agent's row)
	var switchCmd tea.Cmd
	m, switchCmd = m.switchToPane(layout.NewAgentsFocus())
	debug.DebugLog("[ISSUE3] After switchToPane, focus should be on agents pane to show agent card for: %s", agentInfo)

	// Immediately resize the tmux session to current window dimensions
	if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.ready {
		if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
			// For shared sessions, multiply width by agent count so each pane gets full width
			tmuxWidth, tmuxHeight := m.getDetachedTmuxSize(contentWidth, contentHeight)
			if err := currentTmux.SetDetachedSize(tmuxWidth, tmuxHeight); err != nil {
				debug.DebugLog("Failed to resize tmux session after detaching to %dx%d: %v", tmuxWidth, tmuxHeight, err)
				// Continue - terminal will work with current size
			}
		}
	}

	// Resume monitoring and trigger window size recalculation
	var monitorCmd tea.Cmd
	if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
		monitorCmd = waitForTmuxOutput(currentTmux)
	}
	return m, tea.Batch(
		switchCmd,
		monitorCmd,
		tea.WindowSize(), // Trigger complete UI layout recalculation
	)
}

// handleBranchNameGenerated handles branchNameGeneratedMsg
func (m *Model) handleBranchNameGenerated(msg branchNameGeneratedMsg) (*Model, tea.Cmd) {
	// Branch name generation completed - start session creation
	m.creatingSession = true

	// Get prompt from message (if provided via CLI) or from chat input
	prompt := msg.prompt
	if prompt == "" {
		prompt = m.chatInput.GetValue()
	}

	// Get agents from subprocess or chat input
	var agentNames []string
	if m.subprocess != "" {
		// Use agents from subprocess flag
		names := strings.Split(m.subprocess, ",")
		for _, name := range names {
			name = strings.TrimSpace(name)
			agentNames = append(agentNames, name)
		}
	} else {
		// Get from chat input
		agentConfigs := m.chatInput.GetSelectedAgents()
		agentNames = make([]string, 0, len(agentConfigs))
		for _, agent := range agentConfigs {
			agentNames = append(agentNames, agent.Name)
		}
	}

	createCmd := func() tea.Msg {
		// Use stored terminal dimensions if available (from CLI prompt),
		// otherwise use defaults (from manual session creation)
		var newSession *session.Session
		var err error
		if m.initialTermWidth > 0 && m.initialTermHeight > 0 {
			newSession, err = m.sessionManager.CreateSessionWithSize(prompt, msg.branchName, agentNames, m.initialTermWidth, m.initialTermHeight)
		} else {
			newSession, err = m.sessionManager.CreateSession(prompt, msg.branchName, agentNames)
		}
		return sessionCreatedMsg{session: newSession, err: err}
	}

	return m, createCmd
}

// handleDeleteSessionRequest handles panes.DeleteSessionRequestMsg
func (m *Model) handleDeleteSessionRequest(msg panes.DeleteSessionRequestMsg) (*Model, tea.Cmd) {
	// User wants to delete a session - show confirmation dialog
	if msg.Session != nil {
		m.SetOverlay(SessionDeleteOverlay)
		m.sessionConfirm = overlays.NewSessionDeleteConfirmDialog(msg.Session, m.sessionManager)
		if m.sessionConfirm != nil {
			m.sessionConfirm.SetSize(m.layout.GetWidth(), m.layout.GetHeight())
		}
	}
	return m, nil
}

// handleSessionDeleted handles overlays.SessionDeletedMsg
func (m *Model) handleSessionDeleted(msg overlays.SessionDeletedMsg) (*Model, tea.Cmd) {
	// Session deleted successfully (already deleted by overlay)
	m.ClearOverlay()
	m.sessionConfirm = nil

	// Refresh UI to reflect deletion
	if m.repoPane != nil {
		if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
			var snapshot []panes.AgentListItem
			if msg.Session != nil {
				snapshot = repoPane.SnapshotItems()
			}

			if err := repoPane.Refresh(); err != nil {
				debug.DebugLog("Failed to refresh repo pane after session deletion: %v", err)
			}

			// Select the next appropriate session with priority:
			// 1. Session above, 2. Session below, 3. Main worktree
			if msg.Session != nil {
				repoPane.SelectSessionAfterDeletion(msg.Session, snapshot)
			}
		}
	}
	// Update Changes pane
	m.updateChangesPane()

	// Show success toast
	if msg.Session != nil && m.toast != nil {
		branchName := "session"
		if msg.Session.Worktree() != nil {
			branchName = msg.Session.Worktree().Branch
		}
		// Create styled message with green checkmark
		checkmarkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
		checkmark := checkmarkStyle.Render("✓")
		message := checkmark + " Deleted " + branchName
		toastCmd := m.toast.Show(message, 0)
		return m, toastCmd
	}
	return m, nil
}

// Helper functions are defined in model.go:
// - truncateString
// - waitForTmuxOutput
// - combineCmds
