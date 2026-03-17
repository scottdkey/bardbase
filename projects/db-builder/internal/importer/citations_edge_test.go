// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"os"
	"testing"

	"github.com/scottdkey/heminge/projects/db-builder/internal/db"
)

func openEdgeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpFile := t.TempDir() + "/edge_test.db"
	database, err := db.Open(tmpFile)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
		os.Remove(tmpFile)
	})
	return database
}

// seedEdgeDB inserts a standard set of sources + editions for reuse.
func seedEdgeDB(t *testing.T, database *sql.DB) (sourceID, ossEdID, seEdID int64) {
	t.Helper()
	res, err := database.Exec(`INSERT INTO sources (name, short_code, license) VALUES ('Open Source Shakespeare', 'oss', 'PD')`)
	if err != nil {
		t.Fatalf("insert source: %v", err)
	}
	sourceID, _ = res.LastInsertId()
	res, _ = database.Exec(`INSERT INTO editions (source_id, short_code, name) VALUES (?, 'oss_globe', 'OSS Globe')`, sourceID)
	ossEdID, _ = res.LastInsertId()
	res, _ = database.Exec(`INSERT INTO editions (source_id, short_code, name) VALUES (?, 'se_modern', 'SE Modern')`, sourceID)
	seEdID, _ = res.LastInsertId()
	return
}

func TestResolveCitations_EmptyEditions(t *testing.T) {
	database := openEdgeTestDB(t)
	// No editions at all — should return nil with "no editions" path
	err := ResolveCitations(database)
	if err != nil {
		t.Fatalf("expected no error with empty editions, got: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 matches, got %d", count)
	}
}

func TestResolveCitations_EmptyCitations(t *testing.T) {
	database := openEdgeTestDB(t)
	seedEdgeDB(t, database)

	// Editions exist, but no citations
	err := ResolveCitations(database)
	if err != nil {
		t.Fatalf("expected no error with empty citations, got: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 matches, got %d", count)
	}
}

func TestResolveCitations_NoTextLinesForWork(t *testing.T) {
	database := openEdgeTestDB(t)
	_, _, _ = seedEdgeDB(t, database)

	// Create a work and a citation, but NO text_lines
	res, err := database.Exec(`INSERT INTO works (title, work_type) VALUES ('Ghost Play', 'play')`)
	if err != nil {
		t.Fatalf("insert work: %v", err)
	}
	workID, _ := res.LastInsertId()

	res, err = database.Exec(`INSERT INTO lexicon_entries (key, letter, raw_xml) VALUES ('test','T','<xml/>')`)
	if err != nil {
		t.Fatalf("insert entry: %v", err)
	}
	entryID, _ := res.LastInsertId()

	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, act, scene, line, quote_text, raw_bibl)
		VALUES (?, ?, 1, 1, 10, 'some quote', 'Gh. I, 1, 10')`, entryID, workID)

	err = ResolveCitations(database)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 matches (no text_lines), got %d", count)
	}
}

func TestResolveCitations_DuplicateCitations(t *testing.T) {
	database := openEdgeTestDB(t)
	_, ossEdID, _ := seedEdgeDB(t, database)

	res, _ := database.Exec(`INSERT INTO works (title, work_type) VALUES ('Hamlet', 'play')`)
	workID, _ := res.LastInsertId()

	// One text line
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content)
		VALUES (?, ?, 3, 1, 56, 'To be, or not to be, that is the question')`, workID, ossEdID)

	res, _ = database.Exec(`INSERT INTO lexicon_entries (key, letter, raw_xml) VALUES ('Be','B','<xml/>')`)
	entryID, _ := res.LastInsertId()

	// Two identical citations pointing to the same location
	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, act, scene, line, quote_text, raw_bibl)
		VALUES (?, ?, 3, 1, 56, 'to be', 'Ham. III, 1, 56')`, entryID, workID)
	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, act, scene, line, quote_text, raw_bibl)
		VALUES (?, ?, 3, 1, 56, 'to be', 'Ham. III, 1, 56')`, entryID, workID)

	err := ResolveCitations(database)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both citations should match — each gets its own row in citation_matches
	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 matches (one per citation), got %d", count)
	}
}

