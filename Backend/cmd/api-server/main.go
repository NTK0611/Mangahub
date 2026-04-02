package main

import (
	"log"
	"mangahub/internal/auth"
	"mangahub/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.InitDB("./data/mangahub.db")
	defer db.Close()

	database.CreateTables(db)

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "MangaHub is running!"})
	})

	// Auth routes
	r.POST("/auth/register", auth.Register(db))
	r.POST("/auth/login", auth.Login(db))

	log.Println("🚀 Server running on port 8080")
	r.Run(":8080")
}
