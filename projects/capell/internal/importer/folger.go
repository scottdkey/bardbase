// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// ImportFolger parses the Folger Shakespeare TEIsimple XML files and inserts them
// into the database as a new edition ("Folger Shakespeare"). The TEIsimple files
// use word-by-word POS annotation and Folger Through Line Numbers (FTLNs).
//
// License: CC BY-NC 3.0. The edition is tagged with source_key="folger" and
// license_tier="cc-by-nc" so it can be excluded from commercial/paid features.
// Use --exclude folger to omit this step from the build entirely.
func ImportFolger(database *sql.DB, sourcesDir string) error {
	stepBanner("STEP 7: Import Folger Shakespeare (TEIsimple)")
	start := time.Now()

	teisimpleDir := filepath.Join(sourcesDir, "folger", "teisimple")
	entries, err := os.ReadDir(teisimpleDir)
	if err != nil {
		return fmt.Errorf("reading Folger TEIsimple directory: %w", err)
	}

	// Create Folger source and edition.
	sourceID, err := db.GetSourceID(database,
		"Folger Shakespeare Library", "folger",
		"https://shakespeare.folger.edu/",
		"CC BY-NC 3.0",
		"https://creativecommons.org/licenses/by-nc/3.0/deed.en_US",
		"Barbara A. Mowat and Paul Werstine, eds. The Folger Shakespeare. "+
			"Washington, DC: Folger Shakespeare Library. https://shakespeare.folger.edu/",
		true,
		"Modern scholarly edition with word-by-word POS tagging (TEIsimple). "+
			"CC BY-NC 3.0 — not available for commercial use without permission.")
	if err != nil {
		return fmt.Errorf("creating Folger source: %w", err)
	}

	editionID, err := db.GetEditionID(database,
		"Folger Shakespeare", "folger_shakespeare",
		sourceID, 2015,
		"Barbara A. Mowat, Paul Werstine",
		"Modern scholarly edition edited by Mowat & Werstine. "+
			"TEIsimple encoding with MorphAdorner POS tagging and Folger Through Line Numbers.")
	if err != nil {
		return fmt.Errorf("creating Folger edition: %w", err)
	}

	// Tag edition with source_key and license_tier for downstream filtering.
	_, err = database.Exec(
		`UPDATE editions SET source_key = 'folger', license_tier = 'cc-by-nc' WHERE id = ?`,
		editionID)
	if err != nil {
		return fmt.Errorf("tagging Folger edition: %w", err)
	}

	// Build works map: ossID → workInfo
	worksMap, err := buildWorksMap(database)
	if err != nil {
		return fmt.Errorf("building works map: %w", err)
	}

	// Build reverse slug map: folger-slug → ossID
	reverseSlugMap := make(map[string]string)
	for ossID, slug := range constants.FolgerSlugs {
		reverseSlugMap[slug] = ossID
	}

	// Collect XML files, sorted for deterministic output.
	xmlFiles := collectXMLFiles(entries)

	// === Phase 1: Parse all XML files in parallel (CPU-bound) ===
	type parsedPlay struct {
		fname string
		lines []parser.FolgerLine
		err   error
	}

	results := parallelProcess(xmlFiles, func(fname string) parsedPlay {
		data, err := os.ReadFile(filepath.Join(teisimpleDir, fname))
		if err != nil {
			return parsedPlay{fname: fname, err: err}
		}
		lines, err := parser.ParseFolgerTEIsimple(data)
		return parsedPlay{fname: fname, lines: lines, err: err}
	})

	parseResults := make(map[string]parsedPlay, len(results))
	for _, r := range results {
		parseResults[r.fname] = r
	}

	// === Phase 2: Insert each play sequentially (DB writes) ===
	totalLines, totalPlays := 0, 0

	for _, fname := range xmlFiles {
		slug := strings.TrimSuffix(fname, "_TEIsimple_FolgerShakespeare.xml")
		if slug == fname {
			continue
		}

		ossID, ok := reverseSlugMap[slug]
		if !ok {
			fmt.Printf("  SKIP %s — no ossID for slug %q\n", fname, slug)
			continue
		}

		work, ok := worksMap[ossID]
		if !ok {
			fmt.Printf("  SKIP %s — no work for ossID=%s\n", fname, ossID)
			continue
		}

		r := parseResults[fname]
		if r.err != nil {
			fmt.Printf("  ERROR reading/parsing %s: %v\n", fname, r.err)
			continue
		}
		if len(r.lines) == 0 {
			fmt.Printf("  SKIP %s — 0 lines parsed\n", fname)
			continue
		}

		totalPlays++

		clearWorkEditionData(database, work.ID, editionID)

		tx, err := database.Begin()
		if err != nil {
			fmt.Printf("  ERROR starting tx for %s: %v\n", work.Title, err)
			continue
		}

		// Folger-specific INSERT: includes stage_type and stage_who columns
		// that are NULL in all other editions. We use LastInsertId to link
		// word_annotations rows back to the text_lines row we just inserted.
		const folgerLineInsertSQL = `
			INSERT INTO text_lines
				(work_id, edition_id, act, scene, line_number,
				 character_id, char_name, content, content_type, word_count,
				 stage_type, stage_who)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		insertStmt, err := tx.Prepare(folgerLineInsertSQL)
		if err != nil {
			tx.Rollback()
			fmt.Printf("  ERROR preparing insert for %s: %v\n", work.Title, err)
			continue
		}

		wordInsertStmt, err := tx.Prepare(`
			INSERT OR IGNORE INTO word_annotations (line_id, position, word, lemma, pos)
			VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			insertStmt.Close()
			tx.Rollback()
			fmt.Printf("  ERROR preparing word insert for %s: %v\n", work.Title, err)
			continue
		}

		charCache := make(map[string]any)

		type sceneKey struct{ act, scene int }
		verseCounters := make(map[sceneKey]int)
		actScenes := make([][2]int, 0, len(r.lines))

		for _, line := range r.lines {
			sk := sceneKey{line.Act, line.Scene}

			lineNum := line.LineNumber
			if lineNum == 0 {
				lineNum = verseCounters[sk]
			} else if line.LineNumber > verseCounters[sk] {
				verseCounters[sk] = line.LineNumber
			}

			ct := contentType(line.IsStageDirection)
			charName := line.Character
			if line.IsStageDirection {
				charName = ""
			}

			charID := cachedLookupCharacter(database, work.ID, charName, charCache)

			result, err := insertStmt.Exec(
				work.ID, editionID,
				line.Act, line.Scene, lineNum,
				charID, nilIfEmpty(charName),
				line.Text, ct, countWords(line.Text),
				nilIfEmpty(line.StageType), nilIfEmpty(line.StageWho))
			if err != nil {
				actScenes = append(actScenes, [2]int{line.Act, line.Scene})
				continue
			}

			// Insert word-level annotations keyed to this line's row ID.
			if len(line.Words) > 0 {
				lineID, _ := result.LastInsertId()
				for pos, w := range line.Words {
					wordInsertStmt.Exec(lineID, pos+1, w.Word, nilIfEmpty(w.Lemma), nilIfEmpty(w.POS))
				}
			}

			actScenes = append(actScenes, [2]int{line.Act, line.Scene})
		}
		wordInsertStmt.Close()
		insertStmt.Close()

		insertTextDivisions(tx, work.ID, editionID, actScenes)

		if err := tx.Commit(); err != nil {
			fmt.Printf("  ERROR committing %s: %v\n", work.Title, err)
			continue
		}

		totalLines += len(r.lines)
		speeches := 0
		for _, l := range r.lines {
			if !l.IsStageDirection {
				speeches++
			}
		}
		fmt.Printf("  [%2d] %-35s %5d lines (%4d speeches)\n",
			totalPlays, work.Title, len(r.lines), speeches)
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "folger", "import_complete",
		fmt.Sprintf("%d plays", totalPlays), totalLines, elapsed)

	fmt.Printf("\n  ✓ %d lines from %d plays in %.1fs\n", totalLines, totalPlays, elapsed)
	return nil
}
