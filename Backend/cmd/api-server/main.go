package main

import (
	"log"
	"mangahub/internal/auth"
	"mangahub/internal/manga"
	"mangahub/internal/tcp"
	"mangahub/internal/udp"
	"mangahub/internal/user"
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

	// Init Gin router
	r := gin.Default()

	// Public routes
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "MangaHub is running!"})
	})
	r.POST("/auth/register", auth.Register(db))
	r.POST("/auth/login", auth.Login(db))

	// Protected routes
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
	}

	log.Println("🚀 API Server running on port 8080")
	r.Run(":8080")
}
