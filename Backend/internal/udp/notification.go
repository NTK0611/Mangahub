package udp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type NotificationRequest struct {
	MangaID string `json:"manga_id" binding:"required"`
	Message string `json:"message" binding:"required"`
	Type    string `json:"type" binding:"required"`
}

func SendNotification(udpServer *UDPServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req NotificationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		notification := Notification{
			Type:      req.Type,
			MangaID:   req.MangaID,
			Message:   req.Message,
			Timestamp: time.Now().Unix(),
		}

		udpServer.BroadcastNotification(notification)
		c.JSON(http.StatusOK, gin.H{
			"message": "Notification sent",
			"clients": len(udpServer.Clients),
		})
	}
}
