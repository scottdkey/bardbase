// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
)

// ImportAbbott imports E. A. Abbott's Shakespearian Grammar (1877 edition) from
// the plain-text OCR file at sources/abbott/shakespearian-grammar-1877.txt.
//
// The grammar is organised as numbered paragraphs (§1 – §515+). Each paragraph
// begins with a number and period at column 0. The raw paragraph text is stored
// as-is; citations are extracted later by ResolveReferenceCitations.
func ImportAbbott(database *sql.DB, sourcesDir string) error {
	stepBanner("Import Abbott Shakespearian Grammar")

	txtPath := filepath.Join(sourcesDir, "abbott", "shakespearian-grammar-1877.txt")
	if _, err := os.Stat(txtPath); err != nil {
		fmt.Printf("  WARNING: Abbott source not found at %s\n", txtPath)
		fmt.Println("  Skipping Abbott import")
		return nil
	}

	srcID, err := db.GetSourceID(database,
		"Abbott Shakespearian Grammar (1877)", "abbott",
		"https://archive.org/details/shakespeariangrammar00abbo",
		"Public Domain", "", "",
		false,
		"E. A. Abbott, A Shakespearian Grammar, London: Macmillan, 1877. Third edition (public domain). Citations reference the Globe edition of Shakespeare.")
	if err != nil {
		return err
	}

	start := time.Now()

	entries, err := parseAbbottEntries(txtPath)
	if err != nil {
		return fmt.Errorf("parsing Abbott: %w", err)
	}

	fmt.Printf("  Parsed %d paragraphs\n", len(entries))
	if len(entries) == 0 {
		fmt.Println("  No paragraphs found — check file format")
		return nil
	}

	inserted, err := insertReferenceEntries(database, srcID, entries)
	if err != nil {
		return err
	}

	elapsed := time.Since(start).Seconds()
	fmt.Printf("  ✓ Inserted %d Abbott paragraphs in %.1fs\n", inserted, elapsed)

	db.LogImport(database, "abbott", "import_complete",
		fmt.Sprintf("inserted %d paragraphs from %s", inserted, txtPath),
		inserted, elapsed)

	return nil
}

// parseAbbottEntries reads the Abbott OCR text file and splits it into
// numbered grammar paragraphs.
//
// Detection heuristic: a line starts a new paragraph when it:
//   - begins at column 0 (no leading whitespace),
//   - starts with one or more digits followed by a period and a space,
//   - the digit sequence is ≥ 1 and ≤ 520 (paragraph range in Abbott).
//
// The preamble (table of contents, preface, etc.) is discarded.
func parseAbbottEntries(path string) ([]referenceEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		entries    []referenceEntry
		curNum     int
		curLines   []string
		inGrammar  bool // true once we've passed the "GRAMMAR." header
	)

	flush := func() {
		if curNum == 0 || len(curLines) == 0 {
			return
		}
		raw := strings.TrimSpace(strings.Join(curLines, "\n"))
		if raw != "" {
			entries = append(entries, referenceEntry{
				headword: "§" + strconv.Itoa(curNum),
				letter:   "§",
				rawText:  raw,
			})
		}
		curNum = 0
		curLines = nil
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Wait until we pass the "GRAMMAR." section header to skip
		// the table of contents and prefaces.
		if !inGrammar {
			if strings.TrimSpace(line) == "GRAMMAR." {
				inGrammar = true
			}
			continue
		}

		// Stop at the quotation index (end of main grammar text).
		if strings.Contains(line, "INDEX TO THE QUOTATIONS") {
			break
		}

		if n, ok := abbottParaStart(line); ok {
			flush()
			curNum = n
			curLines = []string{line}
		} else if curNum != 0 {
			curLines = append(curLines, line)
		}
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// abbottParaStart returns (paragraphNumber, true) if line begins a new
// numbered paragraph (e.g. "1.  Adjectives...", "52.  Nouns...").
// Returns (0, false) otherwise.
func abbottParaStart(line string) (int, bool) {
	if len(line) == 0 {
		return 0, false
	}
	// Must start at column 0 — no leading whitespace.
	if line[0] == ' ' || line[0] == '\t' {
		return 0, false
	}
	// Must start with a digit.
	if !unicode.IsDigit(rune(line[0])) {
		return 0, false
	}

	// Find the period that terminates the number.
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 {
		return 0, false
	}
	numStr := line[:dotIdx]
	// All characters before the dot must be digits.
	for _, ch := range numStr {
		if !unicode.IsDigit(ch) {
			return 0, false
		}
	}
	n, err := strconv.Atoi(numStr)
	if err != nil || n < 1 || n > 520 {
		return 0, false
	}
	// The character after the dot must be a space (or end of line),
	// distinguishing "1.  Adjectives" from cross-references like "1870."
	rest := line[dotIdx+1:]
	if len(rest) > 0 && rest[0] != ' ' && rest[0] != '\t' {
		return 0, false
	}
	return n, true
}
