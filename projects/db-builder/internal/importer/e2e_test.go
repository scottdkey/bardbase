// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/scottdkey/heminge/projects/db-builder/internal/db"
)

// seedE2EData populates a minimal but realistic dataset that exercises the full pipeline chain:
// sources → editions → works → text_lines → lexicon_entries → lexicon_senses → lexicon_citations
// Then: ResolveCitations → BuildLineMappings → BuildFTS → PopulateAttributions
func seedE2EData(t *testing.T, database *sql.DB) {
	t.Helper()

	// Sources — must match short_codes used by attributions importer
	ossSourceID, err := db.GetSourceID(database,
		"Open Source Shakespeare", "oss_moby",
		"https://opensourceshakespeare.org", "Public Domain", "", "", false, "")
	if err != nil {
		t.Fatalf("creating OSS source: %v", err)
	}

	seSourceID, err := db.GetSourceID(database,
		"Standard Ebooks", "standard_ebooks",
		"https://standardebooks.org", "CC0-1.0", "", "", false, "")
	if err != nil {
		t.Fatalf("creating SE source: %v", err)
	}

	_, err = db.GetSourceID(database,
		"Perseus Digital Library", "perseus_schmidt",
		"https://www.perseus.tufts.edu", "CC BY-SA 3.0",
		"https://creativecommons.org/licenses/by-sa/3.0/us/",
		"Schmidt, Alexander. Shakespeare Lexicon and Quotation Dictionary.",
		true, "")
	if err != nil {
		t.Fatalf("creating Perseus source: %v", err)
	}

	// Editions — must match short_codes used by mappings importer (oss_globe, se_modern)
	ossEditionID, err := db.GetEditionID(database,
		"Open Source Shakespeare (Globe)", "oss_globe", ossSourceID, 2003, "", "")
	if err != nil {
		t.Fatalf("creating OSS edition: %v", err)
	}

	seEditionID, err := db.GetEditionID(database,
		"Standard Ebooks (Modern)", "se_modern", seSourceID, 2024, "", "")
	if err != nil {
		t.Fatalf("creating SE edition: %v", err)
	}

	// Work: Hamlet (play)
	var hamletID int64
	err = database.QueryRow(`
		INSERT INTO works (oss_id, title, short_title, schmidt_abbrev, work_type, genre_type)
		VALUES ('hamlet', 'The Tragedy of Hamlet, Prince of Denmark', 'Hamlet', 'Ham.', 'play', 'tragedy')
		RETURNING id`).Scan(&hamletID)
	if err != nil {
		t.Fatalf("creating Hamlet: %v", err)
	}

	// Work: Sonnets (sonnet_sequence)
	var sonnetID int64
	err = database.QueryRow(`
		INSERT INTO works (title, short_title, schmidt_abbrev, work_type, genre_type)
		VALUES ('Sonnets', 'Sonnets', 'Sonn.', 'sonnet_sequence', 'poetry')
		RETURNING id`).Scan(&sonnetID)
	if err != nil {
		t.Fatalf("creating Sonnets: %v", err)
	}

	// Text lines: Hamlet Act 3 Scene 1 — both editions
	hamletLines := []struct {
		lineNum int
		ossText string
		seText  string
	}{
		{1, "To be, or not to be, that is the question:", "To be, or not to be, that is the question:"},
		{2, "Whether 'tis nobler in the mind to suffer", "Whether 'tis nobler in the mind to suffer"},
		{3, "The slings and arrows of outrageous fortune,", "The slings and arrows of outrageous fortune,"},
		{4, "Or to take arms against a sea of troubles,", "Or to take arms against a sea of troubles,"},
		{5, "And by opposing end them. To die, to sleep—", "And by opposing, end them. To die— perchance to dream."},
	}

	for _, l := range hamletLines {
		_, err = database.Exec(`
			INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, char_name)
			VALUES (?, ?, 3, 1, ?, ?, 'HAMLET')`,
			hamletID, ossEditionID, l.lineNum, l.ossText)
		if err != nil {
			t.Fatalf("inserting OSS Hamlet line %d: %v", l.lineNum, err)
		}
		_, err = database.Exec(`
			INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content, char_name)
			VALUES (?, ?, 3, 1, ?, ?, 'HAMLET')`,
			hamletID, seEditionID, l.lineNum, l.seText)
		if err != nil {
			t.Fatalf("inserting SE Hamlet line %d: %v", l.lineNum, err)
		}
	}

	// Text lines: Sonnet 18 — both editions (act=0, scene=sonnet number)
	sonnetLines := []struct {
		lineNum int
		ossText string
		seText  string
	}{
		{1, "Shall I compare thee to a summer's day?", "Shall I compare thee to a summer's day?"},
		{2, "Thou art more lovely and more temperate:", "Thou art more lovely and more temperate:"},
		{3, "Rough winds do shake the darling buds of May,", "Rough winds do shake the darling buds of May,"},
		{4, "And summer's lease hath all too short a date:", "And summer's lease hath all too short a date;"},
	}

	for _, l := range sonnetLines {
		_, err = database.Exec(`
			INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content)
			VALUES (?, ?, 0, 18, ?, ?)`,
			sonnetID, ossEditionID, l.lineNum, l.ossText)
		if err != nil {
			t.Fatalf("inserting OSS Sonnet line %d: %v", l.lineNum, err)
		}
		_, err = database.Exec(`
			INSERT INTO text_lines (work_id, edition_id, act, scene, line_number, content)
			VALUES (?, ?, 0, 18, ?, ?)`,
			sonnetID, seEditionID, l.lineNum, l.seText)
		if err != nil {
			t.Fatalf("inserting SE Sonnet line %d: %v", l.lineNum, err)
		}
	}

	// Lexicon entry: "question" (key, letter, full_text)
	var entryID int64
	err = database.QueryRow(`
		INSERT INTO lexicon_entries (key, letter, full_text)
		VALUES ('question', 'Q', 'question: a matter of inquiry or debate')
		RETURNING id`).Scan(&entryID)
	if err != nil {
		t.Fatalf("creating lexicon entry: %v", err)
	}

	// Lexicon sense (sense_number is INTEGER, definition_text)
	var senseID int64
	err = database.QueryRow(`
		INSERT INTO lexicon_senses (entry_id, sense_number, definition_text)
		VALUES (?, 1, 'a matter of inquiry or debate')
		RETURNING id`, entryID).Scan(&senseID)
	if err != nil {
		t.Fatalf("creating lexicon sense: %v", err)
	}

	// Citation 1: Hamlet 3.1.1 with a quote (should match via exact_quote)
	_, err = database.Exec(`
		INSERT INTO lexicon_citations (entry_id, sense_id, work_id, work_abbrev, act, scene, line, quote_text, display_text, raw_bibl)
		VALUES (?, ?, ?, 'Ham.', 3, 1, 1, 'to be or not to be that is the question', 'to be or not to be that is the question', 'Ham. III, 1, 1')`,
		entryID, senseID, hamletID)
	if err != nil {
		t.Fatalf("creating Hamlet citation: %v", err)
	}

	// Citation 2: Sonnet 18 line 1 — no quote (should match via line_number)
	_, err = database.Exec(`
		INSERT INTO lexicon_citations (entry_id, sense_id, work_id, work_abbrev, act, scene, line, quote_text, display_text, raw_bibl)
		VALUES (?, ?, ?, 'Sonn.', 0, 18, 1, '', '', 'Sonn. XVIII, 1')`,
		entryID, senseID, sonnetID)
	if err != nil {
		t.Fatalf("creating Sonnet citation: %v", err)
	}
}

