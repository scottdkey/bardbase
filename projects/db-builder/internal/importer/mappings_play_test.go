// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"os"
	"testing"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

// seedPlayMappingDB creates a minimal database with two editions and shared play text lines.
func seedPlayMappingDB(t *testing.T) (string, *os.File) {
	t.Helper()
	tmp, err := os.CreateTemp("", "mappings_play_*.db")
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

	// Sources
	database.Exec(`INSERT INTO sources (id, name, short_code) VALUES (1, 'OSS', 'oss')`)
	database.Exec(`INSERT INTO sources (id, name, short_code) VALUES (2, 'SE', 'se')`)

	// Editions with the exact short_codes BuildLineMappings expects
	database.Exec(`INSERT INTO editions (id, name, short_code, source_id) VALUES (1, 'Globe', 'oss_globe', 1)`)
	database.Exec(`INSERT INTO editions (id, name, short_code, source_id) VALUES (2, 'Modern', 'se_modern', 2)`)

	// Work — a play
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Hamlet', 'play')`)

	database.Close()
	return dbPath, tmp
}

func TestPlayMapping_IdenticalLines(t *testing.T) {
	dbPath, tmp := seedPlayMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Both editions have identical lines for Act 3 Scene 1
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 3, 1, 1, 'To be, or not to be, that is the question')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 1, 3, 1, 2, 'Whether tis nobler in the mind to suffer')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (3, 1, 2, 3, 1, 1, 'To be, or not to be, that is the question')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (4, 1, 2, 3, 1, 2, 'Whether tis nobler in the mind to suffer')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 mapping pairs, got %d", count)
	}

	// Both should be "aligned" with similarity 1.0
	var alignedCount int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE match_type = 'aligned' AND similarity = 1.0`).Scan(&alignedCount)
	if alignedCount != 2 {
		t.Errorf("expected 2 aligned pairs with similarity 1.0, got %d", alignedCount)
	}

	// Verify edition IDs are set correctly
	var edA, edB int64
	database.QueryRow(`SELECT edition_a_id, edition_b_id FROM line_mappings LIMIT 1`).Scan(&edA, &edB)
	if edA != 1 || edB != 2 {
		t.Errorf("expected edition_a=1 edition_b=2, got %d %d", edA, edB)
	}
}

func TestPlayMapping_ModifiedLines(t *testing.T) {
	dbPath, tmp := seedPlayMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// OSS has old spelling, SE has modern — same position, different content
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 1, 1, 1, 'apple banana cherry dog')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 2, 1, 1, 1, 'elephant fox grape house')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var matchType string
	var similarity float64
	database.QueryRow(`SELECT match_type, similarity FROM line_mappings LIMIT 1`).Scan(&matchType, &similarity)
	if matchType != "modified" {
		t.Errorf("expected 'modified' for completely different text, got %q", matchType)
	}
	if similarity >= 0.2 {
		t.Errorf("expected similarity < 0.2, got %f", similarity)
	}
}

func TestPlayMapping_ExtraLineInOneEdition(t *testing.T) {
	dbPath, tmp := seedPlayMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// OSS has 3 lines, SE has 2 (missing the middle one)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 2, 1, 1, 'first line of the scene')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 1, 2, 1, 2, 'stage direction enter ghost')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (3, 1, 1, 2, 1, 3, 'third line of the scene')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (4, 1, 2, 2, 1, 1, 'first line of the scene')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (5, 1, 2, 2, 1, 2, 'third line of the scene')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var total int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&total)
	if total != 3 {
		t.Errorf("expected 3 pairs (2 aligned + 1 only_a), got %d", total)
	}

	var onlyA int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE match_type = 'only_a'`).Scan(&onlyA)
	if onlyA != 1 {
		t.Errorf("expected 1 only_a pair for the extra OSS line, got %d", onlyA)
	}
}

func TestPlayMapping_MultipleScenes(t *testing.T) {
	dbPath, tmp := seedPlayMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Act 1 Scene 1
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 1, 1, 1, 'Who is there')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 2, 1, 1, 1, 'Who is there')`)

	// Act 1 Scene 2
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (3, 1, 1, 1, 2, 1, 'Though yet of Hamlet')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (4, 1, 2, 1, 2, 1, 'Though yet of Hamlet')`)

	// Act 2 Scene 1
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (5, 1, 1, 2, 1, 1, 'Give him this money')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (6, 1, 2, 2, 1, 1, 'Give him this money')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var total int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&total)
	if total != 3 {
		t.Errorf("expected 3 mapping pairs (1 per scene), got %d", total)
	}

	// Verify each scene has its own mapping
	var scene1Count, scene2Count, act2Count int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE act = 1 AND scene = 1`).Scan(&scene1Count)
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE act = 1 AND scene = 2`).Scan(&scene2Count)
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE act = 2 AND scene = 1`).Scan(&act2Count)
	if scene1Count != 1 || scene2Count != 1 || act2Count != 1 {
		t.Errorf("expected 1 pair per scene, got scene1=%d scene2=%d act2=%d", scene1Count, scene2Count, act2Count)
	}
}

func TestPlayMapping_AlignOrder(t *testing.T) {
	dbPath, tmp := seedPlayMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// 3 lines in same scene, both editions
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 1, 1, 1, 'line one text here')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 1, 1, 1, 2, 'line two text here')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (3, 1, 1, 1, 1, 3, 'line three text here')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (4, 1, 2, 1, 1, 1, 'line one text here')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (5, 1, 2, 1, 1, 2, 'line two text here')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (6, 1, 2, 1, 1, 3, 'line three text here')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	// Verify align_order is sequential 1, 2, 3
	rows, _ := database.Query(`SELECT align_order FROM line_mappings ORDER BY align_order`)
	defer rows.Close()
	expected := 1
	for rows.Next() {
		var order int
		rows.Scan(&order)
		if order != expected {
			t.Errorf("expected align_order %d, got %d", expected, order)
		}
		expected++
	}
	if expected != 4 {
		t.Errorf("expected 3 rows, iterated %d", expected-1)
	}
}

func TestPlayMapping_LineIDsPreserved(t *testing.T) {
	dbPath, tmp := seedPlayMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (100, 1, 1, 1, 1, 1, 'To be or not to be')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (200, 1, 2, 1, 1, 1, 'To be or not to be')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var lineAID, lineBID int64
	database.QueryRow(`SELECT line_a_id, line_b_id FROM line_mappings LIMIT 1`).Scan(&lineAID, &lineBID)
	if lineAID != 100 {
		t.Errorf("expected line_a_id=100, got %d", lineAID)
	}
	if lineBID != 200 {
		t.Errorf("expected line_b_id=200, got %d", lineBID)
	}
}

func TestPlayMapping_SceneOnlyInOneEdition(t *testing.T) {
	dbPath, tmp := seedPlayMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Scene only in OSS, not in SE — should produce no mappings
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 5, 1, 1, 'A scene only in OSS')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 mappings for scene only in one edition, got %d", count)
	}
}
