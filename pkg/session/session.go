package session

import (
	"time"

	"agate/pkg/app"
	"agate/pkg/git"
	"agate/pkg/tmux"
)

// Session represents a complete workspace: worktree + tmux session + agent
type Session struct {
	// Identification
	ID          string `json:"id"`
	Name        string `json:"name"`         // Internal name for session
	WorktreeKey string `json:"worktree_key"` // Stable key for worktree identification

	// Session-specific resources
	TmuxSession *tmux.TmuxSession    `json:"-"`            // Not persisted - recreated on startup
	Worktree    *git.WorktreeInfo    `json:"worktree"`     // Worktree information
	Agent       app.AgentConfig      `json:"agent"`        // This session's agent configuration

	// State tracking
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
	IsActive     bool      `json:"is_active"`
}

// Update refreshes the session's last accessed time and sets it as active
func (s *Session) Update() {
	s.LastAccessed = time.Now()
	s.IsActive = true
}

// Deactivate marks the session as inactive
func (s *Session) Deactivate() {
	s.IsActive = false
}

// GetTmuxSessionName returns the stable tmux session name for this session
func (s *Session) GetTmuxSessionName() string {
	if s.TmuxSession != nil {
		return s.TmuxSession.GetSessionName()
	}
	return generateTmuxSessionName(s.Worktree, s.Agent.Name)
}

// generateTmuxSessionName creates a stable, unique tmux session name
// Format: agate_<repo>_<branch>_<agent>_<hash>
func generateTmuxSessionName(worktree *git.WorktreeInfo, agentName string) string {
	if worktree == nil {
		return ""
	}

	// Create a base name combining repo, branch, and agent
	baseName := worktree.RepoName + "_" + worktree.Branch + "_" + agentName

	// Use tmux's existing sanitization which includes hash for uniqueness
	return tmux.SanitizeName(baseName)
}

// generateWorktreeKey creates a stable key for identifying this worktree
func generateWorktreeKey(worktree *git.WorktreeInfo) string {
	if worktree == nil {
		return ""
	}
	// Use repo name and path for uniqueness - branch can change
	return worktree.RepoName + ":" + worktree.Path
}