// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
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

// citMatchTask bundles a citation with the text lines it should be matched against.
type citMatchTask struct {
	cit   citationRow
	lines []textLineRow // already split by edition
}

// citSceneKey indexes text lines within a single work by (act, scene).
// act and scene use COALESCE(x, 0): sonnet lines have act=0 (NULL→0),
// prologue/induction lines have act=0 (explicit). Poems use {0, 0}.
type citSceneKey struct{ act, scene int }

// workLineSet holds all text lines for one work pre-indexed for O(1) lookup
// during citation task building — eliminating per-scene DB queries.
type workLineSet struct {
	byScene map[citSceneKey][]textLineRow // (act, scene) → lines with line_number
	byAct   map[int][]textLineRow         // act → all lines in act (for whole-act fallback)
	all     []textLineRow                 // all lines (for poem/whole-work lookup)
}

// buildCitLineCache loads ALL text_lines with non-null line numbers in a single
// query and indexes them by work_id for O(1) scene/act/work lookup during Phase 1.
// Replaces ~7000 individual loadTextLines calls with one bulk read.
func buildCitLineCache(database *sql.DB) map[int64]*workLineSet {
	cache := make(map[int64]*workLineSet)
	rows, err := database.Query(`
		SELECT tl.id, tl.content, COALESCE(tl.line_number, 0), tl.edition_id, tl.work_id,
		       COALESCE(tl.act, 0), COALESCE(tl.scene, 0)
		FROM text_lines tl
		WHERE tl.line_number IS NOT NULL
		ORDER BY tl.work_id, tl.edition_id, tl.act, tl.scene, tl.line_number, tl.id`)
	if err != nil {
		return cache
	}
	defer rows.Close()
	for rows.Next() {
		var tl textLineRow
		var workID int64
		var act, scene int
		rows.Scan(&tl.ID, &tl.Content, &tl.LineNumber, &tl.EditionID, &workID, &act, &scene)
		wls := cache[workID]
		if wls == nil {
			wls = &workLineSet{
				byScene: make(map[citSceneKey][]textLineRow),
				byAct:   make(map[int][]textLineRow),
			}
			cache[workID] = wls
		}
		sk := citSceneKey{act, scene}
		wls.byScene[sk] = append(wls.byScene[sk], tl)
		wls.byAct[act] = append(wls.byAct[act], tl)
		wls.all = append(wls.all, tl)
	}
	return cache
}

