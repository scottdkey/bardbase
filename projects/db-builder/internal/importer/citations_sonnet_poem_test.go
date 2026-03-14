package importer

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

// setupSonnetDB creates a test DB with sonnet text lines and a lexicon entry.
// Sonnets: scene = sonnet number, act = NULL, line_number = line within sonnet (1-14).
func setupSonnetDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Source and edition
	database.Exec(`INSERT INTO sources (id, name, short_code) VALUES (1, 'Standard Ebooks', 'se')`)
	database.Exec(`INSERT INTO editions (id, source_id, short_code, name) VALUES (1, 1, 'se_modern', 'SE Modern')`)

	// Work: Sonnets
	database.Exec(`INSERT INTO works (id, title, short_title, schmidt_abbrev, work_type) VALUES (1, 'Sonnets', 'Son.', 'son', 'poetry')`)

	// Sonnet 18 — all 14 lines
	sonnet18 := []string{
		"Shall I compare thee to a summer's day?",
		"Thou art more lovely and more temperate:",
		"Rough winds do shake the darling buds of May,",
		"And summer's lease hath all too short a date:",
		"Sometime too hot the eye of heaven shines,",
		"And often is his gold complexion dimm'd;",
		"And every fair from fair sometime declines,",
		"By chance, or nature's changing course untrimm'd;",
		"But thy eternal summer shall not fade,",
		"Nor lose possession of that fair thou ow'st;",
		"Nor shall death brag thou wander'st in his shade,",
		"When in eternal lines to time thou grow'st:",
		"So long as men can breathe, or eyes can see,",
		"So long lives this, and this gives life to thee.",
	}
	for i, line := range sonnet18 {
		database.Exec(`INSERT INTO text_lines (work_id, edition_id, scene, line_number, content)
			VALUES (1, 1, 18, ?, ?)`, i+1, line)
	}

	// Lexicon entry and sense
	database.Exec(`INSERT INTO lexicon_entries (id, key, letter) VALUES (1, 'Compare', 'C')`)
	database.Exec(`INSERT INTO lexicon_senses (id, entry_id, sense_number, definition_text) VALUES (1, 1, 1, 'to liken')`)

	return database
}

func TestResolveCitations_SonnetExactQuote(t *testing.T) {
	database := setupSonnetDB(t)
	defer database.Close()

	// Citation: Son. 18, line 1 with quote text
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, scene, line, quote_text)
		VALUES (1, 1, 1, 1, 18, 1, 'compare thee to a summer')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 match, got %d", count)
	}

	var matchType string
	var confidence float64
	var matchedText string
	database.QueryRow("SELECT match_type, confidence, matched_text FROM citation_matches WHERE citation_id = 1").
		Scan(&matchType, &confidence, &matchedText)

	if matchType != "exact_quote" {
		t.Errorf("expected exact_quote, got %s", matchType)
	}
	if confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", confidence)
	}
	if matchedText == "" {
		t.Error("matched_text should not be empty")
	}
}

func TestResolveCitations_SonnetLineNumber(t *testing.T) {
	database := setupSonnetDB(t)
	defer database.Close()

	// Citation: Son. 18, line 14 — no quote, just line number
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, scene, line, quote_text)
		VALUES (1, 1, 1, 1, 18, 14, '')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var matchType string
	var confidence float64
	database.QueryRow("SELECT match_type, confidence FROM citation_matches WHERE citation_id = 1").
		Scan(&matchType, &confidence)

	if matchType != "line_number" {
		t.Errorf("expected line_number, got %s", matchType)
	}
	if confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", confidence)
	}
}

func TestResolveCitations_SonnetNoMatch(t *testing.T) {
	database := setupSonnetDB(t)
	defer database.Close()

	// Citation to sonnet 99 — doesn't exist
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, scene, line, quote_text)
		VALUES (1, 1, 1, 1, 99, 1, '')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 matches for nonexistent sonnet, got %d", count)
	}
}

func TestResolveCitations_SonnetMultipleEditions(t *testing.T) {
	database := setupSonnetDB(t)
	defer database.Close()

	// Add second edition
	database.Exec(`INSERT INTO sources (id, name, short_code) VALUES (2, 'Open Source Shakespeare', 'oss')`)
	database.Exec(`INSERT INTO editions (id, source_id, short_code, name) VALUES (2, 2, 'oss_globe', 'OSS Globe')`)
	database.Exec(`INSERT INTO text_lines (work_id, edition_id, scene, line_number, content)
		VALUES (1, 2, 18, 1, 'Shall I compare thee to a Summers day?')`)

	// Citation with quote
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, scene, line, quote_text)
		VALUES (1, 1, 1, 1, 18, 1, 'compare thee to a summer')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 matches (one per edition), got %d", count)
	}
}

