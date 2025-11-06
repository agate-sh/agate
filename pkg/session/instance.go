package session

import (
	"time"

	"agate/pkg/app"
	"agate/pkg/git"
	"agate/pkg/naming"
	"agate/pkg/tmux"
)

// AgentInstance represents a single agent working within a multi-agent session
type AgentInstance struct {
	// Identification
	ID          string          `json:"id"`
	AgentConfig app.AgentConfig `json:"agent_config"` // The agent configuration (Claude, Codex, etc.)
	SessionID   string          `json:"session_id"`   // Reference to parent Session

	// Resources
	TmuxSession *tmux.TmuxSession `json:"-"`        // This agent's tmux session - not persisted
	Worktree    *git.WorktreeInfo `json:"worktree"` // This agent's worktree information

	// State tracking
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// Update refreshes the instance's last accessed timestamp
func (i *AgentInstance) Update() {
	i.LastAccessed = time.Now()
}

// Deactivate marks the instance as inactive (no-op for now, kept for compatibility)
func (i *AgentInstance) Deactivate() {
	// Currently a no-op, but kept for interface compatibility
}

// GetTmuxSessionName returns the stable tmux session name for this agent instance
func (i *AgentInstance) GetTmuxSessionName() string {
	if i.TmuxSession != nil {
		return i.TmuxSession.GetSessionName()
	}
	return generateTmuxSessionName(i.Worktree, i.AgentConfig.Name)
}

// generateTmuxSessionName creates a stable, unique tmux session name.
// Format: agate_<repo>_<branch>_<agent>_<hash>
//
// This function uses naming.Generator to ensure consistent name generation.
func generateTmuxSessionName(worktree *git.WorktreeInfo, agentName string) string {
	if worktree == nil {
		return ""
	}

	// Use the naming generator for consistent, sanitized names
	nameGen := naming.NewGenerator()
	return nameGen.GenerateFromWorktree(worktree.RepoName, worktree.Branch, agentName)
}