// citMatchResult holds the output of one citation matching task per edition.
type citMatchResult struct {
	citID      int64
	lineID     int64
	editionID  int64
	matchType  string
	confidence float64
	content    string
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
//
// Phase 1 (load + classify) and Phase 3 (insert) are sequential;
// Phase 2 (findBestMatch) is parallelized across goroutines.
func ResolveCitations(database *sql.DB) error {
	stepBanner("Resolve Lexicon Citations → Text Lines")

	start := time.Now()

	// Clear existing matches for a clean rebuild
	database.Exec("DELETE FROM citation_matches")

	// === Pre-resolution data corrections ===
	// Fix misattributed citations before matching begins.
	fixedCount := fixMisattributedCitations(database)
	if fixedCount > 0 {
		fmt.Printf("  Data corrections: %d citations reassigned\n", fixedCount)
	}

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

	// === Phase 1: Bulk-load ALL text lines, then build match tasks from cache ===
	//
	// Previous approach: one loadTextLines DB query per (work_id, act, scene) group
	// — ~7000 queries totalling 60+ seconds on a cold SQLite cache.
	//
	// New approach: one query loads everything into a map keyed by work_id.
	// Task building then runs entirely in memory with O(1) map lookups.
	citCache := buildCitLineCache(database)
	fmt.Printf("  Text line cache: %d works loaded in %.1fs\n", len(citCache), time.Since(start).Seconds())

	var tasks []citMatchTask

	// Play citations — look up (act, scene) from cache; fall back to whole act.
	for key, sceneCitations := range playCitations {
		wls := citCache[key.WorkID]
		if wls == nil {
			continue
		}
		var lines []textLineRow
		if key.Scene > 0 {
			lines = wls.byScene[citSceneKey{key.Act, key.Scene}]
		} else {
			lines = wls.byAct[key.Act]
		}

		// Fallback 1: scene lookup returned nothing → the 2-part Perseus reference
		// (e.g., "shak. tmp 5.169") was probably act.line, not act.scene.
		// Use the whole-act lines and promote the scene value to a line number.
		if len(lines) == 0 && key.Scene > 0 {
			lines = wls.byAct[key.Act]
			for i := range sceneCitations {
				if sceneCitations[i].Line == nil {
					lineNum := key.Scene
					sceneCitations[i].Line = &lineNum
				}
			}
		}

		// Fallback 2: if any citation's line number doesn't exist in the
		// current line set, widen the search progressively:
		//   scene → act → entire play
		// This handles Globe numbering mismatches where Schmidt's act/scene
		// structure doesn't match the Perseus edition.
		for _, cit := range sceneCitations {
			searchLines := lines
			if cit.Line != nil && len(searchLines) > 0 {
				found := false
				for _, line := range searchLines {
					if line.LineNumber == *cit.Line {
						found = true
						break
					}
				}
				if !found {
					// Try whole act.
					searchLines = wls.byAct[key.Act]
					found = false
					for _, line := range searchLines {
						if line.LineNumber == *cit.Line {
							found = true
							break
						}
					}
					if !found {
						// Try entire play — the act might also be wrong.
						searchLines = wls.all
					}
				}
			}
			tasks = append(tasks, citMatchTask{cit: cit, lines: searchLines})
		}
	}

	// Sonnet citations — act is NULL in DB (coalesced to 0), scene = sonnet number.
	for key, sCitations := range sonnetCitations {
		wls := citCache[key.WorkID]
		if wls == nil {
			continue
		}
		lines := wls.byScene[citSceneKey{0, key.Scene}]
		for _, cit := range sCitations {
			tasks = append(tasks, citMatchTask{cit: cit, lines: lines})
		}
	}

	// Poem citations — no act or scene constraint; use all lines for the work.
	for key, pCitations := range poemCitations {
		wls := citCache[key.WorkID]
		if wls == nil {
			continue
		}
		for _, cit := range pCitations {
			tasks = append(tasks, citMatchTask{cit: cit, lines: wls.all})
		}
	}

	fmt.Printf("  Match tasks: %d (built in %.1fs)\n", len(tasks), time.Since(start).Seconds())

	// === Phase 2: Run findBestMatch in parallel (CPU-bound, no DB access) ===
	workers := workerCount()
	if workers < 1 {
		workers = 1
	}

	taskCh := make(chan citMatchTask, 256)
	resultCh := make(chan citMatchResult, 256)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				// Split lines by edition and match in each
				linesByEdition := make(map[int64][]textLineRow)
				for _, tl := range task.lines {
					linesByEdition[tl.EditionID] = append(linesByEdition[tl.EditionID], tl)
				}
				for editionID, edLines := range linesByEdition {
					matchLine, matchType, confidence := findBestMatch(edLines, task.cit)
					if matchLine != nil {
						resultCh <- citMatchResult{
							citID:      task.cit.ID,
							lineID:     matchLine.ID,
							editionID:  editionID,
							matchType:  matchType,
							confidence: confidence,
							content:    matchLine.Content,
						}
					}
				}
			}
		}()
	}

	// Feed tasks
	go func() {
		for _, t := range tasks {
			taskCh <- t
		}
		close(taskCh)
	}()

	// Close resultCh once all workers finish
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// === Phase 3: Insert results (sequential — single SQLite transaction) ===
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

	for result := range resultCh {
		insertStmt.Exec(result.citID, result.lineID, result.editionID,
			result.matchType, result.confidence, result.content)
		totalMatches++
		switch result.matchType {
		case "exact_quote":
			exactMatches++
		case "line_number":
			lineMatches++
		case "fuzzy_text":
			fuzzyMatches++
		}
	}

	insertStmt.Close()
	tx.Commit()

	fmt.Printf("  Direct matching: %d matches (exact=%d, line=%d, fuzzy=%d) [%d workers]\n",
		totalMatches, exactMatches, lineMatches, fuzzyMatches, workers)

	// === Phase 4: Cross-edition propagation via line_mappings ===
	propagated := propagateCrossEdition(database)
	totalMatches += propagated

	// === Phase 5: Headword search for citations with only act+scene ===
	headwordMatches := matchByHeadword(database)
	totalMatches += headwordMatches

	// === Phase 6: Headword fallback for ALL remaining unmatched citations ===
	// Catches citations where quote/line matching failed (e.g., Shrew Induction
	// where Globe line numbers don't exist in the OSS edition, or archaic spelling
	// prevents quote matching). Uses the same headword search but without the
	// "no line, no quote" restriction.
	fallbackMatches := matchByHeadwordFallback(database)
	totalMatches += fallbackMatches

	// === Phase 7: Cross-play search for remaining unmatched citations ===
	// Some citations have the wrong play attribution in the Perseus source data.
	// For each unmatched citation that has a line number, search ALL plays for
	// a text line near that number containing the headword.
	crossPlayMatches := matchByCrossPlaySearch(database)
	totalMatches += crossPlayMatches

	// === Phase 8: Apply manual corrections from citation_corrections/*.json ===
	manualMatches := applyManualCorrections(database)
	totalMatches += manualMatches

	// Final stats from database
	var finalExact, finalLine, finalFuzzy, finalPropagated, finalHeadword, finalCrossPlay, finalManual, finalTotal int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'exact_quote'").Scan(&finalExact)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'line_number'").Scan(&finalLine)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'fuzzy_text'").Scan(&finalFuzzy)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'propagated'").Scan(&finalPropagated)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'headword'").Scan(&finalHeadword)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'cross_play'").Scan(&finalCrossPlay)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'manual'").Scan(&finalManual)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&finalTotal)

	var finalUnmatched int
	database.QueryRow(`
		SELECT COUNT(*) FROM lexicon_citations
		WHERE work_id IS NOT NULL
		  AND (act IS NOT NULL OR scene IS NOT NULL OR line IS NOT NULL OR quote_text IS NOT NULL)
		  AND id NOT IN (SELECT citation_id FROM citation_matches)`).Scan(&finalUnmatched)

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "citations", "resolve_complete",
		fmt.Sprintf("%d matches (exact=%d, line=%d, fuzzy=%d, propagated=%d, headword=%d, cross_play=%d, manual=%d, unmatched=%d)",
			finalTotal, finalExact, finalLine, finalFuzzy, finalPropagated, finalHeadword, finalCrossPlay, finalManual, finalUnmatched),
		finalTotal, elapsed)

	fmt.Printf("  ✓ %d total matches in %.1fs\n", finalTotal, elapsed)
	fmt.Printf("    exact_quote: %d, line_number: %d, fuzzy_text: %d\n", finalExact, finalLine, finalFuzzy)
	fmt.Printf("    propagated: %d, headword: %d, cross_play: %d, manual: %d\n", finalPropagated, finalHeadword, finalCrossPlay, finalManual)
	fmt.Printf("    unmatched citations: %d of %d (%.1f%%)\n",
		finalUnmatched, len(citations), 100.0*float64(finalUnmatched)/float64(len(citations)))

	// Print unmatched citation details for manual review and write TSV to build dir.
	if finalUnmatched > 0 {
		printUnmatchedReport(database, len(citations))
		writeUnmatchedTSV(database)
	}

	return nil
}

