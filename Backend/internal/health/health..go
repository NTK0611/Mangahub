package health

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"time"

	grpcint "mangahub/internal/grpc"
	"mangahub/internal/tcp"
	"mangahub/internal/udp"
	ws "mangahub/internal/websocket"

	"github.com/gin-gonic/gin"
)

// ServiceStatus holds the health info of a single service.
type ServiceStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "up" | "down"
	Port    string `json:"port,omitempty"`
	Details string `json:"details,omitempty"`
}

// HealthResponse is the JSON body returned by GET /health.
type HealthResponse struct {
	Status    string          `json:"status"` // "healthy" | "degraded"
	Timestamp int64           `json:"timestamp"`
	Services  []ServiceStatus `json:"services"`
}

// HealthCheck returns a Gin handler that probes every service and reports
// the aggregate health of the MangaHub system.
//
//	GET /health
//	200 → all services up
//	503 → one or more services down (response still contains full detail)
func HealthCheck(
	db *sql.DB,
	tcpServer *tcp.TCPServer,
	udpServer *udp.UDPServer,
	hub *ws.Hub,
	grpcClient *grpcint.MangaClient,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		services := make([]ServiceStatus, 0, 6)
		allUp := true

		// ── 1. HTTP REST API ─────────────────────────────────────────────────
		// The fact that this handler is executing means HTTP is up.
		services = append(services, ServiceStatus{
			Name:    "HTTP REST API",
			Status:  "up",
			Port:    "8080",
			Details: "Gin server running",
		})

		// ── 2. SQLite Database ───────────────────────────────────────────────
		dbSvc := checkDatabase(db)
		services = append(services, dbSvc)
		if dbSvc.Status == "down" {
			allUp = false
		}

		// ── 3. TCP Progress Sync ─────────────────────────────────────────────
		tcpSvc := checkTCP(tcpServer)
		services = append(services, tcpSvc)
		if tcpSvc.Status == "down" {
			allUp = false
		}

		// ── 4. UDP Notifications ─────────────────────────────────────────────
		udpSvc := checkUDP(udpServer)
		services = append(services, udpSvc)
		if udpSvc.Status == "down" {
			allUp = false
		}

		// ── 5. WebSocket Chat ────────────────────────────────────────────────
		rooms := hub.GetAllRooms()
		totalUsers := 0
		for _, count := range rooms {
			totalUsers += count
		}
		services = append(services, ServiceStatus{
			Name:    "WebSocket Chat",
			Status:  "up",
			Port:    "8080",
			Details: fmt.Sprintf("%d active rooms, %d users connected", len(rooms), totalUsers),
		})

		// ── 6. gRPC MangaService ─────────────────────────────────────────────
		grpcSvc := checkGRPC(grpcClient)
		services = append(services, grpcSvc)
		if grpcSvc.Status == "down" {
			allUp = false
		}

		// ── Build response ───────────────────────────────────────────────────
		overall := "healthy"
		httpCode := http.StatusOK
		if !allUp {
			overall = "degraded"
			httpCode = http.StatusServiceUnavailable
		}

		c.JSON(httpCode, HealthResponse{
			Status:    overall,
			Timestamp: time.Now().Unix(),
			Services:  services,
		})
	}
}

// ── private probes ────────────────────────────────────────────────────────────

func checkDatabase(db *sql.DB) ServiceStatus {
	if err := db.Ping(); err != nil {
		return ServiceStatus{
			Name:    "SQLite Database",
			Status:  "down",
			Details: err.Error(),
		}
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM manga").Scan(&count)
	return ServiceStatus{
		Name:    "SQLite Database",
		Status:  "up",
		Details: fmt.Sprintf("%d manga entries", count),
	}
}

func checkTCP(s *tcp.TCPServer) ServiceStatus {
	// Dial the TCP port to confirm the listener is accepting connections.
	conn, err := net.DialTimeout("tcp", "localhost:"+s.Port, 2*time.Second)
	if err != nil {
		return ServiceStatus{
			Name:    "TCP Progress Sync",
			Status:  "down",
			Port:    s.Port,
			Details: err.Error(),
		}
	}
	conn.Close()
	return ServiceStatus{
		Name:    "TCP Progress Sync",
		Status:  "up",
		Port:    s.Port,
		Details: fmt.Sprintf("%d active connections", s.GetConnectionCount()),
	}
}

func checkUDP(s *udp.UDPServer) ServiceStatus {
	if s == nil || !s.IsRunning() {
		return ServiceStatus{
			Name:    "UDP Notifications",
			Status:  "down",
			Port:    "9091",
			Details: "server socket not open",
		}
	}
	return ServiceStatus{
		Name:    "UDP Notifications",
		Status:  "up",
		Port:    s.Port,
		Details: fmt.Sprintf("%d registered clients", s.GetClientCount()),
	}
}

func checkGRPC(client *grpcint.MangaClient) ServiceStatus {
	if client == nil {
		return ServiceStatus{
			Name:    "gRPC MangaService",
			Status:  "down",
			Port:    "50051",
			Details: "client not connected — run: go run ./cmd/grpc-server",
		}
	}
	// Fire a lightweight search to confirm the server is responding.
	_, _, err := client.SearchManga("", "", "")
	if err != nil {
		return ServiceStatus{
			Name:    "gRPC MangaService",
			Status:  "down",
			Port:    "50051",
			Details: err.Error(),
		}
	}
	return ServiceStatus{
		Name:    "gRPC MangaService",
		Status:  "up",
		Port:    "50051",
		Details: "MangaService responding",
	}
}
