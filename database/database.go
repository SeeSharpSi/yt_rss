package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "./yt_rss.db")
	if err != nil {
		log.Fatal(err)
	}

	createTables()
}

func createTables() {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		theme TEXT NOT NULL DEFAULT 'rose-pine'
	);
	`
	channelsTable := `
	CREATE TABLE IF NOT EXISTS channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	`

	videoProgressTable := `
	CREATE TABLE IF NOT EXISTS video_progress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		video_id TEXT NOT NULL,
		progress_seconds INTEGER NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id),
		UNIQUE(user_id, video_id)
	);
	`

	summariesTable := `
	CREATE TABLE IF NOT EXISTS summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		video_id TEXT NOT NULL UNIQUE,
		summary TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := DB.Exec(usersTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(channelsTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(videoProgressTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(summariesTable)
	if err != nil {
		log.Fatal(err)
	}
}

// GetSummary retrieves a summary for a video ID
func GetSummary(videoID string) (string, error) {
	var summary string
	err := DB.QueryRow("SELECT summary FROM summaries WHERE video_id = ?", videoID).Scan(&summary)
	if err != nil {
		return "", err
	}
	return summary, nil
}

// SaveSummary saves a summary for a video ID
func SaveSummary(videoID, summary string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO summaries (video_id, summary) VALUES (?, ?)", videoID, summary)
	return err
}
