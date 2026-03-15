// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"path/filepath"
	"testing"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

func TestPlayCitation_ExactQuoteMatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	db.CreateSchema(database)

	// Source + editions
	srcID, _ := db.GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	edOSS, _ := db.GetEditionID(database, "OSS", "oss_globe", srcID, 2000, "", "")
	edSE, _ := db.GetEditionID(database, "SE", "se_modern", srcID, 2020, "", "")

	// Work
	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('The Tempest', 'Tp.', 'comedy')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Tp.'").Scan(&workID)

	// Text lines: Act 1, Scene 1, lines 1-5 in both editions
	lines := []struct {
		content    string
		lineNum    int
		act, scene int
	}{
		{"Boatswain!", 1, 1, 1},
		{"Here, master: what cheer?", 2, 1, 1},
		{"Good, speak to the mariners.", 3, 1, 1},
		{"Fall to't, yarely, or we run ourselves aground.", 4, 1, 1},
		{"Heigh, my hearts! cheerly, cheerly, my hearts!", 5, 1, 1},
	}

	for _, ed := range []int64{edOSS, edSE} {
		for _, l := range lines {
			database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
				VALUES (?, ?, ?, ?, ?, ?, 'speech')`,
				workID, ed, l.act, l.scene, l.lineNum, l.content)
		}
	}

	// Lexicon entry + sense + citation with quote text that matches line 4
	database.Exec(`INSERT INTO lexicon_entries (key, letter, full_text) VALUES ('Aground', 'A', 'test')`)
	var entryID int64
	database.QueryRow("SELECT id FROM lexicon_entries WHERE key = 'Aground'").Scan(&entryID)

	database.Exec(`INSERT INTO lexicon_senses (entry_id, sense_number, definition_text) VALUES (?, 1, 'stranded')`, entryID)
	var senseID int64
	database.QueryRow("SELECT id FROM lexicon_senses WHERE entry_id = ?", entryID).Scan(&senseID)

	database.Exec(`INSERT INTO lexicon_citations (entry_id, sense_id, work_id, work_abbrev, act, scene, line, quote_text)
		VALUES (?, ?, ?, 'Tp.', 1, 1, 4, 'we run ourselves aground')`,
		entryID, senseID, workID)

	// Run resolver
	err = ResolveCitations(database)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	// Should have 2 matches (one per edition), both exact_quote with confidence 1.0
	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 matches (one per edition), got %d", count)
	}

	rows, _ := database.Query(`SELECT edition_id, match_type, confidence, matched_text FROM citation_matches`)
	defer rows.Close()
	for rows.Next() {
		var edID int64
		var matchType, matchedText string
		var confidence float64
		rows.Scan(&edID, &matchType, &confidence, &matchedText)

		if matchType != "exact_quote" {
			t.Errorf("edition %d: expected match_type 'exact_quote', got '%s'", edID, matchType)
		}
		if confidence != 1.0 {
			t.Errorf("edition %d: expected confidence 1.0, got %f", edID, confidence)
		}
		if matchedText != "Fall to't, yarely, or we run ourselves aground." {
			t.Errorf("edition %d: unexpected matched_text: %s", edID, matchedText)
		}
	}
}

func TestPlayCitation_LineNumberMatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	db.CreateSchema(database)

	srcID, _ := db.GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	edOSS, _ := db.GetEditionID(database, "OSS", "oss_globe", srcID, 2000, "", "")

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('The Tempest', 'Tp.', 'comedy')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Tp.'").Scan(&workID)

	// Text lines with no matching quote — forces line_number strategy
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 2, 1, 10, 'Abhorred slave, which any print of goodness will not take.', 'speech')`,
		workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 2, 1, 11, 'Being capable of all ill!', 'speech')`,
		workID, edOSS)

	// Citation with line number but NO quote text
	database.Exec(`INSERT INTO lexicon_entries (key, letter, full_text) VALUES ('Abhorred', 'A', 'test')`)
	var entryID int64
	database.QueryRow("SELECT id FROM lexicon_entries WHERE key = 'Abhorred'").Scan(&entryID)

	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, work_abbrev, act, scene, line)
		VALUES (?, ?, 'Tp.', 2, 1, 10)`, entryID, workID)

	err = ResolveCitations(database)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var matchType string
	var confidence float64
	var matchedText string
	err = database.QueryRow(`SELECT match_type, confidence, matched_text FROM citation_matches`).Scan(&matchType, &confidence, &matchedText)
	if err != nil {
		t.Fatalf("no match found: %v", err)
	}

	if matchType != "line_number" {
		t.Errorf("expected 'line_number', got '%s'", matchType)
	}
	if confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", confidence)
	}
	if matchedText != "Abhorred slave, which any print of goodness will not take." {
		t.Errorf("unexpected matched_text: %s", matchedText)
	}
}

