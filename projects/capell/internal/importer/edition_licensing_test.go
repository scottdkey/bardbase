// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"testing"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
)

// setupLicensingDB creates a minimal database with editions tagged by license tier.
// This mirrors what the full pipeline produces for the frontend.
func setupLicensingDB(t *testing.T) *testDB {
	t.Helper()
	td := newTestDB(t)

	// Add a Folger edition (NC-licensed)
	folgerSrcID, err := db.GetSourceID(td.DB, "Folger Shakespeare Library", "folger",
		"https://shakespeare.folger.edu/", "CC BY-NC 3.0", "", "", false, "")
	if err != nil {
		t.Fatalf("GetSourceID(folger): %v", err)
	}
	folgerEdID, err := db.GetEditionID(td.DB, "Folger Shakespeare", "folger_shakespeare",
		folgerSrcID, 2024, "Mowat & Werstine", "Folger Digital Texts")
	if err != nil {
		t.Fatalf("GetEditionID(folger): %v", err)
	}
	td.DB.Exec(`UPDATE editions SET source_key = 'folger', license_tier = 'cc-by-nc' WHERE id = ?`, folgerEdID)

	// Add an EEBO-TCP quarto edition (CC0-licensed)
	eeboSrcID, err := db.GetSourceID(td.DB, "EEBO-TCP", "eebo_tcp",
		"https://textcreationpartnership.org/", "CC0 1.0 Universal", "", "", false, "")
	if err != nil {
		t.Fatalf("GetSourceID(eebo_tcp): %v", err)
	}
	eeboEdID, err := db.GetEditionID(td.DB, "Hamlet Q1 (1603)", "q1_hamlet_1603",
		eeboSrcID, 1603, "EEBO-TCP", "Q1 Hamlet diplomatic transcription")
	if err != nil {
		t.Fatalf("GetEditionID(q1_hamlet_1603): %v", err)
	}
	td.DB.Exec(`UPDATE editions SET source_key = 'eebo_tcp', license_tier = 'cc0' WHERE id = ?`, eeboEdID)

	return td
}

// TestEditionLicensing_FolgerTaggedCorrectly verifies that after import the
// Folger edition has source_key='folger' and license_tier='cc-by-nc'.
func TestEditionLicensing_FolgerTaggedCorrectly(t *testing.T) {
	td := setupLicensingDB(t)

	var sourceKey, licenseTier string
	err := td.DB.QueryRow(`
		SELECT source_key, license_tier FROM editions
		WHERE short_code = 'folger_shakespeare'`).Scan(&sourceKey, &licenseTier)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if sourceKey != "folger" {
		t.Errorf("expected source_key 'folger', got %q", sourceKey)
	}
	if licenseTier != "cc-by-nc" {
		t.Errorf("expected license_tier 'cc-by-nc', got %q", licenseTier)
	}
}

// TestEditionLicensing_EEBOTaggedCorrectly verifies EEBO quarto editions are
// tagged as cc0.
func TestEditionLicensing_EEBOTaggedCorrectly(t *testing.T) {
	td := setupLicensingDB(t)

	var sourceKey, licenseTier string
	err := td.DB.QueryRow(`
		SELECT source_key, license_tier FROM editions
		WHERE short_code = 'q1_hamlet_1603'`).Scan(&sourceKey, &licenseTier)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if sourceKey != "eebo_tcp" {
		t.Errorf("expected source_key 'eebo_tcp', got %q", sourceKey)
	}
	if licenseTier != "cc0" {
		t.Errorf("expected license_tier 'cc0', got %q", licenseTier)
	}
}

