// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// editionPair holds the two editions being aligned.
type editionPair struct {
	AID   int64
	ACode string
	BID   int64
	BCode string
}

// mappingStats tracks alignment counts for a mapping run.
type mappingStats struct {
	total, aligned, modified, onlyA, onlyB int
}

func (s *mappingStats) record(mt string) {
	s.total++
	switch mt {
	case "aligned":
		s.aligned++
	case "modified":
		s.modified++
	case "only_a":
		s.onlyA++
	case "only_b":
		s.onlyB++
	}
}

func (s *mappingStats) add(other mappingStats) {
	s.total += other.total
	s.aligned += other.aligned
	s.modified += other.modified
	s.onlyA += other.onlyA
	s.onlyB += other.onlyB
}

// alignTask describes one unit of alignment work (a scene, sonnet, or poem
// in a single edition pair).
type alignTask struct {
	pair   editionPair
	workID int64
	act    int
	scene  int
	aLines []parser.AlignableLine
	bLines []parser.AlignableLine
	opts   parser.AlignOptions
}

// alignResult holds the output of one alignment task.
type alignResult struct {
	task  alignTask
	pairs []parser.AlignedPair
}

// lineKey uniquely identifies a (edition, work, act, scene) group.
// act=0 means sonnet (scene = sonnet number) or poem (scene=0).
type lineKey struct {
	editionID, workID int64
	act, scene        int
}

