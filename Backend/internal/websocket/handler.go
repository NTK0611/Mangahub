package websocket

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for development; restrict in production
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWS upgrades an HTTP request to a WebSocket connection and registers
// the client in the specified manga chat room.
//
// URL: GET /ws/chat/:room_id
// Auth: JWT token via query param ?token=<jwt>  (since browsers can't set
//
//	custom headers in WebSocket handshakes)
//
// The room_id is typically the manga ID (e.g. "one-piece"), creating one
// chat room per manga series.
func ServeWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomID := c.Param("room_id")
		if roomID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room_id is required"})
			return
		}

		// Extract user identity set by JWT middleware
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		username, _ := c.Get("username")

		// Upgrade HTTP → WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}

		client := &Client{
			Hub:      hub,
			Conn:     conn,
			Send:     make(chan ChatMessage, sendBufferSize),
			UserID:   userID.(string),
			Username: username.(string),
			RoomID:   roomID,
		}

		// Register client with hub
		hub.Register <- ClientConnection{
			Client: client,
			RoomID: roomID,
		}

		log.Printf("🔌 WebSocket connected: user=%s room=%s", client.Username, roomID)

		// Start pumps in goroutines
		go client.WritePump()
		go client.ReadPump()
	}
}

// GetRooms returns all active chat rooms and their participant counts.
// GET /ws/rooms
func GetRooms(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		rooms := hub.GetAllRooms()

		type RoomInfo struct {
			RoomID       string `json:"room_id"`
			Participants int    `json:"participants"`
		}

		list := make([]RoomInfo, 0, len(rooms))
		for id, count := range rooms {
			list = append(list, RoomInfo{RoomID: id, Participants: count})
		}

		c.JSON(http.StatusOK, gin.H{
			"rooms": list,
			"total": len(list),
		})
	}
}

// GetRoomInfo returns info about a specific room.
// GET /ws/rooms/:room_id
func GetRoomInfo(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomID := c.Param("room_id")
		count, active := hub.GetRoomInfo(roomID)
		c.JSON(http.StatusOK, gin.H{
			"room_id":      roomID,
			"active":       active,
			"participants": count,
		})
	}
}
