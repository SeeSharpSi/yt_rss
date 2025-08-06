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

	_, err := DB.Exec(usersTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(channelsTable)
	if err != nil {
		log.Fatal(err)
	}
}