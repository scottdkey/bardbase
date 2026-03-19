// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// ResolveReferenceCitations parses embedded play citations from reference_entries
// (currently Onions), inserts them into reference_citations, and matches each
// to text_lines (primarily OSS/Globe edition, then propagated to others).
//
// Steps:
//   1. Build abbrev→work_id map from works table + OnionsAbbrevs translation.
//   2. Load all reference_entries.
//   3. Parse citations from each entry's raw_text.
//   4. Bulk-insert into reference_citations.
//   5. Match line numbers against text_lines; insert reference_citation_matches.
//   6. Propagate matches to other editions via line_mappings.
func ResolveReferenceCitations(database *sql.DB) error {
	stepBanner("STEP 10: Resolve Reference Citations → Text Lines")
	start := time.Now()

	// Clear previous run for idempotent rebuild.
	database.Exec("DELETE FROM reference_citation_matches")
	database.Exec("DELETE FROM reference_citations")

	// --- Step 1: build abbrev → work_id ---
	abbrevToWorkID, err := buildAbbrevMap(database)
	if err != nil {
		return fmt.Errorf("building abbrev map: %w", err)
	}
	fmt.Printf("  Abbreviation map: %d entries\n", len(abbrevToWorkID))

	// --- Step 2: load reference entries ---
	type entryRow struct {
		ID       int64
		SourceID int64
		RawText  string
	}
	rows, err := database.Query(`SELECT id, source_id, raw_text FROM reference_entries`)
	if err != nil {
		return fmt.Errorf("loading reference entries: %w", err)
	}
	var entries []entryRow
	for rows.Next() {
		var e entryRow
		rows.Scan(&e.ID, &e.SourceID, &e.RawText)
		entries = append(entries, e)
	}
	rows.Close()

	fmt.Printf("  Reference entries: %d\n", len(entries))
	if len(entries) == 0 {
		fmt.Println("  No reference entries — run onions step first")
		return nil
	}

	// --- Step 3+4: parse citations and bulk-insert ---
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	citStmt, err := tx.Prepare(`
		INSERT INTO reference_citations (entry_id, source_id, work_id, work_abbrev, act, scene, line)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("preparing citation insert: %w", err)
	}

	totalCitations := 0
	unresolved := 0
	for _, e := range entries {
		cits := parser.ParseOnionsCitations(e.RawText)
		for _, c := range cits {
			// Translate Onions abbreviation → Schmidt abbreviation → work_id.
			schmidtAbbrev := resolveOnionsAbbrev(c.WorkAbbrev)
			workID, ok := abbrevToWorkID[schmidtAbbrev]
			var workIDArg any
			if ok {
				workIDArg = workID
			} else {
				workIDArg = nil
				unresolved++
			}

			var actArg, sceneArg, lineArg any
			if c.Act != nil {
				actArg = *c.Act
			}
			if c.Scene != nil {
				sceneArg = *c.Scene
			}
			if c.Line != nil {
				lineArg = *c.Line
			}

			citStmt.Exec(e.ID, e.SourceID, workIDArg, c.WorkAbbrev, actArg, sceneArg, lineArg)
			totalCitations++
		}
	}
	citStmt.Close()
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing citations: %w", err)
	}

	fmt.Printf("  Citations extracted: %d (%d unresolved abbrevs)\n", totalCitations, unresolved)

	if totalCitations == 0 {
		fmt.Println("  No citations to match")
		return nil
	}

	// --- Step 5: match citations to text_lines ---
	matched := matchReferenceCitations(database)

	// --- Step 6: propagate to other editions ---
	propagated := propagateReferenceCitations(database)

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "ref_citations", "resolve_complete",
		fmt.Sprintf("extracted=%d matched=%d propagated=%d", totalCitations, matched, propagated),
		matched+propagated, elapsed)

	fmt.Printf("  ✓ %d citations → %d matches + %d propagated in %.1fs\n",
		totalCitations, matched, propagated, elapsed)
	return nil
}

// buildAbbrevMap returns a map from Schmidt abbreviation (with and without
// trailing period) to work_id, by querying works.schmidt_abbrev.
func buildAbbrevMap(database *sql.DB) (map[string]int64, error) {
	rows, err := database.Query(`SELECT id, schmidt_abbrev FROM works WHERE schmidt_abbrev IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]int64)
	for rows.Next() {
		var id int64
		var abbrev string
		rows.Scan(&id, &abbrev)
		m[abbrev] = id
		// Also index without trailing period for flexibility.
		if trimmed, ok2 := strings.CutSuffix(abbrev, "."); ok2 {
			m[trimmed] = id
		}
	}
	return m, nil
}