// setupPoemDB creates a test DB with poem text lines.
// Poems: act = NULL, scene = NULL, line_number = poem-relative.
func setupPoemDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}

	database.Exec(`INSERT INTO sources (id, name, short_code) VALUES (1, 'Standard Ebooks', 'se')`)
	database.Exec(`INSERT INTO editions (id, source_id, short_code, name) VALUES (1, 1, 'se_modern', 'SE Modern')`)

	// Work: Venus and Adonis
	database.Exec(`INSERT INTO works (id, title, short_title, schmidt_abbrev, work_type) VALUES (1, 'Venus and Adonis', 'Ven.', 'ven', 'poetry')`)

	poemLines := []struct {
		lineNum int
		content string
	}{
		{1, "Even as the sun with purple-colour'd face"},
		{2, "Had ta'en his last leave of the weeping morn,"},
		{3, "Rose-cheek'd Adonis hied him to the chase;"},
		{4, "Hunting he loved, but love he laugh'd to scorn;"},
		{5, "Sick-thoughted Venus makes amain unto him,"},
		{6, "And like a bold-faced suitor 'gins to woo him."},
		{50, "'Thrice-fairer than myself,' thus she began,"},
		{51, "'The field's chief flower, sweet above compare,"},
		{52, "Stain to all nymphs, more lovely than a man,"},
		{53, "More white and red than doves or roses are;"},
	}
	for _, l := range poemLines {
		database.Exec(`INSERT INTO text_lines (work_id, edition_id, line_number, content)
			VALUES (1, 1, ?, ?)`, l.lineNum, l.content)
	}

	// Lexicon entry and sense
	database.Exec(`INSERT INTO lexicon_entries (id, key, letter) VALUES (1, 'Chase', 'C')`)
	database.Exec(`INSERT INTO lexicon_senses (id, entry_id, sense_number, definition_text) VALUES (1, 1, 1, 'the hunt')`)

	return database
}

func TestResolveCitations_PoemLineNumber(t *testing.T) {
	database := setupPoemDB(t)
	defer database.Close()

	// Citation: Ven. 3
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, line, quote_text)
		VALUES (1, 1, 1, 1, 3, '')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 match, got %d", count)
	}

	var matchType string
	var confidence float64
	database.QueryRow("SELECT match_type, confidence FROM citation_matches WHERE citation_id = 1").
		Scan(&matchType, &confidence)

	if matchType != "line_number" {
		t.Errorf("expected line_number, got %s", matchType)
	}
	if confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", confidence)
	}
}

func TestResolveCitations_PoemExactQuote(t *testing.T) {
	database := setupPoemDB(t)
	defer database.Close()

	// Wrong line number but quote exists in text — exact_quote should win
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, line, quote_text)
		VALUES (1, 1, 1, 1, 999, 'Adonis hied him to the chase')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 match, got %d", count)
	}

	var matchType string
	database.QueryRow("SELECT match_type FROM citation_matches WHERE citation_id = 1").Scan(&matchType)
	if matchType != "exact_quote" {
		t.Errorf("expected exact_quote, got %s", matchType)
	}
}

func TestResolveCitations_PoemFuzzyMatch(t *testing.T) {
	database := setupPoemDB(t)
	defer database.Close()

	// Partial words that overlap with line 1 — fuzzy match
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, line, quote_text)
		VALUES (1, 1, 1, 1, 999, 'even sun with face')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 fuzzy match, got %d", count)
	}

	var matchType string
	var confidence float64
	database.QueryRow("SELECT match_type, confidence FROM citation_matches WHERE citation_id = 1").
		Scan(&matchType, &confidence)

	if matchType != "fuzzy_text" {
		t.Errorf("expected fuzzy_text, got %s", matchType)
	}
	if confidence <= 0.25 {
		t.Errorf("expected confidence > 0.25, got %f", confidence)
	}
}

func TestResolveCitations_PoemNoMatch(t *testing.T) {
	database := setupPoemDB(t)
	defer database.Close()

	// Completely unrelated text, invalid line number
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, line, quote_text)
		VALUES (1, 1, 1, 1, 999, 'something completely unrelated xyz abc')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 matches, got %d", count)
	}
}

func TestResolveCitations_PoemQuoteOnlyNoLine(t *testing.T) {
	database := setupPoemDB(t)
	defer database.Close()

	// Quote text only — no line number
	database.Exec(`INSERT INTO lexicon_citations (id, entry_id, sense_id, work_id, quote_text)
		VALUES (1, 1, 1, 1, 'bold-faced suitor')`)

	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations failed: %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 match, got %d", count)
	}

	var matchType string
	database.QueryRow("SELECT match_type FROM citation_matches WHERE citation_id = 1").Scan(&matchType)
	if matchType != "exact_quote" {
		t.Errorf("expected exact_quote, got %s", matchType)
	}
}
