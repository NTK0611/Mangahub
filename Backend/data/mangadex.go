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

// CoverResponse is the response from GET /cover?manga[]=<uuid>
type CoverResponse struct {
	Data []struct {
		Attributes struct {
			FileName string `json:"fileName"`
		} `json:"attributes"`
		Relationships []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"relationships"`
	} `json:"data"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func doGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MangaHub/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func fetchMangaDex(offset int) (*MangaDexResponse, error) {
	apiURL := fmt.Sprintf(
		"https://api.mangadex.org/manga?limit=25&offset=%d&contentRating[]=safe&contentRating[]=suggestive&includes[]=author&includes[]=cover_art",
		offset,
	)
	body, err := doGet(apiURL)
	if err != nil {
		return nil, err
	}
	var result MangaDexResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

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

// fetchCoverForManga calls the /cover endpoint to get the cover for a single manga UUID.
func fetchCoverForManga(mangaUUID string) string {
	time.Sleep(300 * time.Millisecond)
	apiURL := fmt.Sprintf("https://api.mangadex.org/cover?manga[]=%s&limit=1", mangaUUID)
	body, err := doGet(apiURL)
	if err != nil {
		return ""
	}
	var result CoverResponse
	if err := json.Unmarshal(body, &result); err != nil || len(result.Data) == 0 {
		return ""
	}
	fileName := result.Data[0].Attributes.FileName
	if fileName == "" {
		return ""
	}
	return fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.256.jpg", mangaUUID, fileName)
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
			// Skip manga with no cover
			if coverURL == "" {
				continue
			}

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

	// ── Patch pass: fix manga still missing covers ────────────────────────────
	// Some manga in the bulk list response have cover_art relationships with
	// null attributes. We fix them by querying the /cover endpoint individually.
	log.Println("🔍 Patching missing covers...")

	// We need the original MangaDex UUIDs — reconstruct by re-fetching
	// and mapping cleanID → full UUID
	uuidMap := make(map[string]string) // cleanID → full UUID
	for _, offset := range offsets {
		result, err := fetchMangaDex(offset)
		if err != nil {
			continue
		}
		for _, m := range result.Data {
			cleanID := "mdx-" + strings.ReplaceAll(m.ID[:8], "-", "")
			uuidMap[cleanID] = m.ID
		}
	}

	// Find all mdx- manga with empty cover_url
	rows, err := db.Query(`SELECT id FROM manga WHERE (cover_url = '' OR cover_url IS NULL) AND id LIKE 'mdx-%'`)
	if err != nil {
		log.Printf("Failed to query missing covers: %v", err)
		return
	}
	defer rows.Close()

	var toFix []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		toFix = append(toFix, id)
	}
	rows.Close()

	log.Printf("Found %d manga missing covers, fetching individually...", len(toFix))
	fixed := 0
	for _, cleanID := range toFix {
		uuid, ok := uuidMap[cleanID]
		if !ok {
			continue
		}
		coverURL := fetchCoverForManga(uuid)
		if coverURL == "" {
			log.Printf("  ⚠ No cover found for %s", cleanID)
			continue
		}
		db.Exec(`UPDATE manga SET cover_url = ? WHERE id = ?`, coverURL, cleanID)
		fixed++
		log.Printf("  🖼 Fixed cover for %s", cleanID)
	}
	log.Printf("✅ Patched %d/%d missing covers.", fixed, len(toFix))
}
