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
	Prompt          string `json:"prompt"`          // User's initial prompt text
	Description     string `json:"description"`     // AI-generated description (can be empty initially)
	BranchBaseName  string `json:"branch_base_name"` // Base branch name for all worktrees in this session

	// Agent instances
	Instances           map[string]*AgentInstance `json:"instances"`            // agent name -> AgentInstance
	ActiveInstanceIndex int                       `json:"active_instance_index"` // which agent is currently focused

	// State tracking
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// GetActiveInstance returns the currently active agent instance
func (s *Session) GetActiveInstance() *AgentInstance {
	instances := s.GetOrderedInstances()
	if len(instances) == 0 {
		return nil
	}
	if s.ActiveInstanceIndex < 0 || s.ActiveInstanceIndex >= len(instances) {
		return instances[0]
	}
	return instances[s.ActiveInstanceIndex]
}

// GetOrderedInstances returns instances in a consistent order for UI rendering
func (s *Session) GetOrderedInstances() []*AgentInstance {
	if len(s.Instances) == 0 {
		return nil
	}

	// Get all agent names and sort them
	names := make([]string, 0, len(s.Instances))
	for name := range s.Instances {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build ordered list
	instances := make([]*AgentInstance, 0, len(s.Instances))
	for _, name := range names {
		instances = append(instances, s.Instances[name])
	}
	return instances
}

// Update refreshes the session's last accessed time
func (s *Session) Update() {
	s.LastAccessed = time.Now()
}

// Deactivate is a no-op for multi-agent sessions (kept for compatibility)
func (s *Session) Deactivate() {
	// No-op - sessions don't have an IsActive flag in the new model
}

// Compatibility properties for accessing the active instance's resources

// Worktree returns the active agent instance's worktree (for backward compatibility)
func (s *Session) Worktree() *git.WorktreeInfo {
	activeInstance := s.GetActiveInstance()
	if activeInstance == nil {
		return nil
	}
	return activeInstance.Worktree
}

// TmuxSession returns the active agent instance's tmux session (for backward compatibility)
func (s *Session) TmuxSession() *tmux.TmuxSession {
	activeInstance := s.GetActiveInstance()
	if activeInstance == nil {
		return nil
	}
	return activeInstance.TmuxSession
}

// Agent returns the active agent instance's config (for backward compatibility)
func (s *Session) Agent() app.AgentConfig {
	activeInstance := s.GetActiveInstance()
	if activeInstance == nil {
		return app.AgentConfig{}
	}
	return activeInstance.AgentConfig
}

// Name returns a display name for the session (for backward compatibility)
// Uses the branch base name as the session name
func (s *Session) Name() string {
	return s.BranchBaseName
}
