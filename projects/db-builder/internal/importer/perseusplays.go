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

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/parser"
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

	totalLines, totalPlays := 0, 0

	for _, fname := range xmlFiles {
		xmlPath := filepath.Join(perseusDir, fname)
		data, err := os.ReadFile(xmlPath)
		if err != nil {
			fmt.Printf("  ERROR reading %s: %v\n", fname, err)
			continue
		}

		// Parse the TEI XML.
		lines, err := parser.ParsePerseusTEI(data)
		if err != nil {
			fmt.Printf("  ERROR parsing %s: %v\n", fname, err)
			continue
		}

		// Skip poetry files (0 lines from play parser).
		if len(lines) == 0 {
			continue
		}

		// Match to a work by perseus_id (filename without .xml is the ID).
		perseusID := fname[:len(fname)-4] // strip ".xml"
		work, ok := worksMap[perseusID]
		if !ok {
			fmt.Printf("  SKIP %s — no matching work for perseus_id=%s\n", fname, perseusID)
			continue
		}

		totalPlays++

		// Clear existing Perseus data for this work (idempotent re-import).
		database.Exec("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", work.ID, editionID)
		database.Exec("DELETE FROM text_divisions WHERE work_id = ? AND edition_id = ?", work.ID, editionID)

		// Insert lines in a transaction for speed.
		tx, err := database.Begin()
		if err != nil {
			fmt.Printf("  ERROR starting tx for %s: %v\n", work.Title, err)
			continue
		}

		insertStmt, err := tx.Prepare(`
			INSERT INTO text_lines (work_id, edition_id, act, scene, line_number,
				character_id, char_name, content, content_type, word_count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			tx.Rollback()
			fmt.Printf("  ERROR preparing insert for %s: %v\n", work.Title, err)
			continue
		}

		charCache := make(map[string]interface{}) // speaker → *int64 or nil

		// Compute Globe verse line numbers per scene.
		// Schmidt's lexicon cites Globe verse lines which do NOT count stage directions.
		// We use a verse-only counter as line_number so citations match directly.
		type sceneKey struct{ act, scene int }
		verseCounters := make(map[sceneKey]int)

		for _, line := range lines {
			var charID interface{}
			charName := line.Character

			if charName != "" {
				if cached, ok := charCache[charName]; ok {
					charID = cached
				} else {
					charID = lookupCharacter(database, work.ID, charName)
					charCache[charName] = charID
				}
			}

			ct := "speech"
			sk := sceneKey{line.Act, line.Scene}
			var lineNum int

			if line.IsStageDirection {
				ct = "stage_direction"
				charName = ""
				// Stage directions get the current verse counter (sorts with preceding verse line).
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
		}
		insertStmt.Close()

		// Insert text_divisions summary per scene.
		scenes := make(map[[2]int]int)
		for _, line := range lines {
			key := [2]int{line.Act, line.Scene}
			scenes[key]++
		}
		for key, count := range scenes {
			tx.Exec(`INSERT OR IGNORE INTO text_divisions (work_id, edition_id, act, scene, line_count)
				VALUES (?, ?, ?, ?, ?)`,
				work.ID, editionID, key[0], key[1], count)
		}

		if err := tx.Commit(); err != nil {
			fmt.Printf("  ERROR committing %s: %v\n", work.Title, err)
			continue
		}

		totalLines += len(lines)
		speeches := 0
		for _, l := range lines {
			if !l.IsStageDirection {
				speeches++
			}
		}
		fmt.Printf("  [%2d] %-35s %5d lines (%4d speeches)\n",
			totalPlays, work.Title, len(lines), speeches)
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "perseus_plays", "import_complete",
		fmt.Sprintf("%d plays", totalPlays), totalLines, elapsed)

	fmt.Printf("\n  ✓ %d lines from %d plays in %.1fs\n", totalLines, totalPlays, elapsed)
	return nil
}

// perseusWork holds the minimal info needed to match a Perseus XML to a DB work.
type perseusWork struct {
	ID    int64
	Title string
}

// buildPerseusWorksMap queries the works table for all rows with a perseus_id
// and returns a map from perseus_id → perseusWork.
func buildPerseusWorksMap(database *sql.DB) (map[string]perseusWork, error) {
	rows, err := database.Query("SELECT id, title, perseus_id FROM works WHERE perseus_id IS NOT NULL AND perseus_id != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]perseusWork)
	for rows.Next() {
		var id int64
		var title, perseusID string
		if err := rows.Scan(&id, &title, &perseusID); err != nil {
			continue
		}
		m[perseusID] = perseusWork{ID: id, Title: title}
	}
	return m, nil
}
