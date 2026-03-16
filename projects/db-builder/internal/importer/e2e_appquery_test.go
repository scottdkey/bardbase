// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

// setupFullPipelineDB creates a DB, seeds data, and runs the full pipeline.
// Returns the ready-to-query database. This is the state the SvelteKit app receives.
func setupFullPipelineDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "appquery.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening DB: %v", err)
	}
	if err := db.CreateSchema(database); err != nil {
		t.Fatalf("creating schema: %v", err)
	}
	seedE2EData(t, database)

	// Run full pipeline
	if err := ResolveCitations(database); err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}
	if err := BuildLineMappings(database); err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}
	if err := BuildFTS(database); err != nil {
		t.Fatalf("BuildFTS: %v", err)
	}
	if err := PopulateAttributions(database); err != nil {
		t.Fatalf("PopulateAttributions: %v", err)
	}
	return database
}

// TestAppQuery_LexiconWordLookup simulates the primary app use case:
// user looks up a word → sees senses → sees citations resolved to actual text lines.
func TestAppQuery_LexiconWordLookup(t *testing.T) {
	database := setupFullPipelineDB(t)
	defer database.Close()

	// The exact query the SvelteKit app would use for a lexicon page:
	// Given a word, get every sense with its citations resolved to text lines.
	rows, err := database.Query(`
		SELECT
			le.key,
			le.full_text,
			ls.sense_number,
			ls.definition_text,
			lc.raw_bibl,
			lc.work_abbrev,
			lc.act,
			lc.scene,
			lc.line,
			cm.match_type,
			cm.confidence,
			tl.content,
			tl.line_number,
			e.short_code AS edition,
			w.title AS work_title
		FROM lexicon_entries le
		JOIN lexicon_senses ls ON ls.entry_id = le.id
		JOIN lexicon_citations lc ON lc.sense_id = ls.id
		JOIN citation_matches cm ON cm.citation_id = lc.id
		JOIN text_lines tl ON tl.id = cm.text_line_id
		JOIN editions e ON e.id = tl.edition_id
		JOIN works w ON w.id = tl.work_id
		WHERE le.key = ?
		ORDER BY ls.sense_number, lc.id, e.short_code`, "question")
	if err != nil {
		t.Fatalf("lexicon lookup query: %v", err)
	}
	defer rows.Close()

	type result struct {
		key, fullText                   string
		senseNum                        int
		definition, rawBibl, workAbbrev string
		act, scene, line                int
		matchType                       string
		confidence                      float64
		content                         string
		lineNumber                      int
		edition, workTitle              string
	}

	var results []result
	for rows.Next() {
		var r result
		if err := rows.Scan(&r.key, &r.fullText, &r.senseNum, &r.definition,
			&r.rawBibl, &r.workAbbrev, &r.act, &r.scene, &r.line,
			&r.matchType, &r.confidence, &r.content, &r.lineNumber,
			&r.edition, &r.workTitle); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		t.Fatal("lexicon lookup returned 0 rows — the app would show nothing")
	}

	// Verify we get results from both editions
	editions := map[string]bool{}
	for _, r := range results {
		editions[r.edition] = true
	}
	if !editions["oss_globe"] {
		t.Error("missing OSS edition in lexicon lookup")
	}
	if !editions["se_modern"] {
		t.Error("missing SE edition in lexicon lookup")
	}

	// Verify we get results from both works (Hamlet + Sonnets)
	works := map[string]bool{}
	for _, r := range results {
		works[r.workTitle] = true
	}
	if len(works) < 2 {
		t.Errorf("expected citations across multiple works, got %d: %v", len(works), works)
	}

	t.Logf("Lexicon lookup for 'question': %d results across %d editions and %d works",
		len(results), len(editions), len(works))
	for _, r := range results {
		t.Logf("  sense %d → %s %d.%d.%d → %s (%.2f) → \"%s\" [%s]",
			r.senseNum, r.workAbbrev, r.act, r.scene, r.line,
			r.matchType, r.confidence, r.content, r.edition)
	}
}

