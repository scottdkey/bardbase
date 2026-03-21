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

// ImportOnions imports C. T. Onions' Shakespeare Glossary (1911 edition) from
// the plain-text OCR file at sources/onions/shakespeare-glossary-1911.txt.
//
// The file is an OCR scan of a two-column glossary. Each entry begins with a
// headword at column 0 (possibly followed by a colon, parenthetical note, or
// part-of-speech tag) and may span multiple lines. Numbered senses (1, 2, ...)
// and continuation lines are accumulated into the same entry block.
//
// The raw entry text is stored as-is; no attempt is made to parse individual
// senses or citations in this pass.
func ImportOnions(database *sql.DB, sourcesDir string) error {
	stepBanner("Import Onions Shakespeare Glossary")

	txtPath := filepath.Join(sourcesDir, "onions", "shakespeare-glossary-1911.txt")
	if _, err := os.Stat(txtPath); err != nil {
		fmt.Printf("  WARNING: Onions source not found at %s\n", txtPath)
		fmt.Println("  Skipping Onions import")
		return nil
	}

	srcID, err := db.GetSourceID(database,
		"Onions Shakespeare Glossary (1911)", "onions",
		"https://archive.org/details/shakespearegloss00oniouoft",
		"Public Domain", "", "",
		false,
		"C. T. Onions, A Shakespeare Glossary, Oxford: Clarendon Press, 1911. 1911 first edition (public domain). Note: the revised 1986 Oxford edition is still under copyright.")
	if err != nil {
		return err
	}

	start := time.Now()

	entries, err := parseOnionsEntries(txtPath)
	if err != nil {
		return fmt.Errorf("parsing Onions: %w", err)
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
	fmt.Printf("  ✓ Inserted %d Onions entries in %.1fs\n", inserted, elapsed)

	db.LogImport(database, "onions", "import_complete",
		fmt.Sprintf("inserted %d entries from %s", inserted, txtPath),
		inserted, elapsed)

	return nil
}

// parseOnionsEntries reads the Onions OCR text file and splits it into
// individual glossary entries.
//
// Detection heuristic: a line starts a new entry when it:
//   - begins at column 0 (no leading whitespace),
//   - starts with a letter, hyphen, or apostrophe (not a digit),
//   - is not an ALL-CAPS section header,
//   - contains ':' or '(' within the first 80 characters.
//
// Everything from that line until the next entry start is the entry body.
// The glossary preamble (before the first matched entry) is discarded.
func parseOnionsEntries(path string) ([]referenceEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

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

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if isOnionsEntryStart(line) {
			flush()
			curHeadword = extractOnionsHeadword(line)
			curLines = []string{line}
		} else if curHeadword != "" {
			curLines = append(curLines, line)
		}
		// lines before the first entry are discarded
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// isOnionsEntryStart returns true if line looks like the start of a new
// Onions glossary entry.
func isOnionsEntryStart(line string) bool {
	if len(line) == 0 {
		return false
	}

	// Must start at column 0 — leading whitespace means continuation.
	if line[0] == ' ' || line[0] == '\t' {
		return false
	}

	first := rune(line[0])

	// Digits introduce numbered senses (1 to lessen...) — not entry starts.
	if unicode.IsDigit(first) {
		return false
	}

	// Must start with a letter, hyphen, or apostrophe.
	if !unicode.IsLetter(first) && first != '-' && first != '\'' {
		return false
	}

	// ALL-CAPS lines are section headers ("SHAKESPEARE GLOSSAEY", "A-", etc.).
	if isAllCaps(line) {
		return false
	}

	// Require ':' or '(' somewhere in the first 80 characters — this
	// distinguishes headword lines from bare continuation text.
	prefix := line
	if len(prefix) > 80 {
		prefix = prefix[:80]
	}
	return strings.ContainsAny(prefix, ":(")
}

// extractOnionsHeadword returns the headword portion of an entry-start line.
// It takes everything before the first ':' or '(' (or POS tag like "adj.",
// "vb.", "sb."), strips OCR spacing artifacts, and lowercases it.
func extractOnionsHeadword(line string) string {
	// Cut at the first ':' or '('.
	end := strings.IndexAny(line, ":(")
	if end < 0 {
		// Fall back: take first word.
		fields := strings.Fields(line)
		if len(fields) > 0 {
			return cleanHeadword(fields[0])
		}
		return ""
	}
	hw := line[:end]
	return cleanHeadword(hw)
}

// cleanHeadword normalises OCR spacing and trims a headword string.
func cleanHeadword(s string) string {
	// Collapse multiple spaces to one, trim surrounding space.
	fields := strings.Fields(s)
	return strings.ToLower(strings.Join(fields, " "))
}
