// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// setupPerseusTestDB creates a test database with schema, a Perseus source/edition,
// and a sample work with a perseus_id.
func setupPerseusTestDB(t *testing.T) (*sql.DB, int64, int64, func()) {
	t.Helper()
	td := newTestDB(t)

	// Create Perseus source + edition
	td.DB.Exec(`INSERT INTO sources (id, name, short_code, url, license, license_url, attribution_text, attribution_required)
		VALUES (3, 'Perseus Digital Library', 'perseus', 'https://www.perseus.tufts.edu', 'CC BY-SA 3.0 US',
		'https://creativecommons.org/licenses/by-sa/3.0/us/', 'Perseus Digital Library', 1)`)
	td.DB.Exec(`INSERT INTO editions (id, name, short_code, source_id, year, editors, description)
		VALUES (3, 'Perseus Globe', 'perseus_globe', 3, 1864, 'Clark, Wright', 'Globe edition')`)

	// Create a work with a perseus_id
	td.DB.Exec(`INSERT INTO works (id, oss_id, title, perseus_id, work_type)
		VALUES (1, 'tempest', 'The Tempest', 'test-play', 'play')`)

	// Create a character for the work
	td.DB.Exec(`INSERT INTO characters (id, work_id, name, abbrev)
		VALUES (1, 1, 'Prospero', 'Pros.')`)

	return td.DB, 1, 3, func() { td.DB.Close() }
}

func TestImportPerseusPlays_BasicImport(t *testing.T) {
	database, workID, editionID, cleanup := setupPerseusTestDB(t)
	defer cleanup()

	// Create a temp directory with a test XML file
	tmpDir := t.TempDir()
	perseusDir := filepath.Join(tmpDir, "perseus-plays")
	os.MkdirAll(perseusDir, 0755)

	testXML := `<?xml version="1.0" encoding="utf-8"?>
<TEI.2><text lang="en"><body>
<div1 type="act" n="1">
  <div2 type="scene" n="1">
    <stage type="setting">An island.</stage>
    <stage type="entrance">Enter PROSPERO.</stage>
    <sp who="pros-1"><speaker>Pros.</speaker>
      <p><lb ed="F1" n="1" />Now does my project gather to a head:
      <lb ed="G" /><lb ed="F1" n="2" />My charms crack not; my spirits obey.
      <lb ed="G" /></p>
    </sp>
  </div2>
</div1>
</body></text></TEI.2>`

	os.WriteFile(filepath.Join(perseusDir, "test-play.xml"), []byte(testXML), 0644)

	err := ImportPerseusPlays(database, tmpDir)
	if err != nil {
		t.Fatalf("ImportPerseusPlays failed: %v", err)
	}

	// Verify lines were inserted
	var lineCount int
	database.QueryRow("SELECT COUNT(*) FROM text_lines WHERE work_id = ? AND edition_id = ?",
		workID, editionID).Scan(&lineCount)

	if lineCount != 4 {
		t.Errorf("expected 4 text_lines (2 stage dirs + 2 speeches), got %d", lineCount)
	}

	// Verify stage directions
	var sdCount int
	database.QueryRow("SELECT COUNT(*) FROM text_lines WHERE work_id = ? AND edition_id = ? AND content_type = 'stage_direction'",
		workID, editionID).Scan(&sdCount)
	if sdCount != 2 {
		t.Errorf("expected 2 stage_direction lines, got %d", sdCount)
	}

	// Verify speech content
	var content string
	database.QueryRow("SELECT content FROM text_lines WHERE work_id = ? AND edition_id = ? AND content_type = 'speech' ORDER BY line_number LIMIT 1",
		workID, editionID).Scan(&content)
	if content != "Now does my project gather to a head:" {
		t.Errorf("unexpected first speech: %q", content)
	}

	// Verify character matching (Pros. → Prospero)
	var charName sql.NullString
	database.QueryRow("SELECT char_name FROM text_lines WHERE work_id = ? AND edition_id = ? AND content_type = 'speech' LIMIT 1",
		workID, editionID).Scan(&charName)
	if !charName.Valid || charName.String != "Pros." {
		t.Errorf("expected char_name 'Pros.', got %v", charName)
	}

	// Verify text_divisions
	var divCount int
	database.QueryRow("SELECT COUNT(*) FROM text_divisions WHERE work_id = ? AND edition_id = ?",
		workID, editionID).Scan(&divCount)
	if divCount != 1 {
		t.Errorf("expected 1 text_division (Act 1 Scene 1), got %d", divCount)
	}

	// Verify import log
	var logCount int
	database.QueryRow("SELECT COUNT(*) FROM import_log WHERE phase = 'perseus_plays'").Scan(&logCount)
	if logCount != 1 {
		t.Errorf("expected 1 import_log entry, got %d", logCount)
	}
}

