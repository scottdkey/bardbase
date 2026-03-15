// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/parser"
)

// citationRow holds a lexicon citation loaded for resolution.
type citationRow struct {
	ID        int64
	EntryID   int64
	SenseID   *int64
	WorkID    int64
	Act       *int
	Scene     *int
	Line      *int
	QuoteText string
}

// textLineRow holds a text line loaded for citation matching.
type textLineRow struct {
	ID         int64
	Content    string
	LineNumber int
	EditionID  int64
}

// ResolveCitations matches lexicon citations to actual text_lines rows.
//
// Handles three reference types:
//   - Play citations: act + scene + optional line (e.g., "Tp. I, 1, 52")
//   - Sonnet citations: scene (= sonnet number) + line, no act (e.g., "Son. 1, 14")
//   - Poem citations: just line number, no act/scene (e.g., "Ven. 52")
//
// For each citation with a valid work reference, it searches for matching
// text lines in each edition using line numbers and/or quote text.
// Results are stored in the citation_matches table.
func ResolveCitations(database *sql.DB) error {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("STEP 6: Resolve Lexicon Citations → Text Lines")
	fmt.Println("=" + strings.Repeat("=", 59))

	start := time.Now()

	// Clear existing matches for a clean rebuild
	database.Exec("DELETE FROM citation_matches")

	// Get all editions
	type editionInfo struct {
		ID   int64
		Code string
	}
	var editions []editionInfo
	edRows, err := database.Query("SELECT id, short_code FROM editions")
	if err != nil {
		return fmt.Errorf("querying editions: %w", err)
	}
	for edRows.Next() {
		var e editionInfo
		edRows.Scan(&e.ID, &e.Code)
		editions = append(editions, e)
	}
	edRows.Close()

	if len(editions) == 0 {
		fmt.Println("  No editions found, skipping")
		return nil
	}

	// Load ALL citations that have at least a work_id and some location info.
	// Previously required act IS NOT NULL which excluded sonnets and poems.
	citRows, err := database.Query(`
		SELECT id, entry_id, sense_id, work_id, act, scene, line, quote_text
		FROM lexicon_citations
		WHERE work_id IS NOT NULL
		  AND (act IS NOT NULL OR scene IS NOT NULL OR line IS NOT NULL OR quote_text IS NOT NULL)`)
	if err != nil {
		return fmt.Errorf("querying citations: %w", err)
	}

	var citations []citationRow
	for citRows.Next() {
		var c citationRow
		var senseID sql.NullInt64
		var act, scene, line sql.NullInt64
		var quoteText sql.NullString

		citRows.Scan(&c.ID, &c.EntryID, &senseID, &c.WorkID, &act, &scene, &line, &quoteText)

		if senseID.Valid {
			c.SenseID = &senseID.Int64
		}
		if act.Valid {
			v := int(act.Int64)
			c.Act = &v
		}
		if scene.Valid {
			v := int(scene.Int64)
			c.Scene = &v
		}
		if line.Valid {
			v := int(line.Int64)
			c.Line = &v
		}
		if quoteText.Valid {
			c.QuoteText = quoteText.String
		}

		citations = append(citations, c)
	}
	citRows.Close()

	fmt.Printf("  Citations to resolve: %d\n", len(citations))
	if len(citations) == 0 {
		fmt.Println("  No citations to resolve")
		return nil
	}

	// Classify citations into three types for different resolution strategies:
	//   play:   act is set → group by (work_id, act, scene)
	//   sonnet: scene is set, act is nil → group by (work_id, scene)
	//   poem:   only line and/or quote_text → group by (work_id)

	type playKey struct {
		WorkID int64
		Act    int
		Scene  int
	}
	type sonnetKey struct {
		WorkID int64
		Scene  int // sonnet number
	}
	type poemKey struct {
		WorkID int64
	}

	playCitations := make(map[playKey][]citationRow)
	sonnetCitations := make(map[sonnetKey][]citationRow)
	poemCitations := make(map[poemKey][]citationRow)

	for _, c := range citations {
		if c.Act != nil {
			// Play citation: has act (and usually scene)
			scn := 0
			if c.Scene != nil {
				scn = *c.Scene
			}
			key := playKey{WorkID: c.WorkID, Act: *c.Act, Scene: scn}
			playCitations[key] = append(playCitations[key], c)
		} else if c.Scene != nil {
			// Sonnet citation: has scene (sonnet number) but no act
			key := sonnetKey{WorkID: c.WorkID, Scene: *c.Scene}
			sonnetCitations[key] = append(sonnetCitations[key], c)
		} else {
			// Poem citation: only line and/or quote_text
			key := poemKey{WorkID: c.WorkID}
			poemCitations[key] = append(poemCitations[key], c)
		}
	}

	fmt.Printf("    Play citations: %d groups\n", len(playCitations))
	fmt.Printf("    Sonnet citations: %d groups\n", len(sonnetCitations))
	fmt.Printf("    Poem citations: %d groups\n", len(poemCitations))

	// Process all types
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	insertStmt, err := tx.Prepare(`
		INSERT INTO citation_matches (citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("preparing insert: %w", err)
	}

	totalMatches := 0
	exactMatches := 0
	lineMatches := 0
	fuzzyMatches := 0
	noMatches := 0

	matchCitation := func(cit citationRow, lines []textLineRow) {
		// Group lines by edition
		linesByEdition := make(map[int64][]textLineRow)
		for _, tl := range lines {
			linesByEdition[tl.EditionID] = append(linesByEdition[tl.EditionID], tl)
		}

		for editionID, edLines := range linesByEdition {
			matchLine, matchType, confidence := findBestMatch(edLines, cit)
			if matchLine == nil {
				noMatches++
				continue
			}

			insertStmt.Exec(cit.ID, matchLine.ID, editionID, matchType, confidence, matchLine.Content)
			totalMatches++

			switch matchType {
			case "exact_quote":
				exactMatches++
			case "line_number":
				lineMatches++
			case "fuzzy_text":
				fuzzyMatches++
			}
		}
	}

	// === Resolve play citations ===
	for key, sceneCitations := range playCitations {
		var query string
		var args []interface{}
		if key.Scene > 0 {
			query = `SELECT id, content, COALESCE(line_number, 0), edition_id FROM text_lines
				WHERE work_id = ? AND act = ? AND scene = ? AND line_number IS NOT NULL
				ORDER BY edition_id, line_number`
			args = []interface{}{key.WorkID, key.Act, key.Scene}
		} else {
			query = `SELECT id, content, COALESCE(line_number, 0), edition_id FROM text_lines
				WHERE work_id = ? AND act = ? AND line_number IS NOT NULL
				ORDER BY edition_id, line_number`
			args = []interface{}{key.WorkID, key.Act}
		}

		lineRows, err := database.Query(query, args...)
		if err != nil {
			continue
		}

		var lines []textLineRow
		for lineRows.Next() {
			var tl textLineRow
			lineRows.Scan(&tl.ID, &tl.Content, &tl.LineNumber, &tl.EditionID)
			lines = append(lines, tl)
		}
		lineRows.Close()

		for _, cit := range sceneCitations {
			matchCitation(cit, lines)
		}
	}

	// === Resolve sonnet citations ===
	// Sonnet citations have scene = sonnet number, line = line within sonnet
	for key, sCitations := range sonnetCitations {
		lineRows, err := database.Query(`
			SELECT id, content, COALESCE(line_number, 0), edition_id FROM text_lines
			WHERE work_id = ? AND scene = ? AND line_number IS NOT NULL
			ORDER BY edition_id, line_number`,
			key.WorkID, key.Scene)
		if err != nil {
			continue
		}

		var lines []textLineRow
		for lineRows.Next() {
			var tl textLineRow
			lineRows.Scan(&tl.ID, &tl.Content, &tl.LineNumber, &tl.EditionID)
			lines = append(lines, tl)
		}
		lineRows.Close()

		for _, cit := range sCitations {
			matchCitation(cit, lines)
		}
	}

	// === Resolve poem citations ===
	// Poem citations typically have only line number (e.g., "Ven. 52" → line 52)
	// Load all lines for the work and match by line_number or fuzzy text
	for key, pCitations := range poemCitations {
		lineRows, err := database.Query(`
			SELECT id, content, COALESCE(line_number, 0), edition_id FROM text_lines
			WHERE work_id = ? AND line_number IS NOT NULL
			ORDER BY edition_id, line_number`,
			key.WorkID)
		if err != nil {
			continue
		}

		var lines []textLineRow
		for lineRows.Next() {
			var tl textLineRow
			lineRows.Scan(&tl.ID, &tl.Content, &tl.LineNumber, &tl.EditionID)
			lines = append(lines, tl)
		}
		lineRows.Close()

		for _, cit := range pCitations {
			matchCitation(cit, lines)
		}
	}

	insertStmt.Close()
	tx.Commit()

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "citations", "resolve_complete",
		fmt.Sprintf("%d matches (exact=%d, line=%d, fuzzy=%d, unmatched=%d)",
			totalMatches, exactMatches, lineMatches, fuzzyMatches, noMatches),
		totalMatches, elapsed)

	fmt.Printf("  ✓ %d matches resolved in %.1fs\n", totalMatches, elapsed)
	fmt.Printf("    exact_quote: %d, line_number: %d, fuzzy_text: %d, unmatched: %d\n",
		exactMatches, lineMatches, fuzzyMatches, noMatches)
	return nil
}

// findBestMatch finds the best matching text line for a citation.
// Returns the matched line, match type, and confidence score.
func findBestMatch(lines []textLineRow, cit citationRow) (*textLineRow, string, float64) {
	if len(lines) == 0 {
		return nil, "", 0
	}

	// Strategy 1: Exact quote match (highest confidence)
	if cit.QuoteText != "" {
		// Clean up Schmidt quote text (remove -- abbreviations)
		cleanQuote := strings.ReplaceAll(cit.QuoteText, "--", "")
		cleanQuote = strings.TrimSpace(cleanQuote)

		if len(cleanQuote) > 3 {
			for i, line := range lines {
				if parser.ContainsNormalized(line.Content, cleanQuote) {
					return &lines[i], "exact_quote", 1.0
				}
			}
		}
	}

	// Strategy 2: Line number match
	if cit.Line != nil {
		for i, line := range lines {
			if line.LineNumber == *cit.Line {
				// If we also have quote text, verify with fuzzy match
				if cit.QuoteText != "" {
					sim := parser.JaccardSimilarity(line.Content, cit.QuoteText)
					if sim > 0.15 {
						return &lines[i], "line_number", 0.9
					}
					// Line number matched but text is very different — still return but lower confidence
					return &lines[i], "line_number", 0.7
				}
				return &lines[i], "line_number", 0.9
			}
		}

		// Try nearby lines (±3) if exact line number didn't match
		for delta := 1; delta <= 3; delta++ {
			for i, line := range lines {
				if line.LineNumber == *cit.Line+delta || line.LineNumber == *cit.Line-delta {
					if cit.QuoteText != "" {
						sim := parser.JaccardSimilarity(line.Content, cit.QuoteText)
						if sim > 0.2 {
							return &lines[i], "line_number", 0.7
						}
					} else {
						return &lines[i], "line_number", 0.6 - float64(delta)*0.1
					}
				}
			}
		}
	}

	// Strategy 3: Fuzzy text match (last resort)
	if cit.QuoteText != "" {
		bestScore := 0.0
		bestIdx := -1
		for i, line := range lines {
			score := parser.JaccardSimilarity(line.Content, cit.QuoteText)
			if score > bestScore {
				bestScore = score
				bestIdx = i
			}
		}
		if bestScore > 0.25 && bestIdx >= 0 {
			return &lines[bestIdx], "fuzzy_text", bestScore
		}
	}

	return nil, "", 0
}
