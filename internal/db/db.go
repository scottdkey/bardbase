// Package db handles SQLite database connections and schema management.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scottdkey/shakespeare_db/internal/constants"

	_ "modernc.org/sqlite"
)

// Open creates or opens a SQLite database at the given path.
// It sets optimal pragmas for bulk import performance.
func Open(dbPath string) (*sql.DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Performance pragmas for bulk import
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma %q: %w", pragma, err)
		}
	}

	return db, nil
}

// CreateSchema executes the full DDL schema against the database.
func CreateSchema(db *sql.DB) error {
	_, err := db.Exec(constants.SchemaSQL)
	if err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	return nil
}

// RemoveIfExists deletes the database file if it exists (for clean rebuilds).
func RemoveIfExists(dbPath string) error {
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("removing existing database: %w", err)
		}
	}
	// Also remove WAL/SHM files
	os.Remove(dbPath + "-wal")
	os.Remove(dbPath + "-shm")
	return nil
}

// TableCount returns the row count for the given table.
func TableCount(db *sql.DB, table string) (int, error) {
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM [%s]", table)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting %s: %w", table, err)
	}
	return count, nil
}

// GetSourceID returns the ID for a source by short_code, inserting it if needed.
func GetSourceID(db *sql.DB, name, shortCode, url, license, licenseURL, attribution string, required bool, notes string) (int64, error) {
	attrRequired := 0
	if required {
		attrRequired = 1
	}
	_, err := db.Exec(`
		INSERT OR IGNORE INTO sources (name, short_code, url, license, license_url, attribution_text, attribution_required, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		name, shortCode, url, license, licenseURL, attribution, attrRequired, notes)
	if err != nil {
		return 0, fmt.Errorf("inserting source %s: %w", shortCode, err)
	}

	var id int64
	err = db.QueryRow("SELECT id FROM sources WHERE short_code = ?", shortCode).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("fetching source %s: %w", shortCode, err)
	}
	return id, nil
}

// GetEditionID returns the ID for an edition by short_code, inserting it if needed.
func GetEditionID(db *sql.DB, name, shortCode string, sourceID int64, year int, editors, description string) (int64, error) {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO editions (name, short_code, source_id, year, editors, description)
		VALUES (?, ?, ?, ?, ?, ?)`,
		name, shortCode, sourceID, year, editors, description)
	if err != nil {
		return 0, fmt.Errorf("inserting edition %s: %w", shortCode, err)
	}

	var id int64
	err = db.QueryRow("SELECT id FROM editions WHERE short_code = ?", shortCode).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("fetching edition %s: %w", shortCode, err)
	}
	return id, nil
}

// LogImport records a build step in the import_log table.
func LogImport(db *sql.DB, phase, action, details string, count int, durationSecs float64) error {
	_, err := db.Exec(`
		INSERT INTO import_log (phase, action, details, count, duration_secs)
		VALUES (?, ?, ?, ?, ?)`,
		phase, action, details, count, durationSecs)
	return err
}
