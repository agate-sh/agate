package api

import (
	"context"
	"fmt"

	agateclient "agate/sdk/gen"
)

func ListWorktrees(ctx context.Context, client *agateclient.APIClient, repoPath string) ([]agateclient.GitWorktreesList200ResponseWorktreesInner, error) {
	resp, _, err := client.DefaultAPI.GitWorktreesList(ctx).RepoPath(repoPath).Execute()
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}
	return resp.GetWorktrees(), nil
}

func CreateWorktree(ctx context.Context, client *agateclient.APIClient, payload agateclient.GitWorktreeCreateRequest) error {
	_, _, err := client.DefaultAPI.GitWorktreeCreate(ctx).GitWorktreeCreateRequest(payload).Execute()
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}
	return nil
}

func DeleteWorktree(ctx context.Context, client *agateclient.APIClient, payload agateclient.GitWorktreeDeleteRequest) error {
	_, _, err := client.DefaultAPI.GitWorktreeDelete(ctx).GitWorktreeDeleteRequest(payload).Execute()
	if err != nil {
		return fmt.Errorf("delete worktree: %w", err)
	}
	return nil
}
