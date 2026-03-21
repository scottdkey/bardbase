// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// stepCounter tracks the current pipeline step number for auto-incrementing banners.
var stepCounter int

// ResetStepCounter resets the step counter to 0. Called at the start of each
// pipeline run so that single-step (-step) runs start at 1.
func ResetStepCounter() { stepCounter = 0 }

// stepBanner prints a formatted, auto-numbered step header.
// Example: stepBanner("Import OSS Data") prints:
//
//	============================================================
//	STEP 1: Import OSS Data
//	============================================================
func stepBanner(title string) {
	stepCounter++
	printBar()
	fmt.Printf("STEP %d: %s\n", stepCounter, title)
	printBar()
}

// printBar prints a single 60-character line of "=" characters.
// Used as a header/footer for pipeline step output.
func printBar() {
	fmt.Println(strings.Repeat("=", 60))
}

// ─── SQL helpers ─────────────────────────────────────────────────────────────

// nilIfEmpty returns nil if s is empty, otherwise returns s.
// Used when inserting optional TEXT columns that should be NULL not "".
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nilIfZero returns nil if n is zero, otherwise returns n.
// Used when inserting optional INTEGER columns that should be NULL not 0.
func nilIfZero(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

// boolToInt converts a bool to SQLite's 0/1 integer representation.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ─── Character / text helpers ────────────────────────────────────────────────

// lookupCharacter resolves a character name (or abbreviation) to its DB id
// for the given work. Returns nil when not found.
func lookupCharacter(database *sql.DB, workID int64, charName string) any {
	var id int64
	err := database.QueryRow(
		"SELECT id FROM characters WHERE work_id = ? AND UPPER(name) = UPPER(?)",
		workID, charName).Scan(&id)
	if err != nil {
		err = database.QueryRow(
			"SELECT id FROM characters WHERE work_id = ? AND UPPER(abbrev) = UPPER(?)",
			workID, charName).Scan(&id)
	}
	if err != nil || id == 0 {
		return nil
	}
	return id
}

// countWords returns the number of whitespace-delimited words in s.
func countWords(s string) int {
	return len(strings.Fields(s))
}

// ─── Works helpers ───────────────────────────────────────────────────────────

// workInfo holds minimal work metadata for import lookups.
type workInfo struct {
	ID    int64
	Title string
}

// buildWorksMap queries the works table and returns a map from oss_id → workInfo.
// Used by multiple importers that need to match source records to DB works by oss_id.
func buildWorksMap(database *sql.DB) (map[string]workInfo, error) {
	return buildWorksMapByColumn(database, "oss_id")
}

// buildWorksMapByColumn queries the works table and returns a map from the
// specified column → workInfo. Rows with NULL or empty values are skipped.
func buildWorksMapByColumn(database *sql.DB, keyColumn string) (map[string]workInfo, error) {
	query := fmt.Sprintf(
		"SELECT id, %s, title FROM works WHERE %s IS NOT NULL AND %s != ''",
		keyColumn, keyColumn, keyColumn)
	rows, err := database.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying works by %s: %w", keyColumn, err)
	}
	defer rows.Close()

	m := make(map[string]workInfo)
	for rows.Next() {
		var id int64
		var key, title string
		rows.Scan(&id, &key, &title)
		m[key] = workInfo{ID: id, Title: title}
	}
	return m, nil
}

// ─── Parallel helpers ────────────────────────────────────────────────────────

// workerCount returns the number of parallel workers to use.
// Reserves 6 cores for the OS and other tasks to keep the system responsive.
func workerCount() int {
	w := runtime.NumCPU() - 6
	if w < 1 {
		w = 1
	}
	return w
}

// parallelProcess applies fn to each item using up to workerCount() goroutines
// (capped at len(items)). Results are returned in unspecified order; callers that
// need ordered output should key results by a field in O.
func parallelProcess[I, O any](items []I, fn func(I) O) []O {
	if len(items) == 0 {
		return nil
	}
	workers := max(1, min(workerCount(), len(items)))

	ch := make(chan I, len(items))
	for _, item := range items {
		ch <- item
	}
	close(ch)

	resultCh := make(chan O, len(items))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range ch {
				resultCh <- fn(item)
			}
		}()
	}
	go func() { wg.Wait(); close(resultCh) }()

	results := make([]O, 0, len(items))
	for r := range resultCh {
		results = append(results, r)
	}
	return results
}

// ─── Text-line helpers ───────────────────────────────────────────────────────

// contentType returns the text_lines content_type for a line.
// Stage directions return "stage_direction"; all other lines return "speech".
func contentType(isStageDirection bool) string {
	if isStageDirection {
		return "stage_direction"
	}
	return "speech"
}

// collectXMLFiles returns sorted .xml filenames from a directory listing.
func collectXMLFiles(entries []os.DirEntry) []string {
	var files []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".xml" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files
}

