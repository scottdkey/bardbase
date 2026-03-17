// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
)


func TestPopulateAttributions_CreatesAll(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	err := PopulateAttributions(database)
	if err != nil {
		t.Fatalf("PopulateAttributions failed: %v", err)
	}

	count, err := db.TableCount(database, "attributions")
	if err != nil {
		t.Fatalf("TableCount failed: %v", err)
	}
	if count != 6 {
		t.Errorf("expected 6 attributions (oss_moby, perseus_schmidt, standard_ebooks, perseus, eebo_tcp, folger), got %d", count)
	}
}

func TestPopulateAttributions_PerseusIsRequired(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	PopulateAttributions(database)

	var required bool
	err := database.QueryRow(`
		SELECT a.required FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'perseus_schmidt'`).Scan(&required)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if !required {
		t.Error("Perseus attribution should be required (CC BY-SA 3.0)")
	}
}

func TestPopulateAttributions_OSSIsVoluntary(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	PopulateAttributions(database)

	var required bool
	err := database.QueryRow(`
		SELECT a.required FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'oss_moby'`).Scan(&required)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if required {
		t.Error("OSS attribution should be voluntary (Public Domain)")
	}
}

func TestPopulateAttributions_SEIsVoluntary(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	PopulateAttributions(database)

	var required bool
	err := database.QueryRow(`
		SELECT a.required FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'standard_ebooks'`).Scan(&required)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if required {
		t.Error("Standard Ebooks attribution should be voluntary (CC0)")
	}
}

func TestPopulateAttributions_PerseusDisplayRules(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	PopulateAttributions(database)

	var (
		displayFormat        string
		displayContext       string
		displayPriority      int
		requiresLinkBack     bool
		requiresLicense      bool
		requiresAuthorCredit bool
		shareAlikeRequired   bool
		commercialAllowed    bool
		modificationAllowed  bool
	)
	err := database.QueryRow(`
		SELECT a.display_format, a.display_context, a.display_priority,
			a.requires_link_back, a.requires_license_notice,
			a.requires_author_credit, a.share_alike_required,
			a.commercial_use_allowed, a.modification_allowed
		FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'perseus_schmidt'`).Scan(
		&displayFormat, &displayContext, &displayPriority,
		&requiresLinkBack, &requiresLicense,
		&requiresAuthorCredit, &shareAlikeRequired,
		&commercialAllowed, &modificationAllowed)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if displayFormat != "footer" {
		t.Errorf("expected display_format 'footer', got %q", displayFormat)
	}
	if displayContext != "always" {
		t.Errorf("expected display_context 'always', got %q", displayContext)
	}
	if displayPriority != 100 {
		t.Errorf("expected display_priority 100, got %d", displayPriority)
	}
	if !requiresLinkBack {
		t.Error("Perseus should require link back")
	}
	if !requiresLicense {
		t.Error("Perseus should require license notice")
	}
	if !requiresAuthorCredit {
		t.Error("Perseus should require author credit")
	}
	if !shareAlikeRequired {
		t.Error("Perseus should require share-alike")
	}
	if !commercialAllowed {
		t.Error("Perseus should allow commercial use")
	}
	if !modificationAllowed {
		t.Error("Perseus should allow modification")
	}
}

func TestPopulateAttributions_OSSDisplayRules(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	PopulateAttributions(database)

	var (
		displayContext       string
		displayPriority      int
		requiresLinkBack     bool
		requiresLicense      bool
		requiresAuthorCredit bool
		shareAlikeRequired   bool
	)
	err := database.QueryRow(`
		SELECT a.display_context, a.display_priority,
			a.requires_link_back, a.requires_license_notice,
			a.requires_author_credit, a.share_alike_required
		FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'oss_moby'`).Scan(
		&displayContext, &displayPriority,
		&requiresLinkBack, &requiresLicense,
		&requiresAuthorCredit, &shareAlikeRequired)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if displayContext != "on_export" {
		t.Errorf("expected display_context 'on_export', got %q", displayContext)
	}
	if displayPriority != 10 {
		t.Errorf("expected display_priority 10, got %d", displayPriority)
	}
	if requiresLinkBack {
		t.Error("OSS should not require link back")
	}
	if requiresLicense {
		t.Error("OSS should not require license notice")
	}
	if requiresAuthorCredit {
		t.Error("OSS should not require author credit")
	}
	if shareAlikeRequired {
		t.Error("OSS should not require share-alike")
	}
}