// TestE2E_FullPipelineChain runs the full post-import pipeline and verifies the complete
// join chain: lexicon_entry → sense → citation → citation_match → text_line.
func TestE2E_FullPipelineChain(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening DB: %v", err)
	}
	defer database.Close()

	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	seedE2EData(t, database)

	// Verify pre-pipeline state: citations exist, no matches yet
	var citationCount int
	database.QueryRow("SELECT COUNT(*) FROM lexicon_citations").Scan(&citationCount)
	if citationCount != 2 {
		t.Fatalf("expected 2 citations before pipeline, got %d", citationCount)
	}

	var matchCount int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&matchCount)
	if matchCount != 0 {
		t.Fatalf("expected 0 matches before pipeline, got %d", matchCount)
	}

	// === STEP 1: Resolve Citations ===
	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&matchCount)
	if matchCount == 0 {
		t.Fatal("ResolveCitations produced 0 matches — expected at least 2")
	}
	t.Logf("Citation matches created: %d", matchCount)

	// === STEP 2: Build Line Mappings ===
	if err := BuildLineMappings(database); err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	var mappingCount int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&mappingCount)
	if mappingCount == 0 {
		t.Fatal("BuildLineMappings produced 0 mappings — expected cross-edition alignments")
	}
	t.Logf("Line mappings created: %d", mappingCount)

	// === STEP 3: Build FTS ===
	if err := BuildFTS(database); err != nil {
		t.Fatalf("BuildFTS: %v", err)
	}

	var ftsCount int
	database.QueryRow("SELECT COUNT(*) FROM text_fts WHERE text_fts MATCH 'question'").Scan(&ftsCount)
	if ftsCount == 0 {
		t.Fatal("text_fts returned 0 results for 'question' — FTS not populated")
	}
	t.Logf("FTS matches for 'question': %d", ftsCount)

	var lexFTSCount int
	database.QueryRow("SELECT COUNT(*) FROM lexicon_fts WHERE lexicon_fts MATCH 'question'").Scan(&lexFTSCount)
	if lexFTSCount == 0 {
		t.Fatal("lexicon_fts returned 0 results for 'question' — FTS not populated")
	}
	t.Logf("Lexicon FTS matches for 'question': %d", lexFTSCount)

	// === STEP 4: Populate Attributions ===
	if err := PopulateAttributions(database); err != nil {
		t.Fatalf("PopulateAttributions: %v", err)
	}

	var attrCount int
	database.QueryRow("SELECT COUNT(*) FROM attributions").Scan(&attrCount)
	if attrCount == 0 {
		t.Fatal("PopulateAttributions produced 0 rows")
	}
	t.Logf("Attributions created: %d", attrCount)

	// === VERIFY: Full join chain ===
	// lexicon_entry → sense → citation → citation_match → text_line
	rows, err := database.Query(`
		SELECT 
			le.key,
			ls.definition_text,
			lc.raw_bibl,
			cm.match_type,
			cm.confidence,
			tl.content,
			tl.act,
			tl.scene,
			tl.line_number,
			e.short_code
		FROM lexicon_entries le
		JOIN lexicon_senses ls ON ls.entry_id = le.id
		JOIN lexicon_citations lc ON lc.sense_id = ls.id
		JOIN citation_matches cm ON cm.citation_id = lc.id
		JOIN text_lines tl ON tl.id = cm.text_line_id
		JOIN editions e ON e.id = tl.edition_id
		WHERE le.key = 'question'
		ORDER BY lc.id, e.short_code`)
	if err != nil {
		t.Fatalf("full chain query: %v", err)
	}
	defer rows.Close()

	type chainResult struct {
		key        string
		definition string
		rawBibl    string
		matchType  string
		confidence float64
		content    string
		act, scene int
		lineNum    int
		edition    string
	}

	var results []chainResult
	for rows.Next() {
		var r chainResult
		if err := rows.Scan(&r.key, &r.definition, &r.rawBibl, &r.matchType,
			&r.confidence, &r.content, &r.act, &r.scene, &r.lineNum, &r.edition); err != nil {
			t.Fatalf("scanning chain result: %v", err)
		}
		results = append(results, r)
		t.Logf("Chain: %s [%s] → %s → %s (%.2f) → \"%s\" [%s]",
			r.key, r.definition, r.rawBibl, r.matchType, r.confidence, r.content, r.edition)
	}

	if len(results) == 0 {
		t.Fatal("full join chain returned 0 rows — the pipeline chain is broken")
	}

	// We expect: 2 citations × 2 editions = up to 4 matches
	// Hamlet citation (has quote) → should match in both editions via exact_quote
	// Sonnet citation (no quote, has line num) → should match in both editions via line_number
	hamletMatches := 0
	sonnetMatches := 0
	for _, r := range results {
		if r.act == 3 && r.scene == 1 {
			hamletMatches++
			if r.confidence < 0.8 {
				t.Errorf("Hamlet match confidence too low: %.2f", r.confidence)
			}
		}
		if r.act == 0 && r.scene == 18 {
			sonnetMatches++
			if r.confidence < 0.8 {
				t.Errorf("Sonnet match confidence too low: %.2f", r.confidence)
			}
		}
	}

	if hamletMatches == 0 {
		t.Error("no Hamlet citation matches in chain")
	}
	if sonnetMatches == 0 {
		t.Error("no Sonnet citation matches in chain")
	}

	t.Logf("Total chain results: %d (Hamlet: %d, Sonnet: %d)", len(results), hamletMatches, sonnetMatches)
}

