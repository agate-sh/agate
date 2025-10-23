package remote

import (
	"context"
	"errors"
	"fmt"
	"strings"

	agateclient "agate/sdk/gen"
)

// CreateWorktreeOptions configures optional parameters for worktree creation.
type CreateWorktreeOptions struct {
	// ExistingBranch, when set, instructs the server to base the worktree off an existing branch.
	ExistingBranch string
	// UseCOW toggles git's copy-on-write semantics. Nil means "use server default".
	UseCOW *bool
}

// ListWorktrees returns all worktrees for the given repository path.
func (c *Client) ListWorktrees(ctx context.Context, repoPath string) ([]Worktree, error) {
	if c == nil {
		return nil, errors.New("remote: client is nil")
	}

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, errors.New("remote: repoPath is required")
	}

	res, httpResp, err := c.sdk.DefaultAPI.
		GitWorktreesList(ctx).
		RepoPath(repoPath).
		Execute()
	if err != nil {
		return nil, wrapAPIError(err, httpResp, "list worktrees")
	}

	worktrees := make([]Worktree, 0, len(res.GetWorktrees()))
	for _, wt := range res.GetWorktrees() {
		worktrees = append(worktrees, Worktree{
			Path:   wt.GetPath(),
			Branch: wt.GetBranch(),
			Commit: wt.GetCommit(),
		})
	}

	return worktrees, nil
}

// CreateWorktree provisions a new worktree at the requested destination.
func (c *Client) CreateWorktree(
	ctx context.Context,
	repoPath string,
	path string,
	branch string,
	opts CreateWorktreeOptions,
) error {
	if c == nil {
		return errors.New("remote: client is nil")
	}

	repoPath = strings.TrimSpace(repoPath)
	path = strings.TrimSpace(path)
	branch = strings.TrimSpace(branch)

	switch {
	case repoPath == "":
		return errors.New("remote: repoPath is required")
	case path == "":
		return errors.New("remote: worktree path is required")
	case branch == "":
		return errors.New("remote: branch is required")
	}

	req := agateclient.NewGitWorktreeCreateRequest(repoPath, path, branch)
	if strings.TrimSpace(opts.ExistingBranch) != "" {
		req.SetExistingBranch(opts.ExistingBranch)
	}
	if opts.UseCOW != nil {
		req.SetUseCOW(*opts.UseCOW)
	}

	_, httpResp, err := c.sdk.DefaultAPI.
		GitWorktreeCreate(ctx).
		GitWorktreeCreateRequest(*req).
		Execute()
	if err != nil {
		return wrapAPIError(err, httpResp, fmt.Sprintf("create worktree %q", path))
	}

	return nil
}

// DeleteWorktree removes the specified worktree. When force is true the server will pass --force to git.
func (c *Client) DeleteWorktree(ctx context.Context, repoPath string, path string, force bool) error {
	if c == nil {
		return errors.New("remote: client is nil")
	}

	repoPath = strings.TrimSpace(repoPath)
	path = strings.TrimSpace(path)

	switch {
	case repoPath == "":
		return errors.New("remote: repoPath is required")
	case path == "":
		return errors.New("remote: worktree path is required")
	}

	req := agateclient.NewGitWorktreeDeleteRequest(repoPath, path)
	if force {
		req.SetForce(force)
	}

	_, httpResp, err := c.sdk.DefaultAPI.
		GitWorktreeDelete(ctx).
		GitWorktreeDeleteRequest(*req).
		Execute()
	if err != nil {
		return wrapAPIError(err, httpResp, fmt.Sprintf("delete worktree %q", path))
	}

	return nil
}
