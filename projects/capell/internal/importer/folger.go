// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	// FolgerSlugs maps ossID → folger-slug; we need the reverse.
	reverseSlugMap := make(map[string]string) // folger-slug → ossID
	for ossID, slug := range constants.FolgerSlugs {
		reverseSlugMap[slug] = ossID
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
		// Filename format: "{slug}_TEIsimple_FolgerShakespeare.xml"
		slug := strings.TrimSuffix(fname, "_TEIsimple_FolgerShakespeare.xml")
		if slug == fname {
			// Unexpected filename format — skip
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

		xmlPath := filepath.Join(teisimpleDir, fname)
		data, err := os.ReadFile(xmlPath)
		if err != nil {
			fmt.Printf("  ERROR reading %s: %v\n", fname, err)
			continue
		}

		lines, err := parser.ParseFolgerTEIsimple(data)
		if err != nil {
			fmt.Printf("  ERROR parsing %s: %v\n", fname, err)
			continue
		}

		if len(lines) == 0 {
			fmt.Printf("  SKIP %s — 0 lines parsed\n", fname)
			continue
		}

		totalPlays++

		// Clear existing Folger data for this work (idempotent re-import).
		clearWorkEditionData(database, work.ID, editionID)

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

		charCache := make(map[string]interface{})

		// Use a verse-only counter per scene for lines without a Folger line number
		// (stage directions, some prose lines). Lines with explicit FTLNs use those.
		type sceneKey struct{ act, scene int }
		verseCounters := make(map[sceneKey]int)

		for _, line := range lines {
			sk := sceneKey{line.Act, line.Scene}

			lineNum := line.LineNumber
			if lineNum == 0 {
				// No FTLN — use scene counter (stage directions and unnumbered lines)
				lineNum = verseCounters[sk]
			} else {
				// Update counter to track highest seen FTLN for this scene
				if line.LineNumber > verseCounters[sk] {
					verseCounters[sk] = line.LineNumber
				}
			}

			ct := "speech"
			charName := line.Character
			if line.IsStageDirection {
				ct = "stage_direction"
				charName = ""
			}

			charID := cachedLookupCharacter(database, work.ID, charName, charCache)

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
	db.LogImport(database, "folger", "import_complete",
		fmt.Sprintf("%d plays", totalPlays), totalLines, elapsed)

	fmt.Printf("\n  ✓ %d lines from %d plays in %.1fs\n", totalLines, totalPlays, elapsed)
	return nil
}
