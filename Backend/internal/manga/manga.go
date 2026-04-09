package manga

import (
	"database/sql"
	"encoding/json"
	"net/http"

	grpcint "mangahub/internal/grpc"
	"mangahub/pkg/models"

	"github.com/gin-gonic/gin"
)

// ── Direct-DB handlers (fallback when gRPC is unavailable) ───────────────────

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
			if err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description); err != nil {
				continue
			}
			json.Unmarshal([]byte(genresJSON), &m.Genres)
			mangaList = append(mangaList, m)
		}

		if mangaList == nil {
			mangaList = []models.Manga{}
		}

		c.JSON(http.StatusOK, gin.H{"data": mangaList, "count": len(mangaList), "source": "db"})
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

// ── gRPC-backed handlers (HTTP → gRPC → SQLite) ───────────────────────────────
//
// These handlers route manga queries through the gRPC MangaService, demonstrating
// the HTTP ↔ gRPC integration required by the project spec.

// GetAllMangaGRPC handles GET /manga with search/status/genre filters via gRPC.
func GetAllMangaGRPC(grpcClient *grpcint.MangaClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		search := c.Query("search")
		status := c.Query("status")
		genre := c.Query("genre")

		items, count, err := grpcClient.SearchManga(search, status, genre)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gRPC search failed: " + err.Error()})
			return
		}

		mangaList := make([]models.Manga, 0, len(items))
		for _, item := range items {
			mangaList = append(mangaList, models.Manga{
				ID:            item.Id,
				Title:         item.Title,
				Author:        item.Author,
				Genres:        item.Genres,
				Status:        item.Status,
				TotalChapters: int(item.TotalChapters),
				Description:   item.Description,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"data":   mangaList,
			"count":  count,
			"source": "grpc", // shows the request was routed through gRPC
		})
	}
}

// GetMangaByIDGRPC handles GET /manga/:id via gRPC.
func GetMangaByIDGRPC(grpcClient *grpcint.MangaClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		item, found, err := grpcClient.GetManga(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gRPC request failed: " + err.Error()})
			return
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Manga not found"})
			return
		}

		m := models.Manga{
			ID:            item.Id,
			Title:         item.Title,
			Author:        item.Author,
			Genres:        item.Genres,
			Status:        item.Status,
			TotalChapters: int(item.TotalChapters),
			Description:   item.Description,
		}
		c.JSON(http.StatusOK, m)
	}
}
