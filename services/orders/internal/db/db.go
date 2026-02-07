package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var conn *sql.DB

func Init() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	conn = db
	return nil
}

func DB() *sql.DB {
	return conn
}
