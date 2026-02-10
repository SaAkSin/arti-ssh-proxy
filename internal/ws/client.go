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
	Data string `json:"data,omitempty"` // Add Data field for input payload
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
		TLSClientConfig:  nil, // Use default TLS config
	}

	log.Printf("Dialing %s...", c.url)
	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		return err
	}
	
	c.conn = conn
	return nil
}

// WriteData sends data to the server (Text Message for compatibility)
func (c *Client) WriteData(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("connection not established")
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// SendPing sends a ping control message
func (c *Client) SendPing() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("connection not established")
	}
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
			// Try to parse as Control Message
			var msg ControlMessage
			if err := json.Unmarshal(p, &msg); err == nil && msg.Type != "" {
				// It's a valid control message
				if msg.Type == "resize" && onResize != nil {
					onResize(msg.Rows, msg.Cols)
				} else if msg.Type == "input" && onData != nil {
					// Handle input message from gateway
					onData([]byte(msg.Data))
				}
				continue
			}

			// Failure to parse JSON means it's data (input)
			if onData != nil {
				onData(p)
			}
		}
	}
}
