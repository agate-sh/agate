package tui

import (
	"fmt"
	"time"

	"agate/pkg/tui/overlays"
	"agate/pkg/tui/panes"
	"agate/pkg/tui/theme"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// updateOverlay handles messages in ModeOverlay (overlay is active)
func (m *Model) updateOverlay(msg tea.Msg) (*Model, tea.Cmd) {
	// Route based on which overlay is active
	switch m.state.ActiveOverlay {
	case HelpOverlay:
		return m.updateHelpOverlay(msg)
	case DebugOverlay:
		return m.updateDebugOverlay(msg)
	case SessionDeleteOverlay:
		return m.updateSessionDeleteOverlay(msg)
	case MergeOverlay:
		return m.updateMergeOverlay(msg)
	case AgentSelectorOverlay:
		return m.updateAgentSelectorOverlay(msg)
	default:
		// No overlay active - this shouldn't happen but fall back to session mode
		m.ClearOverlay()
		return m.updateSession(msg)
	}
}

// updateHelpOverlay handles the help overlay
func (m *Model) updateHelpOverlay(msg tea.Msg) (*Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		// Any key closes help dialog
		m.ClearOverlay()
		return m, nil
	}
	return m, nil
}

// updateDebugOverlay handles the debug overlay
func (m *Model) updateDebugOverlay(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.debugOverlay != nil {
			var cmd tea.Cmd
			overlay, cmd := m.debugOverlay.Update(msg)
			m.debugOverlay = overlay
			return m, cmd
		}
	case overlays.DebugOverlayClosedMsg:
		// Debug overlay closed
		m.ClearOverlay()
		return m, nil
	}
	return m, nil
}

// updateSessionDeleteOverlay handles the session deletion confirmation overlay
func (m *Model) updateSessionDeleteOverlay(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.sessionConfirm != nil {
			var cmd tea.Cmd
			model, cmd := m.sessionConfirm.Update(msg)
			m.sessionConfirm = model.(*overlays.SessionDeleteConfirmDialog)
			return m, cmd
		}

	case overlays.SessionDeleteCancelledMsg:
		// Session deletion cancelled
		m.ClearOverlay()
		m.sessionConfirm = nil
		return m, nil

	case overlays.SessionDeletionErrorMsg:
		// Session deletion failed
		m.ClearOverlay()
		m.sessionConfirm = nil
		m.err = fmt.Errorf("failed to delete session: %s", msg.Error)
		return m, nil
	}
	return m, nil
}

// updateMergeOverlay handles the merge overlay
func (m *Model) updateMergeOverlay(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Check for esc to close overlay
		if msg.String() == "esc" {
			m.ClearOverlay()
			m.mergeOverlay = nil
			return m, nil
		}

		if m.mergeOverlay != nil {
			var cmd tea.Cmd
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			return m, cmd
		}

	case spinner.TickMsg:
		// Update merge overlay's spinner
		if m.mergeOverlay != nil {
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			if cmd != nil {
				return m, cmd
			}
		}

	case time.Time:
		// Pass time ticks to merge overlay for elapsed time updates
		if m.mergeOverlay != nil {
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			return m, cmd
		}

	case overlays.MergeSuccessMsg:
		// Merge succeeded - show success toast and close overlay
		m.ClearOverlay()
		m.mergeOverlay = nil

		// Show success toast with green checkmark
		checkmarkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
		checkmark := checkmarkStyle.Render("✓")
		message := fmt.Sprintf("%s Merged %s to main", checkmark, msg.SHA)
		toastCmd := m.toast.Show(message, 0)

		// Refresh changes pane to show updated status
		if m.changesPane != nil {
			if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
				changesPane.Refresh()
			}
		}

		return m, toastCmd

	case overlays.MergeMessageGeneratedMsg, overlays.FileDiscardedMsg:
		// Pass overlay-specific messages to merge overlay
		if m.mergeOverlay != nil {
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			return m, cmd
		}

	case overlays.MergeErrorMsg:
		// Merge failed - show error toast and keep overlay open
		toastCmd := m.toast.Show(fmt.Sprintf("✗ Failed to merge: %s", msg.Err.Error()), 0)
		return m, toastCmd
	}

	return m, nil
}

// updateAgentSelectorOverlay handles the agent selector modal
func (m *Model) updateAgentSelectorOverlay(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			if m.chatInput != nil {
				m.chatInput.SetSelectedAgents(m.agentSelector.GetSelectedAgents())
			}
			m.ClearOverlay()
			m.agentSelector = nil
			return m, nil
		}
	}

	// Pass all messages to agent selector (including cursor blink messages)
	if m.agentSelector != nil {
		var cmd tea.Cmd
		m.agentSelector, cmd = m.agentSelector.Update(msg)
		return m, cmd
	}

	return m, nil
}