// TestAppQuery_TextReadingCursorPagination simulates reading a scene with cursor pagination.
// The app loads lines in pages: WHERE id > :cursor ORDER BY id LIMIT :limit.
func TestAppQuery_TextReadingCursorPagination(t *testing.T) {
	database := setupFullPipelineDB(t)
	defer database.Close()

	// Get the OSS edition ID and Hamlet work ID
	var editionID, workID int64
	database.QueryRow("SELECT id FROM editions WHERE short_code = 'oss_globe'").Scan(&editionID)
	database.QueryRow("SELECT id FROM works WHERE short_title = 'Hamlet'").Scan(&workID)

	// Page 1: first 3 lines (cursor = 0)
	rows, err := database.Query(`
		SELECT id, line_number, char_name, content
		FROM text_lines
		WHERE work_id = ? AND edition_id = ? AND act = 3 AND scene = 1 AND id > ?
		ORDER BY id ASC
		LIMIT ?`, workID, editionID, 0, 3)
	if err != nil {
		t.Fatalf("page 1 query: %v", err)
	}

	type line struct {
		id, lineNum int64
		charName    sql.NullString
		content     string
	}

	var page1 []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.id, &l.lineNum, &l.charName, &l.content); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		page1 = append(page1, l)
	}
	rows.Close()

	if len(page1) != 3 {
		t.Fatalf("page 1 expected 3 lines, got %d", len(page1))
	}

	// Verify lines are sequential
	for i := 1; i < len(page1); i++ {
		if page1[i].id <= page1[i-1].id {
			t.Errorf("page 1 lines not in id order: %d <= %d", page1[i].id, page1[i-1].id)
		}
	}

	// Page 2: next lines using last cursor
	lastCursor := page1[len(page1)-1].id
	rows2, err := database.Query(`
		SELECT id, line_number, char_name, content
		FROM text_lines
		WHERE work_id = ? AND edition_id = ? AND act = 3 AND scene = 1 AND id > ?
		ORDER BY id ASC
		LIMIT ?`, workID, editionID, lastCursor, 3)
	if err != nil {
		t.Fatalf("page 2 query: %v", err)
	}

	var page2 []line
	for rows2.Next() {
		var l line
		if err := rows2.Scan(&l.id, &l.lineNum, &l.charName, &l.content); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		page2 = append(page2, l)
	}
	rows2.Close()

	if len(page2) != 2 {
		t.Fatalf("page 2 expected 2 remaining lines, got %d", len(page2))
	}

	// Verify no overlap between pages
	if page2[0].id <= lastCursor {
		t.Errorf("page 2 first id %d should be > cursor %d", page2[0].id, lastCursor)
	}

	// Verify total coverage = 5 lines
	allLines := append(page1, page2...)
	if len(allLines) != 5 {
		t.Errorf("total lines across pages: %d, expected 5", len(allLines))
	}

	t.Logf("Cursor pagination: page1=%d lines (cursor→%d), page2=%d lines",
		len(page1), lastCursor, len(page2))
}

// TestAppQuery_EditionComparison simulates the side-by-side edition view.
// Join line_mappings to both editions' text_lines for a given scene.
func TestAppQuery_EditionComparison(t *testing.T) {
	database := setupFullPipelineDB(t)
	defer database.Close()

	// The exact query for a side-by-side comparison view:
	rows, err := database.Query(`
		SELECT
			lm.align_order,
			lm.match_type,
			lm.similarity,
			tla.content AS oss_text,
			tla.line_number AS oss_line,
			tlb.content AS se_text,
			tlb.line_number AS se_line
		FROM line_mappings lm
		JOIN works w ON w.id = lm.work_id
		LEFT JOIN text_lines tla ON tla.id = lm.line_a_id
		LEFT JOIN text_lines tlb ON tlb.id = lm.line_b_id
		WHERE w.short_title = 'Hamlet' AND lm.act = 3 AND lm.scene = 1
		ORDER BY lm.align_order`)
	if err != nil {
		t.Fatalf("edition comparison query: %v", err)
	}
	defer rows.Close()

	type compRow struct {
		alignOrder      int
		matchType       string
		similarity      float64
		ossText, seText sql.NullString
		ossLine, seLine sql.NullInt64
	}

	var results []compRow
	for rows.Next() {
		var r compRow
		if err := rows.Scan(&r.alignOrder, &r.matchType, &r.similarity,
			&r.ossText, &r.ossLine, &r.seText, &r.seLine); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		t.Fatal("edition comparison returned 0 rows")
	}

	// We have 5 Hamlet lines: 4 identical + 1 different
	if len(results) != 5 {
		t.Errorf("expected 5 comparison rows for Hamlet 3.1, got %d", len(results))
	}

	// Verify align_order is sequential
	for i, r := range results {
		if r.alignOrder != i+1 {
			t.Errorf("row %d: expected align_order %d, got %d", i, i+1, r.alignOrder)
		}
	}

	// Verify identical lines have similarity 1.0
	perfectAligned := 0
	modified := 0
	for _, r := range results {
		if r.similarity == 1.0 && r.matchType == "aligned" {
			perfectAligned++
		}
		if r.matchType == "modified" || (r.matchType == "aligned" && r.similarity < 1.0) {
			modified++
		}
	}

	// Lines 1-4 are identical text → perfect aligned
	// Line 5 differs: "To die, to sleep—" vs "To die— perchance to dream."
	if perfectAligned < 4 {
		t.Errorf("expected at least 4 perfectly aligned lines, got %d", perfectAligned)
	}
	if modified < 1 {
		t.Errorf("expected at least 1 modified/different line (line 5), got %d", modified)
	}

	// Verify both sides have text
	for i, r := range results {
		if !r.ossText.Valid || r.ossText.String == "" {
			t.Errorf("row %d: missing OSS text", i)
		}
		if !r.seText.Valid || r.seText.String == "" {
			t.Errorf("row %d: missing SE text", i)
		}
	}

	t.Logf("Edition comparison: %d rows (%d perfect, %d different)", len(results), perfectAligned, modified)
	for _, r := range results {
		t.Logf("  [%d] %s (%.2f): \"%s\" ↔ \"%s\"",
			r.alignOrder, r.matchType, r.similarity,
			r.ossText.String, r.seText.String)
	}
}

