// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestParseFirstFolioTEI_AllPlays parses the EEBO-TCP First Folio XML (A11954)
// and verifies plays are extracted with non-empty titles.
func TestParseFirstFolioTEI_AllPlays(t *testing.T) {
	folioPath := filepath.Join("..", "..", "..", "sources", "eebo-tcp", "A11954.xml")
	data, err := os.ReadFile(folioPath)
	if err != nil {
		t.Skipf("First Folio XML not found: %v", err)
	}

	lines, err := ParseFirstFolioTEI(data)
	if err != nil {
		t.Fatalf("ParseFirstFolioTEI failed: %v", err)
	}

	if len(lines) == 0 {
		t.Fatal("ParseFirstFolioTEI returned 0 lines")
	}

	// Group lines by play title.
	plays := map[string]int{}
	for _, l := range lines {
		plays[l.PlayTitle]++
	}

	// No empty titles — the Troilus fix ensures every play div gets a title.
	if count, ok := plays[""]; ok {
		t.Errorf("found %d lines with empty play title", count)
	}

	titles := make([]string, 0, len(plays))
	for title := range plays {
		if title != "" {
			titles = append(titles, title)
		}
	}
	sort.Strings(titles)

	// All 36 First Folio plays should be present.
	if len(titles) != 36 {
		t.Errorf("expected 36 plays, got %d:", len(titles))
		for _, title := range titles {
			t.Logf("  %q: %d lines", title, plays[title])
		}
	}

	// Troilus and Cressida — its play div has no direct <head>, so the parser
	// must find the title from the first act div.
	troilusTitle := "the tragedie of troylus and cressida."
	if count, ok := plays[troilusTitle]; !ok {
		t.Errorf("Troilus and Cressida not found in parsed plays")
	} else if count < 100 {
		t.Errorf("Troilus and Cressida has only %d lines, expected at least 100", count)
	}

	// Every parsed play should have a reasonable number of lines.
	for _, title := range titles {
		if plays[title] < 50 {
			t.Errorf("play %q has only %d lines, expected at least 50", title, plays[title])
		}
	}
}