func TestResolveCitations_BelowFuzzyThreshold(t *testing.T) {
	database := openEdgeTestDB(t)
	_, ossEdID, _ := seedEdgeDB(t, database)

	res, _ := database.Exec(`INSERT INTO works (title, work_type) VALUES ('Hamlet', 'play')`)
	workID, _ := res.LastInsertId()

	// Line about something completely unrelated
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content)
		VALUES (?, ?, 1, 1, 1, 'The trumpets sound and cannons fire loudly')`, workID, ossEdID)

	res, _ = database.Exec(`INSERT INTO lexicon_entries (key, letter, raw_xml) VALUES ('xyz','X','<xml/>')`)
	entryID, _ := res.LastInsertId()

	// Citation with quote that has zero word overlap with the line
	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, act, scene, quote_text, raw_bibl)
		VALUES (?, ?, 1, 1, 'purple mountains majesty waves grain', 'Ham. I, 1')`, entryID, workID)

	err := ResolveCitations(database)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 matches (below fuzzy threshold), got %d", count)
	}
}

func TestResolveCitations_NoQuoteNoLine(t *testing.T) {
	database := openEdgeTestDB(t)
	_, ossEdID, _ := seedEdgeDB(t, database)

	res, _ := database.Exec(`INSERT INTO works (title, work_type) VALUES ('Hamlet', 'play')`)
	workID, _ := res.LastInsertId()

	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content)
		VALUES (?, ?, 3, 1, 1, 'To be, or not to be')`, workID, ossEdID)

	res, _ = database.Exec(`INSERT INTO lexicon_entries (key, letter, raw_xml) VALUES ('test','T','<xml/>')`)
	entryID, _ := res.LastInsertId()

	// Citation has act and scene but NO line number and NO quote text
	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, act, scene, raw_bibl)
		VALUES (?, ?, 3, 1, 'Ham. III, 1')`, entryID, workID)

	err := ResolveCitations(database)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 matches (no quote, no line), got %d", count)
	}
}

func TestResolveCitations_MultipleCitationsSameLine(t *testing.T) {
	database := openEdgeTestDB(t)
	_, ossEdID, _ := seedEdgeDB(t, database)

	res, _ := database.Exec(`INSERT INTO works (title, work_type) VALUES ('Hamlet', 'play')`)
	workID, _ := res.LastInsertId()

	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content)
		VALUES (?, ?, 3, 1, 56, 'To be, or not to be, that is the question')`, workID, ossEdID)

	// Two different entries both cite the same line
	res, _ = database.Exec(`INSERT INTO lexicon_entries (key, letter, raw_xml) VALUES ('Be','B','<xml/>')`)
	entry1, _ := res.LastInsertId()
	res, _ = database.Exec(`INSERT INTO lexicon_entries (key, letter, raw_xml) VALUES ('Question','Q','<xml/>')`)
	entry2, _ := res.LastInsertId()

	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, act, scene, line, quote_text, raw_bibl)
		VALUES (?, ?, 3, 1, 56, 'to be', 'Ham. III, 1, 56')`, entry1, workID)
	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, act, scene, line, quote_text, raw_bibl)
		VALUES (?, ?, 3, 1, 56, 'the question', 'Ham. III, 1, 56')`, entry2, workID)

	err := ResolveCitations(database)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both should match to the same text_line
	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 matches (two entries, same line), got %d", count)
	}

	// Verify they point to the same text_line_id
	rows, _ := database.Query("SELECT DISTINCT text_line_id FROM citation_matches")
	var distinctLines int
	for rows.Next() {
		distinctLines++
	}
	rows.Close()
	if distinctLines != 1 {
		t.Errorf("expected both to match same text_line, got %d distinct lines", distinctLines)
	}
}

func TestResolveCitations_Idempotent(t *testing.T) {
	database := openEdgeTestDB(t)
	_, ossEdID, _ := seedEdgeDB(t, database)

	res, _ := database.Exec(`INSERT INTO works (title, work_type) VALUES ('Hamlet', 'play')`)
	workID, _ := res.LastInsertId()

	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content)
		VALUES (?, ?, 3, 1, 56, 'To be, or not to be, that is the question')`, workID, ossEdID)

	res, _ = database.Exec(`INSERT INTO lexicon_entries (key, letter, raw_xml) VALUES ('Be','B','<xml/>')`)
	entryID, _ := res.LastInsertId()
	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, act, scene, line, quote_text, raw_bibl)
		VALUES (?, ?, 3, 1, 56, 'to be', 'Ham. III, 1, 56')`, entryID, workID)

	// Run twice
	ResolveCitations(database)
	ResolveCitations(database)

	// Should have exactly 1 match, not 2 (DELETE clears before rebuild)
	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 match after idempotent run, got %d", count)
	}
}
