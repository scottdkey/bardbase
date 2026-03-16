// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"

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
	Headword  string // lexicon_entries.key — used to expand abbreviated quotes
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
	stepBanner("STEP 6: Resolve Lexicon Citations → Text Lines")

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
	// Join with lexicon_entries to get headword for abbreviation expansion.
	citRows, err := database.Query(`
		SELECT lc.id, lc.entry_id, lc.sense_id, lc.work_id, lc.act, lc.scene, lc.line, lc.quote_text, le.key
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		WHERE lc.work_id IS NOT NULL
		  AND (lc.act IS NOT NULL OR lc.scene IS NOT NULL OR lc.line IS NOT NULL OR lc.quote_text IS NOT NULL)`)
	if err != nil {
		return fmt.Errorf("querying citations: %w", err)
	}

	var citations []citationRow
	for citRows.Next() {
		var c citationRow
		var senseID sql.NullInt64
		var act, scene, line sql.NullInt64
		var quoteText sql.NullString

		citRows.Scan(&c.ID, &c.EntryID, &senseID, &c.WorkID, &act, &scene, &line, &quoteText, &c.Headword)

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
			scn := 0
			if c.Scene != nil {
				scn = *c.Scene
			}
			key := playKey{WorkID: c.WorkID, Act: *c.Act, Scene: scn}
			playCitations[key] = append(playCitations[key], c)
		} else if c.Scene != nil {
			key := sonnetKey{WorkID: c.WorkID, Scene: *c.Scene}
			sonnetCitations[key] = append(sonnetCitations[key], c)
		} else {
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

	// === Resolve play citations (unified via loadTextLines) ===
	for key, sceneCitations := range playCitations {
		var lines []textLineRow
		var err error
		if key.Scene > 0 {
			lines, err = loadTextLines(database, "work_id = ? AND act = ? AND scene = ?",
				key.WorkID, key.Act, key.Scene)
		} else {
			lines, err = loadTextLines(database, "work_id = ? AND act = ?",
				key.WorkID, key.Act)
		}
		if err != nil {
			continue
		}

		// Fallback: if the scene returned 0 lines, the 2-part Perseus reference
		// (e.g., "shak. tmp 5.169") was probably act.line, not act.scene.
		// Reload the entire act and inject the scene value as each citation's line number.
		if len(lines) == 0 && key.Scene > 0 {
			lines, err = loadTextLines(database, "work_id = ? AND act = ?",
				key.WorkID, key.Act)
			if err != nil {
				continue
			}
			// Rewrite each citation in this group: scene was actually the line number.
			for i := range sceneCitations {
				if sceneCitations[i].Line == nil {
					lineNum := key.Scene
					sceneCitations[i].Line = &lineNum
				}
			}
		}

		for _, cit := range sceneCitations {
			matchCitation(cit, lines)
		}
	}

	// === Resolve sonnet citations (unified via loadTextLines) ===
	for key, sCitations := range sonnetCitations {
		lines, err := loadTextLines(database, "work_id = ? AND scene = ?",
			key.WorkID, key.Scene)
		if err != nil {
			continue
		}
		for _, cit := range sCitations {
			matchCitation(cit, lines)
		}
	}

	// === Resolve poem citations (unified via loadTextLines) ===
	for key, pCitations := range poemCitations {
		lines, err := loadTextLines(database, "work_id = ?", key.WorkID)
		if err != nil {
			continue
		}
		for _, cit := range pCitations {
			matchCitation(cit, lines)
		}
	}

	insertStmt.Close()
	tx.Commit()

	fmt.Printf("  Direct matching: %d matches (exact=%d, line=%d, fuzzy=%d, unmatched=%d)\n",
		totalMatches, exactMatches, lineMatches, fuzzyMatches, noMatches)

	// === Phase 2: Cross-edition propagation via line_mappings ===
	// Uses pre-built line_mappings to propagate matches from one edition to another.
	// If a citation matched line X in edition A, and line_mappings says X corresponds
	// to line Y in edition B, create a match for edition B too.
	propagated := propagateCrossEdition(database)
	totalMatches += propagated

	// === Phase 3: Headword search for truly unmatched citations ===
	// For citations with only act+scene (no line/quote), search for the lexicon
	// entry's headword within the scene text.
	headwordMatches := matchByHeadword(database)
	totalMatches += headwordMatches

	// Final stats from database
	var finalExact, finalLine, finalFuzzy, finalPropagated, finalHeadword, finalTotal int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'exact_quote'").Scan(&finalExact)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'line_number'").Scan(&finalLine)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'fuzzy_text'").Scan(&finalFuzzy)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'propagated'").Scan(&finalPropagated)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'headword'").Scan(&finalHeadword)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&finalTotal)

	var finalUnmatched int
	database.QueryRow(`
		SELECT COUNT(*) FROM lexicon_citations
		WHERE work_id IS NOT NULL
		  AND (act IS NOT NULL OR scene IS NOT NULL OR line IS NOT NULL OR quote_text IS NOT NULL)
		  AND id NOT IN (SELECT citation_id FROM citation_matches)`).Scan(&finalUnmatched)

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "citations", "resolve_complete",
		fmt.Sprintf("%d matches (exact=%d, line=%d, fuzzy=%d, propagated=%d, headword=%d, unmatched_citations=%d)",
			finalTotal, finalExact, finalLine, finalFuzzy, finalPropagated, finalHeadword, finalUnmatched),
		finalTotal, elapsed)

	fmt.Printf("  ✓ %d total matches in %.1fs\n", finalTotal, elapsed)
	fmt.Printf("    exact_quote: %d, line_number: %d, fuzzy_text: %d\n", finalExact, finalLine, finalFuzzy)
	fmt.Printf("    propagated: %d, headword: %d\n", finalPropagated, finalHeadword)
	fmt.Printf("    unmatched citations: %d of %d (%.1f%%)\n",
		finalUnmatched, len(citations), 100.0*float64(finalUnmatched)/float64(len(citations)))
	return nil
}

