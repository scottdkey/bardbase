// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// perseusPoetryFiles are Perseus IDs for non-dramatic works (poems, sonnets).
// These use div1 type="section" instead of type="act" and are expected to
// return 0 lines from the play parser.
var perseusPoetryFiles = map[string]bool{
	"1999.03.0061.xml": true, // Venus and Adonis
	"1999.03.0062.xml": true, // The Rape of Lucrece
	"1999.03.0063.xml": true, // The Passionate Pilgrim
	"1999.03.0064.xml": true, // Sonnets
	"1999.03.0065.xml": true, // A Lover's Complaint
	"1999.03.0066.xml": true, // The Phoenix and the Turtle
}

// TestParsePerseusTEI_AllFiles parses every Perseus XML file in sources/perseus-plays/
// to verify the parser handles all 37 plays without errors. The 6 poetry files are
// expected to return 0 lines (they use a different TEI structure).
func TestParsePerseusTEI_AllFiles(t *testing.T) {
	perseusDir := filepath.Join("..", "..", "..", "sources", "perseus-plays")
	entries, err := os.ReadDir(perseusDir)
	if err != nil {
		t.Skipf("Perseus directory not found: %v", err)
	}

	var files []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".xml" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		t.Skip("No Perseus XML files found")
	}

	totalLines, totalSpeeches, totalStage := 0, 0, 0
	playCount := 0

	for _, fname := range files {
		data, err := os.ReadFile(filepath.Join(perseusDir, fname))
		if err != nil {
			t.Errorf("failed to read %s: %v", fname, err)
			continue
		}

		lines, err := ParsePerseusTEI(data)
		if err != nil {
			t.Errorf("failed to parse %s: %v", fname, err)
			continue
		}

		// Poetry files are expected to return 0 lines
		if perseusPoetryFiles[fname] {
			if len(lines) != 0 {
				t.Errorf("%s: expected 0 lines (poetry), got %d", fname, len(lines))
			}
			t.Logf("%-22s    (poetry — skipped)", fname)
			continue
		}

		if len(lines) == 0 {
			t.Errorf("%s: parsed 0 lines (expected play content)", fname)
			continue
		}

		speeches, stage := 0, 0
		for _, l := range lines {
			if l.IsStageDirection {
				stage++
			} else {
				speeches++
			}
		}

		playCount++
		t.Logf("%-22s %5d lines (%4d speeches, %3d stage dirs)", fname, len(lines), speeches, stage)
		totalLines += len(lines)
		totalSpeeches += speeches
		totalStage += stage
	}

	t.Logf("TOTAL: %d plays parsed, %d lines (%d speeches, %d stage dirs)",
		playCount, totalLines, totalSpeeches, totalStage)

	// Sanity checks
	if len(files) != 43 {
		t.Errorf("expected 43 Perseus XMLs, found %d", len(files))
	}
	if playCount != 37 {
		t.Errorf("expected 37 plays (43 - 6 poetry), got %d", playCount)
	}
	if totalLines < 100000 {
		t.Errorf("expected at least 100,000 total lines across all plays, got %d", totalLines)
	}
}
