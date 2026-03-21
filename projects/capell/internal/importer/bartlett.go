// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
)

// ImportBartlett imports John Bartlett's Complete Concordance to Shakespeare
// (1896) from the plain-text OCR file at sources/bartlett/concordance-1896.txt.
//
// The concordance is organized by headword. Each headword group starts at
// column 0 with mixed-case text matching the pattern: word(s) followed by a
// period and two or more spaces (e.g. "Aaron.    Then, Aaron…"). ALL-CAPS
// lines are OCR column headers and are skipped. Each headword group's raw text
// is stored as-is; citations are extracted later by ResolveReferenceCitations.
func ImportBartlett(database *sql.DB, sourcesDir string) error {
	stepBanner("Import Bartlett's Shakespeare Concordance")

	txtPath := filepath.Join(sourcesDir, "bartlett", "concordance-1896.txt")
	if _, err := os.Stat(txtPath); err != nil {
		fmt.Printf("  WARNING: Bartlett source not found at %s\n", txtPath)
		fmt.Println("  Skipping Bartlett import")
		return nil
	}

	srcID, err := db.GetSourceID(database,
		"Bartlett's Shakespeare Concordance (1896)", "bartlett",
		"https://archive.org/details/completeconcordan00bart",
		"Public Domain", "", "",
		false,
		"John Bartlett, A Complete Concordance to Shakespeare's Dramatic Works and Poems, Macmillan, 1896. Concordance prepared from the Globe edition text. Public domain.")
	if err != nil {
		return err
	}

	start := time.Now()

	entries, err := parseBartlettEntries(txtPath)
	if err != nil {
		return fmt.Errorf("parsing Bartlett: %w", err)
	}

	fmt.Printf("  Parsed %d entries\n", len(entries))
	if len(entries) == 0 {
		fmt.Println("  No entries found — check file format")
		return nil
	}

	inserted, err := insertReferenceEntries(database, srcID, entries)
	if err != nil {
		return err
	}

	elapsed := time.Since(start).Seconds()
	fmt.Printf("  ✓ Inserted %d Bartlett entries in %.1fs\n", inserted, elapsed)

	db.LogImport(database, "bartlett", "import_complete",
		fmt.Sprintf("inserted %d entries from %s", inserted, txtPath),
		inserted, elapsed)

	return nil
}

// parseBartlettEntries reads the Bartlett OCR text file and splits it into
// individual concordance headword groups.
func parseBartlettEntries(path string) ([]referenceEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Use a large buffer for long concordance lines.
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, len(buf))

	var (
		entries     []referenceEntry
		curHeadword string
		curLines    []string
	)

	flush := func() {
		if curHeadword == "" {
			return
		}
		raw := strings.TrimSpace(strings.Join(curLines, "\n"))
		if raw != "" {
			entries = append(entries, referenceEntry{
				headword: curHeadword,
				letter:   headwordLetter(curHeadword),
				rawText:  raw,
			})
		}
		curHeadword = ""
		curLines = nil
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Skip blank lines.
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if curHeadword != "" {
				curLines = append(curLines, line)
			}
			continue
		}

		// Skip all-caps lines (OCR column headers like "AARON").
		if isAllCaps(trimmed) {
			continue
		}

		// Skip page-number-only lines (pure digits or roman numerals).
		if isPageNumber(trimmed) {
			continue
		}

		if hw, ok := isBartlettHeadwordStart(line); ok {
			flush()
			curHeadword = hw
			curLines = []string{line}
		} else if curHeadword != "" {
			curLines = append(curLines, line)
		}
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// isBartlettHeadwordStart returns (headword, true) if line starts a new
// Bartlett concordance headword group.
//
// Detection criteria:
//   - No leading whitespace (column 0)
//   - Not all-caps
//   - Starts with uppercase letter
//   - First word(s) followed by ".  " (period + 2+ spaces) within first 60 chars
//   - Headword (text before the ".  ") is at least 3 characters
func isBartlettHeadwordStart(line string) (string, bool) {
	if len(line) == 0 {
		return "", false
	}
	// Must be at column 0 — no leading whitespace.
	if line[0] == ' ' || line[0] == '\t' {
		return "", false
	}
	// Must start with an uppercase letter.
	if !unicode.IsUpper(rune(line[0])) {
		return "", false
	}
	// Must not be all-caps.
	if isAllCaps(line) {
		return "", false
	}

	// Look for ". " (period + 2+ spaces) within the first 60 chars.
	prefix := line
	if len(prefix) > 60 {
		prefix = prefix[:60]
	}
	idx := strings.Index(prefix, ".  ")
	if idx < 0 {
		return "", false
	}
	hw := strings.TrimSpace(line[:idx])
	if len(hw) < 3 {
		return "", false
	}
	return hw, true
}

// isPageNumber returns true if s consists only of digits or Roman numeral
// characters (used to skip page-number-only lines in the OCR output).
func isPageNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
		case r == 'i' || r == 'v' || r == 'x' || r == 'l' ||
			r == 'c' || r == 'd' || r == 'm' ||
			r == 'I' || r == 'V' || r == 'X' || r == 'L' ||
			r == 'C' || r == 'D' || r == 'M':
		default:
			return false
		}
	}
	return true
}