// propagateCrossEdition uses line_mappings to propagate citation matches across editions.
// If a citation matched line X in edition A, and line_mappings maps X → Y in edition B,
// a new "propagated" match is created for edition B.
//
// Runs multiple rounds because a match propagated from A→B in round 1 can then
// propagate from B→C in round 2 (for 3 editions, 2 rounds suffice).
func propagateCrossEdition(database *sql.DB) int {
	totalPropagated := 0

	// Run 2 rounds to handle transitive propagation (A→B, then B→C)
	for round := 0; round < 2; round++ {
		roundCount := 0

		// Forward: match is on line_a_id → propagate to line_b_id
		res, err := database.Exec(`
			INSERT INTO citation_matches (citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
			SELECT sub.citation_id, sub.text_line_id, sub.edition_id, 'propagated', sub.confidence, sub.matched_text
			FROM (
				SELECT
					cm.citation_id,
					lm.line_b_id AS text_line_id,
					lm.edition_b_id AS edition_id,
					cm.confidence * CASE WHEN lm.similarity >= 0.2 THEN lm.similarity ELSE 0.3 END AS confidence,
					tl.content AS matched_text,
					ROW_NUMBER() OVER (
						PARTITION BY cm.citation_id, lm.edition_b_id
						ORDER BY cm.confidence * lm.similarity DESC
					) AS rn
				FROM citation_matches cm
				JOIN line_mappings lm ON lm.line_a_id = cm.text_line_id
					AND lm.line_b_id IS NOT NULL
				JOIN text_lines tl ON tl.id = lm.line_b_id
				WHERE NOT EXISTS (
					SELECT 1 FROM citation_matches cm2
					WHERE cm2.citation_id = cm.citation_id AND cm2.edition_id = lm.edition_b_id
				)
			) sub
			WHERE sub.rn = 1`)
		if err == nil {
			n, _ := res.RowsAffected()
			roundCount += int(n)
		}

		// Reverse: match is on line_b_id → propagate to line_a_id
		res, err = database.Exec(`
			INSERT INTO citation_matches (citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
			SELECT sub.citation_id, sub.text_line_id, sub.edition_id, 'propagated', sub.confidence, sub.matched_text
			FROM (
				SELECT
					cm.citation_id,
					lm.line_a_id AS text_line_id,
					lm.edition_a_id AS edition_id,
					cm.confidence * CASE WHEN lm.similarity >= 0.2 THEN lm.similarity ELSE 0.3 END AS confidence,
					tl.content AS matched_text,
					ROW_NUMBER() OVER (
						PARTITION BY cm.citation_id, lm.edition_a_id
						ORDER BY cm.confidence * lm.similarity DESC
					) AS rn
				FROM citation_matches cm
				JOIN line_mappings lm ON lm.line_b_id = cm.text_line_id
					AND lm.line_a_id IS NOT NULL
				JOIN text_lines tl ON tl.id = lm.line_a_id
				WHERE NOT EXISTS (
					SELECT 1 FROM citation_matches cm2
					WHERE cm2.citation_id = cm.citation_id AND cm2.edition_id = lm.edition_a_id
				)
			) sub
			WHERE sub.rn = 1`)
		if err == nil {
			n, _ := res.RowsAffected()
			roundCount += int(n)
		}

		totalPropagated += roundCount
		fmt.Printf("  Propagation round %d: %d new matches\n", round+1, roundCount)

		if roundCount == 0 {
			break
		}
	}

	fmt.Printf("  Total propagated: %d\n", totalPropagated)
	return totalPropagated
}

