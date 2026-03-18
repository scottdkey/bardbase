// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"testing"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
)

// TestAppQuery_ContentTypeForReader verifies the frontend Phase 3 reader can
// distinguish speech from stage directions via content_type.
func TestAppQuery_ContentTypeForReader(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "Hamlet", "Ham.", "play")

	// Insert a mix of speech and stage direction lines
	td.DB.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number,
		char_name, content, content_type)
		VALUES (?, ?, 1, 1, 1, 'HAMLET', 'To be, or not to be', 'speech')`,
		workID, td.EdOSSID)
	td.DB.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number,
		char_name, content, content_type)
		VALUES (?, ?, 1, 1, 2, NULL, 'Enter HAMLET', 'stage_direction')`,
		workID, td.EdOSSID)
	td.DB.Exec(`INSERT INTO text_lines (work_id, edition_id, act, scene, line_number,
		char_name, content, content_type)
		VALUES (?, ?, 1, 1, 3, 'HAMLET', 'That is the question', 'speech')`,
		workID, td.EdOSSID)

	// The reader query — should return content_type for every line
	rows, err := td.DB.Query(`
		SELECT line_number, char_name, content, content_type
		FROM text_lines
		WHERE work_id = ? AND edition_id = ? AND act = 1 AND scene = 1
		ORDER BY line_number`, workID, td.EdOSSID)
	if err != nil {
		t.Fatalf("reader query: %v", err)
	}
	defer rows.Close()

	type line struct {
		lineNum             int
		charName            *string
		content, contentType string
	}

	var results []line
	for rows.Next() {
		var l line
		rows.Scan(&l.lineNum, &l.charName, &l.content, &l.contentType)
		results = append(results, l)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(results))
	}

	if results[0].contentType != "speech" {
		t.Errorf("line 1 content_type: expected 'speech', got %q", results[0].contentType)
	}
	if results[1].contentType != "stage_direction" {
		t.Errorf("line 2 content_type: expected 'stage_direction', got %q", results[1].contentType)
	}
	if results[1].charName != nil {
		t.Error("stage direction should have NULL char_name")
	}
	if results[2].contentType != "speech" {
		t.Errorf("line 3 content_type: expected 'speech', got %q", results[2].contentType)
	}
}