// TestAppQuery_ReverseCitationLookup simulates clicking a text line and seeing
// which lexicon entries reference it. The "what does this word mean?" feature.
func TestAppQuery_ReverseCitationLookup(t *testing.T) {
	database := setupFullPipelineDB(t)
	defer database.Close()

	// Find Hamlet line 1 (OSS edition) — "To be, or not to be..."
	var targetLineID int64
	err := database.QueryRow(`
		SELECT tl.id FROM text_lines tl
		JOIN works w ON w.id = tl.work_id
		JOIN editions e ON e.id = tl.edition_id
		WHERE w.short_title = 'Hamlet' AND e.short_code = 'oss_globe'
			AND tl.act = 3 AND tl.scene = 1 AND tl.line_number = 1`).Scan(&targetLineID)
	if err != nil {
		t.Fatalf("finding target line: %v", err)
	}

	// Reverse lookup: given a text_line, find all lexicon entries that cite it
	rows, err := database.Query(`
		SELECT
			le.key,
			ls.definition_text,
			lc.raw_bibl,
			cm.match_type,
			cm.confidence
		FROM citation_matches cm
		JOIN lexicon_citations lc ON lc.id = cm.citation_id
		JOIN lexicon_senses ls ON ls.id = lc.sense_id
		JOIN lexicon_entries le ON le.id = ls.entry_id
		WHERE cm.text_line_id = ?
		ORDER BY le.key`, targetLineID)
	if err != nil {
		t.Fatalf("reverse lookup query: %v", err)
	}
	defer rows.Close()

	type reverseResult struct {
		key, definition, rawBibl, matchType string
		confidence                          float64
	}

	var results []reverseResult
	for rows.Next() {
		var r reverseResult
		if err := rows.Scan(&r.key, &r.definition, &r.rawBibl, &r.matchType, &r.confidence); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		t.Fatal("reverse citation lookup returned 0 rows — line should be cited by 'question'")
	}

	// Should find "question" citing this line
	found := false
	for _, r := range results {
		if r.key == "question" {
			found = true
			if r.confidence < 0.8 {
				t.Errorf("reverse lookup confidence for 'question' too low: %.2f", r.confidence)
			}
			t.Logf("Reverse lookup: line %d → %s [%s] via %s (%.2f)",
				targetLineID, r.key, r.definition, r.matchType, r.confidence)
		}
	}

	if !found {
		t.Error("reverse lookup did not find 'question' citing Hamlet 3.1.1")
	}
}

