package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (bytes)
	maxMessageSize = 4096

	// Client send buffer size
	sendBufferSize = 64
)

// IncomingMessage is the structure expected from the client over WebSocket
type IncomingMessage struct {
	Message string `json:"message"`
}

// Client represents a single WebSocket connection
type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan ChatMessage
	UserID   string
	Username string
	RoomID   string
}

// ReadPump pumps messages from the WebSocket connection to the hub.
// Runs in a goroutine per client.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.Printf("WebSocket read error (user=%s): %v", c.Username, err)
			}
			break
		}

		var incoming IncomingMessage
		if err := json.Unmarshal(rawMsg, &incoming); err != nil {
			// Send error back to this client only
			errMsg := ChatMessage{
				Type:      "error",
				RoomID:    c.RoomID,
				UserID:    c.UserID,
				Username:  c.Username,
				Message:   "Invalid message format. Expected: {\"message\": \"your text\"}",
				Timestamp: time.Now().Unix(),
			}
			select {
			case c.Send <- errMsg:
			default:
			}
			continue
		}

		if incoming.Message == "" {
			continue
		}

		outMsg := ChatMessage{
			Type:      "message",
			RoomID:    c.RoomID,
			UserID:    c.UserID,
			Username:  c.Username,
			Message:   incoming.Message,
			Timestamp: time.Now().Unix(),
		}

		// Send to hub for broadcasting; also echo back to sender
		c.Hub.broadcastToRoom(c.RoomID, outMsg, nil)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
// Runs in a goroutine per client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket write error (user=%s): %v", c.Username, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