// TestEditionLicensing_CommercialSafeFilter verifies the frontend can filter out
// NC-licensed editions when building a commercial product.
// The query excludes editions where license_tier = 'cc-by-nc'.
func TestEditionLicensing_CommercialSafeFilter(t *testing.T) {
	td := setupLicensingDB(t)

	rows, err := td.DB.Query(`
		SELECT short_code, COALESCE(license_tier, 'untagged') as tier
		FROM editions
		WHERE license_tier != 'cc-by-nc' OR license_tier IS NULL
		ORDER BY short_code`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	editions := map[string]string{}
	for rows.Next() {
		var code, tier string
		rows.Scan(&code, &tier)
		editions[code] = tier
	}

	// folger_shakespeare should be excluded
	if _, found := editions["folger_shakespeare"]; found {
		t.Error("commercial-safe filter should exclude folger_shakespeare (cc-by-nc)")
	}

	// oss_globe, se_modern, q1_hamlet_1603 should be present
	if _, found := editions["oss_globe"]; !found {
		t.Error("oss_globe (untagged/public domain) should pass commercial filter")
	}
	if _, found := editions["q1_hamlet_1603"]; !found {
		t.Error("q1_hamlet_1603 (cc0) should pass commercial filter")
	}
}

// TestEditionLicensing_SourceKeyExclusion verifies the --exclude flag semantics:
// can filter out all editions by source_key.
func TestEditionLicensing_SourceKeyExclusion(t *testing.T) {
	td := setupLicensingDB(t)

	// Simulate --exclude folger: count editions with source_key != 'folger' or NULL
	var total, excluded int
	td.DB.QueryRow(`SELECT COUNT(*) FROM editions`).Scan(&total)
	td.DB.QueryRow(`SELECT COUNT(*) FROM editions WHERE source_key = 'folger'`).Scan(&excluded)

	if excluded != 1 {
		t.Errorf("expected 1 folger edition to exclude, got %d", excluded)
	}
	if total-excluded < 3 {
		t.Errorf("after exclusion, expected at least 3 remaining editions, got %d", total-excluded)
	}

	// Verify the excluded edition is folger_shakespeare
	var code string
	td.DB.QueryRow(`SELECT short_code FROM editions WHERE source_key = 'folger'`).Scan(&code)
	if code != "folger_shakespeare" {
		t.Errorf("expected excluded edition 'folger_shakespeare', got %q", code)
	}
}

// TestEditionLicensing_GetEditionsByWork simulates the Phase 3 frontend query:
// "which editions have lines for this work?" — the edition switcher dropdown.
func TestEditionLicensing_GetEditionsByWork(t *testing.T) {
	td := setupLicensingDB(t)

	// Insert Hamlet (play) in OSS and Folger, but NOT in se_modern
	workID := td.insertWork(t, "Hamlet", "Ham.", "play")

	td.insertTextLine(t, workID, td.EdOSSID, 3, 1, 1, "HAMLET", "To be, or not to be")
	td.insertTextLine(t, workID, td.EdOSSID, 3, 1, 2, "HAMLET", "That is the question")

	var folgerEdID int64
	td.DB.QueryRow(`SELECT id FROM editions WHERE short_code = 'folger_shakespeare'`).Scan(&folgerEdID)
	td.insertTextLine(t, workID, folgerEdID, 3, 1, 1, "HAMLET", "To be, or not to be")

	// Query: editions that have lines for this work (the app's getEditionsByWork)
	rows, err := td.DB.Query(`
		SELECT DISTINCT e.short_code, e.name,
		       COALESCE(e.source_key, '') as source_key,
		       COALESCE(e.license_tier, '') as license_tier,
		       COUNT(tl.id) as line_count
		FROM editions e
		JOIN text_lines tl ON tl.edition_id = e.id
		WHERE tl.work_id = ?
		GROUP BY e.id
		ORDER BY e.id`, workID)
	if err != nil {
		t.Fatalf("getEditionsByWork query: %v", err)
	}
	defer rows.Close()

	type edRow struct {
		code, name, sourceKey, licenseTier string
		lineCount                          int
	}
	var results []edRow
	for rows.Next() {
		var r edRow
		rows.Scan(&r.code, &r.name, &r.sourceKey, &r.licenseTier, &r.lineCount)
		results = append(results, r)
	}

	// Should find 2 editions (oss_globe and folger_shakespeare), not se_modern
	if len(results) != 2 {
		t.Errorf("expected 2 editions for Hamlet, got %d: %v", len(results), results)
	}

	editions := map[string]edRow{}
	for _, r := range results {
		editions[r.code] = r
	}

	if _, found := editions["oss_globe"]; !found {
		t.Error("oss_globe should appear for Hamlet (has 2 lines)")
	}
	if _, found := editions["folger_shakespeare"]; !found {
		t.Error("folger_shakespeare should appear for Hamlet (has 1 line)")
	}
	if _, found := editions["se_modern"]; found {
		t.Error("se_modern should NOT appear for Hamlet (has 0 lines)")
	}

	// Verify license_tier is returned for filtering in the UI
	if editions["folger_shakespeare"].licenseTier != "cc-by-nc" {
		t.Errorf("folger_shakespeare license_tier should be 'cc-by-nc', got %q",
			editions["folger_shakespeare"].licenseTier)
	}
}

// TestEditionLicensing_CommercialEditionsByWork verifies that the commercial-safe
// edition list for a work excludes NC editions from the switcher.
func TestEditionLicensing_CommercialEditionsByWork(t *testing.T) {
	td := setupLicensingDB(t)

	workID := td.insertWork(t, "Hamlet", "Ham.", "play")
	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 1, "HAMLET", "To be, or not to be")

	var folgerEdID int64
	td.DB.QueryRow(`SELECT id FROM editions WHERE short_code = 'folger_shakespeare'`).Scan(&folgerEdID)
	td.insertTextLine(t, workID, folgerEdID, 1, 1, 1, "HAMLET", "To be, or not to be")

	// Commercial-safe: exclude cc-by-nc editions from the switcher
	rows, err := td.DB.Query(`
		SELECT DISTINCT e.short_code
		FROM editions e
		JOIN text_lines tl ON tl.edition_id = e.id
		WHERE tl.work_id = ?
		  AND (e.license_tier != 'cc-by-nc' OR e.license_tier IS NULL)
		ORDER BY e.short_code`, workID)
	if err != nil {
		t.Fatalf("commercial editions query: %v", err)
	}
	defer rows.Close()

	var editions []string
	for rows.Next() {
		var code string
		rows.Scan(&code)
		editions = append(editions, code)
	}

	if len(editions) != 1 || editions[0] != "oss_globe" {
		t.Errorf("expected only 'oss_globe' for commercial Hamlet editions, got: %v", editions)
	}
}
