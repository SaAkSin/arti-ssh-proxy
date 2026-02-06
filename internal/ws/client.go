package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ControlMessage represents a control packet (Text frame)
type ControlMessage struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
}

// Client handles the WebSocket connection
type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex
	url  string
}

// NewClient creates a new WebSocket client
func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

// Connect establishes the WebSocket connection
func (c *Client) Connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  nil, // Use default TLS config (TLS 1.3 is supported by default in Go)
	}

	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// WriteBinary sends raw data to the server
func (c *Client) WriteBinary(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("connection not established")
	}
	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

// SendPing sends a ping control message
func (c *Client) SendPing() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("connection not established")
	}
	// Using custom ping payload in text/json or standard WS ping?
	// Requirement says "Heartbeat(Ping/Pong)". Standard WS Ping/Pong is best managed by the library,
	// but app-level heartbeat often uses explicit messages.
	// Let's use Standard WS Ping first.
	return c.conn.WriteMessage(websocket.PingMessage, []byte{})
}

// Close closes the connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// ReadLoop reads messages and dispatches them
// onData: callback for PTY data
// onResize: callback for resize events
// returns error when connection closes
func (c *Client) ReadLoop(onData func([]byte), onResize func(uint16, uint16)) error {
	if c.conn == nil {
		return fmt.Errorf("connection not established")
	}

	for {
		msgType, p, err := c.conn.ReadMessage()
		if err != nil {
			return err
		}

		switch msgType {
		case websocket.BinaryMessage:
			if onData != nil {
				onData(p)
			}
		case websocket.TextMessage:
			// Parse Control Message
			var msg ControlMessage
			if err := json.Unmarshal(p, &msg); err != nil {
				log.Printf("Invalid control message: %v", err)
				continue
			}
			if msg.Type == "resize" && onResize != nil {
				onResize(msg.Rows, msg.Cols)
			}
			// Add other control types here (e.g., "ping" if app-level heartbeat)
		}
	}
}
