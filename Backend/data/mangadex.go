package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type MangaDexResponse struct {
	Data []MangaDexManga `json:"data"`
}

type MangaDexManga struct {
	ID         string             `json:"id"`
	Attributes MangaDexAttributes `json:"attributes"`
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

func fetchMangaDex(offset int) (*MangaDexResponse, error) {
	url := fmt.Sprintf(
		"https://api.mangadex.org/manga?limit=25&offset=%d&contentRating[]=safe&contentRating[]=suggestive&includes[]=author",
		offset,
	)

	resp, err := http.Get(url)
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

func main() {
	db, err := sql.Open("sqlite3", "./data/mangahub.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
			// Get title
			title := m.Attributes.Title["en"]
			if title == "" {
				title = m.Attributes.Title["ja-ro"]
			}
			if title == "" {
				continue
			}

			// Get description
			description := m.Attributes.Description["en"]
			if len(description) > 300 {
				description = description[:300] + "..."
			}

			// Get genres from tags
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

			// Get chapter count
			chapters := 0
			if m.Attributes.LastChapter != "" {
				fmt.Sscanf(m.Attributes.LastChapter, "%d", &chapters)
			}

			// Clean ID for database
			cleanID := "mdx-" + strings.ReplaceAll(m.ID[:8], "-", "")

			genresJSON, _ := json.Marshal(genres)
			_, err := db.Exec(
				`INSERT OR IGNORE INTO manga (id, title, author, genres, status, total_chapters, description)
                 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				cleanID, title, "Unknown", string(genresJSON),
				m.Attributes.Status, chapters, description,
			)
			if err != nil {
				log.Printf("Failed to insert %s: %v", title, err)
				continue
			}
			totalInserted++
			log.Printf("✅ Inserted: %s", title)
		}
	}

	log.Printf("🎉 MangaDex sync complete! %d manga added.", totalInserted)
}
