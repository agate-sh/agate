package remote

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	agateclient "agate/sdk/gen"
)

const timestampLayout = time.RFC3339

// ListSessions returns the persisted sessions along with active/default agent metadata.
func (c *Client) ListSessions(ctx context.Context) (*SessionList, error) {
	if c == nil {
		return nil, errors.New("remote: client is nil")
	}

	res, httpResp, err := c.sdk.DefaultAPI.SessionList(ctx).Execute()
	if err != nil {
		return nil, wrapAPIError(err, httpResp, "list sessions")
	}

	items := res.GetSessions()
	sessions := make([]Session, 0, len(items))

	for _, item := range items {
		createdAt, err := parseTimestamp(item.GetCreatedAt())
		if err != nil {
			return nil, fmt.Errorf("remote: parse createdAt for session %q: %w", item.GetId(), err)
		}

		lastAccessed, err := parseTimestamp(item.GetLastAccessed())
		if err != nil {
			return nil, fmt.Errorf("remote: parse lastAccessed for session %q: %w", item.GetId(), err)
		}

		sessions = append(sessions, Session{
			ID:           item.GetId(),
			WorktreeKey:  item.GetWorktreeKey(),
			TmuxName:     item.GetTmuxName(),
			AgentName:    item.GetAgentName(),
			WorktreePath: item.GetWorktreePath(),
			Branch:       item.GetBranch(),
			RepoName:     item.GetRepoName(),
			CreatedAt:    createdAt,
			LastAccessed: lastAccessed,
		})
	}

	return &SessionList{
		Sessions:      sessions,
		ActiveSession: res.GetActiveSession(),
		DefaultAgent:  res.GetDefaultAgent(),
	}, nil
}

// CreateSession provisions a new tmux session for the provided repository directory.
func (c *Client) CreateSession(ctx context.Context, dir string, branchName string, agentName string) (*SessionCreateResult, error) {
	if c == nil {
		return nil, errors.New("remote: client is nil")
	}

	dir = strings.TrimSpace(dir)
	branchName = strings.TrimSpace(branchName)
	agentName = strings.TrimSpace(agentName)

	switch {
	case dir == "":
		return nil, errors.New("remote: dir is required")
	case branchName == "":
		return nil, errors.New("remote: branchName is required")
	case agentName == "":
		return nil, errors.New("remote: agentName is required")
	}

	req := agateclient.NewSessionCreateRequest(dir, branchName, agentName)

	res, httpResp, err := c.sdk.DefaultAPI.
		SessionCreate(ctx).
		SessionCreateRequest(*req).
		Execute()
	if err != nil {
		return nil, wrapAPIError(err, httpResp, fmt.Sprintf("create session for %q", dir))
	}

	return &SessionCreateResult{
		SessionID:    res.GetSessionId(),
		TmuxName:     res.GetTmuxName(),
		WorktreePath: res.GetWorktreePath(),
		BranchName:   res.GetBranchName(),
		RepoName:     res.GetRepoName(),
	}, nil
}

// GetSession fetches runtime tmux metadata for the given session identifier.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	if c == nil {
		return nil, errors.New("remote: client is nil")
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("remote: sessionID is required")
	}

	res, httpResp, err := c.sdk.DefaultAPI.
		SessionGet(ctx, sessionID).
		Execute()
	if err != nil {
		return nil, wrapAPIError(err, httpResp, fmt.Sprintf("get session %q", sessionID))
	}

	info := &SessionInfo{
		ID:        res.GetId(),
		Name:      res.GetName(),
		AgentName: res.GetAgent(),
		CWD:       res.GetCwd(),
		Cols:      int32(res.GetCols()),
		Rows:      int32(res.GetRows()),
		IsAlive:   res.GetIsAlive(),
	}

	if pid, ok := res.GetPidOk(); ok && pid != nil {
		p := int32(*pid)
		info.PID = &p
	}

	return info, nil
}

// DeleteSession terminates the session and removes its persisted metadata.
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	if c == nil {
		return errors.New("remote: client is nil")
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("remote: sessionID is required")
	}

	_, httpResp, err := c.sdk.DefaultAPI.
		SessionDelete(ctx, sessionID).
		Execute()
	if err != nil {
		return wrapAPIError(err, httpResp, fmt.Sprintf("delete session %q", sessionID))
	}

	return nil
}

// ResizeSession updates the tmux session size.
func (c *Client) ResizeSession(ctx context.Context, sessionID string, cols int, rows int) error {
	if c == nil {
		return errors.New("remote: client is nil")
	}

	sessionID = strings.TrimSpace(sessionID)
	switch {
	case sessionID == "":
		return errors.New("remote: sessionID is required")
	case cols <= 0:
		return errors.New("remote: cols must be positive")
	case rows <= 0:
		return errors.New("remote: rows must be positive")
	}

	req := agateclient.NewSessionResizeRequest(float32(cols), float32(rows))

	_, httpResp, err := c.sdk.DefaultAPI.
		SessionResize(ctx, sessionID).
		SessionResizeRequest(*req).
		Execute()
	if err != nil {
		return wrapAPIError(err, httpResp, fmt.Sprintf("resize session %q", sessionID))
	}

	return nil
}

func parseTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(timestampLayout, value)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
