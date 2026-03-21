// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/fetch"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

// sePlayData holds the parsed result of downloading + parsing a single SE play.
type sePlayData struct {
	repoName string
	ossID    string
	lines    []parser.PlayLine
	err      error
}

// ImportSEPlays imports Standard Ebooks play text from GitHub.
//
// When forceDownload is true, play downloads and parsing run in parallel
// (network I/O + CPU), then DB inserts happen sequentially.
func ImportSEPlays(database *sql.DB, cacheDir string, forceDownload bool) error {
	stepBanner("Import Standard Ebooks Plays")

	start := time.Now()
	os.MkdirAll(cacheDir, 0755)

	// Create SE source + edition
	sourceID, err := db.GetSourceID(database,
		"Standard Ebooks", "standard_ebooks",
		"https://standardebooks.org", "CC0 1.0 Universal",
		"https://creativecommons.org/publicdomain/zero/1.0/",
		"Text from Standard Ebooks (standardebooks.org). Released to the public domain under CC0 1.0 Universal.",
		false,
		"Public domain dedication. Based on public domain source texts.")
	if err != nil {
		return err
	}

	editionID, err := db.GetEditionID(database,
		"Standard Ebooks Modern Edition", "se_modern",
		sourceID, 2024, "Standard Ebooks editorial team",
		"Carefully produced modern-spelling editions. CC0.")
	if err != nil {
		return err
	}

	// Build work map
	worksMap, err := buildWorksMap(database)
	if err != nil {
		return err
	}

	// Sort repo names for deterministic ordering
	repoNames := make([]string, 0, len(constants.SEPlayRepos))
	for name := range constants.SEPlayRepos {
		repoNames = append(repoNames, name)
	}
	sort.Strings(repoNames)

	// Phase 1: Download + parse all plays (parallel when downloading, sequential from cache)
	type playEntry struct {
		repoName string
		ossID    string
		work     workInfo
		lines    []parser.PlayLine
	}
	var plays []playEntry

	if forceDownload {
		// Parallel download + parse
		var wg sync.WaitGroup
		resultCh := make(chan sePlayData, len(repoNames))

		// Limit concurrency to avoid GitHub rate limits
		sem := make(chan struct{}, 4)

		for _, repoName := range repoNames {
			ossID := constants.SEPlayRepos[repoName]
			work, ok := worksMap[ossID]
			if !ok {
				continue
			}

			wg.Add(1)
			go func(repoName, ossID string, work workInfo) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				cacheFile := filepath.Join(cacheDir, repoName+".json")
				actsData, err := downloadPlay(cacheFile, repoName, work.Title)
				if err != nil || actsData == nil {
					resultCh <- sePlayData{repoName: repoName, ossID: ossID, err: err}
					return
				}

				var allLines []parser.PlayLine
				actNames := sortedKeys(actsData)
				for _, fname := range actNames {
					content := actsData[fname]
					lines := parser.ParseSEPlay(content)
					allLines = append(allLines, lines...)
				}

				resultCh <- sePlayData{repoName: repoName, ossID: ossID, lines: allLines}
			}(repoName, ossID, work)
		}

		go func() {
			wg.Wait()
			close(resultCh)
		}()

		// Collect results
		resultMap := make(map[string]sePlayData)
		for result := range resultCh {
			resultMap[result.repoName] = result
		}

		// Assemble in deterministic order
		for _, repoName := range repoNames {
			ossID := constants.SEPlayRepos[repoName]
			work, ok := worksMap[ossID]
			if !ok {
				continue
			}
			if result, ok := resultMap[repoName]; ok && result.err == nil && len(result.lines) > 0 {
				plays = append(plays, playEntry{repoName: repoName, ossID: ossID, work: work, lines: result.lines})
			}
		}
	} else {
		// Sequential cache reads (fast, no parallelization needed)
		for num, repoName := range repoNames {
			ossID := constants.SEPlayRepos[repoName]
			work, ok := worksMap[ossID]
			if !ok {
				continue
			}

			cacheFile := filepath.Join(cacheDir, repoName+".json")
			actsData, err := loadFromCache(cacheFile, work.Title, num+1)
			if err != nil || actsData == nil {
				continue
			}

			var allLines []parser.PlayLine
			actNames := sortedKeys(actsData)
			for _, fname := range actNames {
				content := actsData[fname]
				lines := parser.ParseSEPlay(content)
				allLines = append(allLines, lines...)
			}

			if len(allLines) > 0 {
				plays = append(plays, playEntry{repoName: repoName, ossID: ossID, work: work, lines: allLines})
			}
		}
	}

	// Phase 2: Insert into DB (sequential — SQLite single writer)
	totalLines := 0
	totalPlays := 0

	for _, play := range plays {
		totalPlays++

		clearWorkEditionData(database, play.work.ID, editionID)

		charCache := make(map[string]any)
		tx, _ := database.Begin()
		insertStmt, _ := tx.Prepare(`
			INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num, line_number,
				character_id, char_name, content, content_type, word_count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)

		for _, line := range play.lines {
			charName := line.Character
			charID := cachedLookupCharacter(database, play.work.ID, charName, charCache)

			ct := contentType(line.IsStageDirection)
			if line.IsStageDirection {
				charName = ""
			}

			insertStmt.Exec(
				play.work.ID, editionID, line.Act, line.Scene, line.LineInScene, line.LineInScene,
				charID, nilIfEmpty(charName), line.Text, ct, countWords(line.Text))
		}
		insertStmt.Close()

		// Insert divisions
		actScenes := make([][2]int, 0, len(play.lines))
		for _, line := range play.lines {
			actScenes = append(actScenes, [2]int{line.Act, line.Scene})
		}
		insertTextDivisions(tx, play.work.ID, editionID, actScenes)

		tx.Commit()
		totalLines += len(play.lines)
		fmt.Printf("  [%2d/%d] %s: %d lines\n", totalPlays, len(plays), play.work.Title, len(play.lines))
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "se_plays", "import_complete",
		fmt.Sprintf("%d plays", totalPlays), totalLines, elapsed)

	fmt.Printf("  ✓ %d lines from %d plays in %.1fs\n", totalLines, totalPlays, elapsed)
	return nil
}

func loadFromCache(cacheFile, title string, num int) (map[string]string, error) {
	if data, err := os.ReadFile(cacheFile); err == nil {
		var acts map[string]string
		if json.Unmarshal(data, &acts) == nil {
			return acts, nil
		}
	}
	fmt.Printf("  [%2d/37] %s — SKIPPED (no cache; use -force-download to fetch)\n", num, title)
	return nil, nil
}

func downloadPlay(cacheFile, repoName, title string) (map[string]string, error) {
	fmt.Printf("  %s — downloading...\n", title)
	apiURL := fmt.Sprintf("https://api.github.com/repos/standardebooks/%s/contents/src/epub/text", repoName)
	listing, err := fetch.URL(apiURL)
	if err != nil {
		fmt.Printf("    ERROR: %v\n", err)
		return nil, err
	}

	var files []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(listing), &files); err != nil {
		return nil, err
	}

	acts := make(map[string]string)
	for _, f := range files {
		if !strings.HasPrefix(f.Name, "act-") || !strings.HasSuffix(f.Name, ".xhtml") {
			continue
		}
		url := fmt.Sprintf("https://raw.githubusercontent.com/standardebooks/%s/master/src/epub/text/%s",
			repoName, f.Name)
		content, err := fetch.URL(url)
		if err != nil {
			continue
		}
		acts[f.Name] = content
		time.Sleep(200 * time.Millisecond) // Rate limit per-file within a play
	}

	// Save cache
	cacheData, _ := json.Marshal(acts)
	os.MkdirAll(filepath.Dir(cacheFile), 0755)
	os.WriteFile(cacheFile, cacheData, 0644)

	return acts, nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
