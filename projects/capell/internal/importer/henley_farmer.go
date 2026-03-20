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

// ImportHenleyFarmer imports Henley & Farmer's Slang and Its Analogues
// (1890-1904, 7 volumes) from the plain-text OCR files at
// sources/henley-farmer/slang-vol01.txt through slang-vol07.txt.
//
// Only entries that cite Shakespeare (contain "Shak" case-insensitively) are
// imported. Each entry starts with an ALL-CAPS headword at column 0 and runs
// until the next all-caps headword. The raw entry text is stored as-is;
// citations are extracted later by ResolveReferenceCitations.
func ImportHenleyFarmer(database *sql.DB, sourcesDir string) error {
	stepBanner("STEP: Import Henley & Farmer's Slang Dictionary")

	srcID, err := db.GetSourceID(database,
		"Henley & Farmer's Slang and Its Analogues (1890-1904)", "henley_farmer",
		"https://archive.org/details/slangitsanalogues01henl",
		"Public Domain", "", "",
		false,
		"John S. Farmer and W.E. Henley, Slang and Its Analogues Past and Present, 7 vols., 1890-1904. Shakespeare citations extracted. Public domain.")
	if err != nil {
		return err
	}

	start := time.Now()

	totalInserted := 0
	for vol := 1; vol <= 7; vol++ {
		filename := fmt.Sprintf("slang-vol%02d.txt", vol)
		txtPath := filepath.Join(sourcesDir, "henley-farmer", filename)

		if _, err := os.Stat(txtPath); err != nil {
			fmt.Printf("  WARNING: volume not found at %s — skipping\n", txtPath)
			continue
		}

		entries, err := parseHenleyFarmerEntries(txtPath)
		if err != nil {
			return fmt.Errorf("parsing Henley-Farmer vol %d: %w", vol, err)
		}

		fmt.Printf("  Vol %d: %d Shakespeare entries\n", vol, len(entries))

		if len(entries) == 0 {
			continue
		}

		inserted, err := insertReferenceEntries(database, srcID, entries)
		if err != nil {
			return fmt.Errorf("inserting Henley-Farmer vol %d: %w", vol, err)
		}
		totalInserted += inserted
	}

	elapsed := time.Since(start).Seconds()
	fmt.Printf("  ✓ Inserted %d Henley-Farmer Shakespeare entries in %.1fs\n", totalInserted, elapsed)

	db.LogImport(database, "henley_farmer", "import_complete",
		fmt.Sprintf("inserted %d Shakespeare entries from 7 volumes", totalInserted),
		totalInserted, elapsed)

	return nil
}

// parseHenleyFarmerEntries reads a Henley & Farmer volume OCR text file and
// returns only entries that contain "Shak" (Shakespeare citations).
//
// H&F entries start with an ALL-CAPS headword at column 0. Everything up to
// the next all-caps headword is the entry body.
func parseHenleyFarmerEntries(path string) ([]referenceEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 512*1024)
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
		// Only keep entries that reference Shakespeare.
		hw := strings.ToLower(curHeadword)
		if raw != "" && strings.Contains(strings.ToLower(raw), "shak") {
			entries = append(entries, referenceEntry{
				headword: hw,
				letter:   headwordLetter(hw),
				rawText:  raw,
			})
		}
		curHeadword = ""
		curLines = nil
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Detect a new headword: all-caps at column 0, at least 2 chars.
		if isHenleyFarmerHeadwordStart(line) {
			flush()
			// Extract headword: everything before the first non-caps/non-space char
			// or just the first word(s) until we hit definition text.
			curHeadword = extractHenleyFarmerHeadword(line)
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

// isHenleyFarmerHeadwordStart returns true if line starts a new H&F entry.
// Criteria: no leading whitespace, at least 2 characters, all-caps.
func isHenleyFarmerHeadwordStart(line string) bool {
	if len(line) < 2 {
		return false
	}
	// Must be at column 0.
	if line[0] == ' ' || line[0] == '\t' {
		return false
	}
	// Must start with an uppercase letter.
	if !unicode.IsUpper(rune(line[0])) {
		return false
	}
	// Check that the first "word" (sequence of letters up to first space or
	// punctuation other than hyphen/apostrophe) is all-caps.
	word := extractFirstWord(line)
	if len(word) < 2 {
		return false
	}
	return isAllCaps(word)
}

// extractFirstWord returns the first sequence of non-whitespace characters
// from s (i.e. up to the first space).
func extractFirstWord(s string) string {
	for i, r := range s {
		if unicode.IsSpace(r) {
			return s[:i]
		}
	}
	return s
}

// extractHenleyFarmerHeadword returns the headword from an H&F entry-start
// line. It takes the leading ALL-CAPS portion before a comma, period, or
// the start of mixed-case definition text.
func extractHenleyFarmerHeadword(line string) string {
	// Collect leading uppercase / punctuation until we hit lowercase.
	end := len(line)
	for i, r := range line {
		if unicode.IsLower(r) {
			end = i
			break
		}
	}
	hw := strings.TrimRight(line[:end], " ,.-;:()")
	fields := strings.Fields(hw)
	if len(fields) == 0 {
		return strings.ToLower(strings.TrimSpace(line))
	}
	return strings.ToLower(strings.Join(fields, " "))
}
