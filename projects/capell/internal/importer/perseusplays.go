// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// ImportPerseusPlays parses Perseus TEI XML play texts and inserts them into the
// database as a new edition ("Perseus Globe"). Each of the 37 play XMLs is matched
// to an existing work via the works.perseus_id column.
//
// The Perseus texts are the Clark & Wright Globe edition (1864), which is the
// standard scholarly reference for line numbers in Shakespeare studies.
func ImportPerseusPlays(database *sql.DB, sourcesDir string) error {
	stepBanner("Import Perseus Play Texts")
	start := time.Now()

	perseusDir := filepath.Join(sourcesDir, "perseus-plays")
	entries, err := os.ReadDir(perseusDir)
	if err != nil {
		return fmt.Errorf("reading Perseus directory: %w", err)
	}

	// Create Perseus source and Globe edition.
	sourceID, err := db.GetSourceID(database,
		"Perseus Digital Library", "perseus",
		"https://www.perseus.tufts.edu/hopper/",
		"CC BY-SA 3.0 US",
		"https://creativecommons.org/licenses/by-sa/3.0/us/",
		"Text provided by Perseus Digital Library, Tufts University. "+
			"Clark & Wright Globe edition. Funded by NSF/NEH Digital Libraries Initiative.",
		true,
		"Globe edition (1864) with Globe and First Folio line numbers in TEI XML.")
	if err != nil {
		return fmt.Errorf("creating Perseus source: %w", err)
	}

	editionID, err := db.GetEditionID(database,
		"Perseus Globe Edition", "perseus_globe",
		sourceID, 1864,
		"W. G. Clark, W. Aldis Wright",
		"Clark & Wright Globe edition via Perseus Digital Library TEI XML. "+
			"Globe line numbers marked every ~10 lines.")
	if err != nil {
		return fmt.Errorf("creating Perseus edition: %w", err)
	}

	// Build a map from perseus_id → work info.
	worksMap, err := buildWorksMapByColumn(database, "perseus_id")
	if err != nil {
		return fmt.Errorf("building works map: %w", err)
	}

	// Collect XML files, sorted for deterministic output.
	xmlFiles := collectXMLFiles(entries)

	// === Phase 1: Parse all XML files in parallel (CPU-bound) ===
	type parsedPlay struct {
		fname string
		lines []parser.PerseusLine
		err   error
	}

	results := parallelProcess(xmlFiles, func(fname string) parsedPlay {
		data, err := os.ReadFile(filepath.Join(perseusDir, fname))
		if err != nil {
			return parsedPlay{fname: fname, err: err}
		}
		lines, err := parser.ParsePerseusTEI(data)
		return parsedPlay{fname: fname, lines: lines, err: err}
	})

	parseResults := make(map[string]parsedPlay, len(results))
	for _, r := range results {
		parseResults[r.fname] = r
	}

	// === Phase 2: Insert each play sequentially (DB writes) ===
	totalLines, totalPlays := 0, 0

	for _, fname := range xmlFiles {
		r := parseResults[fname]
		if r.err != nil {
			fmt.Printf("  ERROR reading/parsing %s: %v\n", fname, r.err)
			continue
		}
		if len(r.lines) == 0 {
			continue // poetry files return 0 lines from the play parser
		}

		perseusID := fname[:len(fname)-4] // strip ".xml"
		work, ok := worksMap[perseusID]
		if !ok {
			fmt.Printf("  SKIP %s — no matching work for perseus_id=%s\n", fname, perseusID)
			continue
		}

		totalPlays++

		clearWorkEditionData(database, work.ID, editionID)

		tx, err := database.Begin()
		if err != nil {
			fmt.Printf("  ERROR starting tx for %s: %v\n", work.Title, err)
			continue
		}

		insertStmt, err := tx.Prepare(textLinesInsertSQL)
		if err != nil {
			tx.Rollback()
			fmt.Printf("  ERROR preparing insert for %s: %v\n", work.Title, err)
			continue
		}

		charCache := make(map[string]any)
		charNameCache := make(map[string]string)

		// Assign Globe line numbers to every line. Perseus only marks Globe
		// numbers on milestone lines (every 5th/10th), so we interpolate
		// for lines in between. Schmidt's lexicon citations use Globe
		// numbering, so preserving these is essential for citation matching.
		interpolateGlobeNumbers(r.lines)

		actScenes := make([][2]int, 0, len(r.lines))

		for _, line := range r.lines {
			charName := cachedExpandCharName(database, work.ID, line.Character, line.CharID, charNameCache)
			charID := cachedLookupCharacter(database, work.ID, charName, charCache)

			ct := contentType(line.IsStageDirection)
			lineNum := line.GlobeLine

			if line.IsStageDirection {
				charName = ""
			}

			insertStmt.Exec(
				work.ID, editionID,
				line.Act, line.Scene, lineNum,
				charID, nilIfEmpty(charName),
				line.Text, ct, countWords(line.Text))

			actScenes = append(actScenes, [2]int{line.Act, line.Scene})
		}
		insertStmt.Close()

		insertTextDivisions(tx, work.ID, editionID, actScenes)

		if err := tx.Commit(); err != nil {
			fmt.Printf("  ERROR committing %s: %v\n", work.Title, err)
			continue
		}

		totalLines += len(r.lines)
		speeches := 0
		for _, l := range r.lines {
			if !l.IsStageDirection {
				speeches++
			}
		}
		fmt.Printf("  [%2d] %-35s %5d lines (%4d speeches)\n",
			totalPlays, work.Title, len(r.lines), speeches)
	}

	// === Phase 3: Import poems and sonnets (different TEI structure) ===
	poemTypes := map[string]bool{
		"poem": true, "poem_collection": true, "sonnet_sequence": true,
	}
	totalPoems, totalPoemLines := 0, 0

	for _, fname := range xmlFiles {
		perseusID := fname[:len(fname)-4]
		work, ok := worksMap[perseusID]
		if !ok {
			continue
		}
		if !poemTypes[work.WorkType] {
			continue
		}

		data, err := os.ReadFile(filepath.Join(perseusDir, fname))
		if err != nil {
			fmt.Printf("  ERROR reading %s: %v\n", fname, err)
			continue
		}

		poem, err := parser.ParsePerseusPoem(data, work.WorkType)
		if err != nil || poem == nil || len(poem.Lines) == 0 {
			continue
		}

		clearWorkEditionData(database, work.ID, editionID)

		tx, err := database.Begin()
		if err != nil {
			fmt.Printf("  ERROR starting tx for %s: %v\n", work.Title, err)
			continue
		}

		insertStmt, err := tx.Prepare(textLinesInsertSQL)
		if err != nil {
			tx.Rollback()
			continue
		}

		actScenes := make([][2]int, 0, len(poem.Lines))
		for _, line := range poem.Lines {
			// Poems/sonnets: act=1 always, scene=section (sonnet number / 0 for narrative)
			act := 1
			scene := line.Section

			insertStmt.Exec(
				work.ID, editionID,
				act, scene, line.LineNumber,
				nil, nil, // no character
				line.Content, "verse", countWords(line.Content))

			actScenes = append(actScenes, [2]int{act, scene})
		}
		insertStmt.Close()

		insertTextDivisions(tx, work.ID, editionID, actScenes)

		if err := tx.Commit(); err != nil {
			fmt.Printf("  ERROR committing %s: %v\n", work.Title, err)
			continue
		}

		totalPoems++
		totalPoemLines += len(poem.Lines)
		fmt.Printf("  [%2d] %-35s %5d lines\n", totalPlays+totalPoems, work.Title, len(poem.Lines))
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "perseus_plays", "import_complete",
		fmt.Sprintf("%d plays, %d poems", totalPlays, totalPoems), totalLines+totalPoemLines, elapsed)

	fmt.Printf("\n  ✓ %d lines from %d plays + %d poems in %.1fs\n",
		totalLines+totalPoemLines, totalPlays, totalPoems, elapsed)
	return nil
}

