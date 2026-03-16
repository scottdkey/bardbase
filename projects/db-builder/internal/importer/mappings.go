// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/parser"
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

// BuildLineMappings creates cross-edition line alignments for all edition pairs.
//
// For each pair of editions that share works, it aligns lines using text similarity
// (Needleman-Wunsch algorithm with Jaccard word similarity scoring).
//
// Handles three structure types:
//   - Plays: aligns scenes (act + scene) across editions
//   - Sonnets: aligns individual sonnets (scene = sonnet number) across editions
//   - Poems: aligns entire poems (by work_id) across editions
func BuildLineMappings(database *sql.DB) error {
	stepBanner("STEP 7: Build Cross-Edition Line Mappings")

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

	worksProcessed := make(map[int64]bool)
	var totalStats mappingStats

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

	for _, pair := range pairs {
		pairStats := alignEditionPair(database, pair, insertStmt, worksProcessed)
		totalStats.add(pairStats)
		fmt.Printf("    %s ↔ %s: %d pairs (aligned=%d)\n",
			pair.ACode, pair.BCode, pairStats.total, pairStats.aligned)
	}

	insertStmt.Close()
	tx.Commit()

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "mappings", "build_complete",
		fmt.Sprintf("%d pairs across %d works: aligned=%d modified=%d only_a=%d only_b=%d",
			totalStats.total, len(worksProcessed), totalStats.aligned, totalStats.modified,
			totalStats.onlyA, totalStats.onlyB),
		totalStats.total, elapsed)

	fmt.Printf("  ✓ %d alignment pairs across %d works in %.1fs\n",
		totalStats.total, len(worksProcessed), elapsed)
	fmt.Printf("    aligned: %d, modified: %d, only_a: %d, only_b: %d\n",
		totalStats.aligned, totalStats.modified, totalStats.onlyA, totalStats.onlyB)
	return nil
}

// alignEditionPair aligns all shared sections (scenes, sonnets, poems) between two editions.
func alignEditionPair(database *sql.DB, pair editionPair, stmt *sql.Stmt, worksProcessed map[int64]bool) mappingStats {
	var stats mappingStats

	insert := func(workID int64, act, scene, order int, p parser.AlignedPair) {
		var lineAID, lineBID interface{}
		if p.LineA != nil {
			lineAID = p.LineA.ID
		}
		if p.LineB != nil {
			lineBID = p.LineB.ID
		}
		stmt.Exec(workID, act, scene, order,
			pair.AID, pair.BID,
			lineAID, lineBID,
			p.MatchType, p.Similarity)
		stats.record(p.MatchType)
	}

	// === 1. Align play scenes (act > 0) ===
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
	if err != nil {
		return stats
	}

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

		pairs := parser.AlignSequences(aLines, bLines)
		for i, p := range pairs {
			insert(scene.WorkID, scene.Act, scene.Scene, i+1, p)
		}
		worksProcessed[scene.WorkID] = true
	}

	// === 2. Align sonnets (scene = sonnet number, act is null/0) ===
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

			pairs := parser.AlignSequences(aLines, bLines)
			for i, p := range pairs {
				insert(sn.WorkID, 0, sn.Scene, i+1, p)
			}
			worksProcessed[sn.WorkID] = true
		}
	}

	// === 3. Align poems (no act/scene structure, match by work_id) ===
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

			pairs := parser.AlignSequences(aLines, bLines)
			for i, p := range pairs {
				insert(workID, 0, 0, i+1, p)
			}
			worksProcessed[workID] = true
		}
	}

	return stats
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
