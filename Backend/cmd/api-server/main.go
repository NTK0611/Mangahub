package main

import (
	"log"
	"mangahub/internal/auth"
	"mangahub/internal/manga"
	"mangahub/internal/tcp"
	"mangahub/internal/udp"
	"mangahub/internal/user"
	ws "mangahub/internal/websocket"
	"mangahub/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// Init database
	db := database.InitDB("./data/mangahub.db")
	defer db.Close()

	// Create tables
	database.CreateTables(db)

	// Init TCP server
	tcpServer := tcp.NewTCPServer("9090")
	go tcpServer.Start()

	// Init UDP server
	udpServer := udp.NewUDPServer("9091")
	go udpServer.Start()

	// Init WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Init Gin router
	r := gin.Default()

	// CORS middleware (needed for WebSocket upgrade from browsers)
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// ── Public routes ────────────────────────────────────────────────────────
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "MangaHub is running!"})
	})
	r.POST("/auth/register", auth.Register(db))
	r.POST("/auth/login", auth.Login(db))

	// ── Protected routes (Bearer token) ─────────────────────────────────────
	protected := r.Group("/")
	protected.Use(auth.JWTMiddleware())
	{
		// Manga routes
		protected.GET("/manga", manga.GetAllManga(db))
		protected.GET("/manga/:id", manga.GetMangaByID(db))

		// Library routes
		protected.POST("/users/library", user.AddToLibrary(db))
		protected.GET("/users/library", user.GetLibrary(db))
		protected.PUT("/users/progress", user.UpdateProgress(db, tcpServer))

		// Notification route
		protected.POST("/notifications/send", udp.SendNotification(udpServer))

		// Chat room info (REST, requires Bearer auth)
		protected.GET("/ws/rooms", ws.GetRooms(hub))
		protected.GET("/ws/rooms/:room_id", ws.GetRoomInfo(hub))
	}

	// ── WebSocket routes (token via query param) ─────────────────────────────
	// GET /ws/chat/:room_id?token=<jwt>
	// room_id is typically the manga ID, e.g. "one-piece"
	wsGroup := r.Group("/ws")
	wsGroup.Use(ws.WSAuthMiddleware())
	{
		wsGroup.GET("/chat/:room_id", ws.ServeWS(hub))
	}

	log.Println("🚀 API Server running on port 8080")
	log.Println("💬 WebSocket Chat available at ws://localhost:8080/ws/chat/:room_id?token=<jwt>")
	r.Run(":8080")
}
