// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
)

// BuildFTS rebuilds all full-text search indexes.
func BuildFTS(database *sql.DB) error {
	stepBanner("STEP 10: Build Full-Text Search Indexes")

	start := time.Now()

	// Lexicon FTS
	lexiconCount, _ := db.TableCount(database, "lexicon_entries")
	if lexiconCount > 0 {
		fmt.Printf("  Lexicon FTS: %d entries...\n", lexiconCount)
		_, err := database.Exec("INSERT INTO lexicon_fts(lexicon_fts) VALUES('rebuild')")
		if err != nil {
			return fmt.Errorf("rebuilding lexicon FTS: %w", err)
		}
	}

	// Text FTS
	textCount, _ := db.TableCount(database, "text_lines")
	if textCount > 0 {
		fmt.Printf("  Text FTS: %d lines...\n", textCount)
		_, err := database.Exec("INSERT INTO text_fts(text_fts) VALUES('rebuild')")
		if err != nil {
			return fmt.Errorf("rebuilding text FTS: %w", err)
		}
	}

	elapsed := time.Since(start).Seconds()
	fmt.Printf("  ✓ FTS indexes built in %.1fs\n", elapsed)
	return nil
}
