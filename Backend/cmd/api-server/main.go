package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mangahub/data"
	"mangahub/internal/auth"
	grpcint "mangahub/internal/grpc"
	"mangahub/internal/health"
	"mangahub/internal/manga"
	"mangahub/internal/tcp"
	"mangahub/internal/udp"
	"mangahub/internal/user"
	ws "mangahub/internal/websocket"
	"mangahub/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// ── Database ──────────────────────────────────────────────────────────────
	dbPath := getEnv("DB_PATH", "./data/mangahub.db")
	db := database.InitDB(dbPath)
	defer db.Close()
	database.CreateTables(db)

	// ── Auto-seed if database is empty ────────────────────────────────────────
	go data.AutoSeed(db)

	// ── TCP Progress Sync Server ──────────────────────────────────────────────
	tcpServer := tcp.NewTCPServer(getEnv("TCP_PORT", "9090"))
	go tcpServer.Start()

	// ── UDP Notification Server ───────────────────────────────────────────────
	udpServer := udp.NewUDPServer(getEnv("UDP_PORT", "9091"))
	go udpServer.Start()

	// ── WebSocket Chat Hub ────────────────────────────────────────────────────
	hub := ws.NewHub()
	go hub.Run()

	// ── gRPC Client (optional — system degrades gracefully if gRPC is down) ───
	time.Sleep(500 * time.Millisecond)

	grpcAddr := getEnv("GRPC_ADDR", "localhost:50051")
	grpcClient, grpcErr := grpcint.NewMangaClient(grpcAddr)
	if grpcErr != nil {
		log.Printf("⚠️  gRPC client unavailable (%v) — manga routes will use direct DB", grpcErr)
		grpcClient = nil
	} else {
		// grpc.NewClient is lazy — probe with a real call to confirm the server is actually up.
		if probeErr := grpcClient.Probe(); probeErr != nil {
			log.Printf("⚠️  gRPC server unreachable (%v) — manga routes will use direct DB", probeErr)
			grpcClient.Close()
			grpcClient = nil
		} else {
			defer grpcClient.Close()
		}
	}

	// ── HTTP Router ───────────────────────────────────────────────────────────
	r := gin.Default()
	r.Use(corsMiddleware())

	// ── Static Frontend ───────────────────────────────────────────────────────
	frontendPath := getEnv("FRONTEND_PATH", "../Frontend")
	r.StaticFile("/", frontendPath+"/index.html")
	r.StaticFile("/index.html", frontendPath+"/index.html")
	r.StaticFile("/home.html", frontendPath+"/home.html")
	r.StaticFile("/manga.html", frontendPath+"/manga.html")
	r.StaticFile("/library.html", frontendPath+"/library.html")
	r.Static("/css", frontendPath+"/css")
	r.Static("/js", frontendPath+"/js")

	// ── Public / system routes ────────────────────────────────────────────────
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "MangaHub is running!",
			"time":    time.Now().Unix(),
		})
	})

	r.GET("/health", health.HealthCheck(db, tcpServer, udpServer, hub, grpcClient))
	r.POST("/auth/register", auth.Register(db))
	r.POST("/auth/login", auth.Login(db))

	// ── Protected routes (Bearer JWT) ────────────────────────────────────────
	protected := r.Group("/")
	protected.Use(auth.JWTMiddleware())
	{
		if grpcClient != nil {
			log.Println("📡 Manga routes: HTTP → gRPC → SQLite")
			protected.GET("/manga", manga.GetAllMangaGRPC(grpcClient))
			protected.GET("/manga/:id", manga.GetMangaByIDGRPC(grpcClient))
		} else {
			log.Println("📦 Manga routes: HTTP → SQLite (gRPC unavailable)")
			protected.GET("/manga", manga.GetAllManga(db))
			protected.GET("/manga/:id", manga.GetMangaByID(db))
		}

		protected.POST("/users/library", user.AddToLibrary(db))
		protected.GET("/users/library", user.GetLibrary(db))
		protected.PUT("/users/progress", user.UpdateProgress(db, tcpServer, udpServer))
		protected.POST("/notifications/send", udp.SendNotification(udpServer))
		protected.GET("/ws/rooms", ws.GetRooms(hub))
		protected.GET("/ws/rooms/:room_id", ws.GetRoomInfo(hub))
	}

	// ── WebSocket routes ──────────────────────────────────────────────────────
	wsGroup := r.Group("/ws")
	wsGroup.Use(ws.WSAuthMiddleware())
	{
		wsGroup.GET("/chat/:room_id", ws.ServeWS(hub))
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + getEnv("HTTP_PORT", "8080"),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Println("🚀  MangaHub API Server  →  http://localhost:8080")
		log.Println("🌐  Frontend             →  http://localhost:8080/")
		log.Println("❤️   Health Check        →  http://localhost:8080/health")
		log.Println("💬  WebSocket Chat       →  ws://localhost:8080/ws/chat/:room_id?token=<jwt>")
		log.Println("📡  TCP Sync Server      →  localhost:9090")
		log.Println("📢  UDP Notifications    →  localhost:9091")
		log.Println("⚙️   gRPC Service         →  localhost:50051")
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down — draining connections (max 10s)…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Forced shutdown: %v", err)
	}
	log.Println("✅ API server stopped gracefully")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
