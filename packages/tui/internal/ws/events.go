package ws

// EventType represents the type of WebSocket event
type EventType string

const (
	// EventPtyOutput indicates PTY output data from a session
	EventPtyOutput EventType = "pty:output"

	// EventSessionCreated indicates a new session was created
	EventSessionCreated EventType = "session.created"

	// EventSessionDeleted indicates a session was deleted
	EventSessionDeleted EventType = "session.deleted"
)

// Event represents a WebSocket event received from the server
type Event struct {
	Type      EventType
	SessionID string
	Data      string
}
