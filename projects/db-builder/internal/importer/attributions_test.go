// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)


func TestPopulateAttributions_CreatesAllThree(t *testing.T) {
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
	if count != 3 {
		t.Errorf("expected 3 attributions, got %d", count)
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
	if count != 3 {
		t.Errorf("expected 3 rows, got %d", count)
	}
}

func TestPopulateAttributions_Idempotent(t *testing.T) {
	database := newAttributionTestDB(t)
	defer database.Close()

	// Run twice — should not duplicate records (INSERT OR REPLACE)
	PopulateAttributions(database)
	PopulateAttributions(database)

	count, _ := db.TableCount(database, "attributions")
	if count != 3 {
		t.Errorf("expected 3 attributions after double run, got %d", count)
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
