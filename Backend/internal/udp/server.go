package udp

import (
	"encoding/json"
	"log"
	"net"
	"sync"
)

type Notification struct {
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type UDPServer struct {
	Port    string
	Clients []net.UDPAddr
	mutex   sync.Mutex
	conn    *net.UDPConn
}

func NewUDPServer(port string) *UDPServer {
	return &UDPServer{
		Port:    port,
		Clients: make([]net.UDPAddr, 0),
	}
}

func (s *UDPServer) Start() {
	addr, err := net.ResolveUDPAddr("udp", ":"+s.Port)
	if err != nil {
		log.Fatal("Failed to resolve UDP address:", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("Failed to start UDP server:", err)
	}
	s.conn = conn
	defer conn.Close()

	log.Println("✅ UDP server running on port", s.Port)

	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("UDP read error:", err)
			continue
		}

		message := string(buffer[:n])
		log.Printf("UDP message from %s: %s", clientAddr, message)

		if message == "REGISTER" {
			s.registerClient(*clientAddr)
		}
	}
}

func (s *UDPServer) registerClient(addr net.UDPAddr) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if already registered
	for _, client := range s.Clients {
		if client.String() == addr.String() {
			log.Println("Client already registered:", addr.String())
			return
		}
	}

	s.Clients = append(s.Clients, addr)
	log.Println("✅ New UDP client registered:", addr.String())

	// Send confirmation
	if s.conn != nil {
		s.conn.WriteToUDP([]byte("REGISTERED"), &addr)
	}
}

func (s *UDPServer) BroadcastNotification(notification Notification) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := json.Marshal(notification)
	if err != nil {
		log.Println("Failed to marshal notification:", err)
		return
	}

	for _, client := range s.Clients {
		clientCopy := client
		_, err := s.conn.WriteToUDP(data, &clientCopy)
		if err != nil {
			log.Println("Failed to send to client:", client.String())
		} else {
			log.Printf("📢 Notification sent to %s", client.String())
		}
	}
}
