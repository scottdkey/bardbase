package importer

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

// setupLexiconTestDB creates a temp DB with schema and prerequisite data.
func setupLexiconTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	// Insert a work so citations can resolve work_id
	_, err = database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Hamlet', 'Hml.', 'play')`)
	if err != nil {
		t.Fatalf("inserting work: %v", err)
	}
	_, err = database.Exec(`INSERT INTO works (title, schmidt_abbrev, work_type) VALUES ('Othello', 'Oth.', 'play')`)
	if err != nil {
		t.Fatalf("inserting work: %v", err)
	}

	return database, tmpDir
}

func TestImportLexicon_SenseIDWrittenToDB(t *testing.T) {
	database, tmpDir := setupLexiconTestDB(t)
	defer database.Close()

	// Create a minimal XML file with two senses and two citations
	entriesDir := filepath.Join(tmpDir, "entries")
	letterDir := filepath.Join(entriesDir, "T")
	os.MkdirAll(letterDir, 0755)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2><text><body><div1 n="T" type="alphabetic letter">
<entryFree key="Test" type="main"><orth>Test</orth>, 1) first meaning: <cit><quote>first quote</quote> <bibl n="shak. ham 1.1.1">Hml. I, 1, 1</bibl></cit>. 2) second meaning: <cit><quote>second quote</quote> <bibl n="shak. oth 2.3.4">Oth. II, 3, 4</bibl></cit>.
</entryFree></div1></body></text></TEI.2>`

	os.WriteFile(filepath.Join(letterDir, "test.xml"), []byte(xml), 0644)

	err := ImportLexicon(database, entriesDir)
	if err != nil {
		t.Fatalf("ImportLexicon: %v", err)
	}

	// Verify entry exists
	var entryID int64
	err = database.QueryRow("SELECT id FROM lexicon_entries WHERE key = 'Test'").Scan(&entryID)
	if err != nil {
		t.Fatalf("entry not found: %v", err)
	}

	// Verify senses exist
	var senseCount int
	database.QueryRow("SELECT COUNT(*) FROM lexicon_senses WHERE entry_id = ?", entryID).Scan(&senseCount)
	if senseCount != 2 {
		t.Fatalf("expected 2 senses, got %d", senseCount)
	}

	// Verify sense_id is set on citations
	rows, err := database.Query(`
		SELECT lc.id, lc.sense_id, ls.sense_number, lc.work_abbrev
		FROM lexicon_citations lc
		LEFT JOIN lexicon_senses ls ON lc.sense_id = ls.id
		WHERE lc.entry_id = ?
		ORDER BY lc.id`, entryID)
	if err != nil {
		t.Fatalf("querying citations: %v", err)
	}
	defer rows.Close()

	type citResult struct {
		id          int64
		senseID     *int64
		senseNumber *int
		workAbbrev  *string
	}
	var results []citResult
	for rows.Next() {
		var r citResult
		rows.Scan(&r.id, &r.senseID, &r.senseNumber, &r.workAbbrev)
		results = append(results, r)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 citations, got %d", len(results))
	}

	// First citation should be linked to sense 1
	if results[0].senseID == nil {
		t.Error("citation 0: sense_id is NULL, expected it to be set")
	} else if results[0].senseNumber == nil || *results[0].senseNumber != 1 {
		t.Errorf("citation 0: expected sense_number 1, got %v", results[0].senseNumber)
	}

	// Second citation should be linked to sense 2
	if results[1].senseID == nil {
		t.Error("citation 1: sense_id is NULL, expected it to be set")
	} else if results[1].senseNumber == nil || *results[1].senseNumber != 2 {
		t.Errorf("citation 1: expected sense_number 2, got %v", results[1].senseNumber)
	}
}

func TestImportLexicon_WorkIDResolved(t *testing.T) {
	database, tmpDir := setupLexiconTestDB(t)
	defer database.Close()

	entriesDir := filepath.Join(tmpDir, "entries")
	letterDir := filepath.Join(entriesDir, "A")
	os.MkdirAll(letterDir, 0755)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2><text><body><div1 n="A" type="alphabetic letter">
<entryFree key="Abandon" type="main"><orth>Abandon</orth>,
to give up: <cit><quote>abandon the society</quote> <bibl n="shak. ham 3.1.56">Hml. III, 1, 56</bibl></cit>.
</entryFree></div1></body></text></TEI.2>`

	os.WriteFile(filepath.Join(letterDir, "abandon.xml"), []byte(xml), 0644)

	err := ImportLexicon(database, entriesDir)
	if err != nil {
		t.Fatalf("ImportLexicon: %v", err)
	}

	// Verify work_id was resolved
	var workID *int64
	var workAbbrev *string
	err = database.QueryRow(`
		SELECT work_id, work_abbrev FROM lexicon_citations
		WHERE entry_id = (SELECT id FROM lexicon_entries WHERE key = 'Abandon')
		LIMIT 1`).Scan(&workID, &workAbbrev)
	if err != nil {
		t.Fatalf("querying citation: %v", err)
	}

	if workID == nil {
		t.Error("work_id is NULL — citation not resolved to a work")
	}
	if workAbbrev == nil || *workAbbrev != "Hml." {
		t.Errorf("expected work_abbrev 'Hml.', got %v", workAbbrev)
	}
}

