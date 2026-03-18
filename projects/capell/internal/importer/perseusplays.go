// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// ImportPerseusPlays parses Perseus TEI XML play texts and inserts them into the
// database as a new edition ("Perseus Globe"). Each of the 37 play XMLs is matched
// to an existing work via the works.perseus_id column.
//
// The Perseus texts are the Clark & Wright Globe edition (1864), which is the
// standard scholarly reference for line numbers in Shakespeare studies.
func ImportPerseusPlays(database *sql.DB, sourcesDir string) error {
	stepBanner("STEP 5: Import Perseus Play Texts")
	start := time.Now()

	perseusDir := filepath.Join(sourcesDir, "perseus-plays")
	entries, err := os.ReadDir(perseusDir)
	if err != nil {
		return fmt.Errorf("reading Perseus directory: %w", err)
	}

	// Create Perseus source and Globe edition.
	sourceID, err := db.GetSourceID(database,
		"Perseus Digital Library", "perseus",
		"https://www.perseus.tufts.edu/hopper/",
		"CC BY-SA 3.0 US",
		"https://creativecommons.org/licenses/by-sa/3.0/us/",
		"Text provided by Perseus Digital Library, Tufts University. "+
			"Clark & Wright Globe edition. Funded by NSF/NEH Digital Libraries Initiative.",
		true,
		"Globe edition (1864) with Globe and First Folio line numbers in TEI XML.")
	if err != nil {
		return fmt.Errorf("creating Perseus source: %w", err)
	}

	editionID, err := db.GetEditionID(database,
		"Perseus Globe Edition", "perseus_globe",
		sourceID, 1864,
		"W. G. Clark, W. Aldis Wright",
		"Clark & Wright Globe edition via Perseus Digital Library TEI XML. "+
			"Globe line numbers marked every ~10 lines.")
	if err != nil {
		return fmt.Errorf("creating Perseus edition: %w", err)
	}

	// Build a map from perseus_id → work info.
	worksMap, err := buildPerseusWorksMap(database)
	if err != nil {
		return fmt.Errorf("building works map: %w", err)
	}

	// Collect XML files, sorted for deterministic output.
	var xmlFiles []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".xml" {
			xmlFiles = append(xmlFiles, e.Name())
		}
	}
	sort.Strings(xmlFiles)

	// === Phase 1: Parse all XML files in parallel (CPU-bound) ===
	type parsedPlay struct {
		fname string
		lines []parser.PerseusLine
		err   error
	}

	results := parallelProcess(xmlFiles, func(fname string) parsedPlay {
		data, err := os.ReadFile(filepath.Join(perseusDir, fname))
		if err != nil {
			return parsedPlay{fname: fname, err: err}
		}
		lines, err := parser.ParsePerseusTEI(data)
		return parsedPlay{fname: fname, lines: lines, err: err}
	})

	parseResults := make(map[string]parsedPlay, len(results))
	for _, r := range results {
		parseResults[r.fname] = r
	}

	// === Phase 2: Insert each play sequentially (DB writes) ===
	totalLines, totalPlays := 0, 0

	for _, fname := range xmlFiles {
		r := parseResults[fname]
		if r.err != nil {
			fmt.Printf("  ERROR reading/parsing %s: %v\n", fname, r.err)
			continue
		}
		if len(r.lines) == 0 {
			continue // poetry files return 0 lines from the play parser
		}

		perseusID := fname[:len(fname)-4] // strip ".xml"
		work, ok := worksMap[perseusID]
		if !ok {
			fmt.Printf("  SKIP %s — no matching work for perseus_id=%s\n", fname, perseusID)
			continue
		}

		totalPlays++

		clearWorkEditionData(database, work.ID, editionID)

		tx, err := database.Begin()
		if err != nil {
			fmt.Printf("  ERROR starting tx for %s: %v\n", work.Title, err)
			continue
		}

		insertStmt, err := tx.Prepare(textLinesInsertSQL)
		if err != nil {
			tx.Rollback()
			fmt.Printf("  ERROR preparing insert for %s: %v\n", work.Title, err)
			continue
		}

		charCache := make(map[string]any)

		type sceneKey struct{ act, scene int }
		verseCounters := make(map[sceneKey]int)
		actScenes := make([][2]int, 0, len(r.lines))

		for _, line := range r.lines {
			charName := line.Character
			charID := cachedLookupCharacter(database, work.ID, charName, charCache)

			ct := "speech"
			sk := sceneKey{line.Act, line.Scene}
			var lineNum int

			if line.IsStageDirection {
				ct = "stage_direction"
				charName = ""
				lineNum = verseCounters[sk]
			} else {
				verseCounters[sk]++
				lineNum = verseCounters[sk]
			}

			insertStmt.Exec(
				work.ID, editionID,
				line.Act, line.Scene, lineNum,
				charID, nilIfEmpty(charName),
				line.Text, ct, countWords(line.Text))

			actScenes = append(actScenes, [2]int{line.Act, line.Scene})
		}
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
	db.LogImport(database, "perseus_plays", "import_complete",
		fmt.Sprintf("%d plays", totalPlays), totalLines, elapsed)

	fmt.Printf("\n  ✓ %d lines from %d plays in %.1fs\n", totalLines, totalPlays, elapsed)
	return nil
}

// buildPerseusWorksMap queries the works table for all rows with a perseus_id
// and returns a map from perseus_id → workInfo.
func buildPerseusWorksMap(database *sql.DB) (map[string]workInfo, error) {
	rows, err := database.Query("SELECT id, title, perseus_id FROM works WHERE perseus_id IS NOT NULL AND perseus_id != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]workInfo)
	for rows.Next() {
		var id int64
		var title, perseusID string
		if err := rows.Scan(&id, &title, &perseusID); err != nil {
			continue
		}
		m[perseusID] = workInfo{ID: id, Title: title}
	}
	return m, nil
}
