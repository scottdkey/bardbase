// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/scottdkey/heminge/projects/db-builder/internal/parser"
)

// stepBanner prints a formatted step header for pipeline progress output.
// Example: stepBanner("STEP 1: Import OSS Data") prints:
//
//	============================================================
//	STEP 1: Import OSS Data
//	============================================================
func stepBanner(title string) {
	printBar()
	fmt.Println(title)
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
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nilIfZero returns nil if n is zero, otherwise returns n.
// Used when inserting optional INTEGER columns that should be NULL not 0.
func nilIfZero(n int64) interface{} {
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
func lookupCharacter(database *sql.DB, workID int64, charName string) interface{} {
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
	rows, err := database.Query("SELECT id, oss_id, title FROM works")
	if err != nil {
		return nil, fmt.Errorf("querying works: %w", err)
	}
	defer rows.Close()

	m := make(map[string]workInfo)
	for rows.Next() {
		var id int64
		var ossID, title string
		rows.Scan(&id, &ossID, &title)
		m[ossID] = workInfo{ID: id, Title: title}
	}
	return m, nil
}

// ─── Text-line helpers ───────────────────────────────────────────────────────

// clearWorkEditionData deletes all text_lines and text_divisions for a given
// work+edition pair. Called before re-importing to keep imports idempotent.
func clearWorkEditionData(database *sql.DB, workID, editionID int64) {
	database.Exec("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", workID, editionID)
	database.Exec("DELETE FROM text_divisions WHERE work_id = ? AND edition_id = ?", workID, editionID)
}

// cachedLookupCharacter resolves a character name to its DB id, caching the
// result in cache to avoid redundant queries within a single import loop.
// Returns nil when charName is empty or the character is not found.
func cachedLookupCharacter(database *sql.DB, workID int64, charName string, cache map[string]interface{}) interface{} {
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

// queryAlignableLines runs a SELECT against text_lines and returns the results
// as []parser.AlignableLine. The where clause is appended directly; callers
// supply all bind args. Used by the three load*Lines helpers in mappings.go.
func queryAlignableLines(database *sql.DB, where string, args ...interface{}) []parser.AlignableLine {
	query := fmt.Sprintf(
		`SELECT id, content, COALESCE(line_number, 0)
		 FROM text_lines
		 WHERE %s
		 ORDER BY line_number, id`, where)

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var lines []parser.AlignableLine
	for rows.Next() {
		var l parser.AlignableLine
		rows.Scan(&l.ID, &l.Content, &l.LineNumber)
		lines = append(lines, l)
	}
	return lines
}

// loadTextLinesAll is like loadTextLines but does NOT filter out rows with
// NULL line_number. Used for headword search in prologue/chorus scenes where
// Perseus stores content without Globe line numbers.
func loadTextLinesAll(database *sql.DB, where string, args ...interface{}) ([]textLineRow, error) {
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
func loadTextLines(database *sql.DB, where string, args ...interface{}) ([]textLineRow, error) {
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
