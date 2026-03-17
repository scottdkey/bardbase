// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"fmt"
	"os"
	"testing"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
)

// seedEdgeMappingDB creates a minimal database — sources and editions only (no works or lines).
func seedEdgeMappingDB(t *testing.T) (string, *os.File) {
	t.Helper()
	tmp, err := os.CreateTemp("", "mappings_edge_*.db")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	dbPath := tmp.Name()

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	database.Exec(`INSERT INTO sources (id, name, short_code) VALUES (1, 'OSS', 'oss')`)
	database.Exec(`INSERT INTO sources (id, name, short_code) VALUES (2, 'SE', 'se')`)
	database.Exec(`INSERT INTO editions (id, name, short_code, source_id) VALUES (1, 'Globe', 'oss_globe', 1)`)
	database.Exec(`INSERT INTO editions (id, name, short_code, source_id) VALUES (2, 'Modern', 'se_modern', 2)`)

	database.Close()
	return dbPath, tmp
}

// seedEmptyDB creates a DB with schema only — no sources or editions.
func seedEmptyDB(t *testing.T) (string, *os.File) {
	t.Helper()
	tmp, err := os.CreateTemp("", "mappings_empty_*.db")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	dbPath := tmp.Name()

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}
	database.Close()
	return dbPath, tmp
}

func TestEdgeMapping_NoEditions(t *testing.T) {
	dbPath, tmp := seedEmptyDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings should not error with no editions: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 mappings with no editions, got %d", count)
	}
}

func TestEdgeMapping_OnlyOneEdition(t *testing.T) {
	tmp, err := os.CreateTemp("", "mappings_one_ed_*.db")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	database, err := db.Open(tmp.Name())
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	defer database.Close()
	db.CreateSchema(database)

	// Only one edition — OSS but not SE
	database.Exec(`INSERT INTO sources (id, name, short_code) VALUES (1, 'OSS', 'oss')`)
	database.Exec(`INSERT INTO editions (id, name, short_code, source_id) VALUES (1, 'Globe', 'oss_globe', 1)`)
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Hamlet', 'play')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 3, 1, 1, 'To be or not to be')`)

	err = BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings should not error with one edition: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 mappings with one edition, got %d", count)
	}
}

func TestEdgeMapping_NoSharedWorks(t *testing.T) {
	dbPath, tmp := seedEdgeMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Two different works — one per edition, no overlap.
	// With outer-join semantics each work still gets a mapping entry
	// (only_a for Hamlet, only_b for Othello) so neither is silently dropped.
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Hamlet', 'play')`)
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (2, 'Othello', 'play')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 1, 1, 1, 'Who is there?')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 2, 2, 1, 1, 1, 'Never tell me')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	// Expect 2 entries: one only_a (Hamlet in edition 1 only) and
	// one only_b (Othello in edition 2 only).
	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 only_a/only_b mappings for non-overlapping works, got %d", count)
	}

	var onlyA, onlyB int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings WHERE match_type = 'only_a'").Scan(&onlyA)
	database.QueryRow("SELECT COUNT(*) FROM line_mappings WHERE match_type = 'only_b'").Scan(&onlyB)
	if onlyA != 1 || onlyB != 1 {
		t.Errorf("expected 1 only_a and 1 only_b, got only_a=%d only_b=%d", onlyA, onlyB)
	}
}

func TestEdgeMapping_EditionsExistButNoTextLines(t *testing.T) {
	dbPath, tmp := seedEdgeMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Editions exist, a work exists, but no text_lines
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Hamlet', 'play')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 mappings with no text_lines, got %d", count)
	}
}

func TestEdgeMapping_Idempotent(t *testing.T) {
	dbPath, tmp := seedEdgeMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Hamlet', 'play')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 1, 1, 1, 'Who is there?')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 2, 1, 1, 1, 'Who is there?')`)

	// Run twice
	BuildLineMappings(database)
	BuildLineMappings(database)

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 mapping after idempotent rebuild, got %d", count)
	}
}

