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

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/constants"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/parser"
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

// ImportOSS imports the OSS/Moby Shakespeare data from a MySQL dump file.
func ImportOSS(database *sql.DB, sqlPath string) error {
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("STEP 1: Import OSS/Moby Shakespeare")
	fmt.Println("=" + strings.Repeat("=", 59))

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

	// Insert characters
	fmt.Printf("  Characters: %d\n", len(characters))
	for _, c := range characters {
		dbWorkID := workIDMap[c.OSSWorkID]
		_, err := database.Exec(`
			INSERT OR IGNORE INTO characters (char_id, name, abbrev, work_id, oss_work_id, description, speech_count)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			c.CharID, c.Name, c.Abbrev, nilIfZero(dbWorkID), c.OSSWorkID, c.Description, c.SpeechCount)
		if err != nil {
			return fmt.Errorf("inserting character %s: %w", c.CharID, err)
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

		contentType := "speech"
		if p.Type == "d" {
			contentType = "stage_direction"
		}

		_, err := stmt.Exec(
			dbWorkID, editionID, p.Section, p.Chapter, p.ParagraphNum, p.LineNumber,
			nilIfZero(charDBID), nilIfEmpty(charName), p.Text, contentType,
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

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nilIfZero(n int64) interface{} {
	if n == 0 {
		return nil
	}
	return n
}
