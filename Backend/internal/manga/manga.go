package manga

import (
	"database/sql"
	"encoding/json"
	"mangahub/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAllManga(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		search := c.Query("search")
		status := c.Query("status")

		query := "SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE 1=1"
		args := []interface{}{}

		if search != "" {
			query += " AND (title LIKE ? OR author LIKE ?)"
			args = append(args, "%"+search+"%", "%"+search+"%")
		}
		if status != "" {
			query += " AND status = ?"
			args = append(args, status)
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch manga"})
			return
		}
		defer rows.Close()

		var mangaList []models.Manga
		for rows.Next() {
			var m models.Manga
			var genresJSON string
			err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description)
			if err != nil {
				continue
			}
			json.Unmarshal([]byte(genresJSON), &m.Genres)
			mangaList = append(mangaList, m)
		}

		if mangaList == nil {
			mangaList = []models.Manga{}
		}

		c.JSON(http.StatusOK, gin.H{"data": mangaList, "count": len(mangaList)})
	}
}

func GetMangaByID(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var m models.Manga
		var genresJSON string
		err := db.QueryRow(
			"SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE id = ?", id,
		).Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Manga not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch manga"})
			return
		}

		json.Unmarshal([]byte(genresJSON), &m.Genres)
		c.JSON(http.StatusOK, m)
	}
}
