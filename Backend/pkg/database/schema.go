package database

import (
	"database/sql"
	"log"
)

func CreateTables(db *sql.DB) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS manga (
            id TEXT PRIMARY KEY,
            title TEXT NOT NULL,
            author TEXT,
            genres TEXT,
            status TEXT,
            total_chapters INTEGER,
            description TEXT
        );`,
		`CREATE TABLE IF NOT EXISTS user_progress (
            user_id TEXT,
            manga_id TEXT,
            current_chapter INTEGER,
            status TEXT,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (user_id, manga_id)
        );`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatal("Failed to create table:", err)
		}
	}

	log.Println("✅ All tables created successfully!")
}
