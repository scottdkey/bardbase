// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Package importer implements the build steps that populate the Shakespeare database.
package importer

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// OSSWork holds parsed data for a single work from the OSS dump.
type OSSWork struct {
	OSSID           string
	Title           string
	FullTitle       string
	ShortTitle      string
	DateComposed    *int
	GenreType       string
	WorkType        string
	TotalWords      *int
	TotalParagraphs *int
	SourceText      string
}

// OSSCharacter holds parsed character data from the OSS dump.
type OSSCharacter struct {
	CharID      string
	Name        string
	Abbrev      string
	OSSWorkID   string
	Description string
	SpeechCount *int
}

// OSSChapter holds parsed chapter/division data.
type OSSChapter struct {
	WorkID      string
	Section     int
	Chapter     int
	Description string
}

// OSSParagraph holds parsed paragraph/text data.
type OSSParagraph struct {
	WorkID       string
	ParagraphID  int
	ParagraphNum int
	CharID       string
	Text         string
	Type         string
	Section      int
	Chapter      int
	WordCount    int
	LineNumber   int // computed scene-relative line number
}

// splitOSSLines splits a Moby/OSS paragraph on the `\n[p]` line separator.
// The Moby source uses `\n[p]` between verse/prose lines within a speech.
// After MySQL escape decoding, `\n` becomes a real newline character, so
// the separator is newline + `[p]`. Any trailing newlines (from `\n` at
// end of text without `[p]`) are also handled by TrimSpace.
func splitOSSLines(text string) []string {
	parts := strings.Split(text, "\n[p]")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ImportOSS imports the OSS/Moby Shakespeare data from a MySQL dump file.
func ImportOSS(database *sql.DB, sqlPath string) error {
	stepBanner("Import OSS/Moby Shakespeare")

	content, err := os.ReadFile(sqlPath)
	if err != nil {
		return fmt.Errorf("reading SQL dump: %w", err)
	}

	start := time.Now()

	// Create source + edition
	sourceID, err := db.GetSourceID(database,
		"Open Source Shakespeare / Moby", "oss_moby",
		"https://www.opensourceshakespeare.org", "Public Domain", "",
		"Text from Open Source Shakespeare (opensourceshakespeare.org), based on the Moby project. Public domain.", false,
		"Globe-based modern spelling text. Originally from Moby project.")
	if err != nil {
		return err
	}

	editionID, err := db.GetEditionID(database,
		"Open Source Shakespeare (Globe)", "oss_globe",
		sourceID, 2003, "George Mason University",
		"Globe-based text via Moby project")
	if err != nil {
		return err
	}

	// Parse the SQL dump
	sqlContent := string(content)
	statements := parser.ExtractStatements(sqlContent)

	var works []OSSWork
	var characters []OSSCharacter
	chapters := make(map[int]OSSChapter)
	var paragraphs []OSSParagraph

	for _, stmt := range statements {
		upper := strings.ToUpper(stmt)
		if !strings.Contains(upper, "INSERT INTO") {
			continue
		}

		table := parser.GetInsertTable(stmt)
		rows := parser.ParseMySQLValues(stmt)

		switch table {
		case "Works":
			for _, row := range rows {
				if len(row) < 10 {
					continue
				}
				w := OSSWork{
					OSSID:      parser.ValStr(row[0]),
					Title:      parser.DecodeHTMLEntities(parser.ValStr(row[1])),
					FullTitle:  parser.DecodeHTMLEntities(parser.ValStr(row[2])),
					ShortTitle: parser.ValStr(row[3]),
					GenreType:  parser.ValStr(row[5]),
					SourceText: parser.ValStr(row[7]),
				}
				if v := parser.ValStr(row[4]); v != "" {
					if n, err := strconv.Atoi(v); err == nil {
						w.DateComposed = &n
					}
				}
				w.WorkType = constants.GenreMap[w.GenreType]
				if w.WorkType == "" {
					w.WorkType = w.GenreType
				}
				if v := parser.ValStr(row[8]); v != "" {
					if n, err := strconv.Atoi(v); err == nil {
						w.TotalWords = &n
					}
				}
				if v := parser.ValStr(row[9]); v != "" {
					if n, err := strconv.Atoi(v); err == nil {
						w.TotalParagraphs = &n
					}
				}
				works = append(works, w)
			}

		case "Characters":
			for _, row := range rows {
				if len(row) < 6 {
					continue
				}
				c := OSSCharacter{
					CharID:      parser.ValStr(row[0]),
					Name:        parser.DecodeHTMLEntities(parser.ValStr(row[1])),
					Abbrev:      parser.ValStr(row[2]),
					OSSWorkID:   parser.ValStr(row[3]),
					Description: parser.DecodeHTMLEntities(parser.ValStr(row[4])),
				}
				if v := parser.ValStr(row[5]); v != "" {
					if n, err := strconv.Atoi(v); err == nil {
						c.SpeechCount = &n
					}
				}
				characters = append(characters, c)
			}

		case "Chapters":
			for _, row := range rows {
				if len(row) < 5 {
					continue
				}
				chapterID, _ := strconv.Atoi(parser.ValStr(row[1]))
				section, _ := strconv.Atoi(parser.ValStr(row[2]))
				chapter, _ := strconv.Atoi(parser.ValStr(row[3]))
				chapters[chapterID] = OSSChapter{
					WorkID:      parser.ValStr(row[0]),
					Section:     section,
					Chapter:     chapter,
					Description: parser.DecodeHTMLEntities(parser.ValStr(row[4])),
				}
			}

		case "Paragraphs":
			for _, row := range rows {
				if len(row) < 12 {
					continue
				}
				paraID, _ := strconv.Atoi(parser.ValStr(row[1]))
				paraNum, _ := strconv.Atoi(parser.ValStr(row[2]))
				section, _ := strconv.Atoi(parser.ValStr(row[8]))
				chapter, _ := strconv.Atoi(parser.ValStr(row[9]))
				wordCount, _ := strconv.Atoi(parser.ValStr(row[11]))
				paragraphs = append(paragraphs, OSSParagraph{
					WorkID:       parser.ValStr(row[0]),
					ParagraphID:  paraID,
					ParagraphNum: paraNum,
					CharID:       parser.ValStr(row[3]),
					Text:         parser.DecodeHTMLEntities(parser.ValStr(row[4])),
					Type:         parser.ValStr(row[7]),
					Section:      section,
					Chapter:      chapter,
					WordCount:    wordCount,
				})
			}
		}
	}

	// Expand Moby sub-lines: the OSS MySQL dump stores multiple verse/prose lines
	// in a single paragraph, joined by the `n[p]` separator. This comes from the
	// Moby Shakespeare format where line breaks are `\n[p]`; the SQL parser strips
	// the backslash escape and writes `n` literally, so `\n[p]` → `n[p]` in the
	// stored string. Splitting here gives one row per verse/prose line, matching
	// the granularity of SE Modern and Perseus Globe and dramatically improving
	// Jaccard similarity during alignment.
	var expanded []OSSParagraph
	for _, p := range paragraphs {
		parts := splitOSSLines(p.Text)
		if len(parts) <= 1 {
			p.Text = strings.TrimSpace(p.Text)
			expanded = append(expanded, p)
			continue
		}
		for _, line := range parts {
			sub := p
			sub.Text = line
			sub.WordCount = len(strings.Fields(line))
			expanded = append(expanded, sub)
		}
	}
	paragraphs = expanded

	// Compute scene-relative line numbers
	// Sort by (WorkID, Section, Chapter, ParagraphNum) for stable ordering
	sort.Slice(paragraphs, func(i, j int) bool {
		if paragraphs[i].WorkID != paragraphs[j].WorkID {
			return paragraphs[i].WorkID < paragraphs[j].WorkID
		}
		if paragraphs[i].Section != paragraphs[j].Section {
			return paragraphs[i].Section < paragraphs[j].Section
		}
		if paragraphs[i].Chapter != paragraphs[j].Chapter {
			return paragraphs[i].Chapter < paragraphs[j].Chapter
		}
		return paragraphs[i].ParagraphNum < paragraphs[j].ParagraphNum
	})

	prevWork := ""
	prevSection := -1
	prevChapter := -1
	lineNum := 0
	for i := range paragraphs {
		if paragraphs[i].WorkID != prevWork || paragraphs[i].Section != prevSection || paragraphs[i].Chapter != prevChapter {
			lineNum = 0
			prevWork = paragraphs[i].WorkID
			prevSection = paragraphs[i].Section
			prevChapter = paragraphs[i].Chapter
		}
		lineNum++
		paragraphs[i].LineNumber = lineNum
	}

	// Insert works
	fmt.Printf("  Works: %d\n", len(works))
	for _, w := range works {
		schmidt := constants.OSSToSchmidt[w.OSSID]
		var perseusID *string
		if sw, ok := constants.SchmidtWorks[schmidt]; ok {
			perseusID = &sw.PerseusID
		}
		_, err := database.Exec(`
			INSERT OR IGNORE INTO works (oss_id, title, full_title, short_title, schmidt_abbrev,
				work_type, date_composed, genre_type, total_words, total_paragraphs, source_text, perseus_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			w.OSSID, w.Title, w.FullTitle, w.ShortTitle, nilIfEmpty(schmidt),
			w.WorkType, w.DateComposed, w.GenreType,
			w.TotalWords, w.TotalParagraphs, w.SourceText, perseusID)
		if err != nil {
			return fmt.Errorf("inserting work %s: %w", w.OSSID, err)
		}
	}

	// Insert biblical/external reference works cited by Schmidt.
	// These have no text_lines but are needed so lexicon_citations can resolve work_id.
	insertExternalReferenceWorks(database)

	// Build work ID map
	workIDMap := make(map[string]int64)
	rows, err := database.Query("SELECT id, oss_id FROM works")
	if err != nil {
		return fmt.Errorf("querying works: %w", err)
	}
	for rows.Next() {
		var id int64
		var ossID string
		rows.Scan(&id, &ossID)
		workIDMap[ossID] = id
	}
	rows.Close()

	// Insert characters. OSS stores multi-play characters with comma-separated
	// work IDs (e.g., "henry4p1,henry4p2,henry5,merrywives" for Falstaff).
	// Insert one row per work so that character name expansion works for each play.
	fmt.Printf("  Characters: %d\n", len(characters))
	for _, c := range characters {
		ossWorkIDs := strings.Split(c.OSSWorkID, ",")
		for _, ossWID := range ossWorkIDs {
			dbWorkID := workIDMap[ossWID]
			charID := c.CharID
			if len(ossWorkIDs) > 1 {
				charID = c.CharID + "-" + ossWID
			}
			_, err := database.Exec(`
				INSERT OR IGNORE INTO characters (char_id, name, abbrev, work_id, oss_work_id, description, speech_count)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				charID, c.Name, c.Abbrev, nilIfZero(dbWorkID), c.OSSWorkID, c.Description, c.SpeechCount)
			if err != nil {
				return fmt.Errorf("inserting character %s: %w", c.CharID, err)
			}
		}
	}

	// Build character maps
	charIDMap := make(map[string]int64)
	charNameMap := make(map[string]string)
	charRows, _ := database.Query("SELECT id, char_id FROM characters")
	for charRows.Next() {
		var id int64
		var charID string
		charRows.Scan(&id, &charID)
		charIDMap[charID] = id
	}
	charRows.Close()
	for _, c := range characters {
		charNameMap[c.CharID] = c.Name
	}

	// Insert text divisions from chapters
	fmt.Printf("  Chapters (divisions): %d\n", len(chapters))
	for _, ch := range chapters {
		dbWorkID := workIDMap[ch.WorkID]
		if dbWorkID > 0 {
			database.Exec(`
				INSERT OR IGNORE INTO text_divisions (work_id, edition_id, act, scene, description)
				VALUES (?, ?, ?, ?, ?)`,
				dbWorkID, editionID, ch.Section, ch.Chapter, ch.Description)
		}
	}

	// Insert text lines from paragraphs
	fmt.Printf("  Paragraphs (text lines): %d\n", len(paragraphs))
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num, line_number,
			character_id, char_name, content, content_type, word_count, oss_paragraph_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("preparing statement: %w", err)
	}

	for _, p := range paragraphs {
		dbWorkID := workIDMap[p.WorkID]
		if dbWorkID == 0 {
			continue
		}
		charDBID := charIDMap[p.CharID]
		charName := charNameMap[p.CharID]

		ct := contentType(p.Type == "d")

		_, err := stmt.Exec(
			dbWorkID, editionID, p.Section, p.Chapter, p.ParagraphNum, p.LineNumber,
			nilIfZero(charDBID), nilIfEmpty(charName), p.Text, ct,
			p.WordCount, p.ParagraphID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("inserting paragraph %d: %w", p.ParagraphID, err)
		}
	}
	stmt.Close()
	tx.Commit()

	elapsed := time.Since(start).Seconds()
	lineCount, _ := db.TableCount(database, "text_lines")
	db.LogImport(database, "oss", "import_complete",
		fmt.Sprintf("%d works, %d chars", len(works), len(characters)),
		lineCount, elapsed)

	fmt.Printf("  ✓ %d text lines imported in %.1fs\n", lineCount, elapsed)
	return nil
}