// matchByHeadword resolves citations that have act+scene but no line number or quote text.
// For each, it searches the scene's text lines for occurrences of the lexicon entry's headword.
// Returns the number of new matches created.
func matchByHeadword(database *sql.DB) int {
	// Load unmatched citations that have only act+scene
	rows, err := database.Query(`
		SELECT lc.id, lc.work_id, lc.act, lc.scene, le.key
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		WHERE lc.work_id IS NOT NULL
		  AND lc.act IS NOT NULL AND lc.scene IS NOT NULL
		  AND lc.line IS NULL AND lc.quote_text IS NULL
		  AND lc.id NOT IN (SELECT citation_id FROM citation_matches)`)
	if err != nil {
		return 0
	}

	type headwordCit struct {
		ID       int64
		WorkID   int64
		Act      int
		Scene    int
		Key string
	}
	var cits []headwordCit
	for rows.Next() {
		var c headwordCit
		rows.Scan(&c.ID, &c.WorkID, &c.Act, &c.Scene, &c.Key)
		cits = append(cits, c)
	}
	rows.Close()

	if len(cits) == 0 {
		return 0
	}

	fmt.Printf("  Headword search: %d candidates\n", len(cits))

	// Group by scene to avoid redundant queries
	type sceneKey struct {
		WorkID int64
		Act    int
		Scene  int
	}
	sceneGroups := make(map[sceneKey][]headwordCit)
	for _, c := range cits {
		key := sceneKey{c.WorkID, c.Act, c.Scene}
		sceneGroups[key] = append(sceneGroups[key], c)
	}

	tx, err := database.Begin()
	if err != nil {
		return 0
	}
	stmt, err := tx.Prepare(`
		INSERT INTO citation_matches (citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
		VALUES (?, ?, ?, 'headword', 0.4, ?)`)
	if err != nil {
		tx.Rollback()
		return 0
	}

	matched := 0
	for key, groupCits := range sceneGroups {
		lines, err := loadTextLines(database, "work_id = ? AND act = ? AND scene = ?",
			key.WorkID, key.Act, key.Scene)
		if err != nil || len(lines) == 0 {
			continue
		}

		// Group lines by edition once for all citations in this scene
		linesByEdition := make(map[int64][]textLineRow)
		for _, tl := range lines {
			linesByEdition[tl.EditionID] = append(linesByEdition[tl.EditionID], tl)
		}

		for _, cit := range groupCits {
			if cit.Key == "" {
				continue
			}

			// Strip trailing sense number: "Bend2" → "Bend", "Quick1" → "Quick"
			cleanKey := stripSenseNumber(cit.Key)
			if cleanKey == "" {
				continue
			}

			for editionID, edLines := range linesByEdition {
				for _, line := range edLines {
					if parser.ContainsNormalized(line.Content, cleanKey) {
						stmt.Exec(cit.ID, line.ID, editionID, line.Content)
						matched++
						break // one match per edition
					}
				}
			}
		}
	}

	stmt.Close()
	tx.Commit()

	fmt.Printf("  Headword matches: %d\n", matched)
	return matched
}