// printUnmatchedReport outputs a breakdown of unmatched lexicon citations
// grouped by reason, with samples for each category. This helps identify
// systemic parsing/matching issues during development.
func printUnmatchedReport(database *sql.DB, totalCitations int) {
	fmt.Println("\n  === UNMATCHED CITATION REPORT ===")

	// Breakdown by what's missing.
	type bucket struct {
		label string
		query string
	}
	buckets := []bucket{
		{
			"Play citations: have act+scene+line but no match",
			`SELECT le.key, lc.raw_bibl, w.schmidt_abbrev, lc.act, lc.scene, lc.line, lc.quote_text
			 FROM lexicon_citations lc
			 JOIN lexicon_entries le ON le.id = lc.entry_id
			 JOIN works w ON w.id = lc.work_id
			 WHERE w.work_type IN ('comedy','history','tragedy')
			   AND lc.act IS NOT NULL AND lc.scene IS NOT NULL AND lc.line IS NOT NULL
			   AND lc.id NOT IN (SELECT citation_id FROM citation_matches)
			 ORDER BY w.schmidt_abbrev, lc.act, lc.scene, lc.line`,
		},
		{
			"Play citations: have act+scene but NO line",
			`SELECT le.key, lc.raw_bibl, w.schmidt_abbrev, lc.act, lc.scene, lc.line, lc.quote_text
			 FROM lexicon_citations lc
			 JOIN lexicon_entries le ON le.id = lc.entry_id
			 JOIN works w ON w.id = lc.work_id
			 WHERE w.work_type IN ('comedy','history','tragedy')
			   AND lc.act IS NOT NULL AND lc.scene IS NOT NULL AND lc.line IS NULL
			   AND lc.id NOT IN (SELECT citation_id FROM citation_matches)
			 ORDER BY w.schmidt_abbrev, lc.act, lc.scene`,
		},
		{
			"Play citations: have act+line but NO scene",
			`SELECT le.key, lc.raw_bibl, w.schmidt_abbrev, lc.act, lc.scene, lc.line, lc.quote_text
			 FROM lexicon_citations lc
			 JOIN lexicon_entries le ON le.id = lc.entry_id
			 JOIN works w ON w.id = lc.work_id
			 WHERE w.work_type IN ('comedy','history','tragedy')
			   AND lc.act IS NOT NULL AND lc.scene IS NULL AND lc.line IS NOT NULL
			   AND lc.id NOT IN (SELECT citation_id FROM citation_matches)
			 ORDER BY w.schmidt_abbrev, lc.act, lc.line`,
		},
		{
			"Play citations: have act only (no scene, no line)",
			`SELECT le.key, lc.raw_bibl, w.schmidt_abbrev, lc.act, lc.scene, lc.line, lc.quote_text
			 FROM lexicon_citations lc
			 JOIN lexicon_entries le ON le.id = lc.entry_id
			 JOIN works w ON w.id = lc.work_id
			 WHERE w.work_type IN ('comedy','history','tragedy')
			   AND lc.act IS NOT NULL AND lc.scene IS NULL AND lc.line IS NULL
			   AND lc.id NOT IN (SELECT citation_id FROM citation_matches)
			 ORDER BY w.schmidt_abbrev, lc.act`,
		},
		{
			"Sonnet citations: missing scene (sonnet number)",
			`SELECT le.key, lc.raw_bibl, w.schmidt_abbrev, lc.act, lc.scene, lc.line, lc.quote_text
			 FROM lexicon_citations lc
			 JOIN lexicon_entries le ON le.id = lc.entry_id
			 JOIN works w ON w.id = lc.work_id
			 WHERE w.work_type = 'sonnet_sequence'
			   AND lc.scene IS NULL
			   AND lc.id NOT IN (SELECT citation_id FROM citation_matches)
			 ORDER BY le.key`,
		},
		{
			"Poem citations: missing line",
			`SELECT le.key, lc.raw_bibl, w.schmidt_abbrev, lc.act, lc.scene, lc.line, lc.quote_text
			 FROM lexicon_citations lc
			 JOIN lexicon_entries le ON le.id = lc.entry_id
			 JOIN works w ON w.id = lc.work_id
			 WHERE w.work_type = 'poem' AND lc.line IS NULL
			   AND lc.id NOT IN (SELECT citation_id FROM citation_matches)
			 ORDER BY le.key`,
		},
		{
			"Citations with no work_id",
			`SELECT le.key, lc.raw_bibl, '' as abbrev, lc.act, lc.scene, lc.line, lc.quote_text
			 FROM lexicon_citations lc
			 JOIN lexicon_entries le ON le.id = lc.entry_id
			 WHERE lc.work_id IS NULL
			 ORDER BY le.key`,
		},
	}

	for _, b := range buckets {
		rows, err := database.Query(b.query)
		if err != nil {
			continue
		}
		var samples []string
		count := 0
		for rows.Next() {
			var key, rawBibl, abbrev string
			var act, scene, line sql.NullInt64
			var quote sql.NullString
			rows.Scan(&key, &rawBibl, &abbrev, &act, &scene, &line, &quote)
			count++
			if count <= 10 {
				actStr, sceneStr, lineStr := "NULL", "NULL", "NULL"
				if act.Valid {
					actStr = fmt.Sprintf("%d", act.Int64)
				}
				if scene.Valid {
					sceneStr = fmt.Sprintf("%d", scene.Int64)
				}
				if line.Valid {
					lineStr = fmt.Sprintf("%d", line.Int64)
				}
				q := ""
				if quote.Valid && quote.String != "" {
					q = quote.String
					if len(q) > 40 {
						q = q[:40] + "..."
					}
					q = fmt.Sprintf(" quote=%q", q)
				}
				samples = append(samples, fmt.Sprintf("      %-20s %-25s %s act=%s scene=%s line=%s%s",
					key, rawBibl, abbrev, actStr, sceneStr, lineStr, q))
			}
		}
		rows.Close()

		if count == 0 {
			continue
		}

		fmt.Printf("\n  %s: %d\n", b.label, count)
		for _, s := range samples {
			fmt.Println(s)
		}
		if count > 10 {
			fmt.Printf("      ... and %d more\n", count-10)
		}
	}

	// Per-work summary of unmatched play citations.
	fmt.Println("\n  Per-work unmatched play citations:")
	rows, err := database.Query(`
		SELECT w.schmidt_abbrev, COUNT(*) as cnt,
		  SUM(CASE WHEN lc.line IS NULL THEN 1 ELSE 0 END) as no_line,
		  SUM(CASE WHEN lc.scene IS NULL THEN 1 ELSE 0 END) as no_scene,
		  SUM(CASE WHEN lc.quote_text IS NOT NULL AND lc.quote_text != '' THEN 1 ELSE 0 END) as has_quote
		FROM lexicon_citations lc
		JOIN works w ON w.id = lc.work_id
		WHERE w.work_type IN ('comedy','history','tragedy')
		  AND lc.id NOT IN (SELECT citation_id FROM citation_matches)
		GROUP BY w.schmidt_abbrev
		ORDER BY cnt DESC`)
	if err == nil {
		for rows.Next() {
			var abbrev string
			var cnt, noLine, noScene, hasQuote int
			rows.Scan(&abbrev, &cnt, &noLine, &noScene, &hasQuote)
			fmt.Printf("    %-8s %5d unmatched (no_line=%d, no_scene=%d, has_quote=%d)\n",
				abbrev, cnt, noLine, noScene, hasQuote)
		}
		rows.Close()
	}

	fmt.Println()
}

