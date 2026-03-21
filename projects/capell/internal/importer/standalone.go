// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// standaloneEditionDef is the JSON representation of an edition entry in standalone_passages.json.
type standaloneEditionDef struct {
	ShortCode   string  `json:"short_code"`
	Name        string  `json:"name"`
	Year        *int    `json:"year"`
	Description string  `json:"description"`
	SourceKey   string  `json:"source_key"`
	LicenseTier string  `json:"license_tier"`
}

// standalonePassage is the JSON representation of a single passage entry.
type standalonePassage struct {
	WorkID      int64  `json:"work_id"`
	Edition     string `json:"edition"`
	Act         *int   `json:"act,omitempty"`
	Scene       *int   `json:"scene,omitempty"`
	LineNumber  int    `json:"line_number"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// standaloneFile is the top-level structure of standalone_passages.json.
type standaloneFile struct {
	Editions []standaloneEditionDef `json:"editions"`
	Passages []standalonePassage    `json:"passages"`
}

// ImportStandalonePassages reads projects/data/standalone_passages.json and inserts
// text_line rows for passages from non-Shakespeare works cited by Schmidt. These
// passages have no existing DB representation; adding them allows the citation
// resolution pipeline to find matches via manual corrections.
//
// This step must run before the citations step.
func ImportStandalonePassages(database *sql.DB, sourcesDir string) error {
	stepBanner("Import Standalone Passages")
	start := time.Now()

	// Locate standalone_passages.json relative to the data directory.
	// sourcesDir is projects/sources/; data is projects/data/.
	dataDir := filepath.Join(filepath.Dir(sourcesDir), "data")
	jsonPath := filepath.Join(dataDir, "standalone_passages.json")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("reading standalone_passages.json: %w", err)
	}

	var sf standaloneFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return fmt.Errorf("parsing standalone_passages.json: %w", err)
	}

	// Ensure all editions exist; collect short_code → id mapping.
	editionIDs := make(map[string]int64, len(sf.Editions))
	for _, ed := range sf.Editions {
		var edID int64
		err := database.QueryRow("SELECT id FROM editions WHERE short_code = ?", ed.ShortCode).Scan(&edID)
		if err == sql.ErrNoRows {
			// Insert the edition (no source_id; standalone passages have no source record).
			res, err := database.Exec(`
				INSERT INTO editions (name, short_code, year, description, source_key, license_tier)
				VALUES (?, ?, ?, ?, ?, ?)`,
				ed.Name, ed.ShortCode, ed.Year, nilIfEmpty(ed.Description),
				nilIfEmpty(ed.SourceKey), nilIfEmpty(ed.LicenseTier))
			if err != nil {
				return fmt.Errorf("inserting edition %q: %w", ed.ShortCode, err)
			}
			edID, _ = res.LastInsertId()
			fmt.Printf("  Created edition: %s (id=%d)\n", ed.ShortCode, edID)
		} else if err != nil {
			return fmt.Errorf("querying edition %q: %w", ed.ShortCode, err)
		}
		editionIDs[ed.ShortCode] = edID
	}

	// Insert each passage as a text_line (idempotent via INSERT OR IGNORE).
	inserted := 0
	skipped := 0
	for _, p := range sf.Passages {
		edID, ok := editionIDs[p.Edition]
		if !ok {
			fmt.Printf("  WARNING: unknown edition %q for work_id=%d, skipping\n", p.Edition, p.WorkID)
			skipped++
			continue
		}

		var act, scene any
		if p.Act != nil {
			act = *p.Act
		}
		if p.Scene != nil {
			scene = *p.Scene
		}

		contentType := p.ContentType
		if contentType == "" {
			contentType = "verse"
		}

		res, err := database.Exec(`
			INSERT OR IGNORE INTO text_lines
				(work_id, edition_id, act, scene, line_number, content, content_type, word_count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			p.WorkID, edID, act, scene, p.LineNumber,
			p.Content, contentType, countWords(p.Content))
		if err != nil {
			fmt.Printf("  WARNING: inserting passage work_id=%d line=%d: %v\n", p.WorkID, p.LineNumber, err)
			skipped++
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted++
		} else {
			skipped++
		}
	}

	fmt.Printf("  Standalone passages: %d inserted, %d skipped (%.2fs)\n",
		inserted, skipped, time.Since(start).Seconds())
	return nil
}
