package tcp

import (
	"encoding/json"
	"log"
	"net"
	"sync"
)

type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

type TCPServer struct {
	Port        string
	Connections map[string]net.Conn
	Broadcast   chan ProgressUpdate
	mutex       sync.Mutex
}

func NewTCPServer(port string) *TCPServer {
	return &TCPServer{
		Port:        port,
		Connections: make(map[string]net.Conn),
		Broadcast:   make(chan ProgressUpdate, 100),
	}
}

func (s *TCPServer) Start() {
	listener, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		log.Fatal("Failed to start TCP server:", err)
	}
	defer listener.Close()
	log.Println("✅ TCP server running on port", s.Port)

	// Start broadcaster
	go s.handleBroadcast()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	addr := conn.RemoteAddr().String()
	log.Println("New TCP connection:", addr)

	s.mutex.Lock()
	s.Connections[addr] = conn
	s.mutex.Unlock()

	defer func() {
		s.mutex.Lock()
		delete(s.Connections, addr)
		s.mutex.Unlock()
		conn.Close()
		log.Println("TCP connection closed:", addr)
	}()

	// Read messages from client
	decoder := json.NewDecoder(conn)
	for {
		var update ProgressUpdate
		if err := decoder.Decode(&update); err != nil {
			break
		}
		log.Printf("Received progress update from %s: %+v", addr, update)
		s.Broadcast <- update
	}
}

func (s *TCPServer) handleBroadcast() {
	for update := range s.Broadcast {
		s.mutex.Lock()
		for addr, conn := range s.Connections {
			encoder := json.NewEncoder(conn)
			if err := encoder.Encode(update); err != nil {
				log.Println("Failed to send to", addr)
				conn.Close()
				delete(s.Connections, addr)
			}
		}
		s.mutex.Unlock()
	}
}

func (s *TCPServer) BroadcastUpdate(update ProgressUpdate) {
	s.Broadcast <- update
}
