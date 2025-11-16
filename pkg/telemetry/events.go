package telemetry

import "agate/internal/debug"

var distinctID string

// SetDistinctId stores the distinct ID for event tracking
func SetDistinctId(id string) {
	distinctID = id
}

// trackEvent is a helper that tracks an event with the distinct ID
func trackEvent(event string, properties map[string]interface{}) {
	if distinctID == "" {
		debug.DebugLog("Analytics: skipping event %s - no distinct ID", event)
		return
	}

	if err := Track(distinctID, event, properties); err != nil {
		debug.DebugLog("Analytics: failed to track event %s: %v", event, err)
	}
}

// TrackTUIStarted tracks when the TUI is launched
func TrackTUIStarted() {
	trackEvent("tui_started", map[string]interface{}{})
}

// TrackPromptEntered tracks when a user enters a prompt in the chat input
func TrackPromptEntered(prompt string, agentNames []string) {
	// Don't include the actual prompt text for privacy
	trackEvent("prompt_entered", map[string]interface{}{
		"prompt_length": len(prompt),
		"num_agents":    len(agentNames),
	})
}

// TrackAgentsSelected tracks when agents are selected for a session
func TrackAgentsSelected(agentNames []string) {
	trackEvent("agents_selected", map[string]interface{}{
		"agents":     agentNames,
		"num_agents": len(agentNames),
	})
}

// TrackSessionCreated tracks when a new session is created
func TrackSessionCreated(agentNames []string, isFirstSession bool) {
	trackEvent("session_created", map[string]interface{}{
		"agents":           agentNames,
		"num_agents":       len(agentNames),
		"is_first_session": isFirstSession,
	})
}

// TrackSessionDeleted tracks when a session is deleted
func TrackSessionDeleted(agentName string) {
	trackEvent("session_deleted", map[string]interface{}{
		"agent_name": agentName,
	})
}

// TrackSessionActivated tracks when a user switches to a different session
func TrackSessionActivated(agentName string) {
	trackEvent("session_activated", map[string]interface{}{
		"agent_name": agentName,
	})
}

// TrackAgentAttached tracks when a user attaches to an agent's tmux session
func TrackAgentAttached(agentName string) {
	trackEvent("agent_attached", map[string]interface{}{
		"agent_name": agentName,
	})
}

// TrackAgentDetached tracks when a user detaches from an agent's tmux session
func TrackAgentDetached(agentName string) {
	trackEvent("agent_detached", map[string]interface{}{
		"agent_name": agentName,
	})
}

// TrackAgentSwitched tracks when a user tabs between agents in a multi-agent session
func TrackAgentSwitched() {
	// No properties - just count how often this happens
	trackEvent("agent_switched", map[string]interface{}{})
}

// TrackMergeOverlayOpened tracks when the merge overlay is opened
func TrackMergeOverlayOpened() {
	trackEvent("merge_overlay_opened", map[string]interface{}{})
}

// TrackMergeCompleted tracks when a merge is successfully completed
func TrackMergeCompleted(agentName string) {
	trackEvent("merge_completed", map[string]interface{}{
		"agent_name": agentName,
	})
}
