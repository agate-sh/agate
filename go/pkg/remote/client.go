package remote

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	agateclient "agate/sdk/gen"
)

// DefaultBaseURL is the fallback server location used when no base URL is provided.
const DefaultBaseURL = "http://127.0.0.1:24283"

// Options configure the behaviour of the remote client.
type Options struct {
	// BaseURL is the Agate server root (e.g. http://localhost:24283). Defaults to DefaultBaseURL.
	BaseURL string
	// HTTPClient allows callers to supply a custom HTTP client (for timeouts, tracing, etc.).
	HTTPClient *http.Client
	// UserAgent overrides the default SDK user agent.
	UserAgent string
}

// Client provides a thin wrapper around the generated OpenAPI SDK with domain-friendly helpers.
type Client struct {
	baseURL string
	sdk     *agateclient.APIClient
}

// NewClient constructs a new Client instance from the supplied options.
func NewClient(opts Options) (*Client, error) {
	baseURL := strings.TrimSpace(opts.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("remote: invalid base URL %q: %w", baseURL, err)
	}

	sanitized := strings.TrimRight(parsed.String(), "/")

	cfg := agateclient.NewConfiguration()
	cfg.Servers[0].URL = sanitized

	if opts.HTTPClient != nil {
		cfg.HTTPClient = opts.HTTPClient
	}
	if opts.UserAgent != "" {
		cfg.UserAgent = opts.UserAgent
	}

	return &Client{
		baseURL: sanitized,
		sdk:     agateclient.NewAPIClient(cfg),
	}, nil
}

// BaseURL returns the configured server root.
func (c *Client) BaseURL() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}

// HTTPError captures HTTP-level failures returned by the server.
type HTTPError struct {
	StatusCode int
	Message    string
	Body       []byte
	Err        error
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}

	status := e.StatusCode
	message := strings.TrimSpace(e.Message)
	if status == 0 {
		if message == "" && e.Err != nil {
			return e.Err.Error()
		}
		return message
	}

	if message == "" {
		return fmt.Sprintf("agate server returned status %d", status)
	}
	return fmt.Sprintf("agate server returned status %d: %s", status, message)
}

// Unwrap provides access to the original error from the SDK.
func (e *HTTPError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapAPIError(err error, resp *http.Response, context string) error {
	if err == nil {
		return nil
	}

	if resp == nil {
		if context == "" {
			return err
		}
		return fmt.Errorf("%s: %w", context, err)
	}

	httpErr := &HTTPError{
		StatusCode: resp.StatusCode,
		Message:    err.Error(),
		Err:        err,
	}

	if apiErr, ok := err.(*agateclient.GenericOpenAPIError); ok {
		httpErr.Body = apiErr.Body()
	}

	if context == "" {
		return httpErr
	}
	return fmt.Errorf("%s: %w", context, httpErr)
}
