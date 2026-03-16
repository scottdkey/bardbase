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
	"time"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/constants"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/fetch"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/parser"
)

// ImportSEPlays imports Standard Ebooks play text from GitHub.
func ImportSEPlays(database *sql.DB, cacheDir string, forceDownload bool) error {
	stepBanner("STEP 3: Import Standard Ebooks Plays")

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

	totalLines := 0
	totalPlays := 0

	// Sort repo names for deterministic ordering
	repoNames := make([]string, 0, len(constants.SEPlayRepos))
	for name := range constants.SEPlayRepos {
		repoNames = append(repoNames, name)
	}
	sort.Strings(repoNames)

	for _, repoName := range repoNames {
		ossID := constants.SEPlayRepos[repoName]
		work, ok := worksMap[ossID]
		if !ok {
			continue
		}
		totalPlays++

		cacheFile := filepath.Join(cacheDir, repoName+".json")
		actsData, err := loadOrDownloadPlay(cacheFile, repoName, forceDownload, work.Title, totalPlays)
		if err != nil || actsData == nil {
			continue
		}

		// Parse all acts
		var allLines []parser.PlayLine
		actNames := sortedKeys(actsData)
		for _, fname := range actNames {
			content := actsData[fname]
			lines := parser.ParseSEPlay(content)
			allLines = append(allLines, lines...)
		}

		if len(allLines) == 0 {
			continue
		}

		// Clear existing SE data for this work
		clearWorkEditionData(database, work.ID, editionID)

		// Insert lines with character matching
		charCache := make(map[string]interface{}) // name → *int64 or nil
		tx, _ := database.Begin()
		insertStmt, _ := tx.Prepare(`
			INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num, line_number,
				character_id, char_name, content, content_type, word_count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)

		for _, line := range allLines {
			charName := line.Character
			charID := cachedLookupCharacter(database, work.ID, charName, charCache)

			ct := "speech"
			if line.IsStageDirection {
				ct = "stage_direction"
				charName = ""
			}

			// LineInScene is the scene-relative line number — use as both paragraph_num and line_number
			insertStmt.Exec(
				work.ID, editionID, line.Act, line.Scene, line.LineInScene, line.LineInScene,
				charID, nilIfEmpty(charName), line.Text, ct, countWords(line.Text))
		}
		insertStmt.Close()

		// Insert divisions
		scenes := make(map[[2]int]int)
		for _, line := range allLines {
			key := [2]int{line.Act, line.Scene}
			scenes[key]++
		}
		for key, count := range scenes {
			tx.Exec("INSERT OR IGNORE INTO text_divisions (work_id, edition_id, act, scene, line_count) VALUES (?, ?, ?, ?, ?)",
				work.ID, editionID, key[0], key[1], count)
		}

		tx.Commit()
		totalLines += len(allLines)
		fmt.Printf("  [%2d/37] %s: %d lines\n", totalPlays, work.Title, len(allLines))
	}

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "se_plays", "import_complete",
		fmt.Sprintf("%d plays", totalPlays), totalLines, elapsed)

	fmt.Printf("  ✓ %d lines from %d plays in %.1fs\n", totalLines, totalPlays, elapsed)
	return nil
}

type workInfo struct {
	ID    int64
	Title string
}

func loadOrDownloadPlay(cacheFile, repoName string, forceDownload bool, title string, num int) (map[string]string, error) {
	// Use cache unless force-download is requested
	if !forceDownload {
		if data, err := os.ReadFile(cacheFile); err == nil {
			var acts map[string]string
			if json.Unmarshal(data, &acts) == nil {
				return acts, nil
			}
		}
		fmt.Printf("  [%2d/37] %s — SKIPPED (no cache; use -force-download to fetch)\n", num, title)
		return nil, nil
	}

	fmt.Printf("  [%2d/37] %s — downloading...\n", num, title)
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
		time.Sleep(500 * time.Millisecond)
	}

	// Save cache
	cacheData, _ := json.Marshal(acts)
	os.MkdirAll(filepath.Dir(cacheFile), 0755)
	os.WriteFile(cacheFile, cacheData, 0644)

	return acts, nil
}

func lookupCharacter(database *sql.DB, workID int64, charName string) interface{} {
	var id int64
	err := database.QueryRow(
		"SELECT id FROM characters WHERE work_id = ? AND UPPER(name) = UPPER(?)",
		workID, charName).Scan(&id)
	if err != nil {
		err = database.QueryRow(
			"SELECT id FROM characters WHERE work_id = ? AND UPPER(abbrev) = UPPER(?)",
			workID, charName).Scan(&id)
	}
	if err != nil || id == 0 {
		return nil
	}
	return id
}

func countWords(s string) int {
	return len(strings.Fields(s))
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