func TestPlayCitation_FuzzyMatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	db.CreateSchema(database)

	srcID, _ := db.GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	edOSS, _ := db.GetEditionID(database, "OSS", "oss_globe", srcID, 2000, "", "")

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('The Tempest', 'Tp.', 'comedy')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Tp.'").Scan(&workID)

	// Lines in act 3 scene 2 — none with matching line number
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 3, 2, 40, 'Monster, I do smell all horse-piss', 'speech')`,
		workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 3, 2, 41, 'at which my nose is in great indignation.', 'speech')`,
		workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 3, 2, 42, 'So is mine. Do you hear, monster?', 'speech')`,
		workID, edOSS)

	// Citation references line 99 (doesn't exist) but has quote text matching line 40
	database.Exec(`INSERT INTO lexicon_entries (key, letter, full_text) VALUES ('Horse-piss', 'H', 'test')`)
	var entryID int64
	database.QueryRow("SELECT id FROM lexicon_entries WHERE key = 'Horse-piss'").Scan(&entryID)

	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, work_abbrev, act, scene, line, quote_text)
		VALUES (?, ?, 'Tp.', 3, 2, 99, 'I do smell all horse-piss')`, entryID, workID)

	err = ResolveCitations(database)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var matchType string
	var confidence float64
	err = database.QueryRow(`SELECT match_type, confidence FROM citation_matches`).Scan(&matchType, &confidence)
	if err != nil {
		t.Fatalf("no match found: %v", err)
	}

	// Line 99 doesn't exist, nearby ±3 doesn't reach 40, but quote text is a substring of line 40
	// So this should be exact_quote (substring match) with confidence 1.0
	if matchType != "exact_quote" {
		t.Errorf("expected 'exact_quote' (substring of line content), got '%s'", matchType)
	}
}

func TestPlayCitation_MultipleEditionsGetMatches(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	db.CreateSchema(database)

	srcID, _ := db.GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	edOSS, _ := db.GetEditionID(database, "OSS", "oss_globe", srcID, 2000, "", "")
	edSE, _ := db.GetEditionID(database, "SE", "se_modern", srcID, 2020, "", "")

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Hamlet', 'Hml.', 'tragedy')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Hml.'").Scan(&workID)

	// Same line in both editions (slightly different text)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 3, 1, 56, 'To be, or not to be, that is the question:', 'speech')`,
		workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 3, 1, 56, 'To be, or not to be: that is the question:', 'speech')`,
		workID, edSE)

	database.Exec(`INSERT INTO lexicon_entries (key, letter, full_text) VALUES ('Question', 'Q', 'test')`)
	var entryID int64
	database.QueryRow("SELECT id FROM lexicon_entries WHERE key = 'Question'").Scan(&entryID)

	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, work_abbrev, act, scene, line, quote_text)
		VALUES (?, ?, 'Hml.', 3, 1, 56, 'that is the question')`, entryID, workID)

	err = ResolveCitations(database)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	// Should get one match per edition = 2 total
	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 matches (one per edition), got %d", count)
	}

	// Both should match edition-specific IDs
	var editions []int64
	rows, _ := database.Query("SELECT DISTINCT edition_id FROM citation_matches ORDER BY edition_id")
	defer rows.Close()
	for rows.Next() {
		var eid int64
		rows.Scan(&eid)
		editions = append(editions, eid)
	}

	if len(editions) != 2 {
		t.Errorf("expected matches in 2 editions, got %d", len(editions))
	}
	if len(editions) == 2 && (editions[0] != edOSS || editions[1] != edSE) {
		t.Errorf("expected editions [%d, %d], got %v", edOSS, edSE, editions)
	}
}