// textLinesInsertSQL is the shared INSERT statement used by all play importers.
const textLinesInsertSQL = `
	INSERT INTO text_lines (work_id, edition_id, act, scene, line_number,
		character_id, char_name, content, content_type, word_count)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// insertTextDivisions tallies line counts per (act, scene) from actScenes and
// inserts them into text_divisions within tx. Callers build actScenes from their
// line slice: [][2]int{{line.Act, line.Scene}, ...}.
func insertTextDivisions(tx *sql.Tx, workID, editionID int64, actScenes [][2]int) {
	counts := make(map[[2]int]int, len(actScenes))
	for _, as := range actScenes {
		counts[as]++
	}
	for key, count := range counts {
		tx.Exec(`INSERT OR IGNORE INTO text_divisions (work_id, edition_id, act, scene, line_count)
			VALUES (?, ?, ?, ?, ?)`,
			workID, editionID, key[0], key[1], count)
	}
}

// clearWorkEditionData deletes all text_lines and text_divisions for a given
// work+edition pair. Called before re-importing to keep imports idempotent.
func clearWorkEditionData(database *sql.DB, workID, editionID int64) {
	database.Exec("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", workID, editionID)
	database.Exec("DELETE FROM text_divisions WHERE work_id = ? AND edition_id = ?", workID, editionID)
}

// cachedLookupCharacter resolves a character name to its DB id, caching the
// result in cache to avoid redundant queries within a single import loop.
// Returns nil when charName is empty or the character is not found.
func cachedLookupCharacter(database *sql.DB, workID int64, charName string, cache map[string]any) any {
	if charName == "" {
		return nil
	}
	if cached, ok := cache[charName]; ok {
		return cached
	}
	id := lookupCharacter(database, workID, charName)
	cache[charName] = id
	return id
}

// expandCharacterName resolves an abbreviated character name (e.g., "Cal.",
// "Pros.") to its full form (e.g., "Caliban", "Prospero") using the characters
// table. Returns the original name if no match is found.
func expandCharacterName(database *sql.DB, workID int64, abbrev string) string {
	if abbrev == "" {
		return abbrev
	}
	// Try exact match first
	var name string
	err := database.QueryRow(
		"SELECT name FROM characters WHERE work_id = ? AND UPPER(name) = UPPER(?)",
		workID, abbrev).Scan(&name)
	if err == nil {
		return name
	}

	// Try abbreviation match (OSS uses UPPER abbreviations like "CALIBAN")
	err = database.QueryRow(
		"SELECT name FROM characters WHERE work_id = ? AND UPPER(abbrev) = UPPER(?)",
		workID, abbrev).Scan(&name)
	if err == nil {
		return name
	}

	// Try prefix match: strip trailing "." and match against the start of name
	prefix := strings.TrimRight(abbrev, ".")
	if prefix != "" {
		err = database.QueryRow(
			"SELECT name FROM characters WHERE work_id = ? AND LOWER(name) LIKE LOWER(?) || '%' LIMIT 1",
			workID, prefix).Scan(&name)
		if err == nil {
			return name
		}
	}

	return abbrev
}

// cachedExpandCharName expands abbreviated character names with caching.
func cachedExpandCharName(database *sql.DB, workID int64, abbrev string, cache map[string]string) string {
	if abbrev == "" {
		return ""
	}
	if cached, ok := cache[abbrev]; ok {
		return cached
	}
	full := expandCharacterName(database, workID, abbrev)
	cache[abbrev] = full
	return full
}

// ─── Reference-entry helpers ────────────────────────────────────────────────

// referenceEntry holds a parsed headword, letter category, and raw text for
// any reference source (Abbott, Bartlett, Henley-Farmer, Onions).
type referenceEntry struct {
	headword string
	letter   string
	rawText  string
}

// headwordLetter returns the uppercase first letter of hw, or "?" if hw
// contains no letters. Used to categorise reference entries by letter.
func headwordLetter(hw string) string {
	for _, r := range hw {
		if unicode.IsLetter(r) {
			return strings.ToUpper(string(r))
		}
	}
	return "?"
}

// isAllCaps returns true if all letter runes in s are uppercase.
// Short lines that are all-uppercase are treated as section headers.
func isAllCaps(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if unicode.IsLower(r) {
				return false
			}
		}
	}
	return hasLetter
}

// insertReferenceEntries bulk-inserts reference entries into the
// reference_entries table within a single transaction. Returns the number
// of rows inserted.
func insertReferenceEntries(database *sql.DB, srcID int64, entries []referenceEntry) (int, error) {
	tx, err := database.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO reference_entries (source_id, headword, letter, raw_text)
		VALUES (?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, e := range entries {
		if _, err := stmt.Exec(srcID, e.headword, e.letter, e.rawText); err != nil {
			continue
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing transaction: %w", err)
	}
	return inserted, nil
}

// ─── Text-line query helpers ────────────────────────────────────────────────

// loadTextLinesAll is like loadTextLines but does NOT filter out rows with
// NULL line_number. Used for headword search in prologue/chorus scenes where
// Perseus stores content without Globe line numbers.
func loadTextLinesAll(database *sql.DB, where string, args ...any) ([]textLineRow, error) {
	query := fmt.Sprintf(
		`SELECT id, content, COALESCE(line_number, 0), edition_id
		 FROM text_lines
		 WHERE %s
		 ORDER BY edition_id, line_number, id`, where)

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []textLineRow
	for rows.Next() {
		var tl textLineRow
		rows.Scan(&tl.ID, &tl.Content, &tl.LineNumber, &tl.EditionID)
		lines = append(lines, tl)
	}
	return lines, nil
}

// loadTextLines queries text_lines with a parameterized WHERE clause.
// Returns all matching rows with non-null line numbers, ordered by edition
// and line number. This is the single query path used by all citation
// resolution strategies (play, sonnet, poem).
//
// The where parameter should NOT include "AND line_number IS NOT NULL" —
// that is added automatically.
//
// Example:
//
//	lines, err := loadTextLines(db, "work_id = ? AND act = ? AND scene = ?", workID, act, scene)
func loadTextLines(database *sql.DB, where string, args ...any) ([]textLineRow, error) {
	query := fmt.Sprintf(
		`SELECT id, content, COALESCE(line_number, 0), edition_id
		 FROM text_lines
		 WHERE %s AND line_number IS NOT NULL
		 ORDER BY edition_id, line_number`, where)

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []textLineRow
	for rows.Next() {
		var tl textLineRow
		rows.Scan(&tl.ID, &tl.Content, &tl.LineNumber, &tl.EditionID)
		lines = append(lines, tl)
	}
	return lines, nil
}
