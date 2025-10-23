package remote

import "time"

// Worktree represents a git worktree entry returned by the server.
type Worktree struct {
	Path   string
	Branch string
	Commit string
}

// Session captures persisted session metadata from the server.
type Session struct {
	ID           string
	WorktreeKey  string
	TmuxName     string
	AgentName    string
	WorktreePath string
	Branch       string
	RepoName     string
	CreatedAt    time.Time
	LastAccessed time.Time
}

// SessionList bundles the persisted sessions alongside active/default agent information.
type SessionList struct {
	Sessions      []Session
	ActiveSession string
	DefaultAgent  string
}

// SessionCreateResult describes the data returned when a new session is created.
type SessionCreateResult struct {
	SessionID    string
	TmuxName     string
	WorktreePath string
	BranchName   string
	RepoName     string
}

// SessionInfo contains realtime metadata about a tmux session.
type SessionInfo struct {
	ID        string
	Name      string
	AgentName string
	CWD       string
	Cols      int32
	Rows      int32
	PID       *int32
	IsAlive   bool
}