func TestPlayCitation_NoMatchReturnsNothing(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	db.CreateSchema(database)

	srcID, _ := db.GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	db.GetEditionID(database, "OSS", "oss_globe", srcID, 2000, "", "")

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('The Tempest', 'Tp.', 'comedy')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Tp.'").Scan(&workID)

	// Text in act 1 scene 1
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 1, 1, 1, 'Boatswain!', 'speech')`, workID, int64(1))

	// Citation references act 5 scene 3 (doesn't exist in our data)
	database.Exec(`INSERT INTO lexicon_entries (key, letter, full_text) VALUES ('Nonexistent', 'N', 'test')`)
	var entryID int64
	database.QueryRow("SELECT id FROM lexicon_entries WHERE key = 'Nonexistent'").Scan(&entryID)

	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, work_abbrev, act, scene, line, quote_text)
		VALUES (?, ?, 'Tp.', 5, 3, 10, 'totally fake quote')`, entryID, workID)

	err = ResolveCitations(database)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 matches for nonexistent scene, got %d", count)
	}
}

func TestPlayCitation_ActOnlyNoScene(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	db.CreateSchema(database)

	srcID, _ := db.GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	edOSS, _ := db.GetEditionID(database, "OSS", "oss_globe", srcID, 2000, "", "")

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('The Tempest', 'Tp.', 'comedy')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Tp.'").Scan(&workID)

	// Lines in act 4 scene 1
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 4, 1, 5, 'Our revels now are ended.', 'speech')`, workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 4, 1, 6, 'These our actors, as I foretold you, were all spirits.', 'speech')`, workID, edOSS)

	// Citation with act=4 but scene=NULL — should search whole act
	database.Exec(`INSERT INTO lexicon_entries (key, letter, full_text) VALUES ('Revels', 'R', 'test')`)
	var entryID int64
	database.QueryRow("SELECT id FROM lexicon_entries WHERE key = 'Revels'").Scan(&entryID)

	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, work_abbrev, act, scene, line, quote_text)
		VALUES (?, ?, 'Tp.', 4, NULL, 5, 'Our revels now are ended')`, entryID, workID)

	err = ResolveCitations(database)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var matchType string
	var confidence float64
	err = database.QueryRow("SELECT match_type, confidence FROM citation_matches").Scan(&matchType, &confidence)
	if err != nil {
		t.Fatalf("no match: %v", err)
	}

	if matchType != "exact_quote" {
		t.Errorf("expected 'exact_quote', got '%s'", matchType)
	}
	if confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", confidence)
	}
}

func TestPlayCitation_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	db.CreateSchema(database)

	srcID, _ := db.GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	db.GetEditionID(database, "OSS", "oss_globe", srcID, 2000, "", "")

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('The Tempest', 'Tp.', 'comedy')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Tp.'").Scan(&workID)

	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 1, 1, 1, 'Boatswain!', 'speech')`, workID, int64(1))

	database.Exec(`INSERT INTO lexicon_entries (key, letter, full_text) VALUES ('Boatswain', 'B', 'test')`)
	var entryID int64
	database.QueryRow("SELECT id FROM lexicon_entries WHERE key = 'Boatswain'").Scan(&entryID)

	database.Exec(`INSERT INTO lexicon_citations (entry_id, work_id, work_abbrev, act, scene, line, quote_text)
		VALUES (?, ?, 'Tp.', 1, 1, 1, 'Boatswain')`, entryID, workID)

	// Run twice
	ResolveCitations(database)
	ResolveCitations(database)

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 match after two runs (idempotent), got %d", count)
	}
}