// externalRefWork defines a non-Shakespeare work to insert into the works table.
type externalRefWork struct {
	Title    string
	WorkType string
}

// externalWorks maps raw_bibl prefixes to external works referenced in the Schmidt lexicon.
// These works have no text_lines but allow lexicon_citations to resolve work_id.
var externalWorks = map[string]externalRefWork{
	// Biblical references
	"Genesis":              {Title: "Genesis", WorkType: "biblical_reference"},
	"Exodus":               {Title: "Exodus", WorkType: "biblical_reference"},
	"Samuel":               {Title: "Samuel", WorkType: "biblical_reference"},
	"Job":                  {Title: "Job", WorkType: "biblical_reference"},
	"Esther":               {Title: "Esther", WorkType: "biblical_reference"},
	"Ecclesiastes":         {Title: "Ecclesiastes", WorkType: "biblical_reference"},
	"Proverbs":             {Title: "Proverbs", WorkType: "biblical_reference"},
	"Isaiah":               {Title: "Isaiah", WorkType: "biblical_reference"},
	"Jeremiah":             {Title: "Jeremiah", WorkType: "biblical_reference"},
	"Daniel":               {Title: "Daniel", WorkType: "biblical_reference"},
	"Matthew":              {Title: "Matthew", WorkType: "biblical_reference"},
	"Mark":                 {Title: "Mark", WorkType: "biblical_reference"},
	"Luke":                 {Title: "Luke", WorkType: "biblical_reference"},
	"Acts of the Apostles": {Title: "Acts of the Apostles", WorkType: "biblical_reference"},
	"Acts":                 {Title: "Acts of the Apostles", WorkType: "biblical_reference"},
	"1 Corinthians":        {Title: "1 Corinthians", WorkType: "biblical_reference"},
	"Epistle to the Hebr.": {Title: "Epistle to the Hebrews", WorkType: "biblical_reference"},
	// Classical references
	"Hom. Od.": {Title: "Homer, Odyssey", WorkType: "classical_reference"},
	"Pliny":    {Title: "Pliny, Natural History", WorkType: "classical_reference"},
	// Shakespeare apocrypha / disputed
	"Edward III": {Title: "Edward III", WorkType: "apocrypha"},
	// Schmidt lexicon appendix (cross-references within the lexicon itself)
	"Appendix": {Title: "Schmidt Lexicon Appendix", WorkType: "lexicon_appendix"},
	"Append.":  {Title: "Schmidt Lexicon Appendix", WorkType: "lexicon_appendix"},
	"App.":     {Title: "Schmidt Lexicon Appendix", WorkType: "lexicon_appendix"},
}

