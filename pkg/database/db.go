package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("✅ Database connected successfully!")
	return db
}
