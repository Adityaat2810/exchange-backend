package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var conn *sql.DB

// Init initializes a Postgres connection for the auth service.
func Init() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	var lastErr error
	for i := 0; i < 30; i++ { // up to ~30 * 2s = 60s
		if err = db.Ping(); err == nil {
			conn = db
			return nil
		}
		lastErr = err
		log.Printf("waiting for postgres (%d/30): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("ping after retries: %w", lastErr)
}

// DB returns the shared *sql.DB for this service.
func DB() *sql.DB {
	return conn
}