// insertExternalReferenceWorks adds biblical, classical, and other non-Shakespeare works
// to the works table. These are cited in the Schmidt lexicon but have no OSS source.
func insertExternalReferenceWorks(database *sql.DB) {
	// Deduplicate: multiple raw_bibl prefixes can map to the same title.
	seen := make(map[string]bool)
	inserted := 0
	for _, w := range externalWorks {
		if seen[w.Title] {
			continue
		}
		seen[w.Title] = true
		res, err := database.Exec(`
			INSERT OR IGNORE INTO works (title, work_type)
			VALUES (?, ?)`, w.Title, w.WorkType)
		if err == nil {
			n, _ := res.RowsAffected()
			inserted += int(n)
		}
	}
	if inserted > 0 {
		fmt.Printf("  External reference works: %d\n", inserted)
	}
}

// resolveUnmatchedCitations sets work_id on lexicon_citations where the XML parser
// couldn't extract a work abbreviation. Handles two cases:
//  1. External works (biblical, classical, etc.) — matched by raw_bibl prefix.
//  2. Shakespeare poems (Phoen., Lucr.) — matched by raw_bibl prefix against
//     existing works via schmidt_abbrev.
func resolveUnmatchedCitations(database *sql.DB) int {
	total := 0

	// --- Case 1: External works (raw_bibl prefix → external work title) ---
	// Build title → work_id map for external works.
	extRows, err := database.Query(`SELECT id, title FROM works WHERE work_type IN ('biblical_reference', 'classical_reference', 'apocrypha', 'lexicon_appendix')`)
	if err == nil {
		titleToID := make(map[string]int64)
		for extRows.Next() {
			var id int64
			var title string
			extRows.Scan(&id, &title)
			titleToID[title] = id
		}
		extRows.Close()

		for prefix, w := range externalWorks {
			workID, ok := titleToID[w.Title]
			if !ok {
				continue
			}
			res, err := database.Exec(`
				UPDATE lexicon_citations
				SET work_id = ?, work_abbrev = ?
				WHERE work_id IS NULL AND raw_bibl LIKE ?`,
				workID, prefix, prefix+"%")
			if err == nil {
				n, _ := res.RowsAffected()
				total += int(n)
			}
		}
	}

	// --- Case 2: Shakespeare poems with unresolved raw_bibl ---
	// These have raw_bibl like "Phoen. " or "Lucr. 704" but no work_abbrev.
	// Match by raw_bibl prefix against works.schmidt_abbrev.
	schmidtRows, err := database.Query(`SELECT id, schmidt_abbrev FROM works WHERE schmidt_abbrev IS NOT NULL`)
	if err == nil {
		for schmidtRows.Next() {
			var id int64
			var abbrev string
			schmidtRows.Scan(&id, &abbrev)
			res, err := database.Exec(`
				UPDATE lexicon_citations
				SET work_id = ?, work_abbrev = ?
				WHERE work_id IS NULL AND raw_bibl LIKE ?`,
				id, abbrev, abbrev+"%")
			if err == nil {
				n, _ := res.RowsAffected()
				total += int(n)
			}
		}
		schmidtRows.Close()
	}

	// --- Case 3: Extract line numbers from raw_bibl for poem citations ---
	// Citations like "Phoen. 21" or "Lucr. 886" have work_id set but line IS NULL.
	// The line number is the trailing number in raw_bibl after the abbreviation.
	lineRows, err := database.Query(`
		SELECT lc.id, lc.raw_bibl
		FROM lexicon_citations lc
		JOIN works w ON w.id = lc.work_id
		WHERE lc.line IS NULL
		  AND w.work_type = 'poem'
		  AND lc.raw_bibl IS NOT NULL`)
	if err == nil {
		type lineUpdate struct {
			id   int64
			line int
		}
		var updates []lineUpdate
		for lineRows.Next() {
			var id int64
			var rawBibl string
			lineRows.Scan(&id, &rawBibl)
			// Extract trailing number from raw_bibl (e.g., "Phoen. 21" → 21)
			if lineNum := extractTrailingNumber(rawBibl); lineNum > 0 {
				updates = append(updates, lineUpdate{id, lineNum})
			}
		}
		lineRows.Close()

		for _, u := range updates {
			res, err := database.Exec(`UPDATE lexicon_citations SET line = ? WHERE id = ?`, u.line, u.id)
			if err == nil {
				n, _ := res.RowsAffected()
				total += int(n)
			}
		}
	}

	return total
}

// extractTrailingNumber returns the last whitespace-delimited integer from s,
// or 0 if none found. Used to parse "Phoen. 21" → 21, "Lucr. 886" → 886.
func extractTrailingNumber(s string) int {
	fields := strings.Fields(s)
	for i := len(fields) - 1; i >= 0; i-- {
		if v, err := strconv.Atoi(fields[i]); err == nil {
			return v
		}
	}
	return 0
}

