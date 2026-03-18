// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// quartoMeta holds the mapping from EEBO-TCP ID to work and edition metadata.
// Loaded from projects/data/eebo_quartos.json.
type quartoMeta struct {
	OSSID string `json:"oss_id"`
	Title string `json:"title"`
	Year  int    `json:"year"`
}

// ImportEEBOQuartos parses EEBO-TCP early quarto XML files and inserts each as its
// own edition in the database. Early quartos (Q1 Hamlet 1603, Q1 1H4 1598, etc.)
// are textually distinct from the First Folio and provide unique comparison data.
//
// The quarto files are identified by EEBO-TCP IDs (e.g., A11959 = Q1 Hamlet).
// Each quarto gets its own edition short_code (e.g., "q1_hamlet_1603").
func ImportEEBOQuartos(database *sql.DB, sourcesDir string) error {
	stepBanner("STEP 8: Import EEBO-TCP Early Quartos")
	start := time.Now()

	eeboDir := filepath.Join(sourcesDir, "eebo-tcp")

	// Load quarto metadata from projects/data/eebo_quartos.json
	dataDir := constants.DataDir()
	metaPath := filepath.Join(dataDir, "eebo_quartos.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("reading eebo_quartos.json: %w", err)
	}
	var quartos map[string]quartoMeta
	if err := json.Unmarshal(metaData, &quartos); err != nil {
		return fmt.Errorf("parsing eebo_quartos.json: %w", err)
	}

	// Create EEBO-TCP source (shared with First Folio importer).
	sourceID, err := db.GetSourceID(database,
		"EEBO-TCP (Text Creation Partnership)", "eebo_tcp",
		"https://textcreationpartnership.org/",
		"CC0 1.0 Universal",
		"https://creativecommons.org/publicdomain/zero/1.0/",
		"Early English Books Online Text Creation Partnership. "+
			"Phase 1 texts released to public domain 2015.",
		false,
		"Diplomatic transcriptions of early printed editions.")
	if err != nil {
		return fmt.Errorf("creating EEBO-TCP source: %w", err)
	}

	// Build works map: ossID → workInfo
	worksMap, err := buildWorksMap(database)
	if err != nil {
		return fmt.Errorf("building works map: %w", err)
	}

	// Collect and sort eebo IDs for deterministic output; filter to present files.
	eeboIDs := make([]string, 0, len(quartos))
	for id := range quartos {
		eeboIDs = append(eeboIDs, id)
	}
	sort.Strings(eeboIDs)

	var presentIDs []string
	for _, id := range eeboIDs {
		if _, err := os.Stat(filepath.Join(eeboDir, id+".xml")); err == nil {
			presentIDs = append(presentIDs, id)
		} else {
			fmt.Printf("  SKIP %s — file not found\n", id)
		}
	}

	// === Phase 1: Parse all XML files in parallel (CPU-bound) ===
	type parsedQuarto struct {
		eeboID string
		lines  []parser.QuartoLine
		err    error
	}

	results := parallelProcess(presentIDs, func(eeboID string) parsedQuarto {
		data, err := os.ReadFile(filepath.Join(eeboDir, eeboID+".xml"))
		if err != nil {
			return parsedQuarto{eeboID: eeboID, err: err}
		}
		lines, err := parser.ParseEEBOQuartoTEI(data)
		return parsedQuarto{eeboID: eeboID, lines: lines, err: err}
	})

	parseResults := make(map[string]parsedQuarto, len(results))
	for _, r := range results {
		parseResults[r.eeboID] = r
	}

	// === Phase 2: Insert each quarto sequentially (DB writes) ===
	totalLines, totalPlays := 0, 0

	for _, eeboID := range presentIDs {
		meta := quartos[eeboID]

		work, ok := worksMap[meta.OSSID]
		if !ok {
			fmt.Printf("  SKIP %s — no work for ossID=%s\n", eeboID, meta.OSSID)
			continue
		}

		r := parseResults[eeboID]
		if r.err != nil {
			fmt.Printf("  ERROR reading/parsing %s: %v\n", eeboID, r.err)
			continue
		}
		if len(r.lines) == 0 {
			fmt.Printf("  SKIP %s — 0 lines parsed\n", eeboID)
			continue
		}

		// Each quarto gets its own edition (e.g. "q1_hamlet_1603")
		editionShortCode := fmt.Sprintf("q1_%s_%d", meta.OSSID, meta.Year)
		editionID, err := db.GetEditionID(database,
			meta.Title, editionShortCode,
			sourceID, meta.Year,
			"Text Creation Partnership",
			fmt.Sprintf("EEBO-TCP diplomatic transcription of %s (EEBO ID: %s). "+
				"Original spelling, flat structure without act/scene divisions.", meta.Title, eeboID))
		if err != nil {
			fmt.Printf("  ERROR creating edition for %s: %v\n", eeboID, err)
			continue
		}

		_, _ = database.Exec(
			`UPDATE editions SET source_key = 'eebo_tcp', license_tier = 'cc0' WHERE id = ?`,
			editionID)

		totalPlays++

		clearWorkEditionData(database, work.ID, editionID)

		tx, err := database.Begin()
		if err != nil {
			fmt.Printf("  ERROR starting tx for %s: %v\n", meta.Title, err)
			continue
		}

		insertStmt, err := tx.Prepare(textLinesInsertSQL)
		if err != nil {
			tx.Rollback()
			fmt.Printf("  ERROR preparing insert for %s: %v\n", meta.Title, err)
			continue
		}

		charCache := make(map[string]any)

		for _, line := range r.lines {
			ct := "speech"
			charName := line.Character
			if line.IsStageDirection {
				ct = "stage_direction"
				charName = ""
			}

			charID := cachedLookupCharacter(database, work.ID, charName, charCache)

			insertStmt.Exec(
				work.ID, editionID,
				line.Act, line.Scene, line.LineInScene,
				charID, nilIfEmpty(charName),
				line.Text, ct, countWords(line.Text))
		}
		insertStmt.Close()

		speechCount := 0
		for _, l := range r.lines {
			if !l.IsStageDirection {
				speechCount++
			}
		}
		// Flat quartos have a single scene
		tx.Exec(`INSERT OR IGNORE INTO text_divisions (work_id, edition_id, act, scene, line_count)
			VALUES (?, ?, ?, ?, ?)`,
			work.ID, editionID, 1, 1, len(r.lines))

		if err := tx.Commit(); err != nil {
			fmt.Printf("  ERROR committing %s: %v\n", meta.Title, err)
			continue
		}

		totalLines += len(r.lines)
		fmt.Printf("  %-45s %5d lines (%4d speeches)\n",
			meta.Title, len(r.lines), speechCount)
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "eebo_quartos", "import_complete",
		fmt.Sprintf("%d quartos", totalPlays), totalLines, elapsed)

	fmt.Printf("\n  ✓ %d lines from %d quartos in %.1fs\n", totalLines, totalPlays, elapsed)
	return nil
}