// writeUnmatchedTSV writes all unmatched lexicon citations to a TSV file
// in the same directory as the database (build/unmatched_citations.tsv).
func writeUnmatchedTSV(database *sql.DB) {
	// Derive the build directory from the database path.
	var seq int
	var name, dbPath string
	row := database.QueryRow("PRAGMA database_list")
	if err := row.Scan(&seq, &name, &dbPath); err != nil || dbPath == "" {
		fmt.Println("  WARNING: could not determine database path for TSV output")
		return
	}
	outPath := filepath.Join(filepath.Dir(dbPath), "unmatched_citations.tsv")

	rows, err := database.Query(`
		SELECT le.key, lc.raw_bibl, w.schmidt_abbrev, w.work_type,
		       lc.act, lc.scene, lc.line, lc.quote_text
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		LEFT JOIN works w ON w.id = lc.work_id
		WHERE lc.id NOT IN (SELECT citation_id FROM citation_matches)
		  AND lc.work_id IS NOT NULL
		ORDER BY w.schmidt_abbrev, lc.act, lc.scene, lc.line, le.key`)
	if err != nil {
		return
	}
	defer rows.Close()

	f, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("  WARNING: could not write %s: %v\n", outPath, err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "headword\traw_bibl\twork\twork_type\tact\tscene\tline\tquote\n")
	count := 0
	for rows.Next() {
		var key, rawBibl string
		var abbrev, workType sql.NullString
		var act, scene, line sql.NullInt64
		var quote sql.NullString
		rows.Scan(&key, &rawBibl, &abbrev, &workType, &act, &scene, &line, &quote)

		actStr, sceneStr, lineStr := "", "", ""
		if act.Valid {
			actStr = fmt.Sprintf("%d", act.Int64)
		}
		if scene.Valid {
			sceneStr = fmt.Sprintf("%d", scene.Int64)
		}
		if line.Valid {
			lineStr = fmt.Sprintf("%d", line.Int64)
		}
		q := ""
		if quote.Valid {
			q = quote.String
		}

		fmt.Fprintf(f, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			key, rawBibl,
			abbrev.String, workType.String,
			actStr, sceneStr, lineStr, q)
		count++
	}

	fmt.Printf("  Wrote %d unmatched citations to %s\n", count, outPath)
}

// propagateCrossEdition uses line_mappings to propagate citation matches across editions.
// If a citation matched line X in edition A, and line_mappings maps X → Y in edition B,
// a new "propagated" match is created for edition B.
//
// Runs multiple rounds because a match propagated from A→B in round 1 can then
// propagate from B→C in round 2 (for 3 editions, 2 rounds suffice).
func propagateCrossEdition(database *sql.DB) int {
	totalPropagated := 0

	// Run 3 rounds to handle transitive propagation across 4 editions (A→B→C→D)
	for round := 0; round < 3; round++ {
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
func matchByHeadword(database *sql.DB) int {
	return runHeadwordSearch(database, `
		SELECT lc.id, lc.work_id, lc.act, lc.scene, le.key
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		WHERE lc.work_id IS NOT NULL
		  AND lc.act IS NOT NULL AND lc.scene IS NOT NULL
		  AND lc.line IS NULL AND (lc.quote_text IS NULL OR lc.quote_text = '')
		  AND lc.id NOT IN (SELECT citation_id FROM citation_matches)`,
		0.4, "Headword search", "Headword matches")
}

// matchByHeadwordFallback is a last-resort pass for ALL remaining unmatched citations.
// Unlike matchByHeadword, it does not require line IS NULL / quote_text IS NULL.
func matchByHeadwordFallback(database *sql.DB) int {
	return runHeadwordSearch(database, `
		SELECT lc.id, lc.work_id, lc.act, lc.scene, le.key
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		WHERE lc.work_id IS NOT NULL
		  AND lc.act IS NOT NULL AND lc.scene IS NOT NULL
		  AND lc.id NOT IN (SELECT citation_id FROM citation_matches)`,
		0.3, "Headword fallback", "Headword fallback matches")
}

// runHeadwordSearch is the shared implementation for matchByHeadword and
// matchByHeadwordFallback. It runs a 3-tier headword search (exact substring →
// word-prefix → stem-prefix) against scene text lines in parallel, then inserts
// results with the given confidence score.
//
//   - citQuery: SELECT returning (id, work_id, act, scene, key)
//   - confidence: inserted into citation_matches.confidence
//   - candidateLabel: printed before the candidate count
//   - matchLabel: printed after insertion ("X matches")
func runHeadwordSearch(database *sql.DB, citQuery string, confidence float64, candidateLabel, matchLabel string) int {
	rows, err := database.Query(citQuery)
	if err != nil {
		return 0
	}

	type hwCit struct {
		ID     int64
		WorkID int64
		Act    int
		Scene  int
		Key    string
	}
	var cits []hwCit
	for rows.Next() {
		var c hwCit
		rows.Scan(&c.ID, &c.WorkID, &c.Act, &c.Scene, &c.Key)
		cits = append(cits, c)
	}
	rows.Close()

	if len(cits) == 0 {
		return 0
	}

	fmt.Printf("  %s: %d candidates\n", candidateLabel, len(cits))

	// Group by scene to avoid redundant text-line loads.
	type sceneKey struct {
		WorkID int64
		Act    int
		Scene  int
	}
	sceneGroups := make(map[sceneKey][]hwCit)
	for _, c := range cits {
		sceneGroups[sceneKey{c.WorkID, c.Act, c.Scene}] = append(sceneGroups[sceneKey{c.WorkID, c.Act, c.Scene}], c)
	}

	// Phase 1: Pre-load all scene text lines (sequential DB reads).
	type hwTask struct {
		cit            hwCit
		linesByEdition map[int64][]textLineRow
	}
	var tasks []hwTask

	for key, groupCits := range sceneGroups {
		lines, err := loadTextLinesAll(database, "work_id = ? AND act = ? AND scene = ?",
			key.WorkID, key.Act, key.Scene)
		if err != nil {
			continue
		}
		// Fall back to whole-act search when the specific scene has no text lines.
		if len(lines) == 0 {
			lines, err = loadTextLinesAll(database, "work_id = ? AND act = ?",
				key.WorkID, key.Act)
			if err != nil || len(lines) == 0 {
				continue
			}
		}

		linesByEdition := make(map[int64][]textLineRow)
		for _, tl := range lines {
			linesByEdition[tl.EditionID] = append(linesByEdition[tl.EditionID], tl)
		}
		for _, cit := range groupCits {
			tasks = append(tasks, hwTask{cit: cit, linesByEdition: linesByEdition})
		}
	}

	// Phase 2: Run 3-tier headword search in parallel (CPU-bound).
	type hwResult struct {
		citID     int64
		lineID    int64
		editionID int64
		content   string
	}

	workers := max(1, workerCount())
	taskCh := make(chan hwTask, 256)
	resultCh := make(chan hwResult, 256)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				cleanKey := stripSenseNumber(task.cit.Key)
				if cleanKey == "" {
					continue
				}
				for editionID, edLines := range task.linesByEdition {
					found := false
					for _, line := range edLines {
						if parser.ContainsNormalized(line.Content, cleanKey) {
							resultCh <- hwResult{task.cit.ID, line.ID, editionID, line.Content}
							found = true
							break
						}
					}
					if !found {
						for _, line := range edLines {
							if parser.ContainsWordPrefix(line.Content, cleanKey) {
								resultCh <- hwResult{task.cit.ID, line.ID, editionID, line.Content}
								found = true
								break
							}
						}
					}
					if !found {
						for _, line := range edLines {
							if parser.ContainsStemPrefix(line.Content, cleanKey) {
								resultCh <- hwResult{task.cit.ID, line.ID, editionID, line.Content}
								break
							}
						}
					}
				}
			}
		}()
	}

	go func() {
		for _, t := range tasks {
			taskCh <- t
		}
		close(taskCh)
	}()
	go func() { wg.Wait(); close(resultCh) }()

	// Phase 3: Insert results (sequential).
	tx, err := database.Begin()
	if err != nil {
		return 0
	}
	stmt, err := tx.Prepare(fmt.Sprintf(`
		INSERT INTO citation_matches (citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
		VALUES (?, ?, ?, 'headword', %g, ?)`, confidence))
	if err != nil {
		tx.Rollback()
		return 0
	}

	matched := 0
	for result := range resultCh {
		stmt.Exec(result.citID, result.lineID, result.editionID, result.content)
		matched++
	}
	stmt.Close()
	tx.Commit()

	fmt.Printf("  %s: %d\n", matchLabel, matched)
	return matched
}

