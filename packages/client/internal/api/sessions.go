package api

import (
	"context"
	"fmt"

	agateclient "agate/sdk/gen"
)

func ListSessions(ctx context.Context, client *agateclient.APIClient) (*agateclient.SessionList200Response, error) {
	resp, _, err := client.DefaultAPI.SessionList(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return resp, nil
}

func CreateSession(ctx context.Context, client *agateclient.APIClient, req agateclient.SessionCreateRequest) (*agateclient.SessionCreate201Response, error) {
	resp, _, err := client.DefaultAPI.SessionCreate(ctx).SessionCreateRequest(req).Execute()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return resp, nil
}

func GetSession(ctx context.Context, client *agateclient.APIClient, id string) (*agateclient.SessionGet200Response, error) {
	resp, _, err := client.DefaultAPI.SessionGet(ctx, id).Execute()
	if err != nil {
		return nil, fmt.Errorf("get session %s: %w", id, err)
	}
	return resp, nil
}

func DeleteSession(ctx context.Context, client *agateclient.APIClient, id string) error {
	_, _, err := client.DefaultAPI.SessionDelete(ctx, id).Execute()
	if err != nil {
		return fmt.Errorf("delete session %s: %w", id, err)
	}
	return nil
}

func ResizeSession(ctx context.Context, client *agateclient.APIClient, id string, cols, rows int) error {
	payload := agateclient.NewSessionResizeRequest(float32(cols), float32(rows))
	_, _, err := client.DefaultAPI.SessionResize(ctx, id).SessionResizeRequest(*payload).Execute()
	if err != nil {
		return fmt.Errorf("resize session %s: %w", id, err)
	}
	return nil
}