// TestAppQuery_FTSToStructuredData simulates searching text and joining back to metadata.
// The app search feature: user types a query, gets structured results with work/edition context.
func TestAppQuery_FTSToStructuredData(t *testing.T) {
	database := setupFullPipelineDB(t)
	defer database.Close()

	// Text search: find lines containing "slings" with full metadata.
	// FTS5 content table uses content_rowid='id', so fts.rowid = text_lines.id.
	rows, err := database.Query(`
		SELECT
			tl.id,
			tl.content,
			tl.act,
			tl.scene,
			tl.line_number,
			COALESCE(tl.char_name, '') AS char_name,
			w.title AS work_title,
			w.work_type,
			e.short_code AS edition
		FROM text_fts fts
		JOIN text_lines tl ON tl.id = fts.rowid
		JOIN works w ON w.id = tl.work_id
		JOIN editions e ON e.id = tl.edition_id
		WHERE text_fts MATCH 'slings'
		ORDER BY w.title, e.short_code`)
	if err != nil {
		t.Fatalf("FTS structured query: %v", err)
	}
	defer rows.Close()

	type searchResult struct {
		id                          int64
		content                     string
		act, scene, lineNum         int
		charName                    string
		workTitle, workType, edition string
	}

	var results []searchResult
	for rows.Next() {
		var r searchResult
		if err := rows.Scan(&r.id, &r.content, &r.act, &r.scene, &r.lineNum,
			&r.charName, &r.workTitle, &r.workType, &r.edition); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		t.Fatal("FTS search for 'slings' returned 0 structured results")
	}

	// Should find results in both editions
	editions := map[string]bool{}
	for _, r := range results {
		editions[r.edition] = true
	}
	if len(editions) < 2 {
		t.Errorf("expected FTS results across 2 editions, got %d", len(editions))
	}

	// Every result should have full metadata
	for _, r := range results {
		if r.workTitle == "" {
			t.Error("FTS result missing work title")
		}
		if r.edition == "" {
			t.Error("FTS result missing edition")
		}
		if r.content == "" {
			t.Error("FTS result missing content")
		}
		t.Logf("FTS result: \"%s\" — %s %d.%d.%d [%s] (%s)",
			r.content, r.workTitle, r.act, r.scene, r.lineNum, r.edition, r.charName)
	}
}

// TestAppQuery_LexiconBrowseByLetter simulates browsing the lexicon alphabetically
// with cursor pagination (the lexicon index page).
func TestAppQuery_LexiconBrowseByLetter(t *testing.T) {
	database := setupFullPipelineDB(t)
	defer database.Close()

	// Browse letter Q with cursor pagination
	rows, err := database.Query(`
		SELECT id, key, full_text
		FROM lexicon_entries
		WHERE letter = ? AND id > ?
		ORDER BY id ASC
		LIMIT ?`, "Q", 0, 10)
	if err != nil {
		t.Fatalf("lexicon browse query: %v", err)
	}
	defer rows.Close()

	type entry struct {
		id       int64
		key      string
		fullText string
	}

	var results []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.key, &e.fullText); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		results = append(results, e)
	}

	if len(results) == 0 {
		t.Fatal("lexicon browse for letter Q returned 0 entries")
	}

	// Should find "question"
	found := false
	for _, e := range results {
		if e.key == "question" {
			found = true
			if e.fullText == "" {
				t.Error("lexicon entry for 'question' has empty full_text")
			}
		}
	}
	if !found {
		t.Error("browsing letter Q did not find 'question'")
	}

	t.Logf("Lexicon browse Q: %d entries", len(results))
}

// TestAppQuery_AttributionDisplay simulates loading attribution requirements
// for displaying proper credits on a page.
func TestAppQuery_AttributionDisplay(t *testing.T) {
	database := setupFullPipelineDB(t)
	defer database.Close()

	// The app needs to know what attributions to display. Query all required ones.
	rows, err := database.Query(`
		SELECT
			s.name AS source_name,
			s.short_code,
			a.required,
			a.attribution_text,
			a.display_format,
			a.display_context,
			a.requires_link_back,
			COALESCE(a.link_back_url, '') AS link_back_url,
			a.share_alike_required
		FROM attributions a
		JOIN sources s ON s.id = a.source_id
		ORDER BY a.display_priority DESC, a.required DESC`)
	if err != nil {
		t.Fatalf("attribution query: %v", err)
	}
	defer rows.Close()

	type attr struct {
		sourceName, shortCode      string
		required                   bool
		text, format, context      string
		linkBack                   bool
		linkURL                    string
		shareAlike                 bool
	}

	var results []attr
	for rows.Next() {
		var a attr
		if err := rows.Scan(&a.sourceName, &a.shortCode, &a.required,
			&a.text, &a.format, &a.context, &a.linkBack, &a.linkURL, &a.shareAlike); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		results = append(results, a)
	}

	if len(results) == 0 {
		t.Fatal("attribution query returned 0 rows — app has no credits to display")
	}

	// Verify Perseus is required with share-alike
	perseusFound := false
	for _, a := range results {
		if a.shortCode == "perseus_schmidt" {
			perseusFound = true
			if !a.required {
				t.Error("Perseus should be required")
			}
			if !a.shareAlike {
				t.Error("Perseus should require share-alike")
			}
			if a.text == "" {
				t.Error("Perseus attribution text is empty")
			}
		}
	}
	if !perseusFound {
		t.Error("Perseus not found in attributions")
	}

	t.Logf("Attributions: %d total", len(results))
	for _, a := range results {
		req := "voluntary"
		if a.required {
			req = "REQUIRED"
		}
		t.Logf("  [%s] %s — %s (%s/%s)", req, a.sourceName, a.text, a.format, a.context)
	}
}
