package importer

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/parser"
)

// BuildLineMappings creates cross-edition line alignments for side-by-side comparison.
//
// Handles three structure types:
//   - Plays: aligns scenes (act + scene) across editions
//   - Sonnets: aligns individual sonnets (scene = sonnet number) across editions
//   - Poems: aligns entire poems (by work_id) across editions
//
// For each shared section, it aligns lines using text similarity
// (Needleman-Wunsch algorithm with Jaccard word similarity scoring).
func BuildLineMappings(database *sql.DB) error {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("STEP 7: Build Cross-Edition Line Mappings")
	fmt.Println("=" + strings.Repeat("=", 59))

	start := time.Now()

	// Clear existing mappings
	database.Exec("DELETE FROM line_mappings")

	// Get edition IDs
	var ossEditionID, seEditionID int64
	database.QueryRow("SELECT id FROM editions WHERE short_code = 'oss_globe'").Scan(&ossEditionID)
	database.QueryRow("SELECT id FROM editions WHERE short_code = 'se_modern'").Scan(&seEditionID)

	if ossEditionID == 0 || seEditionID == 0 {
		fmt.Println("  Need both oss_globe and se_modern editions. Skipping.")
		return nil
	}

	totalPairs := 0
	alignedCount := 0
	modifiedCount := 0
	onlyACount := 0
	onlyBCount := 0
	worksProcessed := make(map[int64]bool)

	// Prepare insert statement
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

	insertPair := func(workID int64, act, scene, order int, pair parser.AlignedPair) {
		var lineAID, lineBID interface{}
		if pair.LineA != nil {
			lineAID = pair.LineA.ID
		}
		if pair.LineB != nil {
			lineBID = pair.LineB.ID
		}

		insertStmt.Exec(
			workID, act, scene, order,
			ossEditionID, seEditionID,
			lineAID, lineBID,
			pair.MatchType, pair.Similarity)

		totalPairs++
		switch pair.MatchType {
		case "aligned":
			alignedCount++
		case "modified":
			modifiedCount++
		case "only_a":
			onlyACount++
		case "only_b":
			onlyBCount++
		}
	}

	// === 1. Align play scenes (act > 0) ===
	sceneRows, err := database.Query(`
		SELECT DISTINCT t1.work_id, t1.act, t1.scene
		FROM text_lines t1
		WHERE t1.edition_id = ? AND t1.act IS NOT NULL AND t1.act > 0 AND t1.scene IS NOT NULL
		  AND EXISTS (
			SELECT 1 FROM text_lines t2
			WHERE t2.work_id = t1.work_id AND t2.act = t1.act AND t2.scene = t1.scene
			  AND t2.edition_id = ?
		  )
		ORDER BY t1.work_id, t1.act, t1.scene`, ossEditionID, seEditionID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("querying shared scenes: %w", err)
	}

	type sceneRef struct {
		WorkID int64
		Act    int
		Scene  int
	}
	var scenes []sceneRef
	for sceneRows.Next() {
		var s sceneRef
		sceneRows.Scan(&s.WorkID, &s.Act, &s.Scene)
		scenes = append(scenes, s)
	}
	sceneRows.Close()

	fmt.Printf("  Play scenes to align: %d\n", len(scenes))

	for _, scene := range scenes {
		ossLines := loadSceneLines(database, scene.WorkID, ossEditionID, scene.Act, scene.Scene)
		seLines := loadSceneLines(database, scene.WorkID, seEditionID, scene.Act, scene.Scene)

		if len(ossLines) == 0 && len(seLines) == 0 {
			continue
		}

		pairs := parser.AlignSequences(ossLines, seLines)
		for i, pair := range pairs {
			insertPair(scene.WorkID, scene.Act, scene.Scene, i+1, pair)
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
		ORDER BY t1.work_id, t1.scene`, ossEditionID, seEditionID)
	if err == nil {
		var sonnetScenes []sceneRef
		for sonnetRows.Next() {
			var s sceneRef
			sonnetRows.Scan(&s.WorkID, &s.Scene)
			sonnetScenes = append(sonnetScenes, s)
		}
		sonnetRows.Close()

		if len(sonnetScenes) > 0 {
			fmt.Printf("  Sonnets to align: %d\n", len(sonnetScenes))
			for _, sn := range sonnetScenes {
				ossLines := loadSonnetLines(database, sn.WorkID, ossEditionID, sn.Scene)
				seLines := loadSonnetLines(database, sn.WorkID, seEditionID, sn.Scene)

				if len(ossLines) == 0 && len(seLines) == 0 {
					continue
				}

				pairs := parser.AlignSequences(ossLines, seLines)
				for i, pair := range pairs {
					insertPair(sn.WorkID, 0, sn.Scene, i+1, pair)
				}
				worksProcessed[sn.WorkID] = true
			}
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
		ORDER BY t1.work_id`, ossEditionID, seEditionID)
	if err == nil {
		var poemWorks []int64
		for poemRows.Next() {
			var workID int64
			poemRows.Scan(&workID)
			poemWorks = append(poemWorks, workID)
		}
		poemRows.Close()

		if len(poemWorks) > 0 {
			fmt.Printf("  Poems to align: %d\n", len(poemWorks))
			for _, workID := range poemWorks {
				ossLines := loadPoemLines(database, workID, ossEditionID)
				seLines := loadPoemLines(database, workID, seEditionID)

				if len(ossLines) == 0 && len(seLines) == 0 {
					continue
				}

				pairs := parser.AlignSequences(ossLines, seLines)
				for i, pair := range pairs {
					insertPair(workID, 0, 0, i+1, pair)
				}
				worksProcessed[workID] = true
			}
		}
	}

	insertStmt.Close()
	tx.Commit()

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "mappings", "build_complete",
		fmt.Sprintf("%d pairs across %d works: aligned=%d modified=%d only_a=%d only_b=%d",
			totalPairs, len(worksProcessed), alignedCount, modifiedCount, onlyACount, onlyBCount),
		totalPairs, elapsed)

	fmt.Printf("  ✓ %d alignment pairs across %d works in %.1fs\n", totalPairs, len(worksProcessed), elapsed)
	fmt.Printf("    aligned: %d, modified: %d, only_oss: %d, only_se: %d\n",
		alignedCount, modifiedCount, onlyACount, onlyBCount)
	return nil
}