// stripSenseNumber removes trailing digit suffixes from lexicon headword keys.
// Schmidt's keys like "Bend2", "Act1", "Quick1" have sense numbers that prevent
// substring matching against text. "Bend2" → "Bend", "A1" → "A", "Go" → "Go".
func stripSenseNumber(key string) string {
	s := strings.TrimRightFunc(key, unicode.IsDigit)
	if s == "" {
		return key // all digits — keep as-is
	}
	return s
}

// expandQuoteAbbreviation expands Schmidt's abbreviated headword in citation quotes.
// Schmidt systematically abbreviates the cited word using these patterns:
//   - "b. thee to it"       → first letter + "." → expand to headword ("betake thee to it")
//   - "--s these together"  → "--" + suffix → expand to headword+suffix ("tickles these together")
//   - "I have --ed him"     → "--" + suffix → expand to headword+suffix ("dogged him")
//   - ". . ."               → elision of headword within quote
//
// Returns the expanded quote and whether any expansion was made.
func expandQuoteAbbreviation(quote, headword string) (string, bool) {
	if headword == "" || quote == "" {
		return quote, false
	}

	headwordLower := strings.ToLower(headword)

	// Pattern 1: "--" + suffix → headword + suffix
	// e.g., "--s" → "tickles", "--ed" → "dogged", "--ing" → "making"
	if idx := strings.Index(quote, "--"); idx >= 0 {
		// Find the suffix after "--"
		rest := quote[idx+2:]
		suffixEnd := 0
		for i, c := range rest {
			if c == ' ' || c == ',' || c == '.' || c == ';' || c == ':' || c == '!' || c == '?' {
				break
			}
			suffixEnd = i + 1
		}
		suffix := rest[:suffixEnd]
		expanded := quote[:idx] + headwordLower + suffix + rest[suffixEnd:]
		return expanded, true
	}

	// Pattern 2: single letter + "." where the letter matches headword start
	// e.g., "b." → "betake", "s." → "strip", "d." → "defence"
	// Try prefix lengths 1-3
	for prefixLen := 1; prefixLen <= 3 && prefixLen < len(headwordLower); prefixLen++ {
		prefix := headwordLower[:prefixLen]
		// Look for "X." as a word boundary (preceded by space/start, followed by space/end)
		target := prefix + "."
		lowerQuote := strings.ToLower(quote)
		idx := strings.Index(lowerQuote, target)
		if idx < 0 {
			continue
		}
		// Verify it's at a word boundary (not middle of "e.g." or similar)
		if idx > 0 {
			prev := rune(lowerQuote[idx-1])
			if prev != ' ' && prev != ',' && prev != ';' && prev != ':' && prev != '(' {
				continue
			}
		}
		expanded := quote[:idx] + headwordLower + quote[idx+len(target):]
		return expanded, true
	}

	return quote, false
}

