package user

import (
	"database/sql"
	"mangahub/internal/tcp"
	"mangahub/pkg/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func AddToLibrary(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		var req models.AddToLibraryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		// Check if manga exists
		var mangaID string
		err := db.QueryRow("SELECT id FROM manga WHERE id = ?", req.MangaID).Scan(&mangaID)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Manga not found"})
			return
		}

		// Add to library
		_, err = db.Exec(
			`INSERT INTO user_progress (user_id, manga_id, current_chapter, status, updated_at)
             VALUES (?, ?, ?, ?, ?)
             ON CONFLICT(user_id, manga_id) DO UPDATE SET
             current_chapter = excluded.current_chapter,
             status = excluded.status,
             updated_at = excluded.updated_at`,
			userID, req.MangaID, req.CurrentChapter, req.Status, time.Now().UTC(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to library"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Added to library successfully"})
	}
}

func GetLibrary(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		rows, err := db.Query(
			`SELECT p.manga_id, p.current_chapter, p.status, p.updated_at,
                    m.title, m.author, m.total_chapters
             FROM user_progress p
             JOIN manga m ON p.manga_id = m.id
             WHERE p.user_id = ?`, userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch library"})
			return
		}
		defer rows.Close()

		var library []models.UserProgress
		for rows.Next() {
			var p models.UserProgress
			err := rows.Scan(
				&p.MangaID, &p.CurrentChapter, &p.Status, &p.LastUpdated,
				&p.Title, &p.Author, &p.TotalChapters,
			)
			if err != nil {
				continue
			}
			p.UserID = userID
			library = append(library, p)
		}

		if library == nil {
			library = []models.UserProgress{}
		}

		c.JSON(http.StatusOK, gin.H{"data": library, "count": len(library)})
	}
}

func UpdateProgress(db *sql.DB, tcpServer *tcp.TCPServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		var req models.UpdateProgressRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		result, err := db.Exec(
			`UPDATE user_progress SET current_chapter = ?, status = ?, updated_at = ?
             WHERE user_id = ? AND manga_id = ?`,
			req.CurrentChapter, req.Status, time.Now().UTC(), userID, req.MangaID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Manga not in library"})
			return
		}

		// Broadcast update via TCP
		tcpServer.BroadcastUpdate(tcp.ProgressUpdate{
			UserID:    userID,
			MangaID:   req.MangaID,
			Chapter:   req.CurrentChapter,
			Status:    req.Status,
			Timestamp: time.Now().Unix(),
		})

		c.JSON(http.StatusOK, gin.H{"message": "Progress updated successfully"})
	}
}