func TestPopulateAttributions_HasHTMLVersions(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	PopulateAttributions(database)

	// All 3 should have non-empty attribution_html
	rows, err := database.Query(`
		SELECT s.short_code, a.attribution_html
		FROM attributions a
		JOIN sources s ON a.source_id = s.id`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var code string
		var html sql.NullString
		rows.Scan(&code, &html)
		if !html.Valid || html.String == "" {
			t.Errorf("%s should have attribution_html set", code)
		}
		count++
	}
	if count != 6 {
		t.Errorf("expected 6 rows, got %d", count)
	}
}

func TestPopulateAttributions_Idempotent(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	// Run twice — should not duplicate records (INSERT OR REPLACE)
	PopulateAttributions(database)
	PopulateAttributions(database)

	count, _ := db.TableCount(database, "attributions")
	if count != 6 {
		t.Errorf("expected 6 attributions after double run, got %d", count)
	}
}

func TestPopulateAttributions_MissingSourceSkipped(t *testing.T) {
	// Create DB with only 1 of the 3 sources
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, _ := db.Open(dbPath)
	defer database.Close()
	db.CreateSchema(database)

	db.GetSourceID(database, "Perseus Schmidt Lexicon", "perseus_schmidt", "", "CC BY-SA 3.0", "", "", true, "")

	err := PopulateAttributions(database)
	if err != nil {
		t.Fatalf("PopulateAttributions should not fail with missing sources: %v", err)
	}

	count, _ := db.TableCount(database, "attributions")
	if count != 1 {
		t.Errorf("expected 1 attribution (only perseus), got %d", count)
	}
}

func TestPopulateAttributions_PerseusLinkBackURL(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	PopulateAttributions(database)

	var linkBackURL sql.NullString
	database.QueryRow(`
		SELECT a.link_back_url FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'perseus_schmidt'`).Scan(&linkBackURL)

	if !linkBackURL.Valid || linkBackURL.String == "" {
		t.Error("Perseus should have a link_back_url set")
	}
	if linkBackURL.Valid && linkBackURL.String != "http://www.perseus.tufts.edu" {
		t.Errorf("expected Perseus link_back_url 'http://www.perseus.tufts.edu', got %q", linkBackURL.String)
	}
}

func TestPopulateAttributions_PerseusAuthorCredit(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	PopulateAttributions(database)

	var authorCredit sql.NullString
	database.QueryRow(`
		SELECT a.author_credit_text FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'perseus_schmidt'`).Scan(&authorCredit)

	if !authorCredit.Valid || authorCredit.String != "Alexander Schmidt" {
		t.Errorf("expected author_credit_text 'Alexander Schmidt', got %v", authorCredit)
	}
}

// ---- New source attribution tests ----

func TestPopulateAttributions_FolgerIsRequiredNonCommercial(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()
	PopulateAttributions(database)

	var required, commercial bool
	err := database.QueryRow(`
		SELECT a.required, a.commercial_use_allowed
		FROM attributions a JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'folger'`).Scan(&required, &commercial)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if !required {
		t.Error("Folger attribution should be required (CC BY-NC 3.0)")
	}
	if commercial {
		t.Error("Folger should NOT allow commercial use (NC license)")
	}
}

func TestPopulateAttributions_FolgerDisplayRules(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()
	PopulateAttributions(database)

	var format, context string
	var priority int
	var requiresLink, requiresAuthor bool
	err := database.QueryRow(`
		SELECT a.display_format, a.display_context, a.display_priority,
		       a.requires_link_back, a.requires_author_credit
		FROM attributions a JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'folger'`).Scan(
		&format, &context, &priority, &requiresLink, &requiresAuthor)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if format != "inline" {
		t.Errorf("Folger display_format should be 'inline', got %q", format)
	}
	if context != "always" {
		t.Errorf("Folger display_context should be 'always', got %q", context)
	}
	if priority != 110 {
		t.Errorf("Folger display_priority should be 110, got %d", priority)
	}
	if !requiresLink {
		t.Error("Folger should require link back to shakespeare.folger.edu")
	}
	if !requiresAuthor {
		t.Error("Folger should require author credit (Mowat & Werstine)")
	}
}

