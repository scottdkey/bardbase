// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
)

// CreateExternalReferenceLines generates stub text_lines for non-Shakespeare
// works (lexicon_appendix, classical_reference, biblical_reference, apocrypha)
// so that lexicon citations referencing these works can resolve to a text_line.
//
// Each unique (work_id, act, scene, line) from lexicon_citations is inserted
// as a text_line with a placeholder edition (id=0) and content derived from
// the citation's raw_bibl and headword.
//
// This must run BEFORE ResolveCitations so that findBestMatch has lines to
// match against.
func CreateExternalReferenceLines(database *sql.DB) error {
	stepBanner("Create External Reference Lines")

	// Ensure a placeholder edition exists for external references.
	database.Exec(`INSERT OR IGNORE INTO editions (id, name, short_code, source_id, year)
		VALUES (0, 'External Reference', 'ext_ref', 1, 0)`)

	// Find all distinct locations cited in non-Shakespeare works.
	rows, err := database.Query(`
		SELECT DISTINCT lc.work_id, lc.act, lc.scene, lc.line,
			lc.raw_bibl, le.key
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		JOIN works w ON w.id = lc.work_id
		WHERE w.work_type IN ('lexicon_appendix', 'classical_reference', 'biblical_reference', 'apocrypha')
		  AND lc.line IS NOT NULL
		ORDER BY lc.work_id, lc.act, lc.scene, lc.line`)
	if err != nil {
		return fmt.Errorf("querying external citations: %w", err)
	}
	defer rows.Close()

	type extLine struct {
		WorkID  int64
		Act     sql.NullInt64
		Scene   sql.NullInt64
		Line    int
		RawBibl string
		Headword string
	}

	var lines []extLine
	for rows.Next() {
		var l extLine
		rows.Scan(&l.WorkID, &l.Act, &l.Scene, &l.Line, &l.RawBibl, &l.Headword)
		lines = append(lines, l)
	}

	if len(lines) == 0 {
		fmt.Println("  No external reference citations to create lines for")
		return nil
	}

	// Also handle citations without line numbers — create a line 1 stub.
	noLineRows, err := database.Query(`
		SELECT DISTINCT lc.work_id, lc.act, lc.scene,
			lc.raw_bibl, le.key
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		JOIN works w ON w.id = lc.work_id
		WHERE w.work_type IN ('lexicon_appendix', 'classical_reference', 'biblical_reference', 'apocrypha')
		  AND lc.line IS NULL
		ORDER BY lc.work_id, lc.act, lc.scene`)
	if err == nil {
		defer noLineRows.Close()
		for noLineRows.Next() {
			var l extLine
			noLineRows.Scan(&l.WorkID, &l.Act, &l.Scene, &l.RawBibl, &l.Headword)
			l.Line = 1 // stub line number
			lines = append(lines, l)
		}
	}

	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO text_lines
			(work_id, edition_id, act, scene, line_number, content, content_type, char_name)
		VALUES (?, 0, ?, ?, ?, ?, 'reference', NULL)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("preparing insert: %w", err)
	}

	inserted := 0
	seen := make(map[[4]int64]bool)
	for _, l := range lines {
		act := int64(0)
		if l.Act.Valid {
			act = l.Act.Int64
		}
		scene := int64(0)
		if l.Scene.Valid {
			scene = l.Scene.Int64
		}
		key := [4]int64{l.WorkID, act, scene, int64(l.Line)}
		if seen[key] {
			continue
		}
		seen[key] = true

		content := fmt.Sprintf("[%s] %s", l.RawBibl, l.Headword)
		_, err := stmt.Exec(l.WorkID, nullInt(act), nullInt(scene), l.Line, content)
		if err == nil {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	fmt.Printf("  Created %d stub text_lines for external references\n", inserted)
	return nil
}

func nullInt(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
