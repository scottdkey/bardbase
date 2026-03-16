// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"strings"
)

// stepBanner prints a formatted step header for pipeline progress output.
// Example: stepBanner("STEP 1: Import OSS Data") prints:
//
//	============================================================
//	STEP 1: Import OSS Data
//	============================================================
func stepBanner(title string) {
	bar := "=" + strings.Repeat("=", 59)
	fmt.Println()
	fmt.Println(bar)
	fmt.Println(title)
	fmt.Println(bar)
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