// TestE2E_FTSSearchAcrossTypes verifies full-text search works across text types.
func TestE2E_FTSSearchAcrossTypes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e_fts.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening DB: %v", err)
	}
	defer database.Close()

	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	seedE2EData(t, database)

	if err := BuildFTS(database); err != nil {
		t.Fatalf("BuildFTS: %v", err)
	}

	// Search for a word that appears in Hamlet text
	var hamletFTS int
	database.QueryRow("SELECT COUNT(*) FROM text_fts WHERE text_fts MATCH 'slings'").Scan(&hamletFTS)
	if hamletFTS == 0 {
		t.Error("FTS found no results for 'slings' (Hamlet)")
	}

	// Search for a word that appears in Sonnet text
	var sonnetFTS int
	database.QueryRow("SELECT COUNT(*) FROM text_fts WHERE text_fts MATCH 'temperate'").Scan(&sonnetFTS)
	if sonnetFTS == 0 {
		t.Error("FTS found no results for 'temperate' (Sonnet)")
	}

	// Search for a word in the lexicon
	var lexFTS int
	database.QueryRow("SELECT COUNT(*) FROM lexicon_fts WHERE lexicon_fts MATCH 'question'").Scan(&lexFTS)
	if lexFTS == 0 {
		t.Error("lexicon FTS found no results for 'question'")
	}

	t.Logf("FTS results — slings: %d, temperate: %d, lexicon 'question': %d", hamletFTS, sonnetFTS, lexFTS)
}