// BuildLineMappings creates cross-edition line alignments for all edition pairs.
//
// For each pair of editions that share works, it aligns lines using text similarity
// (Needleman-Wunsch algorithm with Jaccard word similarity scoring).
//
// Handles three structure types:
//   - Plays: aligns scenes (act + scene) across editions
//   - Sonnets: aligns individual sonnets (scene = sonnet number) across editions
//   - Poems: aligns entire poems (by work_id) across editions
//
// Loading phase: ONE bulk DB query loads all alignable lines into an in-memory
// cache keyed by (edition_id, work_id, act, scene). Alignment tasks are built
// from this cache without any further DB access during loading.
//
// Alignment computation is parallelized across goroutines since it is CPU-bound,
// while DB reads (loading lines) and writes (inserting mappings) remain sequential.
func BuildLineMappings(database *sql.DB) error {
	stepBanner("Build Cross-Edition Line Mappings")

	start := time.Now()

	// Clear existing mappings
	database.Exec("DELETE FROM line_mappings")

	// Get all editions that have text_lines
	type edInfo struct {
		ID   int64
		Code string
	}
	var editions []edInfo
	edRows, err := database.Query(`
		SELECT DISTINCT e.id, e.short_code
		FROM editions e
		JOIN text_lines tl ON tl.edition_id = e.id
		ORDER BY e.id`)
	if err != nil {
		return fmt.Errorf("querying editions: %w", err)
	}
	for edRows.Next() {
		var e edInfo
		edRows.Scan(&e.ID, &e.Code)
		editions = append(editions, e)
	}
	edRows.Close()

	if len(editions) < 2 {
		fmt.Println("  Need at least 2 editions. Skipping.")
		return nil
	}

	// Build edition → work set from a quick DB query so we can skip pairs
	// that share no works. This eliminates nonsensical comparisons like
	// q1_titus_1594↔q1_henry6p2_1594 (completely different plays).
	editionWorkSet := buildEditionWorkSet(database)

	var pairs []editionPair
	for i := 0; i < len(editions); i++ {
		for j := i + 1; j < len(editions); j++ {
			if !sharesWork(editionWorkSet, editions[i].ID, editions[j].ID) {
				continue
			}
			pairs = append(pairs, editionPair{
				AID: editions[i].ID, ACode: editions[i].Code,
				BID: editions[j].ID, BCode: editions[j].Code,
			})
		}
	}

	// Count distinct works that have at least one text line in any edition.
	var totalWorks int
	database.QueryRow(`
		SELECT COUNT(DISTINCT work_id) FROM text_lines
		WHERE act IS NOT NULL AND act > 0`).Scan(&totalWorks)

	fmt.Printf("  Editions: %d  →  %d meaningful pairs (skipped editions with no shared works)\n",
		len(editions), len(pairs))
	fmt.Printf("  Works with play text: %d\n", totalWorks)

	// === Phase 1: Bulk-load ALL alignable lines in ONE query ===
	//
	// Previous approach: loadAlignTasks made ~2 DB queries per scene per pair
	// (one per edition), totalling ~50k queries for a full 9-edition build.
	//
	// New approach: one query loads everything into a map keyed by
	// (edition_id, work_id, act, scene). loadAlignTasksFromCache then
	// assembles tasks entirely from this in-memory map.
	lineCache := buildLineCache(database)
	// Also build a set of all (work_id, work_type) for poem/sonnet detection
	workTypes := buildWorkTypes(database)

	fmt.Printf("  Line cache: %d groups loaded in %.1fs\n",
		len(lineCache), time.Since(start).Seconds())

	// Phase 1b: Build alignment tasks from cache (no DB access)
	var tasks []alignTask
	for _, pair := range pairs {
		tasks = append(tasks, loadAlignTasksFromCache(lineCache, workTypes, pair)...)
	}

	// Report task size distribution
	var bigTasks int
	for _, t := range tasks {
		cells := int64(len(t.aLines)) * int64(len(t.bLines))
		if cells > 100_000 {
			bigTasks++
		}
	}
	fmt.Printf("  Alignment tasks: %d (%d large) (loading done in %.1fs)\n",
		len(tasks), bigTasks, time.Since(start).Seconds())

	// Phase 2: Run alignments in parallel (CPU-bound, no DB access).
	workers := workerCount()

	// Sort tasks largest-first so big tasks start early and don't become tail latency.
	sort.Slice(tasks, func(i, j int) bool {
		sizeI := int64(len(tasks[i].aLines)) * int64(len(tasks[i].bLines))
		sizeJ := int64(len(tasks[j].aLines)) * int64(len(tasks[j].bLines))
		return sizeI > sizeJ
	})

	taskCh := make(chan alignTask, len(tasks))
	resultCh := make(chan alignResult, 256) // small buffer to reduce memory pressure

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				aligned := parser.AlignSequences(task.aLines, task.bLines, task.opts)
				resultCh <- alignResult{task: task, pairs: aligned}
			}
		}()
	}

	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	// Close resultCh once all workers finish.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Phase 3: Insert results (sequential — single SQLite transaction).
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	insertStmt, err := tx.Prepare(`
		INSERT INTO line_mappings (work_id, act, scene, align_order, edition_a_id, edition_b_id,
			line_a_id, line_b_id, match_type, similarity)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("preparing insert: %w", err)
	}

	worksProcessed := make(map[int64]bool)
	var totalStats mappingStats
	pairStats := make(map[string]*mappingStats)

	for result := range resultCh {
		t := result.task
		pairKey := fmt.Sprintf("%s↔%s", t.pair.ACode, t.pair.BCode)
		ps, ok := pairStats[pairKey]
		if !ok {
			ps = &mappingStats{}
			pairStats[pairKey] = ps
		}

		for i, p := range result.pairs {
			var lineAID, lineBID any
			if p.LineA != nil {
				lineAID = p.LineA.ID
			}
			if p.LineB != nil {
				lineBID = p.LineB.ID
			}
			insertStmt.Exec(t.workID, t.act, t.scene, i+1,
				t.pair.AID, t.pair.BID,
				lineAID, lineBID,
				p.MatchType, p.Similarity)
			ps.record(p.MatchType)
		}
		worksProcessed[t.workID] = true
	}

	insertStmt.Close()
	tx.Commit()

	// Aggregate and print per-pair stats
	for _, pair := range pairs {
		pairKey := fmt.Sprintf("%s↔%s", pair.ACode, pair.BCode)
		if ps, ok := pairStats[pairKey]; ok {
			totalStats.add(*ps)
			fmt.Printf("    %s: %d pairs (aligned=%d)\n",
				pairKey, ps.total, ps.aligned)
		}
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "mappings", "build_complete",
		fmt.Sprintf("%d pairs across %d works: aligned=%d modified=%d only_a=%d only_b=%d",
			totalStats.total, len(worksProcessed), totalStats.aligned, totalStats.modified,
			totalStats.onlyA, totalStats.onlyB),
		totalStats.total, elapsed)

	fmt.Printf("  ✓ %d alignment pairs across %d works in %.1fs (%d workers, %d CPUs)\n",
		totalStats.total, len(worksProcessed), elapsed, workers, runtime.NumCPU())
	fmt.Printf("    aligned: %d, modified: %d, only_a: %d, only_b: %d\n",
		totalStats.aligned, totalStats.modified, totalStats.onlyA, totalStats.onlyB)
	return nil
}

// buildEditionWorkSet returns a map from edition_id → set of work_ids that
// have at least one text line in that edition.
func buildEditionWorkSet(database *sql.DB) map[int64]map[int64]bool {
	m := make(map[int64]map[int64]bool)
	rows, err := database.Query(`SELECT DISTINCT edition_id, work_id FROM text_lines`)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var edID, wID int64
		rows.Scan(&edID, &wID)
		if m[edID] == nil {
			m[edID] = make(map[int64]bool)
		}
		m[edID][wID] = true
	}
	return m
}

// sharesWork returns true if editions A and B have at least one work in common.
func sharesWork(editionWorks map[int64]map[int64]bool, aID, bID int64) bool {
	for wID := range editionWorks[aID] {
		if editionWorks[bID][wID] {
			return true
		}
	}
	return false
}

// buildLineCache loads ALL text lines in a single DB query and indexes them by
// (edition_id, work_id, act, scene). act and scene are stored as 0 when NULL.
//
// This replaces the previous per-scene DB queries in loadAlignTasks, reducing
// ~50k individual queries (for a 9-edition build) to a single bulk read.
func buildLineCache(database *sql.DB) map[lineKey][]parser.AlignableLine {
	cache := make(map[lineKey][]parser.AlignableLine)

	// Exclude 'scene' content_type rows (OSS scene-header lines such as
	// "SCENE I. A forest." that exist only in the OSS edition and have no
	// counterpart in other editions, inflating only_a counts).
	//
	// Normalize act to 0 for non-play works (sonnet_sequence, poem). The OSS
	// edition stores sonnets and poems with act=1 (a dummy section value) while
	// other editions use act=NULL. The alignment task builder classifies lines
	// by act: act>0 → play scene, act=0 → sonnet or poem. Normalising here
	// ensures OSS sonnets/poems land in the correct bucket regardless of source.
	rows, err := database.Query(`
		SELECT tl.id, tl.content, COALESCE(tl.line_number, 0), tl.edition_id, tl.work_id,
		       CASE WHEN w.work_type IN ('sonnet_sequence', 'poem') THEN 0
		            ELSE COALESCE(tl.act, 0) END,
		       COALESCE(tl.scene, 0),
		       COALESCE(tl.content_type, 'speech')
		FROM text_lines tl
		JOIN works w ON w.id = tl.work_id
		WHERE tl.content_type != 'scene' OR tl.content_type IS NULL
		ORDER BY tl.edition_id, tl.work_id, tl.act, tl.scene, tl.line_number, tl.id`)
	if err != nil {
		return cache
	}
	defer rows.Close()

	for rows.Next() {
		var l parser.AlignableLine
		var editionID, workID int64
		var act, scene int
		rows.Scan(&l.ID, &l.Content, &l.LineNumber, &editionID, &workID, &act, &scene, &l.ContentType)
		l.Words = parser.WordSet(l.Content)
		k := lineKey{editionID, workID, act, scene}
		cache[k] = append(cache[k], l)
	}
	return cache
}

// buildWorkTypes returns a map from work_id → work type string.
func buildWorkTypes(database *sql.DB) map[int64]string {
	m := make(map[int64]string)
	rows, err := database.Query("SELECT id, work_type FROM works")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var wt string
		rows.Scan(&id, &wt)
		m[id] = wt
	}
	return m
}

// loadAlignTasksFromCache builds alignment tasks for a pair entirely from the
// in-memory line cache — no DB access. Replaces loadAlignTasks.
// isGlobeEdition returns true for editions whose line numbers follow the Globe
// numbering scheme (oss_globe, perseus_globe). When both editions in a pair use
// Globe numbering, line-number affinity is enabled to anchor NW alignment and
// reduce drift in scenes with many short or common words.
func isGlobeEdition(code string) bool {
	return strings.Contains(code, "globe")
}

func loadAlignTasksFromCache(
	lineCache map[lineKey][]parser.AlignableLine,
	workTypes map[int64]string,
	pair editionPair,
) []alignTask {
	// Build per-pair alignment options.
	var opts parser.AlignOptions
	if isGlobeEdition(pair.ACode) && isGlobeEdition(pair.BCode) {
		// Both editions share Globe line numbering: add affinity so that NW
		// strongly prefers matching lines with the same (or adjacent) line numbers.
		opts.LineNumberAffinity = 0.15
	}
	// Collect all keys visible from this pair (present in either edition A or B)
	// and group them by content type: plays, sonnets, poems.
	type sceneRef struct {
		workID     int64
		act, scene int
	}

	playScenesSet := make(map[sceneRef]bool)
	sonnetScenesSet := make(map[sceneRef]bool)
	poemWorksSet := make(map[int64]bool)

	for k := range lineCache {
		if k.editionID != pair.AID && k.editionID != pair.BID {
			continue
		}
		wt := workTypes[k.workID]
		if k.act > 0 {
			// Play scene
			playScenesSet[sceneRef{k.workID, k.act, k.scene}] = true
		} else if k.scene > 0 && wt == "sonnet_sequence" {
			// Sonnet
			sonnetScenesSet[sceneRef{k.workID, 0, k.scene}] = true
		} else if k.scene == 0 && k.act == 0 && wt == "poem" {
			// Poem
			poemWorksSet[k.workID] = true
		}
	}

	var tasks []alignTask

	// === 1. Play scenes ===
	// Group scenes by work so we can detect flat editions (e.g. EEBO-TCP quartos
	// that store all lines in act=1, scene=1). When one edition is flat and the
	// other is structured, aligning scene-by-scene yields 0 matches because the
	// entire quarto sits in a single (1,1) bucket while the other edition spreads
	// across 20+ scenes. In that case we merge all scenes of the structured edition
	// into one work-level task so simpleAlign can do positional matching.
	workToScenes := make(map[int64][]sceneRef)
	for ref := range playScenesSet {
		workToScenes[ref.workID] = append(workToScenes[ref.workID], ref)
	}

	for workID, refs := range workToScenes {
		// Sort refs by (act, scene) so merged line slices are in narrative order.
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].act != refs[j].act {
				return refs[i].act < refs[j].act
			}
			return refs[i].scene < refs[j].scene
		})

		// Count how many distinct scenes each edition has for this work.
		aScenes, bScenes := 0, 0
		for _, ref := range refs {
			if len(lineCache[lineKey{pair.AID, workID, ref.act, ref.scene}]) > 0 {
				aScenes++
			}
			if len(lineCache[lineKey{pair.BID, workID, ref.act, ref.scene}]) > 0 {
				bScenes++
			}
		}

		// Detect when two editions have incompatible scene structures and
		// fall back to work-level alignment. This catches three cases:
		//
		// 1. One edition is truly flat (1 scene) — e.g. EEBO-TCP quartos.
		// 2. One edition has partial act/scene divisions — e.g. the First
		//    Folio's Hamlet has 4 scene entries vs OSS Globe's 20. Per-scene
		//    alignment would leave most scenes matched against nothing.
		// 3. Scenes don't overlap — editions use different numbering schemes.
		//
		// We measure scene overlap: the fraction of scenes present in BOTH
		// editions out of the total distinct scenes across both. When overlap
		// is below 50%, per-scene alignment wastes most lines as only_a/only_b.
		//
		// NOTE: when one edition has no lines at all for this work (aScenes==0 or
		// bScenes==0), we fall through to the per-scene path, which produces
		// only_a/only_b entries for the present edition's lines. This is correct:
		// a play present in OSS but absent from SE should appear as only_a rows in
		// the comparison, allowing the front-end to render one side empty.
		var sharedScenes int
		for _, ref := range refs {
			hasA := len(lineCache[lineKey{pair.AID, workID, ref.act, ref.scene}]) > 0
			hasB := len(lineCache[lineKey{pair.BID, workID, ref.act, ref.scene}]) > 0
			if hasA && hasB {
				sharedScenes++
			}
		}
		totalScenes := len(refs) // distinct (act, scene) across both editions
		sceneOverlap := 0.0
		if totalScenes > 0 {
			sceneOverlap = float64(sharedScenes) / float64(totalScenes)
		}
		structureMismatch := aScenes > 0 && bScenes > 0 && sceneOverlap < 0.5

		if structureMismatch {
			// Scene structures are incompatible: merge all lines for this work
			// into one work-level task (act=0, scene=0 sentinel, same as poems).
			var aAll, bAll []parser.AlignableLine
			for _, ref := range refs {
				aAll = append(aAll, lineCache[lineKey{pair.AID, workID, ref.act, ref.scene}]...)
				bAll = append(bAll, lineCache[lineKey{pair.BID, workID, ref.act, ref.scene}]...)
			}
			if len(aAll) > 0 || len(bAll) > 0 {
				tasks = append(tasks, alignTask{
					pair: pair, workID: workID,
					act: 0, scene: 0,
					aLines: aAll, bLines: bAll,
					opts: opts,
				})
			}
		} else {
			// Both structured or both flat: align per scene as normal.
			for _, ref := range refs {
				aLines := lineCache[lineKey{pair.AID, workID, ref.act, ref.scene}]
				bLines := lineCache[lineKey{pair.BID, workID, ref.act, ref.scene}]
				if len(aLines) == 0 && len(bLines) == 0 {
					continue
				}
				tasks = append(tasks, alignTask{
					pair: pair, workID: workID,
					act: ref.act, scene: ref.scene,
					aLines: aLines, bLines: bLines,
					opts: opts,
				})
			}
		}
	}

	// === 2. Sonnets ===
	for ref := range sonnetScenesSet {
		aLines := lineCache[lineKey{pair.AID, ref.workID, 0, ref.scene}]
		bLines := lineCache[lineKey{pair.BID, ref.workID, 0, ref.scene}]
		if len(aLines) == 0 && len(bLines) == 0 {
			continue
		}
		tasks = append(tasks, alignTask{
			pair: pair, workID: ref.workID,
			act: 0, scene: ref.scene,
			aLines: aLines, bLines: bLines,
			opts: opts,
		})
	}

	// === 3. Poems ===
	for workID := range poemWorksSet {
		aLines := lineCache[lineKey{pair.AID, workID, 0, 0}]
		bLines := lineCache[lineKey{pair.BID, workID, 0, 0}]
		if len(aLines) == 0 && len(bLines) == 0 {
			continue
		}
		tasks = append(tasks, alignTask{
			pair: pair, workID: workID,
			act: 0, scene: 0,
			aLines: aLines, bLines: bLines,
			opts: opts,
		})
	}

	return tasks
}
