package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

func Connect(databaseURL string) (*sql.DB, error) {
	databaseURL = normalizeDatabaseURL(databaseURL)
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return conn, nil
}

func normalizeDatabaseURL(databaseURL string) string {
	if databaseURL == "" || strings.Contains(databaseURL, "sslmode=") {
		return databaseURL
	}
	sep := "?"
	if strings.Contains(databaseURL, "?") {
		sep = "&"
	}
	return databaseURL + sep + "sslmode=disable"
}
