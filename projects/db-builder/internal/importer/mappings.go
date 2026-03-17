// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/scottdkey/heminge/projects/db-builder/internal/db"
	"github.com/scottdkey/heminge/projects/db-builder/internal/parser"
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
}

// alignResult holds the output of one alignment task.
type alignResult struct {
	task  alignTask
	pairs []parser.AlignedPair
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
// Alignment computation is parallelized across goroutines since it is CPU-bound,
// while DB reads (loading lines) and writes (inserting mappings) remain sequential.
func BuildLineMappings(database *sql.DB) error {
	stepBanner("STEP 8: Build Cross-Edition Line Mappings")

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

	// Build all edition pairs
	var pairs []editionPair
	for i := 0; i < len(editions); i++ {
		for j := i + 1; j < len(editions); j++ {
			pairs = append(pairs, editionPair{
				AID: editions[i].ID, ACode: editions[i].Code,
				BID: editions[j].ID, BCode: editions[j].Code,
			})
		}
	}

	fmt.Printf("  Edition pairs: %d\n", len(pairs))

	// Phase 1: Load all alignment tasks from DB (sequential — SQLite reads).
	var tasks []alignTask
	for _, pair := range pairs {
		tasks = append(tasks, loadAlignTasks(database, pair)...)
	}

	fmt.Printf("  Alignment tasks: %d (loading done in %.1fs)\n",
		len(tasks), time.Since(start).Seconds())

	// Phase 2: Run alignments in parallel (CPU-bound, no DB access).
	workers := min(runtime.NumCPU(), 8)
	if workers < 1 {
		workers = 1
	}

	taskCh := make(chan alignTask, len(tasks))
	resultCh := make(chan alignResult, len(tasks))

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				aligned := parser.AlignSequences(task.aLines, task.bLines)
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
			var lineAID, lineBID interface{}
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

	fmt.Printf("  ✓ %d alignment pairs across %d works in %.1fs (%d workers)\n",
		totalStats.total, len(worksProcessed), elapsed, workers)
	fmt.Printf("    aligned: %d, modified: %d, only_a: %d, only_b: %d\n",
		totalStats.aligned, totalStats.modified, totalStats.onlyA, totalStats.onlyB)
	return nil
}

// loadAlignTasks queries the DB for all alignment units (scenes, sonnets, poems)
// shared between a pair of editions, pre-loads their text lines, and returns them
// as ready-to-align tasks. This keeps all DB I/O sequential.
func loadAlignTasks(database *sql.DB, pair editionPair) []alignTask {
	var tasks []alignTask

	// === 1. Play scenes (act > 0) ===
	type sceneRef struct {
		WorkID     int64
		Act, Scene int
	}

	sceneRows, err := database.Query(`
		SELECT DISTINCT t1.work_id, t1.act, t1.scene
		FROM text_lines t1
		WHERE t1.edition_id = ? AND t1.act IS NOT NULL AND t1.act > 0 AND t1.scene IS NOT NULL
		  AND EXISTS (
			SELECT 1 FROM text_lines t2
			WHERE t2.work_id = t1.work_id AND t2.act = t1.act AND t2.scene = t1.scene
			  AND t2.edition_id = ?
		  )
		ORDER BY t1.work_id, t1.act, t1.scene`, pair.AID, pair.BID)
	if err == nil {
		var scenes []sceneRef
		for sceneRows.Next() {
			var s sceneRef
			sceneRows.Scan(&s.WorkID, &s.Act, &s.Scene)
			scenes = append(scenes, s)
		}
		sceneRows.Close()

		for _, scene := range scenes {
			aLines := loadSceneLines(database, scene.WorkID, pair.AID, scene.Act, scene.Scene)
			bLines := loadSceneLines(database, scene.WorkID, pair.BID, scene.Act, scene.Scene)
			if len(aLines) == 0 && len(bLines) == 0 {
				continue
			}
			tasks = append(tasks, alignTask{
				pair: pair, workID: scene.WorkID,
				act: scene.Act, scene: scene.Scene,
				aLines: aLines, bLines: bLines,
			})
		}
	}

	// === 2. Sonnets (scene = sonnet number, act is null/0) ===
	sonnetRows, err := database.Query(`
		SELECT DISTINCT t1.work_id, t1.scene
		FROM text_lines t1
		JOIN works w ON t1.work_id = w.id
		WHERE t1.edition_id = ? AND (t1.act IS NULL OR t1.act = 0) AND t1.scene IS NOT NULL AND t1.scene > 0
		  AND w.work_type = 'sonnet_sequence'
		  AND EXISTS (
			SELECT 1 FROM text_lines t2
			WHERE t2.work_id = t1.work_id AND t2.scene = t1.scene
			  AND (t2.act IS NULL OR t2.act = 0)
			  AND t2.edition_id = ?
		  )
		ORDER BY t1.work_id, t1.scene`, pair.AID, pair.BID)
	if err == nil {
		var sonnetScenes []sceneRef
		for sonnetRows.Next() {
			var s sceneRef
			sonnetRows.Scan(&s.WorkID, &s.Scene)
			sonnetScenes = append(sonnetScenes, s)
		}
		sonnetRows.Close()

		for _, sn := range sonnetScenes {
			aLines := loadSonnetLines(database, sn.WorkID, pair.AID, sn.Scene)
			bLines := loadSonnetLines(database, sn.WorkID, pair.BID, sn.Scene)
			if len(aLines) == 0 && len(bLines) == 0 {
				continue
			}
			tasks = append(tasks, alignTask{
				pair: pair, workID: sn.WorkID,
				act: 0, scene: sn.Scene,
				aLines: aLines, bLines: bLines,
			})
		}
	}

	// === 3. Poems (no act/scene structure, match by work_id) ===
	poemRows, err := database.Query(`
		SELECT DISTINCT t1.work_id
		FROM text_lines t1
		JOIN works w ON t1.work_id = w.id
		WHERE t1.edition_id = ? AND (t1.act IS NULL OR t1.act = 0) AND (t1.scene IS NULL OR t1.scene = 0)
		  AND w.work_type = 'poem'
		  AND EXISTS (
			SELECT 1 FROM text_lines t2
			WHERE t2.work_id = t1.work_id
			  AND (t2.act IS NULL OR t2.act = 0) AND (t2.scene IS NULL OR t2.scene = 0)
			  AND t2.edition_id = ?
		  )
		ORDER BY t1.work_id`, pair.AID, pair.BID)
	if err == nil {
		var poemWorks []int64
		for poemRows.Next() {
			var workID int64
			poemRows.Scan(&workID)
			poemWorks = append(poemWorks, workID)
		}
		poemRows.Close()

		for _, workID := range poemWorks {
			aLines := loadPoemLines(database, workID, pair.AID)
			bLines := loadPoemLines(database, workID, pair.BID)
			if len(aLines) == 0 && len(bLines) == 0 {
				continue
			}
			tasks = append(tasks, alignTask{
				pair: pair, workID: workID,
				act: 0, scene: 0,
				aLines: aLines, bLines: bLines,
			})
		}
	}

	return tasks
}

// loadSceneLines loads play text lines for a given scene into AlignableLine format.
func loadSceneLines(database *sql.DB, workID, editionID int64, act, scene int) []parser.AlignableLine {
	return queryAlignableLines(database,
		"work_id = ? AND edition_id = ? AND act = ? AND scene = ?",
		workID, editionID, act, scene)
}

// loadSonnetLines loads lines for a specific sonnet (scene = sonnet number).
func loadSonnetLines(database *sql.DB, workID, editionID int64, sonnetNum int) []parser.AlignableLine {
	return queryAlignableLines(database,
		"work_id = ? AND edition_id = ? AND scene = ? AND (act IS NULL OR act = 0)",
		workID, editionID, sonnetNum)
}

// loadPoemLines loads all lines for a poem (no act/scene structure).
func loadPoemLines(database *sql.DB, workID, editionID int64) []parser.AlignableLine {
	return queryAlignableLines(database,
		"work_id = ? AND edition_id = ? AND (act IS NULL OR act = 0) AND (scene IS NULL OR scene = 0)",
		workID, editionID)
}
