// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"path/filepath"
	"testing"

	"github.com/scottdkey/heminge/projects/db-builder/internal/db"
)

func TestPoemAlignment_IdenticalLines(t *testing.T) {
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

	// Poem work (act=0, scene=0, work_type='poem')
	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Venus and Adonis', 'Ven.', 'poem')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Ven.'").Scan(&workID)

	// Identical lines in both editions
	lines := []string{
		"Even as the sun with purple-colour'd face",
		"Had ta'en his last leave of the weeping morn,",
		"Rose-cheek'd Adonis hied him to the chase;",
	}
	for i, content := range lines {
		database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
			VALUES (?, ?, 0, 0, ?, ?, 'verse')`, workID, edOSS, i+1, content)
		database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
			VALUES (?, ?, 0, 0, ?, ?, 'verse')`, workID, edSE, i+1, content)
	}

	err = BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings WHERE work_id = ?", workID).Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 mapping pairs, got %d", count)
	}

	// All should be "aligned" with similarity 1.0
	rows, _ := database.Query(`SELECT match_type, similarity FROM line_mappings WHERE work_id = ? ORDER BY align_order`, workID)
	defer rows.Close()
	for rows.Next() {
		var matchType string
		var similarity float64
		rows.Scan(&matchType, &similarity)
		if matchType != "aligned" {
			t.Errorf("expected 'aligned', got '%s'", matchType)
		}
		if similarity != 1.0 {
			t.Errorf("expected similarity 1.0, got %f", similarity)
		}
	}
}

func TestPoemAlignment_ModifiedLines(t *testing.T) {
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

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Venus and Adonis', 'Ven.', 'poem')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Ven.'").Scan(&workID)

	// Completely different text in each edition
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 1, 'Alpha beta gamma delta epsilon', 'verse')`, workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 1, 'Zeta eta theta iota kappa', 'verse')`, workID, edSE)

	err = BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	var matchType string
	var similarity float64
	database.QueryRow(`SELECT match_type, similarity FROM line_mappings WHERE work_id = ?`, workID).Scan(&matchType, &similarity)

	if matchType != "modified" {
		t.Errorf("expected 'modified', got '%s'", matchType)
	}
	if similarity >= 0.5 {
		t.Errorf("expected low similarity for completely different text, got %f", similarity)
	}
}

func TestPoemAlignment_ExtraLineInOneEdition(t *testing.T) {
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

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Venus and Adonis', 'Ven.', 'poem')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Ven.'").Scan(&workID)

	// OSS has 3 lines, SE has 2 (missing the middle one)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 1, 'Even as the sun with purple face', 'verse')`, workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 2, 'Had taken his last leave of the weeping morn', 'verse')`, workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 3, 'Rose cheeked Adonis hied him to the chase', 'verse')`, workID, edOSS)

	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 1, 'Even as the sun with purple face', 'verse')`, workID, edSE)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 2, 'Rose cheeked Adonis hied him to the chase', 'verse')`, workID, edSE)

	err = BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	var total int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings WHERE work_id = ?", workID).Scan(&total)
	if total != 3 {
		t.Errorf("expected 3 mapping pairs (2 aligned + 1 only_a), got %d", total)
	}

	var onlyACount int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE work_id = ? AND match_type = 'only_a'`, workID).Scan(&onlyACount)
	if onlyACount != 1 {
		t.Errorf("expected 1 only_a gap, got %d", onlyACount)
	}
}

func TestPoemAlignment_MultiplePoems(t *testing.T) {
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

	// Two separate poems
	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Venus and Adonis', 'Ven.', 'poem')`)
	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('The Rape of Lucrece', 'Lucr.', 'poem')`)
	var venID, lucrID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Ven.'").Scan(&venID)
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Lucr.'").Scan(&lucrID)

	// 2 lines each poem, both editions identical
	for _, wID := range []int64{venID, lucrID} {
		for _, edID := range []int64{edOSS, edSE} {
			database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
				VALUES (?, ?, 0, 0, 1, 'First line of poem', 'verse')`, wID, edID)
			database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
				VALUES (?, ?, 0, 0, 2, 'Second line of poem', 'verse')`, wID, edID)
		}
	}

	err = BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	// 2 lines per poem × 2 poems = 4 total mappings
	var total int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&total)
	if total != 4 {
		t.Errorf("expected 4 total mappings (2 per poem), got %d", total)
	}

	// Each poem should have exactly 2
	var venCount, lucrCount int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings WHERE work_id = ?", venID).Scan(&venCount)
	database.QueryRow("SELECT COUNT(*) FROM line_mappings WHERE work_id = ?", lucrID).Scan(&lucrCount)
	if venCount != 2 {
		t.Errorf("Venus: expected 2 mappings, got %d", venCount)
	}
	if lucrCount != 2 {
		t.Errorf("Lucrece: expected 2 mappings, got %d", lucrCount)
	}
}

func TestPoemAlignment_PoemOnlyInOneEdition(t *testing.T) {
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
	db.GetEditionID(database, "SE", "se_modern", srcID, 2020, "", "")

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Venus and Adonis', 'Ven.', 'poem')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Ven.'").Scan(&workID)

	// Only in OSS, not in SE
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 1, 'Even as the sun', 'verse')`, workID, edOSS)

	err = BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	// No shared poem → 0 mappings
	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings WHERE work_id = ?", workID).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 mappings for poem only in one edition, got %d", count)
	}
}

func TestPoemAlignment_LineIDsPreserved(t *testing.T) {
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

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Venus and Adonis', 'Ven.', 'poem')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Ven.'").Scan(&workID)

	// Insert one line each
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 1, 'Even as the sun with purple face', 'verse')`, workID, edOSS)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
		VALUES (?, ?, 0, 0, 1, 'Even as the sun with purple face', 'verse')`, workID, edSE)

	// Get the actual line IDs
	var ossLineID, seLineID int64
	database.QueryRow("SELECT id FROM text_lines WHERE edition_id = ?", edOSS).Scan(&ossLineID)
	database.QueryRow("SELECT id FROM text_lines WHERE edition_id = ?", edSE).Scan(&seLineID)

	err = BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	var lineAID, lineBID int64
	database.QueryRow(`SELECT line_a_id, line_b_id FROM line_mappings WHERE work_id = ?`, workID).Scan(&lineAID, &lineBID)

	if lineAID != ossLineID {
		t.Errorf("expected line_a_id = %d (OSS), got %d", ossLineID, lineAID)
	}
	if lineBID != seLineID {
		t.Errorf("expected line_b_id = %d (SE), got %d", seLineID, lineBID)
	}
}

func TestPoemAlignment_AlignOrder(t *testing.T) {
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

	database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Venus and Adonis', 'Ven.', 'poem')`)
	var workID int64
	database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Ven.'").Scan(&workID)

	// 4 identical lines in both editions
	lines := []string{"Line one", "Line two", "Line three", "Line four"}
	for i, content := range lines {
		database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
			VALUES (?, ?, 0, 0, ?, ?, 'verse')`, workID, edOSS, i+1, content)
		database.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, content_type)
			VALUES (?, ?, 0, 0, ?, ?, 'verse')`, workID, edSE, i+1, content)
	}

	err = BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	// Verify align_order is sequential 1,2,3,4
	rows, _ := database.Query(`SELECT align_order FROM line_mappings WHERE work_id = ? ORDER BY align_order`, workID)
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
	if expected != 5 {
		t.Errorf("expected 4 rows, got %d", expected-1)
	}
}
