package api

import (
	"fmt"
	"net/url"
	"os"

	agateclient "agate/sdk/gen"
)

const (
	envServerURL       = "AGATE_SERVER_URL"
	defaultServerURL   = "http://localhost:24283"
	defaultUserAgent   = "agate-client/0.0.1"
)

// New creates a configured API client for talking to the Agate server.
func New() (*agateclient.APIClient, error) {
	baseURL := os.Getenv(envServerURL)
	if baseURL == "" {
		baseURL = defaultServerURL
	}

	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid %s %q: %w", envServerURL, baseURL, err)
	}

	cfg := agateclient.NewConfiguration()
	cfg.Servers = agateclient.ServerConfigurations{
		{
			URL:         baseURL,
			Description: "Agate Server",
		},
	}
	cfg.UserAgent = defaultUserAgent

	return agateclient.NewAPIClient(cfg), nil
}
