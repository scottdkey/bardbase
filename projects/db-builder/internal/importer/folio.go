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

	"github.com/scottdkey/heminge/projects/db-builder/internal/constants"
	"github.com/scottdkey/heminge/projects/db-builder/internal/db"
	"github.com/scottdkey/heminge/projects/db-builder/internal/parser"
)

// ImportFirstFolio parses the EEBO-TCP First Folio TEI XML (A11954) and inserts
// all 35 plays into the database as a new edition ("First Folio 1623").
//
// The First Folio is a diplomatic transcription in original spelling (1623),
// with long-s (ſ) normalized to s by the parser. Verse lines are per-<l> element;
// prose speeches are per-<p> element. Attribution: Text Creation Partnership, CC0.
func ImportFirstFolio(database *sql.DB, sourcesDir string) error {
	stepBanner("STEP 6: Import First Folio (EEBO-TCP A11954)")
	start := time.Now()

	folioPath := filepath.Join(sourcesDir, "eebo-tcp", "A11954.xml")
	data, err := os.ReadFile(folioPath)
	if err != nil {
		return fmt.Errorf("reading First Folio XML: %w", err)
	}

	// Create source and edition records.
	sourceID, err := db.GetSourceID(database,
		"EEBO-TCP (Text Creation Partnership)", "eebo_tcp",
		"https://textcreationpartnership.org/",
		"CC0 1.0 Universal",
		"https://creativecommons.org/publicdomain/zero/1.0/",
		"Transcribed and encoded by the Text Creation Partnership. "+
			"Phase 1 texts released to public domain 1 January 2015.",
		false,
		"Diplomatic transcription of early English books in TEI P5 XML.")
	if err != nil {
		return fmt.Errorf("creating EEBO-TCP source: %w", err)
	}

	editionID, err := db.GetEditionID(database,
		"First Folio (1623)", "first_folio",
		sourceID, 1623,
		"John Heminge, Henry Condell (eds.); EEBO-TCP transcription",
		"First Folio — Mr. William Shakespeares Comedies, Histories, & Tragedies (1623). "+
			"Original spelling, diplomatic transcription. TCP ID A11954, STC 22273.")
	if err != nil {
		return fmt.Errorf("creating First Folio edition: %w", err)
	}

	// Parse the full First Folio XML.
	fmt.Println("  Parsing First Folio TEI XML...")
	lines, err := parser.ParseFirstFolioTEI(data)
	if err != nil {
		return fmt.Errorf("parsing First Folio TEI: %w", err)
	}
	fmt.Printf("  Parsed %d total lines\n", len(lines))

	// Build works map (oss_id → workInfo).
	worksMap, err := buildWorksMap(database)
	if err != nil {
		return fmt.Errorf("building works map: %w", err)
	}

	// Group lines by play title for import.
	type playLines struct {
		ossID string
		lines []parser.FolioLine
	}
	playGroups := make(map[string][]parser.FolioLine)
	for _, line := range lines {
		playGroups[line.PlayTitle] = append(playGroups[line.PlayTitle], line)
	}

	// Sort play titles for deterministic output.
	titles := make([]string, 0, len(playGroups))
	for t := range playGroups {
		titles = append(titles, t)
	}
	sort.Strings(titles)

	totalLines, totalPlays := 0, 0
	unmatched := []string{}

	for _, title := range titles {
		playLineSlice := playGroups[title]

		// Look up OSS ID from folio title map.
		ossID, ok := constants.FolioPlayTitles[title]
		if !ok {
			unmatched = append(unmatched, title)
			continue
		}

		work, ok := worksMap[ossID]
		if !ok {
			fmt.Printf("  SKIP %q — oss_id=%q not in works table\n", title, ossID)
			continue
		}

		totalPlays++

		// Clear existing First Folio data for this work (idempotent).
		database.Exec("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", work.ID, editionID)
		database.Exec("DELETE FROM text_divisions WHERE work_id = ? AND edition_id = ?", work.ID, editionID)

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

		for i, line := range playLineSlice {
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
			if line.IsStageDirection {
				ct = "stage_direction"
				charName = ""
			}

			insertStmt.Exec(
				work.ID, editionID,
				line.Act, line.Scene, i+1,
				charID, nilIfEmpty(charName),
				line.Text, ct, countWords(line.Text))
		}
		insertStmt.Close()

		// Insert text_divisions summary per scene.
		scenes := make(map[[2]int]int)
		for _, line := range playLineSlice {
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

		totalLines += len(playLineSlice)
		fmt.Printf("  [%2d] %-35s %5d lines\n", totalPlays, work.Title, len(playLineSlice))
	}

	if len(unmatched) > 0 {
		fmt.Printf("  WARNING: %d unmatched play titles:\n", len(unmatched))
		for _, t := range unmatched {
			fmt.Printf("    %q\n", t)
		}
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "first_folio", "import_complete",
		fmt.Sprintf("%d plays", totalPlays), totalLines, elapsed)

	fmt.Printf("\n  ✓ %d lines from %d plays in %.1fs\n", totalLines, totalPlays, elapsed)
	return nil
}

