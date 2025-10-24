package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client manages a WebSocket connection to the Agate server
type Client struct {
	url       string
	conn      *websocket.Conn
	eventChan chan Event
	mu        sync.RWMutex
	connected bool
	stopChan  chan struct{}
}

// wsMessage represents a WebSocket message sent to/from the server
type wsMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId,omitempty"`
	Data      string `json:"data,omitempty"`
}

// NewClient creates a new WebSocket client
func NewClient(url string) *Client {
	return &Client{
		url:       url,
		eventChan: make(chan Event, 100), // Buffered channel to prevent blocking
		stopChan:  make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the server
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil // Already connected
	}

	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	c.conn = conn
	c.connected = true

	// Start reading messages in a goroutine
	go c.readLoop()

	return nil
}

// readLoop continuously reads messages from the WebSocket
func (c *Client) readLoop() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.stopChan:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				// Connection closed or error - could trigger reconnection here
				return
			}

			var msg wsMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue // Skip malformed messages
			}

			// Convert to Event and send to channel
			event := Event{
				Type:      EventType(msg.Type),
				SessionID: msg.SessionID,
				Data:      msg.Data,
			}

			// Non-blocking send to event channel
			select {
			case c.eventChan <- event:
			default:
				// Channel full, drop event (or could log here)
			}
		}
	}
}

// Subscribe sends a subscribe message for a session
func (c *Client) Subscribe(sessionID string) error {
	msg := wsMessage{
		Type:      "subscribe",
		SessionID: sessionID,
	}
	return c.sendMessage(msg)
}

// Unsubscribe sends an unsubscribe message
func (c *Client) Unsubscribe() error {
	msg := wsMessage{
		Type: "unsubscribe",
	}
	return c.sendMessage(msg)
}

// SendInput sends input to a session's PTY
func (c *Client) SendInput(sessionID, data string) error {
	msg := wsMessage{
		Type:      "pty:input",
		SessionID: sessionID,
		Data:      data,
	}
	return c.sendMessage(msg)
}

// sendMessage sends a message over the WebSocket connection
func (c *Client) sendMessage(msg wsMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Events returns the channel for receiving WebSocket events
func (c *Client) Events() <-chan Event {
	return c.eventChan
}

// IsConnected returns whether the client is currently connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Close closes the WebSocket connection
func (c *Client) Close() error {
	close(c.stopChan)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.connected = false
		return err
	}

	return nil
}

// ConnectWithRetry attempts to connect with exponential backoff
func (c *Client) ConnectWithRetry(maxRetries int) error {
	backoff := 100 * time.Millisecond
	maxBackoff := 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		err := c.Connect()
		if err == nil {
			return nil
		}

		// Wait with exponential backoff
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return fmt.Errorf("failed to connect after %d retries", maxRetries)
}
