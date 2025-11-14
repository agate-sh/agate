package analytics

import (
	"os"
	"sync"

	"github.com/posthog/posthog-go"
)

var (
	client     posthog.Client
	clientOnce sync.Once
	disabled   bool
)

// Init initializes the PostHog analytics client.
// Must be called before using Track or Identify.
func Init(apiKey string) error {
	var err error
	clientOnce.Do(func() {
		// Check if telemetry is disabled
		if os.Getenv("AGATE_DISABLE_TELEMETRY") != "" {
			disabled = true
			return
		}

		if apiKey == "" {
			disabled = true
			return
		}

		client, err = posthog.NewWithConfig(apiKey, posthog.Config{
			Endpoint: "https://us.i.posthog.com",
		})
	})
	return err
}

// Track records an analytics event.
func Track(distinctID, event string, properties map[string]interface{}) error {
	if disabled || client == nil {
		return nil
	}

	return client.Enqueue(posthog.Capture{
		DistinctId: distinctID,
		Event:      event,
		Properties: properties,
	})
}

// Identify associates user properties with a distinct ID.
func Identify(distinctID string, properties map[string]interface{}) error {
	if disabled || client == nil {
		return nil
	}

	return client.Enqueue(posthog.Identify{
		DistinctId: distinctID,
		Properties: properties,
	})
}

// Close flushes any pending events and closes the client.
func Close() error {
	if disabled || client == nil {
		return nil
	}
	return client.Close()
}