// splitOnEllipsis splits a Schmidt quote on ellipsis markers ("..." or "…").
// Schmidt frequently elides text mid-quote (e.g. "to be ... not to be"),
// making the full string unmatchable as a substring. Each non-empty fragment
// can be tried independently as a shorter, matchable substring.
// applyManualCorrections inserts citation matches from the manually curated
// corrections files (projects/data/citation_corrections/*.json). These handle
// citations that can't be matched automatically due to spelling variants,
// matchByCrossPlaySearch resolves remaining unmatched citations by searching
// across ALL plays. Some citations in the Perseus source data have the wrong
// play attribution (e.g., "Tooth: H4A IV, 4, 375" is actually in Wint.).
//
// For each unmatched citation with a line number, this searches every play's
// text for a line near that number containing the headword.
func matchByCrossPlaySearch(database *sql.DB) int {
	// Load unmatched citations that have a line number.
	rows, err := database.Query(`
		SELECT lc.id, le.key, lc.line, lc.act
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		WHERE lc.work_id IS NOT NULL
		  AND lc.line IS NOT NULL
		  AND lc.id NOT IN (SELECT citation_id FROM citation_matches)`)
	if err != nil {
		return 0
	}

	type crossCit struct {
		id       int64
		headword string
		line     int
		act      int
	}
	var cits []crossCit
	for rows.Next() {
		var c crossCit
		var act sql.NullInt64
		rows.Scan(&c.id, &c.headword, &c.line, &act)
		if act.Valid {
			c.act = int(act.Int64)
		}
		cits = append(cits, c)
	}
	rows.Close()

	if len(cits) == 0 {
		return 0
	}

	fmt.Printf("  Cross-play search: %d candidates\n", len(cits))

	tx, err := database.Begin()
	if err != nil {
		return 0
	}

	insertStmt, err := tx.Prepare(`
		INSERT INTO citation_matches (citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
		SELECT ?, tl.id, tl.edition_id, 'cross_play', 0.6, tl.content
		FROM text_lines tl
		WHERE tl.id = ?
		  AND NOT EXISTS (
			SELECT 1 FROM citation_matches cm
			WHERE cm.citation_id = ? AND cm.edition_id = tl.edition_id
		  )`)
	if err != nil {
		tx.Rollback()
		return 0
	}

	matched := 0
	for _, c := range cits {
		headNorm := parser.NormalizeForMatch(stripSenseNumber(c.headword))
		if len(headNorm) < 2 {
			continue
		}

		// Search all plays for lines near this line number containing the headword.
		// Use a ±10 window around the cited line number.
		searchRows, err := database.Query(`
			SELECT tl.id, tl.content, tl.line_number, tl.edition_id
			FROM text_lines tl
			JOIN editions e ON e.id = tl.edition_id
			WHERE e.short_code = 'perseus_globe'
			  AND tl.line_number BETWEEN ? AND ?
			  AND (? = 0 OR tl.act = ?)
			ORDER BY ABS(tl.line_number - ?)
			LIMIT 100`, c.line-10, c.line+10, c.act, c.act, c.line)
		if err != nil {
			continue
		}

		bestLineID := int64(0)
		bestDelta := 999
		for searchRows.Next() {
			var lineID int64
			var content string
			var lineNum int
			var editionID int64
			searchRows.Scan(&lineID, &content, &lineNum, &editionID)

			contentNorm := parser.NormalizeForMatch(content)
			if strings.Contains(contentNorm, headNorm) {
				delta := lineNum - c.line
				if delta < 0 {
					delta = -delta
				}
				if delta < bestDelta {
					bestDelta = delta
					bestLineID = lineID
				}
			}
		}
		searchRows.Close()

		if bestLineID > 0 {
			insertStmt.Exec(c.id, bestLineID, c.id)
			matched++
		}
	}

	insertStmt.Close()
	tx.Commit()

	fmt.Printf("  Cross-play matches: %d\n", matched)
	return matched
}