// loadSceneLines loads play text lines for a given scene into AlignableLine format.
func loadSceneLines(database *sql.DB, workID, editionID int64, act, scene int) []parser.AlignableLine {
	rows, err := database.Query(`
		SELECT id, content, COALESCE(line_number, 0)
		FROM text_lines
		WHERE work_id = ? AND edition_id = ? AND act = ? AND scene = ?
		ORDER BY line_number, id`, workID, editionID, act, scene)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var lines []parser.AlignableLine
	for rows.Next() {
		var l parser.AlignableLine
		rows.Scan(&l.ID, &l.Content, &l.LineNumber)
		lines = append(lines, l)
	}
	return lines
}

// loadSonnetLines loads lines for a specific sonnet (scene = sonnet number).
func loadSonnetLines(database *sql.DB, workID, editionID int64, sonnetNum int) []parser.AlignableLine {
	rows, err := database.Query(`
		SELECT id, content, COALESCE(line_number, 0)
		FROM text_lines
		WHERE work_id = ? AND edition_id = ? AND scene = ? AND (act IS NULL OR act = 0)
		ORDER BY line_number, id`, workID, editionID, sonnetNum)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var lines []parser.AlignableLine
	for rows.Next() {
		var l parser.AlignableLine
		rows.Scan(&l.ID, &l.Content, &l.LineNumber)
		lines = append(lines, l)
	}
	return lines
}

// loadPoemLines loads all lines for a poem (no act/scene structure).
func loadPoemLines(database *sql.DB, workID, editionID int64) []parser.AlignableLine {
	rows, err := database.Query(`
		SELECT id, content, COALESCE(line_number, 0)
		FROM text_lines
		WHERE work_id = ? AND edition_id = ? AND (act IS NULL OR act = 0) AND (scene IS NULL OR scene = 0)
		ORDER BY line_number, id`, workID, editionID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var lines []parser.AlignableLine
	for rows.Next() {
		var l parser.AlignableLine
		rows.Scan(&l.ID, &l.Content, &l.LineNumber)
		lines = append(lines, l)
	}
	return lines
}
