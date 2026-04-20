package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type MangaDexResponse struct {
	Data []MangaDexManga `json:"data"`
}

type MangaDexManga struct {
	ID            string             `json:"id"`
	Attributes    MangaDexAttributes `json:"attributes"`
	Relationships []MangaDexRel      `json:"relationships"`
}

type MangaDexAttributes struct {
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	Status      string            `json:"status"`
	Tags        []MangaDexTag     `json:"tags"`
	LastChapter string            `json:"lastChapter"`
}

type MangaDexTag struct {
	Attributes struct {
		Name map[string]string `json:"name"`
	} `json:"attributes"`
}

type MangaDexRel struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

func fetchMangaDex(offset int) (*MangaDexResponse, error) {
	apiURL := fmt.Sprintf(
		"https://api.mangadex.org/manga?limit=25&offset=%d&contentRating[]=safe&contentRating[]=suggestive&includes[]=author&includes[]=cover_art",
		offset,
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MangaHub/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result MangaDexResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// extractCoverURL builds the MangaDex CDN cover URL from the relationships array.
func extractCoverURL(mangaID string, rels []MangaDexRel) string {
	for _, rel := range rels {
		if rel.Type == "cover_art" && rel.Attributes != nil {
			if fileName, ok := rel.Attributes["fileName"].(string); ok && fileName != "" {
				return fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.256.jpg", mangaID, fileName)
			}
		}
	}
	return ""
}

func main() {
	db, err := sql.Open("sqlite3", "./data/mangahub.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, _ = db.Exec(`ALTER TABLE manga ADD COLUMN cover_url TEXT`)

	totalInserted := 0
	offsets := []int{0, 25, 50, 75}

	for _, offset := range offsets {
		log.Printf("Fetching manga from MangaDex (offset: %d)...", offset)

		result, err := fetchMangaDex(offset)
		if err != nil {
			log.Printf("Failed to fetch at offset %d: %v", offset, err)
			continue
		}

		for _, m := range result.Data {
			title := m.Attributes.Title["en"]
			if title == "" {
				title = m.Attributes.Title["ja-ro"]
			}
			if title == "" {
				continue
			}

			description := m.Attributes.Description["en"]
			if len(description) > 300 {
				description = description[:300] + "..."
			}

			var genres []string
			for _, tag := range m.Attributes.Tags {
				name := tag.Attributes.Name["en"]
				if name != "" {
					genres = append(genres, name)
				}
			}
			if len(genres) == 0 {
				genres = []string{"Unknown"}
			}

			chapters := 0
			if m.Attributes.LastChapter != "" {
				fmt.Sscanf(m.Attributes.LastChapter, "%d", &chapters)
			}

			coverURL := extractCoverURL(m.ID, m.Relationships)

			cleanID := "mdx-" + strings.ReplaceAll(m.ID[:8], "-", "")

			genresJSON, _ := json.Marshal(genres)
			_, err := db.Exec(
				`INSERT OR REPLACE INTO manga (id, title, author, genres, status, total_chapters, description, cover_url)
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				cleanID, title, "Unknown", string(genresJSON),
				m.Attributes.Status, chapters, description, coverURL,
			)
			if err != nil {
				log.Printf("Failed to insert %s: %v", title, err)
				continue
			}
			totalInserted++
			log.Printf("✅ Inserted: %s (cover: %v)", title, coverURL != "")
		}
	}

	log.Printf("🎉 MangaDex sync complete! %d manga added.", totalInserted)
}