// resolveOnionsAbbrev translates an Onions 1911 abbreviation to the
// corresponding Schmidt abbreviation. Falls through to the raw abbreviation
// for the majority that are identical in both systems.
func resolveOnionsAbbrev(abbrev string) string {
	if mapped, ok := constants.OnionsAbbrevs[abbrev]; ok {
		return mapped
	}
	return abbrev
}

// matchReferenceCitations loads each reference_citation with a known work_id
// and attempts to find the matching text_line by line number.
// Returns the number of matches inserted.
func matchReferenceCitations(database *sql.DB) int {
	// Load citations that resolved to a work.
	type refCit struct {
		ID     int64
		WorkID int64
		Act    *int
		Scene  *int
		Line   *int
	}

	citRows, err := database.Query(`
		SELECT id, work_id, act, scene, line
		FROM reference_citations
		WHERE work_id IS NOT NULL AND line IS NOT NULL`)
	if err != nil {
		return 0
	}
	var cits []refCit
	for citRows.Next() {
		var c refCit
		var act, scene, line sql.NullInt64
		citRows.Scan(&c.ID, &c.WorkID, &act, &scene, &line)
		if act.Valid {
			v := int(act.Int64)
			c.Act = &v
		}
		if scene.Valid {
			v := int(scene.Int64)
			c.Scene = &v
		}
		if line.Valid {
			v := int(line.Int64)
			c.Line = &v
		}
		cits = append(cits, c)
	}
	citRows.Close()

	if len(cits) == 0 {
		return 0
	}

	// Group by (work_id, act, scene) to batch text_line loads.
	type sceneKey struct {
		WorkID int64
		Act    int // 0 when nil
		Scene  int // 0 when nil (poems)
	}
	type citGroup struct {
		cits  []refCit
		lines []textLineRow // loaded once per group
	}
	groups := make(map[sceneKey]*citGroup)
	for _, c := range cits {
		act := 0
		if c.Act != nil {
			act = *c.Act
		}
		scene := 0
		if c.Scene != nil {
			scene = *c.Scene
		}
		k := sceneKey{c.WorkID, act, scene}
		if groups[k] == nil {
			groups[k] = &citGroup{}
		}
		groups[k].cits = append(groups[k].cits, c)
	}

	// Load text lines per group.
	for k, g := range groups {
		var lines []textLineRow
		var err error
		if k.Act > 0 && k.Scene > 0 {
			lines, err = loadTextLines(database,
				"work_id = ? AND act = ? AND scene = ?",
				k.WorkID, k.Act, k.Scene)
		} else if k.Scene > 0 {
			// Sonnet: scene = sonnet number, no act.
			lines, err = loadTextLines(database,
				"work_id = ? AND scene = ?",
				k.WorkID, k.Scene)
		} else {
			// Poem: no act/scene, just line number.
			lines, err = loadTextLines(database,
				"work_id = ?",
				k.WorkID)
		}
		if err != nil || len(lines) == 0 {
			// Fallback: broaden to full act when scene yields nothing.
			if k.Act > 0 && k.Scene > 0 {
				lines, err = loadTextLines(database,
					"work_id = ? AND act = ?",
					k.WorkID, k.Act)
			}
		}
		if err == nil {
			g.lines = lines
		}
	}

	// Match each citation by line number, insert results.
	tx, err := database.Begin()
	if err != nil {
		return 0
	}
	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO reference_citation_matches
			(ref_citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
		VALUES (?, ?, ?, 'line_number', ?, ?)`)
	if err != nil {
		tx.Rollback()
		return 0
	}

	matched := 0
	for _, g := range groups {
		if len(g.lines) == 0 {
			continue
		}
		// Split lines by edition for per-edition matching.
		linesByEdition := make(map[int64][]textLineRow)
		for _, tl := range g.lines {
			linesByEdition[tl.EditionID] = append(linesByEdition[tl.EditionID], tl)
		}
		for _, c := range g.cits {
			if c.Line == nil {
				continue
			}
			for editionID, edLines := range linesByEdition {
				if tl := findLineByNumber(edLines, *c.Line); tl != nil {
					stmt.Exec(c.ID, tl.ID, editionID, 1.0, tl.Content)
					matched++
				}
			}
		}
	}
	stmt.Close()
	tx.Commit()

	fmt.Printf("  Direct line matches: %d\n", matched)
	return matched
}

// findLineByNumber returns the textLineRow whose LineNumber equals target,
// or the closest one within ±10 if no exact match exists (confidence still 1.0
// since Globe line numbers are generally reliable). Returns nil if lines is empty.
func findLineByNumber(lines []textLineRow, target int) *textLineRow {
	var best *textLineRow
	bestDelta := -1
	for i := range lines {
		d := lines[i].LineNumber - target
		if d < 0 {
			d = -d
		}
		if best == nil || d < bestDelta {
			bestDelta = d
			best = &lines[i]
		}
		if d == 0 {
			break
		}
	}
	if bestDelta > 10 {
		return nil // too far off — skip rather than produce a bad match
	}
	return best
}

// propagateReferenceCitations uses line_mappings to spread existing
// reference_citation_matches to other editions. One round suffices for
// the typical 2-4 edition layout (OSS → SE, Perseus, Folio, Folger).
func propagateReferenceCitations(database *sql.DB) int {
	total := 0

	// Two passes: forward (line_a→line_b) and backward (line_b→line_a).
	queries := []string{
		`INSERT OR IGNORE INTO reference_citation_matches
			(ref_citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
		SELECT sub.ref_citation_id, sub.text_line_id, sub.edition_id, 'propagated',
		       sub.confidence, sub.matched_text
		FROM (
			SELECT
				rcm.ref_citation_id,
				lm.line_b_id AS text_line_id,
				lm.edition_b_id AS edition_id,
				rcm.confidence * CASE WHEN lm.similarity >= 0.2 THEN lm.similarity ELSE 0.3 END AS confidence,
				tl.content AS matched_text,
				ROW_NUMBER() OVER (
					PARTITION BY rcm.ref_citation_id, lm.edition_b_id
					ORDER BY rcm.confidence * lm.similarity DESC
				) AS rn
			FROM reference_citation_matches rcm
			JOIN line_mappings lm ON lm.line_a_id = rcm.text_line_id
				AND lm.line_b_id IS NOT NULL
			JOIN text_lines tl ON tl.id = lm.line_b_id
			WHERE NOT EXISTS (
				SELECT 1 FROM reference_citation_matches rcm2
				WHERE rcm2.ref_citation_id = rcm.ref_citation_id
				  AND rcm2.edition_id = lm.edition_b_id
			)
		) sub WHERE sub.rn = 1`,

		`INSERT OR IGNORE INTO reference_citation_matches
			(ref_citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
		SELECT sub.ref_citation_id, sub.text_line_id, sub.edition_id, 'propagated',
		       sub.confidence, sub.matched_text
		FROM (
			SELECT
				rcm.ref_citation_id,
				lm.line_a_id AS text_line_id,
				lm.edition_a_id AS edition_id,
				rcm.confidence * CASE WHEN lm.similarity >= 0.2 THEN lm.similarity ELSE 0.3 END AS confidence,
				tl.content AS matched_text,
				ROW_NUMBER() OVER (
					PARTITION BY rcm.ref_citation_id, lm.edition_a_id
					ORDER BY rcm.confidence * lm.similarity DESC
				) AS rn
			FROM reference_citation_matches rcm
			JOIN line_mappings lm ON lm.line_b_id = rcm.text_line_id
				AND lm.line_a_id IS NOT NULL
			JOIN text_lines tl ON tl.id = lm.line_a_id
			WHERE NOT EXISTS (
				SELECT 1 FROM reference_citation_matches rcm2
				WHERE rcm2.ref_citation_id = rcm.ref_citation_id
				  AND rcm2.edition_id = lm.edition_a_id
			)
		) sub WHERE sub.rn = 1`,
	}

	for _, q := range queries {
		res, err := database.Exec(q)
		if err == nil {
			n, _ := res.RowsAffected()
			total += int(n)
		}
	}

	fmt.Printf("  Propagated: %d\n", total)
	return total
}
