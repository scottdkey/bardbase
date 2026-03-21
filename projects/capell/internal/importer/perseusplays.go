// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	stepBanner("Import Perseus Play Texts")
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
	worksMap, err := buildWorksMapByColumn(database, "perseus_id")
	if err != nil {
		return fmt.Errorf("building works map: %w", err)
	}

	// Collect XML files, sorted for deterministic output.
	xmlFiles := collectXMLFiles(entries)

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

		// Assign Globe line numbers to every line. Perseus only marks Globe
		// numbers on milestone lines (every 5th/10th), so we interpolate
		// for lines in between. Schmidt's lexicon citations use Globe
		// numbering, so preserving these is essential for citation matching.
		interpolateGlobeNumbers(r.lines)

		actScenes := make([][2]int, 0, len(r.lines))

		for _, line := range r.lines {
			charName := line.Character
			charID := cachedLookupCharacter(database, work.ID, charName, charCache)

			ct := contentType(line.IsStageDirection)
			lineNum := line.GlobeLine

			if line.IsStageDirection {
				charName = ""
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

// interpolateGlobeNumbers fills in Globe line numbers for all lines.
// Perseus only marks Globe numbers on milestone lines (every 5th/10th via
// <lb ed="G" n="10"/>). Lines between milestones get GlobeLine=0.
//
// This function walks through the lines and interpolates: between two
// milestones at Globe numbers G1 and G2 with N lines between them,
// each line gets G1+1, G1+2, etc. Lines before the first milestone
// count backwards. Stage directions share the number of the preceding line.
func interpolateGlobeNumbers(lines []parser.PerseusLine) {
	if len(lines) == 0 {
		return
	}

	// Pass 1: assign Globe numbers to speech lines (not stage directions)
	// using milestones as anchors.
	current := 0
	for i := range lines {
		if lines[i].GlobeLine > 0 {
			// This is a milestone — use its number as the anchor.
			current = lines[i].GlobeLine
		} else if !lines[i].IsStageDirection {
			// Increment from last known Globe number.
			current++
			lines[i].GlobeLine = current
		}
	}

	// Pass 2: fill in lines before the first milestone by counting backwards.
	firstMilestone := -1
	for i := range lines {
		if lines[i].GlobeLine > 0 {
			firstMilestone = i
			break
		}
	}
	if firstMilestone > 0 {
		num := lines[firstMilestone].GlobeLine
		for i := firstMilestone - 1; i >= 0; i-- {
			if !lines[i].IsStageDirection {
				num--
				if num < 1 {
					num = 1
				}
				lines[i].GlobeLine = num
			}
		}
	}

	// Pass 3: stage directions get the same number as the preceding speech line.
	lastNum := 0
	for i := range lines {
		if lines[i].IsStageDirection {
			lines[i].GlobeLine = lastNum
		} else if lines[i].GlobeLine > 0 {
			lastNum = lines[i].GlobeLine
		}
	}
}

