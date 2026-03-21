// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// ImportLexicon imports Schmidt lexicon XML entries from the given directory.
func ImportLexicon(database *sql.DB, entriesDir string) error {
	stepBanner("Import Schmidt Lexicon")

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

	// Group by letter directory (needed for per-letter transactions in insert phase)
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

	// === Phase 1: Parse all XML files in parallel (CPU-bound) ===
	// File reads and ParseEntryXML are independent — safe to parallelize.
	// No DB access in this phase.
	type parsedEntry struct {
		entry     *parser.LexiconEntry
		letterDir string
	}

	parsed := parallelProcess(xmlFiles, func(xmlPath string) parsedEntry {
		content, err := os.ReadFile(xmlPath)
		if err != nil {
			return parsedEntry{}
		}
		entry, err := parser.ParseEntryXML(content, filepath.Base(xmlPath))
		if err != nil || entry == nil {
			return parsedEntry{}
		}
		return parsedEntry{entry: entry, letterDir: filepath.Dir(xmlPath)}
	})

	// Collect parsed entries grouped by letter (preserves per-letter transaction structure)
	letterEntries := make(map[string][]*parser.LexiconEntry)
	errors := 0
	for _, pf := range parsed {
		if pf.entry == nil {
			errors++
			continue
		}
		letter := filepath.Base(pf.letterDir)
		letterEntries[letter] = append(letterEntries[letter], pf.entry)
	}

	fmt.Printf("  Parsed in %.1fs (%d errors)\n",
		time.Since(start).Seconds(), errors)

	// === Phase 2: Insert all entries sequentially (DB writes, per-letter transactions) ===
	totalEntries := 0
	totalCitations := 0
	totalSenses := 0

	// Look up the Perseus Globe edition ID — the primary edition for lexicon
	// citations since Schmidt's Lexicon is sourced from Perseus Digital Library.
	// Falls back to any edition if Perseus isn't available.
	var perseusEditionID int64
	err = database.QueryRow(
		"SELECT id FROM editions WHERE short_code = 'perseus_globe'").Scan(&perseusEditionID)
	if err != nil || perseusEditionID == 0 {
		// Fallback: use whichever edition has the most text lines.
		database.QueryRow(
			"SELECT edition_id FROM text_lines GROUP BY edition_id ORDER BY COUNT(*) DESC LIMIT 1").Scan(&perseusEditionID)
	}

	// Prepared statements to resolve missing scene from act+line.
	// sceneLookupExact: tries exact line match across all editions.
	sceneLookupExact, err := database.Prepare(`
		SELECT DISTINCT scene FROM text_lines
		WHERE work_id = ? AND act = ? AND line_number = ?
		LIMIT 1`)
	if err != nil {
		return fmt.Errorf("preparing scene lookup: %w", err)
	}
	defer sceneLookupExact.Close()

	// sceneLookupRange: range match against Perseus Globe (marks every ~10th line).
	// Finds the scene whose line range brackets the target line number.
	sceneLookupRange, err := database.Prepare(`
		SELECT scene FROM text_lines
		WHERE work_id = ? AND act = ? AND edition_id = ?
		GROUP BY scene
		HAVING MIN(line_number) <= ? AND MAX(line_number) >= ?
		LIMIT 1`)
	if err != nil {
		return fmt.Errorf("preparing scene range lookup: %w", err)
	}
	defer sceneLookupRange.Close()

	// sceneLookupNearest: fallback for lines before the first Globe marker in a scene
	// (e.g., line 3 in a scene whose first Globe marker is line 10).
	sceneLookupNearest, err := database.Prepare(`
		SELECT scene FROM text_lines
		WHERE work_id = ? AND act = ? AND edition_id = ? AND line_number >= ?
		ORDER BY line_number
		LIMIT 1`)
	if err != nil {
		return fmt.Errorf("preparing scene nearest lookup: %w", err)
	}
	defer sceneLookupNearest.Close()

	for _, letterDir := range sortedDirs {
		letter := filepath.Base(letterDir)
		entries := letterEntries[letter]
		if len(entries) == 0 {
			continue
		}

		tx, err := database.Begin()
		if err != nil {
			return fmt.Errorf("starting transaction: %w", err)
		}

		letterCount := 0
		for _, entry := range entries {
			// Split key into base_key + sense_group:
			// "Abandon" → ("Abandon", NULL), "A1" → ("A", 1), "Like3" → ("Like", 3)
			baseKey, senseGroup := splitSenseKey(entry.Key)

			result, err := tx.Exec(`
				INSERT OR IGNORE INTO lexicon_entries (key, base_key, sense_group, letter, orthography, entry_type, full_text, source_file)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				entry.Key, baseKey, senseGroup, entry.Letter, entry.Orthography, entry.EntryType,
				entry.FullText, entry.SourceFile)
			if err != nil {
				continue
			}

			entryID, err := result.LastInsertId()
			if entryID == 0 {
				tx.QueryRow("SELECT id FROM lexicon_entries WHERE key = ?", entry.Key).Scan(&entryID)
			}
			if entryID == 0 {
				continue
			}

			senseIDMap := make(map[int]int64)
			for _, sense := range entry.Senses {
				var subSense interface{}
				if sense.SubSense != "" {
					subSense = sense.SubSense
				}
				sResult, sErr := tx.Exec(
					`INSERT OR IGNORE INTO lexicon_senses (entry_id, sense_number, sub_sense, definition_text) VALUES (?, ?, ?, ?)`,
					entryID, sense.Number, subSense, sense.Text)
				if sErr == nil {
					senseID, _ := sResult.LastInsertId()
					if senseID == 0 {
						tx.QueryRow(
							"SELECT id FROM lexicon_senses WHERE entry_id = ? AND sense_number = ? AND COALESCE(sub_sense, '') = COALESCE(?, '')",
							entryID, sense.Number, subSense).Scan(&senseID)
					}
					if senseID > 0 {
						senseIDMap[sense.Number] = senseID
					}
				}
				totalSenses++
			}

			for _, cit := range entry.Citations {
				var workID interface{}
				var workIDInt int64
				if cit.WorkAbbrev != "" {
					if id, ok := workMap[cit.WorkAbbrev]; ok {
						workID = id
						workIDInt = id
					}
				}

				var senseID interface{}
				if cit.SenseNumber > 0 {
					if sid, ok := senseIDMap[cit.SenseNumber]; ok {
						senseID = sid
					}
				}

				// For poems/sonnets: set act=1 if missing (text_lines always has act=1).
				// Sonnets: scene = sonnet number. Poems: scene = 0.
				if workIDInt > 0 && cit.Act == nil {
					if sw, ok := constants.SchmidtWorks[cit.WorkAbbrev]; ok {
						one := 1
						switch sw.WorkType {
						case "sonnet_sequence":
							cit.Act = &one
						case "poem", "poem_collection":
							cit.Act = &one
							if cit.Scene == nil {
								zero := 0
								cit.Scene = &zero
							}
						}
					}
				}

				// Resolve missing scene from act+line.
				// Many citations have act and line but no scene — the scene can be
				// determined by looking up which scene contains that line number.
				// Strategy: exact match → range match → nearest line match.
				if workIDInt > 0 && cit.Act != nil && cit.Scene == nil && cit.Line != nil {
					var scene int
					err := sceneLookupExact.QueryRow(workIDInt, *cit.Act, *cit.Line).Scan(&scene)
					if err != nil {
						err = sceneLookupRange.QueryRow(workIDInt, *cit.Act, perseusEditionID, *cit.Line, *cit.Line).Scan(&scene)
					}
					if err != nil {
						err = sceneLookupNearest.QueryRow(workIDInt, *cit.Act, perseusEditionID, *cit.Line).Scan(&scene)
					}
					if err == nil {
						cit.Scene = &scene
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

			letterCount++
		}

		tx.Commit()
		if letterCount > 0 {
			totalEntries += letterCount
			fmt.Printf("  %s: %d entries\n", letter, letterCount)
		}
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "lexicon", "import_complete",
		fmt.Sprintf("%d entries, %d citations, %d errors", totalEntries, totalCitations, errors),
		totalEntries, elapsed)

	// Resolve external work references that the XML parser couldn't handle
	// (biblical, classical, appendix, and Shakespeare poems like Phoen./Lucr.).
	externalResolved := resolveUnmatchedCitations(database)
	if externalResolved > 0 {
		fmt.Printf("  Unmatched citations resolved: %d\n", externalResolved)
	}

	fmt.Printf("  ✓ %d entries, %d citations, %d senses in %.1fs\n",
		totalEntries, totalCitations, totalSenses, elapsed)
	return nil
}

// buildWorkMap creates a map from Schmidt abbreviation → database work ID.
// splitSenseKey splits a lexicon key like "A1" into base key "A" and sense group 1.
// Keys without trailing digits (e.g., "Abandon") return the key unchanged and nil.
func splitSenseKey(key string) (string, interface{}) {
	// Find where trailing digits start.
	i := len(key)
	for i > 0 && key[i-1] >= '0' && key[i-1] <= '9' {
		i--
	}
	if i == len(key) || i == 0 {
		// No trailing digits, or entire key is digits.
		return key, nil
	}
	base := key[:i]
	num, err := strconv.Atoi(key[i:])
	if err != nil {
		return key, nil
	}
	return base, num
}

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