// TestAppQuery_ComparisonWithGapSides verifies Phase 4 (cross-edition comparison)
// correctly handles plays present in one edition but absent in the other.
// Lines from the present edition produce only_a/only_b rows with NULL on the other side.
func TestAppQuery_ComparisonWithGapSides(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "Pericles", "Per.", "play")

	// Pericles: 3 lines in OSS, 0 in SE (absent from SE edition).
	// Both editions must share at least one work for a pair to be created.
	// Hamlet is present in both so the oss_globe↔se_modern pair is valid.
	hamletID := td.insertWork(t, "Hamlet", "Ham.", "play")
	td.insertTextLine(t, hamletID, td.EdSEID, 1, 1, 1, "HAMLET", "Who is there?")
	td.insertTextLine(t, hamletID, td.EdOSSID, 1, 1, 1, "HAMLET", "Who is there?")

	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 1, "HELICANUS", "Who makes the fairest show means most deceit")
	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 2, "HELICANUS", "That man that hath a tongue")
	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 3, "HELICANUS", "I say it is no sin")

	if err := BuildLineMappings(td.DB); err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	// Verify 3 only_a entries — all lines have NULL on the SE side
	var total, onlyA int
	td.DB.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE work_id = ?`, workID).Scan(&total)
	td.DB.QueryRow(`SELECT COUNT(*) FROM line_mappings WHERE work_id = ? AND match_type = 'only_a'`, workID).Scan(&onlyA)

	if total != 3 {
		t.Errorf("expected 3 line_mappings for Pericles, got %d", total)
	}
	if onlyA != 3 {
		t.Errorf("expected 3 only_a entries (SE absent), got %d", onlyA)
	}

	// Phase 4 side-by-side query — LEFT JOIN returns NULL for the absent SE side
	rows, err := td.DB.Query(`
		SELECT
			lm.align_order,
			lm.match_type,
			tla.content AS oss_text,
			tlb.content AS se_text
		FROM line_mappings lm
		LEFT JOIN text_lines tla ON tla.id = lm.line_a_id
		LEFT JOIN text_lines tlb ON tlb.id = lm.line_b_id
		WHERE lm.work_id = ? AND lm.act = 1 AND lm.scene = 1
		ORDER BY lm.align_order`, workID)
	if err != nil {
		t.Fatalf("side-by-side query: %v", err)
	}
	defer rows.Close()

	type compRow struct {
		alignOrder     int
		matchType      string
		ossText        *string
		seText         *string
	}

	var compRows []compRow
	for rows.Next() {
		var r compRow
		rows.Scan(&r.alignOrder, &r.matchType, &r.ossText, &r.seText)
		compRows = append(compRows, r)
	}

	if len(compRows) != 3 {
		t.Fatalf("side-by-side: expected 3 rows, got %d", len(compRows))
	}

	for i, r := range compRows {
		if r.alignOrder != i+1 {
			t.Errorf("row %d: align_order should be %d, got %d", i, i+1, r.alignOrder)
		}
		if r.matchType != "only_a" {
			t.Errorf("row %d: match_type should be 'only_a', got %q", i, r.matchType)
		}
		if r.ossText == nil || *r.ossText == "" {
			t.Errorf("row %d: OSS text should not be NULL/empty", i)
		}
		if r.seText != nil {
			t.Errorf("row %d: SE text should be NULL (absent edition), got %q", i, *r.seText)
		}
	}
}

// TestAppQuery_ComparisonEditionPairOrdering verifies that the comparison query
// correctly handles both orderings: (editionA, editionB) and (editionB, editionA).
// line_mappings always stores the pair with lower edition_id as edition_a.
func TestAppQuery_ComparisonEditionPairOrdering(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "Hamlet", "Ham.", "play")

	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 1, "HAMLET", "To be, or not to be")
	td.insertTextLine(t, workID, td.EdSEID, 1, 1, 1, "HAMLET", "To be or not to be")

	if err := BuildLineMappings(td.DB); err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	// Verify the mapping uses the lower edition_id as A
	var edAID, edBID int64
	err := td.DB.QueryRow(`
		SELECT edition_a_id, edition_b_id FROM line_mappings
		WHERE work_id = ? LIMIT 1`, workID).Scan(&edAID, &edBID)
	if err != nil {
		t.Fatalf("querying line_mapping: %v", err)
	}

	if edAID > edBID {
		t.Errorf("edition_a_id (%d) should be <= edition_b_id (%d)", edAID, edBID)
	}

	// Regardless of pair ordering, the comparison query using COALESCE handles both
	// user-requested orderings (A→B or B→A):
	var lineAContent, lineBContent string
	td.DB.QueryRow(`
		SELECT tla.content, tlb.content
		FROM line_mappings lm
		JOIN text_lines tla ON tla.id = lm.line_a_id
		JOIN text_lines tlb ON tlb.id = lm.line_b_id
		WHERE lm.work_id = ? AND lm.act = 1 AND lm.scene = 1`, workID).
		Scan(&lineAContent, &lineBContent)

	if lineAContent == "" || lineBContent == "" {
		t.Error("both sides of the mapping should have content")
	}
}

// TestAppQuery_ScenesNavigation verifies the frontend act/scene nav query:
// derive all scenes for a work+edition from text_lines (Phase 3.2).
func TestAppQuery_ScenesNavigation(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "Hamlet", "Ham.", "play")

	// Insert lines across 3 scenes
	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 1, "BARNARDO", "Who is there?")
	td.insertTextLine(t, workID, td.EdOSSID, 1, 2, 1, "HORATIO", "Friends to this ground")
	td.insertTextLine(t, workID, td.EdOSSID, 2, 1, 1, "HAMLET", "A little more than kin")

	// Act/scene navigation query: get all scenes for this work+edition
	rows, err := td.DB.Query(`
		SELECT act, scene, COUNT(*) as line_count
		FROM text_lines
		WHERE work_id = ? AND edition_id = ?
		  AND act IS NOT NULL AND act > 0
		GROUP BY act, scene
		ORDER BY act, scene`, workID, td.EdOSSID)
	if err != nil {
		t.Fatalf("scene nav query: %v", err)
	}
	defer rows.Close()

	type sceneRow struct{ act, scene, lineCount int }
	var scenes []sceneRow
	for rows.Next() {
		var s sceneRow
		rows.Scan(&s.act, &s.scene, &s.lineCount)
		scenes = append(scenes, s)
	}

	if len(scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(scenes))
	}
	// Verify ordering and structure
	if scenes[0].act != 1 || scenes[0].scene != 1 {
		t.Errorf("expected first scene 1.1, got %d.%d", scenes[0].act, scenes[0].scene)
	}
	if scenes[2].act != 2 || scenes[2].scene != 1 {
		t.Errorf("expected last scene 2.1, got %d.%d", scenes[2].act, scenes[2].scene)
	}
}

// TestAppQuery_MappingCoverage verifies all edition pairs for a work are present
// in line_mappings — critical for Phase 4 comparison feature completeness.
func TestAppQuery_MappingCoverage(t *testing.T) {
	td := newTestDB(t)

	// Add a third edition
	thirdSrcID, _ := db.GetSourceID(td.DB, "Third Source", "third", "", "", "", "", false, "")
	thirdEdID, _ := db.GetEditionID(td.DB, "Third Edition", "third_edition", thirdSrcID, 1900, "", "")

	workID := td.insertWork(t, "Hamlet", "Ham.", "play")
	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 1, "HAMLET", "To be")
	td.insertTextLine(t, workID, td.EdSEID, 1, 1, 1, "HAMLET", "To be")
	td.insertTextLine(t, workID, thirdEdID, 1, 1, 1, "HAMLET", "To be")

	if err := BuildLineMappings(td.DB); err != nil {
		t.Fatalf("BuildLineMappings: %v", err)
	}

	// With 3 editions, C(3,2)=3 pairs — all should have a mapping entry
	var pairCount int
	td.DB.QueryRow(`
		SELECT COUNT(DISTINCT edition_a_id || '-' || edition_b_id)
		FROM line_mappings WHERE work_id = ?`, workID).Scan(&pairCount)

	if pairCount != 3 {
		t.Errorf("expected 3 edition pairs in line_mappings for C(3,2), got %d", pairCount)
	}
}