// line number mismatches, or missing text in all editions.
//
// Citations are identified by (headword, raw_bibl) rather than database row ID,
// making corrections stable across rebuilds.
func applyManualCorrections(database *sql.DB) int {
	corrections := constants.CitationCorrections
	if len(corrections) == 0 {
		return 0
	}

	tx, err := database.Begin()
	if err != nil {
		return 0
	}

	matched := 0
	for _, c := range corrections {
		if c.Edition == "" || c.WorkID == nil || c.LineNumber == nil {
			continue // no-match entry; skip
		}
		if c.Headword == "" || c.RawBibl == "" {
			continue // missing lookup key
		}

		// Find the target text line by semantic coordinates.
		tlWhere := "tl.work_id = ? AND e.short_code = ? AND tl.line_number = ?"
		tlArgs := []any{*c.WorkID, c.Edition, *c.LineNumber}
		if c.Act != nil {
			tlWhere += " AND tl.act = ?"
			tlArgs = append(tlArgs, *c.Act)
		}
		if c.Scene != nil {
			tlWhere += " AND tl.scene = ?"
			tlArgs = append(tlArgs, *c.Scene)
		}

		var lineID, editionID int64
		var content string
		err := tx.QueryRow(
			`SELECT tl.id, tl.edition_id, tl.content
			 FROM text_lines tl
			 JOIN editions e ON e.id = tl.edition_id
			 WHERE `+tlWhere+` LIMIT 1`, tlArgs...).Scan(&lineID, &editionID, &content)
		if err != nil {
			continue // line not found in this build
		}

		// Find all citation IDs matching (headword, raw_bibl). Multiple senses
		// of the same word may cite the same passage — correct them all.
		rows, err := tx.Query(`
			SELECT lc.id FROM lexicon_citations lc
			JOIN lexicon_entries le ON le.id = lc.entry_id
			WHERE le.key = ? AND lc.raw_bibl = ?`, c.Headword, c.RawBibl)
		if err != nil {
			continue
		}

		for rows.Next() {
			var citID int64
			if rows.Scan(&citID) != nil {
				continue
			}
			res, err := tx.Exec(`
				INSERT INTO citation_matches (citation_id, text_line_id, edition_id, match_type, confidence, matched_text)
				SELECT ?, ?, ?, 'manual', ?, ?
				WHERE NOT EXISTS (
					SELECT 1 FROM citation_matches cm
					WHERE cm.citation_id = ? AND cm.edition_id = ?
				)`, citID, lineID, editionID, c.Confidence, content,
				citID, editionID)
			if err == nil {
				if n, _ := res.RowsAffected(); n > 0 {
					matched++
				}
			}
		}
		rows.Close()
	}

	tx.Commit()

	fmt.Printf("  Manual corrections: %d matches\n", matched)
	return matched
}