func TestEdgeMapping_MixedWorkTypes(t *testing.T) {
	dbPath, tmp := seedEdgeMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Play, sonnet, and poem — all in one DB
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Hamlet', 'play')`)
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (2, 'Sonnets', 'sonnet_sequence')`)
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (3, 'Venus and Adonis', 'poem')`)

	// Play — Act 1 Scene 1
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 1, 1, 1, 'Who is there?')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 2, 1, 1, 1, 'Who is there?')`)

	// Sonnet 18
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (3, 2, 1, 0, 18, 1, 'Shall I compare thee to a summers day?')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (4, 2, 2, 0, 18, 1, 'Shall I compare thee to a summers day?')`)

	// Poem
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (5, 3, 1, 0, 0, 1, 'Even as the sun with purple-colourd face')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (6, 3, 2, 0, 0, 1, 'Even as the sun with purple-colourd face')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	// Should get 3 mappings — one from each work type
	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 mappings (play + sonnet + poem), got %d", count)
	}

	// Verify each type is represented
	var playCount, sonnetCount, poemCount int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE act = 1 AND scene = 1`).Scan(&playCount)
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings lm JOIN works w ON lm.work_id = w.id WHERE w.work_type = 'sonnet_sequence'`).Scan(&sonnetCount)
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings lm JOIN works w ON lm.work_id = w.id WHERE w.work_type = 'poem'`).Scan(&poemCount)

	if playCount != 1 {
		t.Errorf("expected 1 play mapping, got %d", playCount)
	}
	if sonnetCount != 1 {
		t.Errorf("expected 1 sonnet mapping, got %d", sonnetCount)
	}
	if poemCount != 1 {
		t.Errorf("expected 1 poem mapping, got %d", poemCount)
	}
}

func TestEdgeMapping_LargeScene(t *testing.T) {
	dbPath, tmp := seedEdgeMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Hamlet', 'play')`)

	// 100 lines per edition — some identical, some completely different
	for i := 1; i <= 100; i++ {
		if i%10 == 0 {
			// Every 10th line is completely different text between editions (Jaccard < 0.2)
			ossContent := fmt.Sprintf("ancient marble fortress crumbling slowly line%d", i)
			seContent := fmt.Sprintf("bright golden butterfly dancing wildly verse%d", i)
			database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 3, 1, ?, ?)`, i, ossContent)
			database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content) VALUES (1, 2, 3, 1, ?, ?)`, i, seContent)
		} else {
			// Identical text in both editions
			content := fmt.Sprintf("To be or not to be that is the question line %d", i)
			database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 3, 1, ?, ?)`, i, content)
			database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content) VALUES (1, 2, 3, 1, ?, ?)`, i, content)
		}
	}

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed on large scene: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 100 {
		t.Errorf("expected 100 mapping pairs for 100 lines, got %d", count)
	}

	// The 90 identical lines should be aligned
	var alignedCount int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE match_type = 'aligned' AND similarity = 1.0`).Scan(&alignedCount)
	if alignedCount != 90 {
		t.Errorf("expected 90 identical aligned pairs, got %d", alignedCount)
	}

	// The 10 different lines should be modified (similarity < 0.2)
	var modifiedCount int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE match_type = 'modified'`).Scan(&modifiedCount)
	if modifiedCount != 10 {
		t.Errorf("expected 10 modified pairs, got %d", modifiedCount)
	}
}

func TestEdgeMapping_WorkInOneEditionOnly(t *testing.T) {
	dbPath, tmp := seedEdgeMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Hamlet', 'play')`)

	// Lines only in edition 1 (OSS), nothing in edition 2 (SE)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 1, 1, 1, 'Who is there?')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 1, 1, 1, 2, 'Nay answer me')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 mappings when work only in one edition, got %d", count)
	}
}