func TestImportLexicon_ActSceneLineSet(t *testing.T) {
	database, tmpDir := setupLexiconTestDB(t)
	defer database.Close()

	entriesDir := filepath.Join(tmpDir, "entries")
	letterDir := filepath.Join(entriesDir, "B")
	os.MkdirAll(letterDir, 0755)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2><text><body><div1 n="B" type="alphabetic letter">
<entryFree key="Bold" type="main"><orth>Bold</orth>,
daring: <cit><quote>be bold</quote> <bibl n="shak. ham 5.2.10">Hml. V, 2, 10</bibl></cit>.
</entryFree></div1></body></text></TEI.2>`

	os.WriteFile(filepath.Join(letterDir, "bold.xml"), []byte(xml), 0644)

	err := ImportLexicon(database, entriesDir)
	if err != nil {
		t.Fatalf("ImportLexicon: %v", err)
	}

	var act, scene, line *int
	err = database.QueryRow(`
		SELECT act, scene, line FROM lexicon_citations
		WHERE entry_id = (SELECT id FROM lexicon_entries WHERE key = 'Bold')
		LIMIT 1`).Scan(&act, &scene, &line)
	if err != nil {
		t.Fatalf("querying citation: %v", err)
	}

	if act == nil || *act != 5 {
		t.Errorf("expected act 5, got %v", act)
	}
	if scene == nil || *scene != 2 {
		t.Errorf("expected scene 2, got %v", scene)
	}
	if line == nil || *line != 10 {
		t.Errorf("expected line 10, got %v", line)
	}
}

func TestImportLexicon_EmptyDirectory(t *testing.T) {
	database, tmpDir := setupLexiconTestDB(t)
	defer database.Close()

	entriesDir := filepath.Join(tmpDir, "entries")
	os.MkdirAll(entriesDir, 0755)

	err := ImportLexicon(database, entriesDir)
	if err != nil {
		t.Fatalf("ImportLexicon on empty dir should not error: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM lexicon_entries").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 entries, got %d", count)
	}
}

func TestImportLexicon_MissingDirectory(t *testing.T) {
	database, tmpDir := setupLexiconTestDB(t)
	defer database.Close()

	err := ImportLexicon(database, filepath.Join(tmpDir, "nonexistent"))
	if err != nil {
		t.Fatalf("ImportLexicon on missing dir should not error: %v", err)
	}
}

func TestImportLexicon_Idempotent(t *testing.T) {
	database, tmpDir := setupLexiconTestDB(t)
	defer database.Close()

	entriesDir := filepath.Join(tmpDir, "entries")
	letterDir := filepath.Join(entriesDir, "C")
	os.MkdirAll(letterDir, 0755)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2><text><body><div1 n="C" type="alphabetic letter">
<entryFree key="Call" type="main"><orth>Call</orth>,
to name: <cit><quote>call me</quote> <bibl n="shak. ham 1.1.1">Hml. I, 1, 1</bibl></cit>.
</entryFree></div1></body></text></TEI.2>`

	os.WriteFile(filepath.Join(letterDir, "call.xml"), []byte(xml), 0644)

	// Run twice
	ImportLexicon(database, entriesDir)
	ImportLexicon(database, entriesDir)

	var count int
	database.QueryRow("SELECT COUNT(*) FROM lexicon_entries WHERE key = 'Call'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 entry after double import, got %d", count)
	}
}