// findBestMatch finds the best matching text line for a citation.
// Returns the matched line, match type, and confidence score.
func findBestMatch(lines []textLineRow, cit citationRow) (*textLineRow, string, float64) {
	if len(lines) == 0 {
		return nil, "", 0
	}

	// Strategy 1: Exact quote match (highest confidence)
	if cit.QuoteText != "" {
		cleanQuote := strings.ReplaceAll(cit.QuoteText, "--", "")
		cleanQuote = strings.TrimSpace(cleanQuote)

		// Also prepare an expanded quote where Schmidt's headword abbreviation
		// (e.g., "b." for "betake", "--s" for "doctors") is expanded.
		headword := stripSenseNumber(cit.Headword)
		expandedQuote, wasExpanded := expandQuoteAbbreviation(cit.QuoteText, headword)
		if wasExpanded {
			expandedQuote = strings.ReplaceAll(expandedQuote, "--", "")
			expandedQuote = strings.TrimSpace(expandedQuote)
		}

		if len(cleanQuote) > 3 {
			// 1a: Single-line substring match (original quote).
			for i, line := range lines {
				if parser.ContainsNormalized(line.Content, cleanQuote) {
					return &lines[i], "exact_quote", 1.0
				}
			}

			// 1b: Single-line match with expanded abbreviation.
			if wasExpanded && len(expandedQuote) > 3 {
				for i, line := range lines {
					if parser.ContainsNormalized(line.Content, expandedQuote) {
						return &lines[i], "exact_quote", 0.95
					}
				}
			}

			// 1c: Multi-line match — Schmidt quotes often span two verse lines.
			if len(lines) > 1 {
				for i := 0; i < len(lines)-1; i++ {
					combined := lines[i].Content + " " + lines[i+1].Content
					if parser.ContainsNormalized(combined, cleanQuote) {
						return &lines[i], "exact_quote", 0.95
					}
					if wasExpanded && parser.ContainsNormalized(combined, expandedQuote) {
						return &lines[i], "exact_quote", 0.90
					}
				}
			}
		}
	}

	// Strategy 2: Line number match
	if cit.Line != nil {
		for i, line := range lines {
			if line.LineNumber == *cit.Line {
				if cit.QuoteText != "" {
					sim := parser.JaccardSimilarity(line.Content, cit.QuoteText)
					if sim > 0.15 {
						return &lines[i], "line_number", 0.9
					}
					return &lines[i], "line_number", 0.7
				}
				return &lines[i], "line_number", 0.9
			}
		}

		// Try nearby lines if exact line number didn't match.
		// With a quote: scan ±20 and return the best-scoring candidate (handles
		// Globe vs SE line-number offset caused by stage directions).
		// Without a quote: scan ±10 with decreasing confidence.
		if cit.QuoteText != "" {
			const maxDeltaQuote = 20
			bestSim := 0.0
			bestIdx := -1
			for i, line := range lines {
				d := line.LineNumber - *cit.Line
				if d < 0 {
					d = -d
				}
				if d > 0 && d <= maxDeltaQuote {
					sim := parser.JaccardSimilarity(line.Content, cit.QuoteText)
					if sim > bestSim {
						bestSim = sim
						bestIdx = i
					}
				}
			}
			if bestSim > 0.2 && bestIdx >= 0 {
				return &lines[bestIdx], "line_number", 0.7
			}
		} else {
			for delta := 1; delta <= 10; delta++ {
				for i, line := range lines {
					if line.LineNumber == *cit.Line+delta || line.LineNumber == *cit.Line-delta {
						conf := 0.6 - float64(delta)*0.1
						if conf < 0.1 {
							conf = 0.1
						}
						return &lines[i], "line_number", conf
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
		if bestScore > 0.20 && bestIdx >= 0 {
			return &lines[bestIdx], "fuzzy_text", bestScore
		}
	}

	return nil, "", 0
}