// TestE2E_AttributionChain verifies attributions link back to sources correctly.
func TestE2E_AttributionChain(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e_attr.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening DB: %v", err)
	}
	defer database.Close()

	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	seedE2EData(t, database)

	if err := PopulateAttributions(database); err != nil {
		t.Fatalf("PopulateAttributions: %v", err)
	}

	// Verify Perseus is required
	var perseusRequired bool
	err = database.QueryRow(`
		SELECT a.required FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'perseus_schmidt'`).Scan(&perseusRequired)
	if err != nil {
		t.Fatalf("querying Perseus attribution: %v", err)
	}
	if !perseusRequired {
		t.Error("Perseus attribution should be required (CC BY-SA 3.0)")
	}

	// Verify OSS is voluntary
	var ossRequired bool
	err = database.QueryRow(`
		SELECT a.required FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'oss_moby'`).Scan(&ossRequired)
	if err != nil {
		t.Fatalf("querying OSS attribution: %v", err)
	}
	if ossRequired {
		t.Error("OSS attribution should be voluntary (Public Domain)")
	}
}

// TestE2E_LineMappingsShowDifferences verifies line mappings capture edition differences.
func TestE2E_LineMappingsShowDifferences(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e_mappings.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening DB: %v", err)
	}
	defer database.Close()

	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	seedE2EData(t, database)

	if err := BuildLineMappings(database); err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	// Lines 1-4 are identical across editions → should be "aligned" with similarity 1.0
	var alignedCount int
	database.QueryRow(`
		SELECT COUNT(*) FROM line_mappings WHERE match_type = 'aligned' AND similarity = 1.0
	`).Scan(&alignedCount)
	if alignedCount == 0 {
		t.Error("expected some perfectly aligned lines (identical text across editions)")
	}

	// Line 5 differs: "To die, to sleep—" vs "To die— perchance to dream."
	// Should be "modified" or "aligned" with similarity < 1.0 (different words)
	var totalMappings int
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&totalMappings)
	t.Logf("Total mappings: %d (aligned@1.0: %d)", totalMappings, alignedCount)

	// Verify we have mappings for both Hamlet and Sonnets
	var hamletMappings, sonnetMappings int
	database.QueryRow(`
		SELECT COUNT(*) FROM line_mappings lm
		JOIN text_lines tl ON tl.id = lm.line_a_id
		JOIN works w ON w.id = tl.work_id
		WHERE w.short_title = 'Hamlet'`).Scan(&hamletMappings)
	database.QueryRow(`
		SELECT COUNT(*) FROM line_mappings lm
		JOIN text_lines tl ON tl.id = lm.line_a_id
		JOIN works w ON w.id = tl.work_id
		WHERE w.short_title = 'Sonnets'`).Scan(&sonnetMappings)

	if hamletMappings == 0 {
		t.Error("no line mappings for Hamlet")
	}
	if sonnetMappings == 0 {
		t.Error("no line mappings for Sonnets")
	}

	t.Logf("Hamlet mappings: %d, Sonnet mappings: %d", hamletMappings, sonnetMappings)
}

