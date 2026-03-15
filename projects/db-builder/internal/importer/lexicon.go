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

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/constants"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/parser"
)

// ImportLexicon imports Schmidt lexicon XML entries from the given directory.
func ImportLexicon(database *sql.DB, entriesDir string) error {
	stepBanner("STEP 2: Import Schmidt Lexicon")

	info, err := os.Stat(entriesDir)
	if err != nil || !info.IsDir() {
		fmt.Printf("  WARNING: Lexicon entries not found at %s\n", entriesDir)
		fmt.Println("  Skipping lexicon import (scraper may still be running)")
		return nil
	}

	// Create Perseus source
	_, err = db.GetSourceID(database,
		"Perseus Digital Library — Schmidt Shakespeare Lexicon", "perseus_schmidt",
		"http://www.perseus.tufts.edu", "CC BY-SA 3.0",
		"https://creativecommons.org/licenses/by-sa/3.0/",
		"Alexander Schmidt, Shakespeare Lexicon and Quotation Dictionary. Provided by the Perseus Digital Library, Tufts University. Licensed under CC BY-SA 3.0.",
		true,
		"Schmidt lexicon entries scraped from Perseus TEI XML.")
	if err != nil {
		return err
	}

	start := time.Now()

	// Find all XML files
	xmlFiles, err := filepath.Glob(filepath.Join(entriesDir, "*", "*.xml"))
	if err != nil {
		return fmt.Errorf("globbing XML files: %w", err)
	}
	sort.Strings(xmlFiles)
	fmt.Printf("  Found %d XML files\n", len(xmlFiles))

	if len(xmlFiles) == 0 {
		fmt.Println("  No XML files found, skipping")
		return nil
	}

	// Build work abbreviation → DB ID map
	workMap := buildWorkMap(database)

	totalEntries := 0
	totalCitations := 0
	totalSenses := 0
	errors := 0

	// Group by letter directory
	letterDirs := make(map[string][]string)
	for _, f := range xmlFiles {
		dir := filepath.Dir(f)
		letterDirs[dir] = append(letterDirs[dir], f)
	}

	sortedDirs := make([]string, 0, len(letterDirs))
	for d := range letterDirs {
		sortedDirs = append(sortedDirs, d)
	}
	sort.Strings(sortedDirs)

	for _, letterDir := range sortedDirs {
		letter := filepath.Base(letterDir)
		files := letterDirs[letterDir]
		letterEntries := 0

		tx, err := database.Begin()
		if err != nil {
			return fmt.Errorf("starting transaction: %w", err)
		}

		for _, xmlPath := range files {
			content, err := os.ReadFile(xmlPath)
			if err != nil {
				errors++
				continue
			}

			entry, err := parser.ParseEntryXML(content, filepath.Base(xmlPath))
			if err != nil || entry == nil {
				errors++
				continue
			}

			// Insert entry
			result, err := tx.Exec(`
				INSERT OR IGNORE INTO lexicon_entries (key, letter, orthography, entry_type, full_text, raw_xml, source_file)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				entry.Key, entry.Letter, entry.Orthography, entry.EntryType,
				entry.FullText, entry.RawXML, entry.SourceFile)
			if err != nil {
				continue
			}

			entryID, err := result.LastInsertId()
			if entryID == 0 {
				// Already existed (OR IGNORE), look it up
				tx.QueryRow("SELECT id FROM lexicon_entries WHERE key = ?", entry.Key).Scan(&entryID)
			}
			if entryID == 0 {
				continue
			}

			// Insert senses and build sense_number → sense_id map
			senseIDMap := make(map[int]int64)
			for _, sense := range entry.Senses {
				sResult, sErr := tx.Exec(`INSERT OR IGNORE INTO lexicon_senses (entry_id, sense_number, definition_text) VALUES (?, ?, ?)`,
					entryID, sense.Number, sense.Text)
				if sErr == nil {
					senseID, _ := sResult.LastInsertId()
					if senseID == 0 {
						tx.QueryRow("SELECT id FROM lexicon_senses WHERE entry_id = ? AND sense_number = ?",
							entryID, sense.Number).Scan(&senseID)
					}
					if senseID > 0 {
						senseIDMap[sense.Number] = senseID
					}
				}
				totalSenses++
			}

			// Insert citations with sense_id linkage
			for _, cit := range entry.Citations {
				var workID interface{}
				if cit.WorkAbbrev != "" {
					if id, ok := workMap[cit.WorkAbbrev]; ok {
						workID = id
					}
				}

				// Resolve sense_id from the citation's assigned sense number
				var senseID interface{}
				if cit.SenseNumber > 0 {
					if sid, ok := senseIDMap[cit.SenseNumber]; ok {
						senseID = sid
					}
				}

				tx.Exec(`
					INSERT INTO lexicon_citations (entry_id, sense_id, work_id, work_abbrev, perseus_ref,
						act, scene, line, quote_text, display_text, raw_bibl)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					entryID, senseID, workID, nilIfEmpty(cit.WorkAbbrev), nilIfEmpty(cit.PerseusRef),
					cit.Act, cit.Scene, cit.Line,
					nilIfEmpty(cit.QuoteText), nilIfEmpty(cit.DisplayText), nilIfEmpty(cit.RawBibl))
				totalCitations++
			}

			letterEntries++
		}

		tx.Commit()
		if letterEntries > 0 {
			totalEntries += letterEntries
			fmt.Printf("  %s: %d entries\n", letter, letterEntries)
		}
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "lexicon", "import_complete",
		fmt.Sprintf("%d entries, %d citations, %d errors", totalEntries, totalCitations, errors),
		totalEntries, elapsed)

	fmt.Printf("  ✓ %d entries, %d citations, %d senses in %.1fs\n",
		totalEntries, totalCitations, totalSenses, elapsed)
	return nil
}

// buildWorkMap creates a map from Schmidt abbreviation → database work ID.
func buildWorkMap(database *sql.DB) map[string]int64 {
	workMap := make(map[string]int64)
	rows, err := database.Query("SELECT id, schmidt_abbrev FROM works WHERE schmidt_abbrev IS NOT NULL")
	if err != nil {
		return workMap
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var abbrev string
		rows.Scan(&id, &abbrev)
		workMap[abbrev] = id
		workMap[strings.TrimRight(abbrev, ".")] = id
	}

	// Also map all known aliases
	for abbrev := range constants.SchmidtWorks {
		title := constants.SchmidtWorks[abbrev].Title
		for existingAbbrev, id := range workMap {
			if sw, ok := constants.SchmidtWorks[existingAbbrev]; ok && sw.Title == title {
				workMap[abbrev] = id
			}
		}
	}

	return workMap
}
