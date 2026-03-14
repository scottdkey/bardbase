package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndCreateSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer database.Close()

	err = CreateSchema(database)
	if err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Verify tables exist
	tables := []string{"works", "characters", "text_lines", "lexicon_entries", "editions", "sources"}
	for _, table := range tables {
		count, err := TableCount(database, table)
		if err != nil {
			t.Errorf("TableCount(%s) failed: %v", table, err)
		}
		if count != 0 {
			t.Errorf("expected 0 rows in %s, got %d", table, count)
		}
	}
}

func TestRemoveIfExists(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a file
	f, _ := os.Create(dbPath)
	f.Close()

	err := RemoveIfExists(dbPath)
	if err != nil {
		t.Fatalf("RemoveIfExists failed: %v", err)
	}

	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Error("expected file to be removed")
	}
}

func TestRemoveIfExists_NoFile(t *testing.T) {
	err := RemoveIfExists("/tmp/nonexistent-shakespeare-test.db")
	if err != nil {
		t.Errorf("RemoveIfExists on nonexistent file should not error: %v", err)
	}
}

func TestGetSourceID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, _ := Open(dbPath)
	defer database.Close()
	CreateSchema(database)

	id, err := GetSourceID(database, "Test Source", "test", "http://test.com", "MIT", "", "", false, "")
	if err != nil {
		t.Fatalf("GetSourceID failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Second call should return same ID
	id2, err := GetSourceID(database, "Test Source", "test", "http://test.com", "MIT", "", "", false, "")
	if err != nil {
		t.Fatalf("second GetSourceID failed: %v", err)
	}
	if id != id2 {
		t.Errorf("expected same ID %d, got %d", id, id2)
	}
}

func TestGetEditionID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, _ := Open(dbPath)
	defer database.Close()
	CreateSchema(database)

	srcID, _ := GetSourceID(database, "Test", "test", "", "", "", "", false, "")
	edID, err := GetEditionID(database, "Test Edition", "test_ed", srcID, 2024, "Editor", "Desc")
	if err != nil {
		t.Fatalf("GetEditionID failed: %v", err)
	}
	if edID <= 0 {
		t.Errorf("expected positive ID, got %d", edID)
	}
}

func TestLogImport(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, _ := Open(dbPath)
	defer database.Close()
	CreateSchema(database)

	err := LogImport(database, "test", "test_action", "test details", 42, 1.5)
	if err != nil {
		t.Fatalf("LogImport failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM import_log WHERE phase = 'test'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 log entry, got %d", count)
	}
}
