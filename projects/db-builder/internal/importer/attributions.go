package importer

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

// PopulateAttributions creates attribution records for all sources.
// This step runs after all source data is imported so all sources exist.
// Attribution records document the display rules for each source,
// whether attribution is legally required or not.
func PopulateAttributions(database *sql.DB) error {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("STEP 5: Populate Attribution Rules")
	fmt.Println("=" + strings.Repeat("=", 59))

	start := time.Now()

	type attrDef struct {
		SourceCode            string
		Required              bool
		AttributionText       string
		AttributionHTML       string
		DisplayFormat         string
		DisplayContext        string
		DisplayPriority       int
		RequiresLinkBack      bool
		LinkBackURL           string
		RequiresLicenseNotice bool
		LicenseNoticeText     string
		RequiresAuthorCredit  bool
		AuthorCreditText      string
		ShareAlikeRequired    bool
		CommercialAllowed     bool
		ModificationAllowed   bool
		Notes                 string
	}

	attributions := []attrDef{
		{
			SourceCode:      "oss_moby",
			Required:        false,
			AttributionText: "Text from Open Source Shakespeare (opensourceshakespeare.org), based on the Moby project. Public domain.",
			AttributionHTML: `Text from <a href="https://www.opensourceshakespeare.org">Open Source Shakespeare</a>, based on the Moby project. Public domain.`,
			DisplayFormat:   "footer",
			DisplayContext:  "on_export",
			DisplayPriority: 10,
			RequiresLinkBack:      false,
			LinkBackURL:           "https://www.opensourceshakespeare.org",
			RequiresLicenseNotice: false,
			LicenseNoticeText:     "",
			RequiresAuthorCredit:  false,
			AuthorCreditText:      "",
			ShareAlikeRequired:    false,
			CommercialAllowed:     true,
			ModificationAllowed:   true,
			Notes:                 "Public domain. Attribution is voluntary but encouraged as good practice.",
		},
		{
			SourceCode:      "perseus_schmidt",
			Required:        true,
			AttributionText: "Alexander Schmidt, Shakespeare Lexicon and Quotation Dictionary. Provided by the Perseus Digital Library, Tufts University. Licensed under CC BY-SA 3.0.",
			AttributionHTML: `Alexander Schmidt, <em>Shakespeare Lexicon and Quotation Dictionary</em>. Provided by the <a href="http://www.perseus.tufts.edu">Perseus Digital Library</a>, Tufts University. Licensed under <a href="https://creativecommons.org/licenses/by-sa/3.0/">CC BY-SA 3.0</a>.`,
			DisplayFormat:   "footer",
			DisplayContext:  "always",
			DisplayPriority: 100,
			RequiresLinkBack:      true,
			LinkBackURL:           "http://www.perseus.tufts.edu",
			RequiresLicenseNotice: true,
			LicenseNoticeText:     "This work is licensed under the Creative Commons Attribution-ShareAlike 3.0 Unported License. To view a copy of this license, visit https://creativecommons.org/licenses/by-sa/3.0/",
			RequiresAuthorCredit:  true,
			AuthorCreditText:      "Alexander Schmidt",
			ShareAlikeRequired:    true,
			CommercialAllowed:     true,
			ModificationAllowed:   true,
			Notes:                 "CC BY-SA 3.0 REQUIRES: attribution, share-alike. Link back to Perseus. Author credit to Alexander Schmidt. License notice when distributing.",
		},
		{
			SourceCode:      "standard_ebooks",
			Required:        false,
			AttributionText: "Text from Standard Ebooks (standardebooks.org). Released to the public domain under CC0 1.0 Universal.",
			AttributionHTML: `Text from <a href="https://standardebooks.org">Standard Ebooks</a>. Released to the public domain under <a href="https://creativecommons.org/publicdomain/zero/1.0/">CC0 1.0 Universal</a>.`,
			DisplayFormat:   "footer",
			DisplayContext:  "on_export",
			DisplayPriority: 10,
			RequiresLinkBack:      false,
			LinkBackURL:           "https://standardebooks.org",
			RequiresLicenseNotice: false,
			LicenseNoticeText:     "",
			RequiresAuthorCredit:  false,
			AuthorCreditText:      "Standard Ebooks editorial team",
			ShareAlikeRequired:    false,
			CommercialAllowed:     true,
			ModificationAllowed:   true,
			Notes:                 "CC0 1.0 Universal — no rights reserved. Attribution is voluntary but encouraged as good practice. Standard Ebooks appreciates credit.",
		},
	}

	inserted := 0
	for _, attr := range attributions {
		// Look up source ID
		var sourceID int64
		err := database.QueryRow("SELECT id FROM sources WHERE short_code = ?", attr.SourceCode).Scan(&sourceID)
		if err != nil {
			fmt.Printf("  WARNING: Source %q not found, skipping attribution\n", attr.SourceCode)
			continue
		}

		boolToInt := func(b bool) int {
			if b {
				return 1
			}
			return 0
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