func TestPopulateAttributions_EEBOTCPIsVoluntaryCC0(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()
	PopulateAttributions(database)

	var required, commercial bool
	var context string
	err := database.QueryRow(`
		SELECT a.required, a.commercial_use_allowed, a.display_context
		FROM attributions a JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'eebo_tcp'`).Scan(&required, &commercial, &context)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if required {
		t.Error("EEBO-TCP attribution should be voluntary (CC0)")
	}
	if !commercial {
		t.Error("EEBO-TCP (CC0) should allow commercial use")
	}
	if context != "on_export" {
		t.Errorf("EEBO-TCP display_context should be 'on_export', got %q", context)
	}
}

func TestPopulateAttributions_PerseusGlobeIsRequiredShareAlike(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()
	PopulateAttributions(database)

	var required, shareAlike, commercial bool
	var authorCredit string
	err := database.QueryRow(`
		SELECT a.required, a.share_alike_required, a.commercial_use_allowed,
		       a.author_credit_text
		FROM attributions a JOIN sources s ON a.source_id = s.id
		WHERE s.short_code = 'perseus'`).Scan(
		&required, &shareAlike, &commercial, &authorCredit)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if !required {
		t.Error("Perseus Globe (Clark & Wright) attribution should be required (CC BY-SA 3.0)")
	}
	if !shareAlike {
		t.Error("Perseus Globe (CC BY-SA) should require share-alike")
	}
	if !commercial {
		t.Error("Perseus Globe (CC BY-SA) should allow commercial use")
	}
	if authorCredit != "W. G. Clark and W. Aldis Wright" {
		t.Errorf("expected author 'W. G. Clark and W. Aldis Wright', got %q", authorCredit)
	}
}

// TestPopulateAttributions_RequiredSources verifies exactly which sources
// require mandatory attribution (for the frontend's always-visible credits).
func TestPopulateAttributions_RequiredSources(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()
	PopulateAttributions(database)

	rows, err := database.Query(`
		SELECT s.short_code FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE a.required = 1
		ORDER BY s.short_code`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	var required []string
	for rows.Next() {
		var code string
		rows.Scan(&code)
		required = append(required, code)
	}

	// Exactly 3 sources require mandatory attribution
	expected := map[string]bool{"folger": true, "perseus": true, "perseus_schmidt": true}
	if len(required) != 3 {
		t.Errorf("expected 3 required attributions, got %d: %v", len(required), required)
	}
	for _, code := range required {
		if !expected[code] {
			t.Errorf("unexpected required attribution: %s", code)
		}
	}
}

// TestPopulateAttributions_DisplayContextFiltering verifies the display_context
// split: some are 'always' (shown on every page), some 'on_export' (exports only).
func TestPopulateAttributions_DisplayContextFiltering(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()
	PopulateAttributions(database)

	var alwaysCount, exportCount int
	database.QueryRow(`SELECT COUNT(*) FROM attributions WHERE display_context = 'always'`).Scan(&alwaysCount)
	database.QueryRow(`SELECT COUNT(*) FROM attributions WHERE display_context = 'on_export'`).Scan(&exportCount)

	// folger, perseus_schmidt, perseus → 'always'
	if alwaysCount != 3 {
		t.Errorf("expected 3 'always' attributions (folger, perseus_schmidt, perseus), got %d", alwaysCount)
	}
	// oss_moby, standard_ebooks, eebo_tcp → 'on_export'
	if exportCount != 3 {
		t.Errorf("expected 3 'on_export' attributions (oss_moby, standard_ebooks, eebo_tcp), got %d", exportCount)
	}
}

// TestPopulateAttributions_CommercialUseSafeFilter verifies the frontend can
// identify which sources are safe for commercial features.
func TestPopulateAttributions_CommercialUseSafeFilter(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()
	PopulateAttributions(database)

	// Only folger is non-commercial
	var nonCommercialCodes []string
	rows, err := database.Query(`
		SELECT s.short_code FROM attributions a
		JOIN sources s ON a.source_id = s.id
		WHERE a.commercial_use_allowed = 0`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		rows.Scan(&code)
		nonCommercialCodes = append(nonCommercialCodes, code)
	}

	if len(nonCommercialCodes) != 1 || nonCommercialCodes[0] != "folger" {
		t.Errorf("expected only 'folger' to be non-commercial, got: %v", nonCommercialCodes)
	}
}