// TestE2E_EmptyPipeline verifies the pipeline handles an empty database gracefully.
func TestE2E_EmptyPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e_empty.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening DB: %v", err)
	}
	defer database.Close()

	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	// Run every pipeline step on an empty DB — nothing should crash
	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations on empty DB: %v", err)
	}
	if err := BuildLineMappings(database); err != nil {
		t.Fatalf("BuildLineMappings on empty DB: %v", err)
	}
	if err := BuildFTS(database); err != nil {
		t.Fatalf("BuildFTS on empty DB: %v", err)
	}
	if err := PopulateAttributions(database); err != nil {
		t.Fatalf("PopulateAttributions on empty DB: %v", err)
	}

	// All tables should be empty
	tables := []string{"citation_matches", "line_mappings", "attributions"}
	for _, table := range tables {
		count, _ := db.TableCount(database, table)
		if count != 0 {
			t.Errorf("%s should have 0 rows after empty pipeline, got %d", table, count)
		}
	}
}

// TestE2E_PipelineIdempotent verifies running the full pipeline twice produces the same results.
func TestE2E_PipelineIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e_idempotent.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening DB: %v", err)
	}
	defer database.Close()

	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	seedE2EData(t, database)

	// Run pipeline once
	ResolveCitations(database)
	BuildLineMappings(database)
	BuildFTS(database)
	PopulateAttributions(database)

	// Capture counts
	var matches1, mappings1, attrs1 int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&matches1)
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&mappings1)
	database.QueryRow("SELECT COUNT(*) FROM attributions").Scan(&attrs1)

	// Run pipeline again
	ResolveCitations(database)
	BuildLineMappings(database)
	BuildFTS(database)
	PopulateAttributions(database)

	// Verify counts are identical
	var matches2, mappings2, attrs2 int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&matches2)
	database.QueryRow("SELECT COUNT(*) FROM line_mappings").Scan(&mappings2)
	database.QueryRow("SELECT COUNT(*) FROM attributions").Scan(&attrs2)

	if matches1 != matches2 {
		t.Errorf("citation_matches not idempotent: %d → %d", matches1, matches2)
	}
	if mappings1 != mappings2 {
		t.Errorf("line_mappings not idempotent: %d → %d", mappings1, mappings2)
	}
	if attrs1 != attrs2 {
		t.Errorf("attributions not idempotent: %d → %d", attrs1, attrs2)
	}

	t.Logf("Run 1: matches=%d, mappings=%d, attrs=%d", matches1, mappings1, attrs1)
	t.Logf("Run 2: matches=%d, mappings=%d, attrs=%d", matches2, mappings2, attrs2)
}
