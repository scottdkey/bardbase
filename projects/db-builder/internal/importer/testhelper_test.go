// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

// testDB is a test helper factory that provides a fresh SQLite database
// with schema, source, and editions pre-configured. Eliminates the repeated
// setup boilerplate across 100+ test functions.
//
// Usage:
//
//	td := newTestDB(t)
//	workID := td.insertWork(t, "The Tempest", "Tp.", "comedy")
//	td.DB.Exec(`INSERT INTO text_lines ...`, workID, td.EdOSSID, ...)
type testDB struct {
	DB       *sql.DB
	SourceID int64
	EdOSSID  int64
	EdSEID   int64
}

// newTestDB creates a fresh test database with schema, a test source,
// and two editions (OSS Globe and SE Modern). The database is automatically
// closed when the test completes via t.Cleanup.
func newTestDB(t *testing.T) *testDB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("newTestDB: Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("newTestDB: CreateSchema: %v", err)
	}

	srcID, err := db.GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	if err != nil {
		t.Fatalf("newTestDB: GetSourceID: %v", err)
	}
	edOSS, err := db.GetEditionID(database, "OSS", "oss_globe", srcID, 2000, "", "")
	if err != nil {
		t.Fatalf("newTestDB: GetEditionID(OSS): %v", err)
	}
	edSE, err := db.GetEditionID(database, "SE", "se_modern", srcID, 2020, "", "")
	if err != nil {
		t.Fatalf("newTestDB: GetEditionID(SE): %v", err)
	}

	return &testDB{DB: database, SourceID: srcID, EdOSSID: edOSS, EdSEID: edSE}
}

// insertWork inserts a work and returns its ID.
func (td *testDB) insertWork(t *testing.T, title, schmidtAbbrev, workType string) int64 {
	t.Helper()
	_, err := td.DB.Exec(
		`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES (?, ?, ?)`,
		title, schmidtAbbrev, workType)
	if err != nil {
		t.Fatalf("insertWork: %v", err)
	}
	var id int64
	err = td.DB.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = ?", schmidtAbbrev).Scan(&id)
	if err != nil {
		t.Fatalf("insertWork: fetch ID: %v", err)
	}
	return id
}

// insertTextLine inserts a text line and returns its ID.
func (td *testDB) insertTextLine(t *testing.T, workID, editionID int64, act, scene, lineNum int, charName, content string) int64 {
	t.Helper()
	res, err := td.DB.Exec(
		`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, char_name, content, content_type)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'speech')`,
		workID, editionID, act, scene, lineNum, charName, content)
	if err != nil {
		t.Fatalf("insertTextLine: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// insertLexiconEntry inserts a lexicon entry and returns its ID.
func (td *testDB) insertLexiconEntry(t *testing.T, key, letter string) int64 {
	t.Helper()
	res, err := td.DB.Exec(
		`INSERT INTO lexicon_entries (key, letter) VALUES (?, ?)`, key, letter)
	if err != nil {
		t.Fatalf("insertLexiconEntry: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// insertSense inserts a lexicon sense and returns its ID.
func (td *testDB) insertSense(t *testing.T, entryID int64, senseNum int, definition string) int64 {
	t.Helper()
	res, err := td.DB.Exec(
		`INSERT INTO lexicon_senses (entry_id, sense_number, definition_text) VALUES (?, ?, ?)`,
		entryID, senseNum, definition)
	if err != nil {
		t.Fatalf("insertSense: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// insertCitation inserts a lexicon citation and returns its ID.
func (td *testDB) insertCitation(t *testing.T, entryID int64, senseID *int64, workID int64, act, scene, line *int, quoteText string) int64 {
	t.Helper()
	res, err := td.DB.Exec(
		`INSERT INTO lexicon_citations (entry_id, sense_id, work_id, act, scene, line, quote_text)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entryID, senseID, workID, act, scene, line, quoteText)
	if err != nil {
		t.Fatalf("insertCitation: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// newAttributionTestDB creates a test database with the 3 sources that
// PopulateAttributions expects (oss_moby, perseus_schmidt, standard_ebooks).
func newAttributionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("newAttributionTestDB: Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("newAttributionTestDB: CreateSchema: %v", err)
	}

	sources := []struct {
		name, code, url, license string
	}{
		{"Open Source Shakespeare", "oss_moby", "https://www.opensourceshakespeare.org/", "Public Domain"},
		{"Perseus Schmidt Lexicon", "perseus_schmidt", "http://www.perseus.tufts.edu", "CC BY-SA 3.0"},
		{"Standard Ebooks", "standard_ebooks", "https://standardebooks.org", "CC0 1.0"},
	}
	for _, s := range sources {
		if _, err := db.GetSourceID(database, s.name, s.code, s.url, s.license, "", "", false, ""); err != nil {
			t.Fatalf("newAttributionTestDB: inserting source %s: %v", s.code, err)
		}
	}

	return database
}

// intPtr returns a pointer to an int value. Useful for nullable test parameters.
func intPtr(v int) *int { return &v }

// int64Ptr returns a pointer to an int64 value.
func int64Ptr(v int64) *int64 { return &v }