func TestImportPerseusPlays_IdempotentReimport(t *testing.T) {
	database, workID, editionID, cleanup := setupPerseusTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()
	perseusDir := filepath.Join(tmpDir, "perseus-plays")
	os.MkdirAll(perseusDir, 0755)

	testXML := `<?xml version="1.0" encoding="utf-8"?>
<TEI.2><text lang="en"><body>
<div1 type="act" n="1">
  <div2 type="scene" n="1">
    <sp who="a-1"><speaker>A.</speaker>
      <p><lb ed="F1" n="1" />Line one.
      <lb ed="G" /></p>
    </sp>
  </div2>
</div1>
</body></text></TEI.2>`

	os.WriteFile(filepath.Join(perseusDir, "test-play.xml"), []byte(testXML), 0644)

	// Import twice
	ImportPerseusPlays(database, tmpDir)
	ImportPerseusPlays(database, tmpDir)

	// Should still have exactly 1 line (not doubled)
	var count int
	database.QueryRow("SELECT COUNT(*) FROM text_lines WHERE work_id = ? AND edition_id = ?",
		workID, editionID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 line after re-import, got %d", count)
	}
}

func TestImportPerseusPlays_NoMatchingWork(t *testing.T) {
	database, _, _, cleanup := setupPerseusTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()
	perseusDir := filepath.Join(tmpDir, "perseus-plays")
	os.MkdirAll(perseusDir, 0755)

	// This file's perseus_id ("unknown-play") won't match any work
	testXML := `<?xml version="1.0" encoding="utf-8"?>
<TEI.2><text lang="en"><body>
<div1 type="act" n="1">
  <div2 type="scene" n="1">
    <sp who="a-1"><speaker>A.</speaker>
      <p><lb ed="F1" n="1" />Orphan line.
      <lb ed="G" /></p>
    </sp>
  </div2>
</div1>
</body></text></TEI.2>`

	os.WriteFile(filepath.Join(perseusDir, "unknown-play.xml"), []byte(testXML), 0644)

	err := ImportPerseusPlays(database, tmpDir)
	if err != nil {
		t.Fatalf("should not error on unmatched work: %v", err)
	}

	// No lines should be inserted
	var count int
	database.QueryRow("SELECT COUNT(*) FROM text_lines").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 lines for unmatched work, got %d", count)
	}
}

func TestImportPerseusPlays_VerseLines(t *testing.T) {
	database, workID, editionID, cleanup := setupPerseusTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()
	perseusDir := filepath.Join(tmpDir, "perseus-plays")
	os.MkdirAll(perseusDir, 0755)

	// Use <l> verse format (like King John)
	testXML := `<?xml version="1.0" encoding="utf-8"?>
<TEI.2><text lang="en"><body>
<div1 type="act" n="2">
  <div2 type="scene" n="3">
    <sp who="x-1"><speaker>X.</speaker>
      <l>First verse line of the scene.
      <lb ed="G" /><lb ed="F1" n="1" /></l>
      <l>Second verse line with number.
      <lb ed="G" n="10" /><lb ed="F1" n="2" /></l>
    </sp>
  </div2>
</div1>
</body></text></TEI.2>`

	os.WriteFile(filepath.Join(perseusDir, "test-play.xml"), []byte(testXML), 0644)

	err := ImportPerseusPlays(database, tmpDir)
	if err != nil {
		t.Fatalf("ImportPerseusPlays failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM text_lines WHERE work_id = ? AND edition_id = ?",
		workID, editionID).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 verse lines, got %d", count)
	}

	// Verify act/scene
	var act, scene int
	database.QueryRow("SELECT act, scene FROM text_lines WHERE work_id = ? AND edition_id = ? LIMIT 1",
		workID, editionID).Scan(&act, &scene)
	if act != 2 || scene != 3 {
		t.Errorf("expected act=2 scene=3, got act=%d scene=%d", act, scene)
	}
}

func TestImportPerseusPlays_EmptyDirectory(t *testing.T) {
	database, _, _, cleanup := setupPerseusTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()
	perseusDir := filepath.Join(tmpDir, "perseus-plays")
	os.MkdirAll(perseusDir, 0755)

	err := ImportPerseusPlays(database, tmpDir)
	if err != nil {
		t.Fatalf("should handle empty directory gracefully: %v", err)
	}
}