func splitOnEllipsis(s string) []string {
	s = strings.ReplaceAll(s, "…", "...")
	parts := strings.Split(s, "...")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
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

// fixMisattributedCitations corrects known data errors in the Schmidt TEI XML
// where citations are assigned to the wrong play. Returns the number of corrections.
//
// Two classes of error are fixed:
//
//  1. Shr. Ind. citations: Schmidt's raw_bibl says "Shr. Ind. X, Y" (Taming of the
//     Shrew, Induction scene X, line Y) but the TEI bibl ref points to the wrong play.
//     These are reassigned to Shr. with act=0 (the Induction act).
//
//  2. Phantom prologue citations: Schmidt cites "III Prol. Y" for plays like Antony,
//     Cymbeline, Hamlet etc. that have no act-level prologues. These are actually
//     Pericles Gower choruses, H5 choruses, or Romeo prologues. The work_id is
//     reassigned to whichever chorus play has text at that act+scene=0.
func fixMisattributedCitations(database *sql.DB) int {
	fixed := 0

	// --- Fix 1: Shr. Ind. citations ---
	// Find the Shr. work_id
	var shrID int64
	err := database.QueryRow("SELECT id FROM works WHERE schmidt_abbrev = 'Shr.'").Scan(&shrID)
	if err == nil {
		res, err := database.Exec(`
			UPDATE lexicon_citations
			SET work_id = ?, act = 0
			WHERE raw_bibl LIKE 'Shr. Ind%'
			  AND work_id != ?`, shrID, shrID)
		if err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				fmt.Printf("    Shr. Ind.: reassigned %d citations to Shrew Induction\n", n)
				fixed += int(n)
			}
		}
	}

	// --- Fix 2: Correct act/scene for "I Chor." / "I Prol." citations ---
	// The parser correctly assigns these to Romeo/H5 but with act=1, scene=0
	// (literal parse of "I Chor."). The actual text lives at act=0, scene=1
	// (the prologue before act 1). Without this fix, Fix 3 (phantom prologue)
	// sees "no text at act=1 scene=0" and reassigns them to Pericles.
	actSceneFixes := []struct {
		sql   string
		label string
	}{
		{`UPDATE lexicon_citations SET act = 0, scene = 1 WHERE raw_bibl = 'Rom. I Chor.' AND act = 1 AND scene = 0`, "Rom. I Chor."},
		{`UPDATE lexicon_citations SET act = 0, scene = 1 WHERE raw_bibl = 'Rom. I Prol.' AND act = 1 AND scene = 0`, "Rom. I Prol."},
		{`UPDATE lexicon_citations SET act = 0, scene = 1 WHERE raw_bibl = 'H5 I Chor.' AND act = 1 AND scene = 0`, "H5 I Chor."},
	}
	for _, af := range actSceneFixes {
		res, err := database.Exec(af.sql)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			fmt.Printf("    %s: corrected act/scene for %d citations\n", af.label, n)
			fixed += int(n)
		}
	}

	// --- Fix 3: Phantom prologue citations ---
	// Find plays that actually have scene=0 text (choruses/prologues).
	chorusPlays, err := database.Query(`
		SELECT DISTINCT w.id, tl.act
		FROM text_lines tl
		JOIN works w ON w.id = tl.work_id
		WHERE tl.scene = 0
		GROUP BY w.id, tl.act
		HAVING COUNT(*) > 5`)
	if err != nil {
		return fixed
	}
	type chorusKey struct {
		WorkID int64
		Act    int
	}
	var chorusScenes []chorusKey
	for chorusPlays.Next() {
		var ck chorusKey
		chorusPlays.Scan(&ck.WorkID, &ck.Act)
		chorusScenes = append(chorusScenes, ck)
	}
	chorusPlays.Close()

	// For each unmatched scene=0 citation where the play has NO scene=0 text,
	// try to reassign to a chorus play that does.
	rows, err := database.Query(`
		SELECT lc.id, lc.work_id, lc.act, lc.line, lc.quote_text, le.key
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		WHERE lc.scene = 0
		  AND NOT EXISTS (
			SELECT 1 FROM text_lines tl
			WHERE tl.work_id = lc.work_id AND tl.act = lc.act AND tl.scene = 0
		  )`)
	if err != nil {
		return fixed
	}

	type prologueCit struct {
		ID       int64
		WorkID   int64
		Act      int
		Line     *int
		Quote    string
		Headword string
	}
	var pCits []prologueCit
	for rows.Next() {
		var pc prologueCit
		var line sql.NullInt64
		var quote sql.NullString
		rows.Scan(&pc.ID, &pc.WorkID, &pc.Act, &line, &quote, &pc.Headword)
		if line.Valid {
			v := int(line.Int64)
			pc.Line = &v
		}
		if quote.Valid {
			pc.Quote = quote.String
		}
		pCits = append(pCits, pc)
	}
	rows.Close()

	if len(pCits) == 0 {
		return fixed
	}

	// For each phantom prologue citation, find the correct chorus play.
	// Strategy: try each chorus play that has the same act+scene=0.
	// Pick the one where the headword appears in the text (or line number exists).
	stmt, err := database.Prepare("UPDATE lexicon_citations SET work_id = ? WHERE id = ?")
	if err != nil {
		return fixed
	}
	defer stmt.Close()

	for _, pc := range pCits {
		// Find candidate chorus plays for this act
		var candidates []int64
		for _, ck := range chorusScenes {
			if ck.Act == pc.Act {
				candidates = append(candidates, ck.WorkID)
			}
		}
		if len(candidates) == 0 {
			continue
		}

		bestWork := int64(0)

		// Try matching by headword in each candidate's chorus text
		cleanKey := stripSenseNumber(pc.Headword)
		for _, workID := range candidates {
			lines, err := loadTextLinesAll(database, "work_id = ? AND act = ? AND scene = 0",
				workID, pc.Act)
			if err != nil || len(lines) == 0 {
				continue
			}

			// Check headword match
			for _, line := range lines {
				if parser.ContainsNormalized(line.Content, cleanKey) {
					bestWork = workID
					break
				}
			}
			if bestWork != 0 {
				break
			}

			// Check line number match
			if pc.Line != nil {
				for _, line := range lines {
					if line.LineNumber == *pc.Line {
						bestWork = workID
						break
					}
				}
			}
			if bestWork != 0 {
				break
			}
		}

		// If only one candidate, assign even without text confirmation
		if bestWork == 0 && len(candidates) == 1 {
			bestWork = candidates[0]
		}

		if bestWork != 0 && bestWork != pc.WorkID {
			stmt.Exec(bestWork, pc.ID)
			fixed++
		}
	}

	if fixed > 0 {
		fmt.Printf("    Phantom prologues: reassigned %d citations to correct chorus plays\n", fixed)
	}

	return fixed
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
// This function is safe for concurrent use — it has no side effects.
//
// Performance: all text normalization is done ONCE up front. The previous
// implementation re-normalized each line's content 5-9 times across the
// different matching strategies, dominating CPU time.
func findBestMatch(lines []textLineRow, cit citationRow) (*textLineRow, string, float64) {
	if len(lines) == 0 {
		return nil, "", 0
	}

	// === Pre-compute all normalizations ONCE ===
	// This eliminates millions of redundant NormalizeForMatch calls.
	// Each line is normalized exactly once; each quote variant is normalized once.

	// Normalized text for each line (used by substring search and word containment).
	lineNorms := make([]string, len(lines))
	for i := range lines {
		lineNorms[i] = parser.NormalizeForMatch(lines[i].Content)
	}

	// Pre-compute word sets for each line (used by Jaccard similarity).
	lineWordSets := make([]map[string]bool, len(lines))
	for i, norm := range lineNorms {
		words := strings.Fields(norm)
		set := make(map[string]bool, len(words))
		for _, w := range words {
			set[w] = true
		}
		lineWordSets[i] = set
	}

	// Line-number index for O(1) lookup (used by Strategy 2).
	lineNumIdx := make(map[int]int, len(lines))
	for i, line := range lines {
		lineNumIdx[line.LineNumber] = i
	}

	// Pre-compute quote variants (normalized).
	var normQuote, normExpanded string
	var quoteWordSet map[string]bool
	wasExpanded := false

	if cit.QuoteText != "" {
		cleanQuote := strings.TrimSpace(strings.ReplaceAll(cit.QuoteText, "--", ""))
		normQuote = parser.NormalizeForMatch(cleanQuote)

		headword := stripSenseNumber(cit.Headword)
		expanded, exp := expandQuoteAbbreviation(cit.QuoteText, headword)
		if exp {
			wasExpanded = true
			expanded = strings.TrimSpace(strings.ReplaceAll(expanded, "--", ""))
			normExpanded = parser.NormalizeForMatch(expanded)
		}

		// Word set for quote (used by Jaccard and word containment).
		qwords := strings.Fields(normQuote)
		quoteWordSet = make(map[string]bool, len(qwords))
		for _, w := range qwords {
			quoteWordSet[w] = true
		}
	}

	// jaccardFromSets computes Jaccard index from pre-computed word sets.
	jaccardSets := func(lineIdx int) float64 {
		setA := lineWordSets[lineIdx]
		setB := quoteWordSet
		if len(setA) == 0 && len(setB) == 0 {
			return 1.0
		}
		if len(setA) == 0 || len(setB) == 0 {
			return 0.0
		}
		intersection := 0
		for w := range setA {
			if setB[w] {
				intersection++
			}
		}
		union := len(setA) + len(setB) - intersection
		if union == 0 {
			return 0.0
		}
		return float64(intersection) / float64(union)
	}

	// Strategy 1: Exact quote match (highest confidence)
	if normQuote != "" && len(normQuote) > 3 {
		// 1a: Single-line substring match (original quote).
		for i := range lines {
			if strings.Contains(lineNorms[i], normQuote) {
				return &lines[i], "exact_quote", 1.0
			}
		}

		// 1b: Single-line match with expanded abbreviation.
		if wasExpanded && len(normExpanded) > 3 {
			for i := range lines {
				if strings.Contains(lineNorms[i], normExpanded) {
					return &lines[i], "exact_quote", 0.95
				}
			}
		}

		// 1c: Multi-line match — Schmidt quotes often span two verse lines.
		if len(lines) > 1 {
			for i := 0; i < len(lines)-1; i++ {
				combined := lineNorms[i] + " " + lineNorms[i+1]
				if strings.Contains(combined, normQuote) {
					return &lines[i], "exact_quote", 0.95
				}
				if wasExpanded && strings.Contains(combined, normExpanded) {
					return &lines[i], "exact_quote", 0.90
				}
			}
		}

		// 1d: Ellipsis split — Schmidt uses "..." to elide text within quotes.
		fragments := splitOnEllipsis(normQuote)
		if len(fragments) > 1 {
			for _, frag := range fragments {
				frag = strings.TrimSpace(frag)
				if len(frag) <= 3 {
					continue
				}
				for i := range lines {
					if strings.Contains(lineNorms[i], frag) {
						return &lines[i], "exact_quote", 0.85
					}
				}
			}
		}

		// 1e: Word-set containment — for short quotes (2–6 words), check whether
		// every word in the quote appears somewhere in the line (any order).
		quoteWords := strings.Fields(normQuote)
		if len(quoteWords) >= 2 && len(quoteWords) <= 6 {
			for i := range lines {
				allFound := true
				for _, w := range quoteWords {
					if !strings.Contains(lineNorms[i], w) {
						allFound = false
						break
					}
				}
				if allFound {
					return &lines[i], "exact_quote", 0.80
				}
			}
		}
	}

	// Strategy 2: Line number match with headword verification.
	//
	// Schmidt uses Globe line numbers, but different editions number lines
	// differently (stage directions, scene breaks, etc. shift numbering).
	// When the line at the cited number doesn't contain the headword, we
	// search nearby lines (±20) for one that does.
	if cit.Line != nil {
		headword := strings.ToLower(stripSenseNumber(cit.Headword))
		headwordNorm := parser.NormalizeForMatch(headword)

		// lineContainsHeadword checks if a line contains the headword.
		lineHasWord := func(idx int) bool {
			if headwordNorm == "" || len(headwordNorm) < 2 {
				return false // too short to verify
			}
			return strings.Contains(lineNorms[idx], headwordNorm)
		}

		// Can we verify via headword? Need at least 2 normalized chars.
		canVerifyHeadword := len(headwordNorm) >= 2

		// Try exact line number first.
		if idx, ok := lineNumIdx[*cit.Line]; ok {
			// Headword is in the line → accept.
			if canVerifyHeadword && lineHasWord(idx) {
				return &lines[idx], "line_number", 0.9
			}
			// Headword check passed via Jaccard similarity with the quote.
			if !canVerifyHeadword && quoteWordSet != nil {
				sim := jaccardSets(idx)
				if sim > 0.3 {
					return &lines[idx], "line_number", 0.9
				}
			}
			// If we can't verify by headword at all, accept the line number match.
			if !canVerifyHeadword && quoteWordSet == nil {
				return &lines[idx], "line_number", 0.9
			}
			// Line number matches but headword isn't there.
			// Don't accept yet — fall through to search nearby lines.
		}

		// Search nearby lines (±20) for one containing the headword.
		if canVerifyHeadword {
			const maxDelta = 20
			bestIdx := -1
			bestScore := 0.0

			for delta := 1; delta <= maxDelta; delta++ {
				for _, d := range []int{-delta, delta} {
					idx, ok := lineNumIdx[*cit.Line+d]
					if !ok {
						continue
					}

					if !lineHasWord(idx) {
						continue // headword must be present
					}

					var score float64
					if quoteWordSet != nil {
						score = jaccardSets(idx) + 0.5
					} else {
						score = 1.0 - float64(delta)*0.02
					}

					if score > bestScore {
						bestScore = score
						bestIdx = idx
					}
				}
			}

			if bestIdx >= 0 {
				if bestScore > 0.5 {
					return &lines[bestIdx], "line_number", 0.8
				}
				return &lines[bestIdx], "line_number", 0.6
			}
		}

		// Fall back to exact line number — only if we can't verify headword
		// (headword too short). If we CAN verify but didn't find it, don't
		// return a bad match — let later phases (headword search) handle it.
		if !canVerifyHeadword {
			if idx, ok := lineNumIdx[*cit.Line]; ok {
				return &lines[idx], "line_number", 0.7
			}
		}
	}

	// Strategy 3: Fuzzy text match (last resort) — uses pre-computed word sets.
	// Prefer lines that contain the headword.
	if quoteWordSet != nil {
		headword := strings.ToLower(stripSenseNumber(cit.Headword))
		hwNorm := parser.NormalizeForMatch(headword)
		canVerify := len(hwNorm) >= 2

		bestScore := 0.0
		bestIdx := -1
		for i := range lines {
			score := jaccardSets(i)
			if canVerify && !strings.Contains(lineNorms[i], hwNorm) {
				score *= 0.3 // heavy penalty for missing headword
			}
			if score > bestScore {
				bestScore = score
				bestIdx = i
			}
		}
		if bestScore > 0.15 && bestIdx >= 0 {
			return &lines[bestIdx], "fuzzy_text", bestScore
		}
	}

	return nil, "", 0
}
