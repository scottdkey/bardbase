// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Command api serves the bardbase Shakespeare database over HTTP.
//
// Environment variables:
//
//	DB_PATH   path to bardbase.db (default: ./bardbase.db)
//	PORT      listen port (default: 8080)
//	API_KEY   shared secret for request auth (optional, disabled if empty)
//
// Usage:
//
//	./api                                    # defaults
//	API_KEY=secret PORT=8080 ./api           # with auth
//	make api run                             # via Makefile
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/scottdkey/bardbase/projects/capell/internal/api"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := envOr("DB_PATH", "./bardbase.db")
	port := envOr("PORT", "8080")
	apiKey := os.Getenv("API_KEY")

	db, err := openReadOnly(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	srv := api.NewServer(db, apiKey)
	addr := ":" + port
	if apiKey != "" {
		log.Printf("bardbase API listening on %s (db: %s, auth: enabled)", addr, dbPath)
	} else {
		log.Printf("bardbase API listening on %s (db: %s, auth: disabled)", addr, dbPath)
	}
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func openReadOnly(dbPath string) (*sql.DB, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database file not found: %s", dbPath)
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA cache_size=-64000",
		"PRAGMA mmap_size=268435456",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma %q: %w", p, err)
		}
	}

	return db, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
