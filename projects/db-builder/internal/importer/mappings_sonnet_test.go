// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"os"
	"testing"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

// seedSonnetMappingDB creates a minimal database with two editions and a sonnet_sequence work.
func seedSonnetMappingDB(t *testing.T) (string, *os.File) {
	t.Helper()
	tmp, err := os.CreateTemp("", "mappings_sonnet_*.db")
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

	// Work — a sonnet sequence
	database.Exec(`INSERT INTO works (id, title, work_type) VALUES (1, 'Sonnets', 'sonnet_sequence')`)

	database.Close()
	return dbPath, tmp
}

func TestSonnetMapping_IdenticalLines(t *testing.T) {
	dbPath, tmp := seedSonnetMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Sonnet 18, both editions identical (act=0, scene=18)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 0, 18, 1, 'Shall I compare thee to a summers day')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 1, 0, 18, 2, 'Thou art more lovely and more temperate')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (3, 1, 2, 0, 18, 1, 'Shall I compare thee to a summers day')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (4, 1, 2, 0, 18, 2, 'Thou art more lovely and more temperate')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 mapping pairs, got %d", count)
	}

	var alignedCount int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE match_type = 'aligned' AND similarity = 1.0`).Scan(&alignedCount)
	if alignedCount != 2 {
		t.Errorf("expected 2 aligned pairs with similarity 1.0, got %d", alignedCount)
	}

	// Verify act=0 and scene=18 on the mapping
	var act, scene int
	database.QueryRow(`SELECT act, scene FROM line_mappings LIMIT 1`).Scan(&act, &scene)
	if act != 0 || scene != 18 {
		t.Errorf("expected act=0 scene=18, got act=%d scene=%d", act, scene)
	}
}

func TestSonnetMapping_ModifiedLines(t *testing.T) {
	dbPath, tmp := seedSonnetMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Sonnet 1: different spelling across editions
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 0, 1, 1, 'apple banana cherry dog elephant')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 2, 0, 1, 1, 'fox grape house igloo jungle kite')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var matchType string
	var similarity float64
	database.QueryRow(`SELECT match_type, similarity FROM line_mappings LIMIT 1`).Scan(&matchType, &similarity)
	if matchType != "modified" {
		t.Errorf("expected 'modified', got %q", matchType)
	}
	if similarity >= 0.2 {
		t.Errorf("expected similarity < 0.2, got %f", similarity)
	}
}

func TestSonnetMapping_MultipleSonnets(t *testing.T) {
	dbPath, tmp := seedSonnetMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Sonnet 18 — 2 lines each
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 0, 18, 1, 'Shall I compare thee')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 1, 0, 18, 2, 'Thou art more lovely')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (3, 1, 2, 0, 18, 1, 'Shall I compare thee')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (4, 1, 2, 0, 18, 2, 'Thou art more lovely')`)

	// Sonnet 130 — 2 lines each
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (5, 1, 1, 0, 130, 1, 'My mistress eyes are nothing like the sun')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (6, 1, 1, 0, 130, 2, 'Coral is far more red than her lips red')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (7, 1, 2, 0, 130, 1, 'My mistress eyes are nothing like the sun')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (8, 1, 2, 0, 130, 2, 'Coral is far more red than her lips red')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var total int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&total)
	if total != 4 {
		t.Errorf("expected 4 mapping pairs (2 per sonnet), got %d", total)
	}

	// Verify each sonnet is separate
	var s18, s130 int
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE scene = 18`).Scan(&s18)
	database.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE scene = 130`).Scan(&s130)
	if s18 != 2 || s130 != 2 {
		t.Errorf("expected 2 per sonnet, got s18=%d s130=%d", s18, s130)
	}
}

func TestSonnetMapping_ExtraLineInOneEdition(t *testing.T) {
	dbPath, tmp := seedSonnetMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Sonnet 73: OSS has 3 lines, SE has 2
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 0, 73, 1, 'That time of year thou mayst in me behold')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 1, 0, 73, 2, 'When yellow leaves or none or few do hang')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (3, 1, 1, 0, 73, 3, 'Upon those boughs which shake against the cold')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (4, 1, 2, 0, 73, 1, 'That time of year thou mayst in me behold')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (5, 1, 2, 0, 73, 2, 'Upon those boughs which shake against the cold')`)

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
		t.Errorf("expected 1 only_a pair, got %d", onlyA)
	}
}

func TestSonnetMapping_SonnetOnlyInOneEdition(t *testing.T) {
	dbPath, tmp := seedSonnetMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// Sonnet 154 only in OSS, not SE
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, 0, 154, 1, 'The little Love-god lying once asleep')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 mappings for sonnet only in one edition, got %d", count)
	}
}

func TestSonnetMapping_NullActTreatedAsZero(t *testing.T) {
	dbPath, tmp := seedSonnetMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	// OSS uses act=NULL, SE uses act=0 — both should match
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (1, 1, 1, NULL, 29, 1, 'When in disgrace with fortune')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (2, 1, 2, 0, 29, 1, 'When in disgrace with fortune')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 mapping (NULL act should match 0 act), got %d", count)
	}
}

func TestSonnetMapping_LineIDsPreserved(t *testing.T) {
	dbPath, tmp := seedSonnetMappingDB(t)
	defer os.Remove(tmp.Name())

	database, _ := db.Open(dbPath)
	defer database.Close()

	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (500, 1, 1, 0, 55, 1, 'Not marble nor the gilded monuments')`)
	database.Exec(`INSERT INTO text_lines (id, work_id, edition_id, act, scene, line_number, content) VALUES (600, 1, 2, 0, 55, 1, 'Not marble nor the gilded monuments')`)

	err := BuildLineMappings(database)
	if err != nil {
		t.Fatalf("BuildLineMappings failed: %v", err)
	}

	var lineAID, lineBID int64
	database.QueryRow(`SELECT line_a_id, line_b_id FROM line_mappings LIMIT 1`).Scan(&lineAID, &lineBID)
	if lineAID != 500 {
		t.Errorf("expected line_a_id=500, got %d", lineAID)
	}
	if lineBID != 600 {
		t.Errorf("expected line_b_id=600, got %d", lineBID)
	}
}
