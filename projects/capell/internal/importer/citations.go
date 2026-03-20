// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"runtime"
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
	stepBanner("STEP 9: Resolve Lexicon Citations → Text Lines")

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

		// Fallback: scene lookup returned nothing → the 2-part Perseus reference
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

		for _, cit := range sceneCitations {
			tasks = append(tasks, citMatchTask{cit: cit, lines: lines})
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
	workers := runtime.NumCPU()
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

	// === Phase 7: Apply manual corrections from citation_corrections/*.json ===
	manualMatches := applyManualCorrections(database)
	totalMatches += manualMatches

	// Final stats from database
	var finalExact, finalLine, finalFuzzy, finalPropagated, finalHeadword, finalManual, finalTotal int
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'exact_quote'").Scan(&finalExact)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'line_number'").Scan(&finalLine)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'fuzzy_text'").Scan(&finalFuzzy)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'propagated'").Scan(&finalPropagated)
	database.QueryRow("SELECT COUNT(*) FROM citation_matches WHERE match_type = 'headword'").Scan(&finalHeadword)
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
		fmt.Sprintf("%d matches (exact=%d, line=%d, fuzzy=%d, propagated=%d, headword=%d, manual=%d, unmatched_citations=%d)",
			finalTotal, finalExact, finalLine, finalFuzzy, finalPropagated, finalHeadword, finalManual, finalUnmatched),
		finalTotal, elapsed)

	fmt.Printf("  ✓ %d total matches in %.1fs\n", finalTotal, elapsed)
	fmt.Printf("    exact_quote: %d, line_number: %d, fuzzy_text: %d\n", finalExact, finalLine, finalFuzzy)
	fmt.Printf("    propagated: %d, headword: %d, manual: %d\n", finalPropagated, finalHeadword, finalManual)
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

	workers := max(1, runtime.NumCPU())
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

			// 1d: Ellipsis split — Schmidt uses "..." to elide text within quotes.
			// Try each fragment independently as a substring match.
			fragments := splitOnEllipsis(cleanQuote)
			if len(fragments) > 1 {
				for _, frag := range fragments {
					if len(frag) <= 3 {
						continue
					}
					for i, line := range lines {
						if parser.ContainsNormalized(line.Content, frag) {
							return &lines[i], "exact_quote", 0.85
						}
					}
				}
			}

			// 1e: Word-set containment — for short quotes (2–6 words), check whether
			// every word in the quote appears somewhere in the line (any order).
			// Catches cases where the quote is not a contiguous substring.
			quoteWords := strings.Fields(parser.NormalizeForMatch(cleanQuote))
			if len(quoteWords) >= 2 && len(quoteWords) <= 6 {
				for i, line := range lines {
					lineNorm := parser.NormalizeForMatch(line.Content)
					allFound := true
					for _, w := range quoteWords {
						if !strings.Contains(lineNorm, w) {
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
			for delta := 1; delta <= 20; delta++ {
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

			// Final fallback: when Schmidt's line number exceeds the scene's range
			// (e.g., citing Troilus line 268 in a scene that only has 168 lines),
			// return the line with the smallest absolute distance from the target.
			// This handles plays where Schmidt uses continuous line numbering across
			// scenes while Perseus resets per scene. Confidence is very low (0.1).
			closestIdx := -1
			closestDelta := -1
			for i, line := range lines {
				d := line.LineNumber - *cit.Line
				if d < 0 {
					d = -d
				}
				if closestIdx == -1 || d < closestDelta {
					closestDelta = d
					closestIdx = i
				}
			}
			if closestIdx >= 0 {
				return &lines[closestIdx], "line_number", 0.1
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
		if bestScore > 0.15 && bestIdx >= 0 {
			return &lines[bestIdx], "fuzzy_text", bestScore
		}
	}

	return nil, "", 0
}
