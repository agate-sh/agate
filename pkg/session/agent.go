package session

import (
	"time"

	"agate/pkg/app"
	"agate/pkg/git"
	"agate/pkg/naming"
)

// Agent represents a single agent working within a multi-agent session.
// Each agent has its own worktree but shares a tmux session (in different panes).
type Agent struct {
	// Identification
	ID          string          `json:"id"`
	AgentConfig app.AgentConfig `json:"agent_config"` // The agent configuration (Claude, Codex, etc.)
	SessionID   string          `json:"session_id"`   // Reference to parent Session

	// Resources
	Worktree  *git.WorktreeInfo `json:"worktree"`   // This agent's worktree information
	PaneIndex int               `json:"pane_index"` // Which pane in the shared tmux session

	// State tracking
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// Update refreshes the agent's last accessed timestamp
func (a *Agent) Update() {
	a.LastAccessed = time.Now()
}

// Deactivate marks the agent as inactive (no-op for now, kept for compatibility)
func (a *Agent) Deactivate() {
	// Currently a no-op, but kept for interface compatibility
}

// generateTmuxSessionName creates a stable, unique tmux session name for a session.
// Format: agate_<repo>_<branch>_<hash>
// Note: No longer includes agent name since one session is shared by all agents
//
// This function uses naming.Generator to ensure consistent name generation.
func generateTmuxSessionName(worktree *git.WorktreeInfo, sessionBaseName string) string {
	if worktree == nil {
		return ""
	}

	// Use the naming generator for consistent, sanitized names
	// Use session base name instead of individual agent names
	nameGen := naming.NewGenerator()
	return nameGen.GenerateFromWorktree(worktree.RepoName, sessionBaseName, "")
}
