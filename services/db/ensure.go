package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/lib/pq"
)

func EnsureDatabase(databaseURL string) error {
	databaseURL = normalizeDatabaseURL(databaseURL)
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	dbName, adminURL, err := splitDatabaseURL(databaseURL)
	if err != nil {
		return err
	}
	if dbName == "" || dbName == "postgres" {
		return nil
	}

	conn, err := sql.Open("postgres", adminURL)
	if err != nil {
		return fmt.Errorf("open admin database: %w", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		return fmt.Errorf("ping admin database: %w", err)
	}

	var exists bool
	if err := conn.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists); err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	if exists {
		return nil
	}

	if !isValidDatabaseName(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}

	if _, err := conn.Exec(`CREATE DATABASE ` + quoteIdent(dbName)); err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}

	return nil
}

func splitDatabaseURL(databaseURL string) (dbName, adminURL string, err error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", "", fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	dbName = strings.TrimPrefix(u.Path, "/")
	if i := strings.Index(dbName, "/"); i >= 0 {
		dbName = dbName[:i]
	}

	admin := *u
	admin.Path = "/postgres"
	return dbName, admin.String(), nil
}

func isValidDatabaseName(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		case r == '-' && i > 0 && i < len(name)-1:
		default:
			return false
		}
	}
	return true
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
