package websocket

import (
	"log"
	"sync"
	"time"
)

// ChatMessage represents a message sent in a chat room
type ChatMessage struct {
	Type      string `json:"type"` // "message", "join", "leave", "error"
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// ClientConnection holds info about a connecting client
type ClientConnection struct {
	Client *Client
	RoomID string
}

// Room represents a manga chat room
type Room struct {
	ID      string
	MangaID string
	Clients map[*Client]bool
	mu      sync.RWMutex
}

func newRoom(id, mangaID string) *Room {
	return &Room{
		ID:      id,
		MangaID: mangaID,
		Clients: make(map[*Client]bool),
	}
}

func (r *Room) addClient(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Clients[c] = true
}

func (r *Room) removeClient(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Clients, c)
}

func (r *Room) clientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients)
}

// Hub manages all chat rooms and client connections
type Hub struct {
	// rooms maps roomID -> Room
	rooms map[string]*Room
	mu    sync.RWMutex

	// channels
	Register   chan ClientConnection
	Unregister chan *Client
	Broadcast  chan ChatMessage
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		Register:   make(chan ClientConnection, 64),
		Unregister: make(chan *Client, 64),
		Broadcast:  make(chan ChatMessage, 256),
	}
}

// Run starts the hub event loop
func (h *Hub) Run() {
	log.Println("💬 WebSocket Hub started")
	for {
		select {

		case conn := <-h.Register:
			room := h.getOrCreateRoom(conn.RoomID, conn.RoomID)
			room.addClient(conn.Client)
			conn.Client.RoomID = conn.RoomID

			// Notify room that user joined
			joinMsg := ChatMessage{
				Type:      "join",
				RoomID:    conn.RoomID,
				UserID:    conn.Client.UserID,
				Username:  conn.Client.Username,
				Message:   conn.Client.Username + " has joined the room",
				Timestamp: time.Now().Unix(),
			}
			h.broadcastToRoom(conn.RoomID, joinMsg, nil)
			log.Printf("👤 User %s joined room %s (total: %d)", conn.Client.Username, conn.RoomID, room.clientCount())

		case client := <-h.Unregister:
			h.removeClient(client)

		case msg := <-h.Broadcast:
			h.broadcastToRoom(msg.RoomID, msg, nil)
		}
	}
}

// getOrCreateRoom returns existing room or creates a new one
func (h *Hub) getOrCreateRoom(roomID, mangaID string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[roomID]; ok {
		return room
	}
	room := newRoom(roomID, mangaID)
	h.rooms[roomID] = room
	log.Printf("🏠 Room created: %s", roomID)
	return room
}

// removeClient removes a client from its room
func (h *Hub) removeClient(client *Client) {
	h.mu.RLock()
	room, ok := h.rooms[client.RoomID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	room.removeClient(client)

	leaveMsg := ChatMessage{
		Type:      "leave",
		RoomID:    client.RoomID,
		UserID:    client.UserID,
		Username:  client.Username,
		Message:   client.Username + " has left the room",
		Timestamp: time.Now().Unix(),
	}
	h.broadcastToRoom(client.RoomID, leaveMsg, nil)
	log.Printf("👋 User %s left room %s (remaining: %d)", client.Username, client.RoomID, room.clientCount())

	// Clean up empty rooms
	if room.clientCount() == 0 {
		h.mu.Lock()
		delete(h.rooms, client.RoomID)
		h.mu.Unlock()
		log.Printf("🗑️  Room %s deleted (empty)", client.RoomID)
	}
}

// broadcastToRoom sends a message to all clients in a room except the sender
func (h *Hub) broadcastToRoom(roomID string, msg ChatMessage, sender *Client) {
	h.mu.RLock()
	room, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for client := range room.Clients {
		// Skip the sender for regular messages to avoid echo
		// (join/leave are broadcast to everyone including sender)
		if sender != nil && client == sender {
			continue
		}
		select {
		case client.Send <- msg:
		default:
			// Client send buffer full - skip (client will be cleaned up via disconnect)
			log.Printf("⚠️  Send buffer full for user %s, dropping message", client.Username)
		}
	}
}

// GetRoomInfo returns basic info about a room (for REST API)
func (h *Hub) GetRoomInfo(roomID string) (int, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, ok := h.rooms[roomID]
	if !ok {
		return 0, false
	}
	return room.clientCount(), true
}

// GetAllRooms returns a snapshot of all active rooms
func (h *Hub) GetAllRooms() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[string]int, len(h.rooms))
	for id, room := range h.rooms {
		result[id] = room.clientCount()
	}
	return result
}
