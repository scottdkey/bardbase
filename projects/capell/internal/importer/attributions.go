// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/db"
)

// PopulateAttributions creates attribution records for all sources.
// This step runs after all source data is imported so all sources exist.
// Attribution records document the display rules for each source,
// whether attribution is legally required or not.
//
// Data is loaded from projects/data/attributions.json via the constants package.
func PopulateAttributions(database *sql.DB) error {
	stepBanner("STEP 7: Populate Attribution Rules")

	start := time.Now()

	// Ensure reference data is loaded
	constants.EnsureLoaded()

	inserted := 0
	for _, attr := range constants.Attributions {
		// Look up source ID
		var sourceID int64
		err := database.QueryRow("SELECT id FROM sources WHERE short_code = ?", attr.SourceCode).Scan(&sourceID)
		if err != nil {
			fmt.Printf("  WARNING: Source %q not found, skipping attribution\n", attr.SourceCode)
			continue
		}

		_, err = database.Exec(`
			INSERT OR REPLACE INTO attributions (
				source_id, required, attribution_text, attribution_html,
				display_format, display_context, display_priority,
				requires_link_back, link_back_url,
				requires_license_notice, license_notice_text,
				requires_author_credit, author_credit_text,
				share_alike_required, commercial_use_allowed, modification_allowed,
				notes
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sourceID,
			boolToInt(attr.Required),
			attr.AttributionText,
			attr.AttributionHTML,
			attr.DisplayFormat,
			attr.DisplayContext,
			attr.DisplayPriority,
			boolToInt(attr.RequiresLinkBack),
			nilIfEmpty(attr.LinkBackURL),
			boolToInt(attr.RequiresLicenseNotice),
			nilIfEmpty(attr.LicenseNoticeText),
			boolToInt(attr.RequiresAuthorCredit),
			nilIfEmpty(attr.AuthorCreditText),
			boolToInt(attr.ShareAlikeRequired),
			boolToInt(attr.CommercialAllowed),
			boolToInt(attr.ModificationAllowed),
			attr.Notes,
		)
		if err != nil {
			fmt.Printf("  ERROR inserting attribution for %s: %v\n", attr.SourceCode, err)
			continue
		}
		inserted++
		fmt.Printf("  %s: %s (required=%v, priority=%d)\n",
			attr.SourceCode, attr.DisplayFormat, attr.Required, attr.DisplayPriority)
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "attributions", "populate_complete",
		fmt.Sprintf("%d attribution records", inserted), inserted, elapsed)

	fmt.Printf("  ✓ %d attribution records in %.1fs\n", inserted, elapsed)
	return nil
}
