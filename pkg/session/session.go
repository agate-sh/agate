package session

import (
	"sort"
	"time"

	"agate/pkg/app"
	"agate/pkg/git"
	"agate/pkg/tmux"
)

// Session represents a multi-agent workspace with multiple concurrent agents
type Session struct {
	// Identification
	ID              string `json:"id"`
	Prompt          string `json:"prompt"`           // User's initial prompt text
	Description     string `json:"description"`      // AI-generated description (can be empty initially)
	BranchBaseName  string `json:"branch_base_name"` // Base branch name for all worktrees in this session

	// Agents and resources
	Agents           map[string]*Agent `json:"agents"`             // agent name -> Agent
	SharedTmux       *tmux.TmuxSession `json:"-"`                  // Shared tmux session for all agents (not persisted)
	ActiveAgentIndex int               `json:"active_agent_index"` // which agent is currently focused

	// Terminal dimensions
	TermWidth  int `json:"term_width"`  // Terminal width detected at creation
	TermHeight int `json:"term_height"` // Terminal height detected at creation

	// State tracking
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// GetActiveAgent returns the currently active agent
func (s *Session) GetActiveAgent() *Agent {
	agents := s.GetOrderedAgents()
	if len(agents) == 0 {
		return nil
	}
	if s.ActiveAgentIndex < 0 || s.ActiveAgentIndex >= len(agents) {
		return agents[0]
	}
	return agents[s.ActiveAgentIndex]
}

// GetOrderedAgents returns agents in a consistent order for UI rendering
func (s *Session) GetOrderedAgents() []*Agent {
	if len(s.Agents) == 0 {
		return nil
	}

	// Get all agent names and sort them
	names := make([]string, 0, len(s.Agents))
	for name := range s.Agents {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build ordered list
	agents := make([]*Agent, 0, len(s.Agents))
	for _, name := range names {
		agents = append(agents, s.Agents[name])
	}
	return agents
}


// Update refreshes the session's last accessed time
func (s *Session) Update() {
	s.LastAccessed = time.Now()
}

// Deactivate is a no-op for multi-agent sessions (kept for compatibility)
func (s *Session) Deactivate() {
	// No-op - sessions don't have an IsActive flag in the new model
}

// Compatibility properties for accessing the active agent's resources

// Worktree returns the active agent's worktree (for backward compatibility)
func (s *Session) Worktree() *git.WorktreeInfo {
	activeAgent := s.GetActiveAgent()
	if activeAgent == nil {
		return nil
	}
	return activeAgent.Worktree
}

// TmuxSession returns the shared tmux session (for backward compatibility)
// Note: Now returns the session-level tmux session, not per-agent
func (s *Session) TmuxSession() *tmux.TmuxSession {
	return s.SharedTmux
}

// Agent returns the active agent's config (for backward compatibility)
func (s *Session) Agent() app.AgentConfig {
	activeAgent := s.GetActiveAgent()
	if activeAgent == nil {
		return app.AgentConfig{}
	}
	return activeAgent.AgentConfig
}

// Name returns a display name for the session (for backward compatibility)
// Uses the branch base name as the session name
func (s *Session) Name() string {
	return s.BranchBaseName
}