// interpolateGlobeNumbers fills in Globe line numbers for all lines.
// Perseus only marks Globe numbers on milestone lines (every 5th/10th via
// <lb ed="G" n="10"/>). Lines between milestones get GlobeLine=0.
//
// Processing is done per-scene to avoid numbers bleeding across scene
// boundaries. Within each scene, milestones anchor the numbering and
// lines between milestones are interpolated linearly.
func interpolateGlobeNumbers(lines []parser.PerseusLine) {
	if len(lines) == 0 {
		return
	}

	// Split into per-scene slices and process each independently.
	start := 0
	for start < len(lines) {
		act, scene := lines[start].Act, lines[start].Scene
		end := start + 1
		for end < len(lines) && lines[end].Act == act && lines[end].Scene == scene {
			end++
		}
		interpolateSceneGlobeNumbers(lines[start:end])
		start = end
	}
}

// interpolateSceneGlobeNumbers assigns Globe line numbers to all lines in a
// single scene. It uses milestone lines (those with GlobeLine set by the parser)
// as anchors and linearly interpolates between them.
func interpolateSceneGlobeNumbers(lines []parser.PerseusLine) {
	// Collect indices of speech lines (not stage directions).
	speechIdx := make([]int, 0, len(lines))
	for i := range lines {
		if !lines[i].IsStageDirection {
			speechIdx = append(speechIdx, i)
		}
	}
	if len(speechIdx) == 0 {
		return
	}

	// Find milestones: speech lines that already have a Globe number from the parser.
	type milestone struct {
		speechPos int // position within speechIdx
		globeN    int
	}
	var milestones []milestone
	for si, idx := range speechIdx {
		if lines[idx].GlobeLine > 0 {
			milestones = append(milestones, milestone{si, lines[idx].GlobeLine})
		}
	}

	if len(milestones) == 0 {
		// No milestones in this scene — number sequentially from 1.
		for i, idx := range speechIdx {
			lines[idx].GlobeLine = i + 1
		}
	} else {
		// Interpolate before the first milestone by counting backwards.
		first := milestones[0]
		for i := 0; i < first.speechPos; i++ {
			lines[speechIdx[i]].GlobeLine = max(1, first.globeN-(first.speechPos-i))
		}

		// Interpolate between consecutive milestones using linear interpolation.
		for m := 0; m < len(milestones)-1; m++ {
			startPos := milestones[m].speechPos
			endPos := milestones[m+1].speechPos
			startN := milestones[m].globeN
			endN := milestones[m+1].globeN

			span := endPos - startPos
			for i := startPos; i < endPos; i++ {
				n := startN + (endN-startN)*(i-startPos)/span
				lines[speechIdx[i]].GlobeLine = n
			}
		}

		// Interpolate after the last milestone by counting forwards.
		last := milestones[len(milestones)-1]
		for i := last.speechPos; i < len(speechIdx); i++ {
			lines[speechIdx[i]].GlobeLine = last.globeN + (i - last.speechPos)
		}
	}

	// Stage directions get the same number as the preceding speech line.
	lastNum := 0
	for i := range lines {
		if lines[i].IsStageDirection {
			lines[i].GlobeLine = lastNum
		} else if lines[i].GlobeLine > 0 {
			lastNum = lines[i].GlobeLine
		}
	}
}

