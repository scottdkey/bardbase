package importer

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/scottdkey/shakespeare_db/internal/db"
	"github.com/scottdkey/shakespeare_db/internal/parser"
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
// For each citation with a valid work + act + scene reference, it searches
// for matching text lines in each edition using line numbers and/or quote text.
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

	// Load citations that have at least work_id and act set
	citRows, err := database.Query(`
		SELECT id, entry_id, sense_id, work_id, act, scene, line, quote_text
		FROM lexicon_citations
		WHERE work_id IS NOT NULL AND act IS NOT NULL`)
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

	// Group citations by (work_id, act, scene) for batch loading
	type sceneKey struct {
		WorkID int64
		Act    int
		Scene  int
	}
	citationsByScene := make(map[sceneKey][]citationRow)
	for _, c := range citations {
		if c.Act == nil {
			continue
		}
		scn := 0
		if c.Scene != nil {
			scn = *c.Scene
		}
		key := sceneKey{WorkID: c.WorkID, Act: *c.Act, Scene: scn}
		citationsByScene[key] = append(citationsByScene[key], c)
	}

	// Process each scene group
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

	for key, sceneCitations := range citationsByScene {
		// Load text lines for this scene from all editions
		var query string
		var args []interface{}
		if key.Scene > 0 {
			query = "SELECT id, content, line_number, edition_id FROM text_lines WHERE work_id = ? AND act = ? AND scene = ? AND line_number IS NOT NULL ORDER BY edition_id, line_number"
			args = []interface{}{key.WorkID, key.Act, key.Scene}
		} else {
			query = "SELECT id, content, line_number, edition_id FROM text_lines WHERE work_id = ? AND act = ? AND line_number IS NOT NULL ORDER BY edition_id, line_number"
			args = []interface{}{key.WorkID, key.Act}
		}

		lineRows, err := database.Query(query, args...)
		if err != nil {
			continue
		}

		// Group lines by edition
		linesByEdition := make(map[int64][]textLineRow)
		for lineRows.Next() {
			var tl textLineRow
			lineRows.Scan(&tl.ID, &tl.Content, &tl.LineNumber, &tl.EditionID)
			linesByEdition[tl.EditionID] = append(linesByEdition[tl.EditionID], tl)
		}
		lineRows.Close()

		// Match each citation against each edition's lines
		for _, cit := range sceneCitations {
			for editionID, lines := range linesByEdition {
				matchLine, matchType, confidence := findBestMatch(lines, cit)
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
