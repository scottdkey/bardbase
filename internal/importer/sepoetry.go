package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/scottdkey/shakespeare_db/internal/constants"
	"github.com/scottdkey/shakespeare_db/internal/db"
	"github.com/scottdkey/shakespeare_db/internal/fetch"
	"github.com/scottdkey/shakespeare_db/internal/parser"
)

// ImportSEPoetry imports Standard Ebooks poetry, sonnets, and Folger URLs.
func ImportSEPoetry(database *sql.DB, cacheDir string, skipDownload bool) error {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("STEP 4: Import Poetry + Folger URLs")
	fmt.Println("=" + strings.Repeat("=", 59))

	start := time.Now()
	os.MkdirAll(cacheDir, 0755)

	// Get edition ID
	var editionID int64
	err := database.QueryRow("SELECT id FROM editions WHERE short_code = 'se_modern'").Scan(&editionID)
	if err != nil {
		fmt.Println("  ERROR: se_modern edition not found. Run step 3 first.")
		return fmt.Errorf("se_modern edition not found: %w", err)
	}

	// Build works map
	works := make(map[string]workInfo)
	rows, _ := database.Query("SELECT id, oss_id, title FROM works")
	for rows.Next() {
		var id int64
		var ossID, title string
		rows.Scan(&id, &ossID, &title)
		works[ossID] = workInfo{ID: id, Title: title}
	}
	rows.Close()

	base := "https://raw.githubusercontent.com/standardebooks/william-shakespeare_poetry/master/src/epub/text"
	totalImported := 0

	// === Poetry ===
	poetryCache := filepath.Join(cacheDir, "se-poetry.xhtml")
	poetryHTML := loadOrDownloadFile(poetryCache, base+"/poetry.xhtml", skipDownload, "Poetry")

	if poetryHTML != "" {
		poems := parser.ParseSEPoetry(poetryHTML)
		for articleID, lines := range poems {
			ossID, ok := constants.SEPoetryMap[articleID]
			if !ok {
				continue
			}
			work, ok := works[ossID]
			if !ok {
				continue
			}

			database.Exec("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", work.ID, editionID)
			tx, _ := database.Begin()
			for _, line := range lines {
				tx.Exec(`
					INSERT INTO text_lines (work_id, edition_id, paragraph_num, content, content_type, word_count, stanza)
					VALUES (?, ?, ?, ?, 'verse', ?, ?)`,
					work.ID, editionID, line.LineNumber, line.Text, countWords(line.Text), line.Stanza)
			}
			tx.Commit()
			totalImported += len(lines)
			fmt.Printf("  %s: %d lines\n", work.Title, len(lines))
		}
	}

	// === Sonnets ===
	sonnetsCache := filepath.Join(cacheDir, "se-sonnets.xhtml")
	sonnetsHTML := loadOrDownloadFile(sonnetsCache, base+"/sonnets.xhtml", skipDownload, "Sonnets")

	if sonnetsHTML != "" {
		data := parser.ParseSESonnets(sonnetsHTML)

		// Sonnets proper
		if work, ok := works["sonnets"]; ok {
			database.Exec("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", work.ID, editionID)
			tx, _ := database.Begin()
			sortOrder := 0

			sonnetNums := make([]int, 0, len(data.Sonnets))
			for num := range data.Sonnets {
				sonnetNums = append(sonnetNums, num)
			}
			sort.Ints(sonnetNums)

			for _, snum := range sonnetNums {
				for _, line := range data.Sonnets[snum] {
					sortOrder++
					tx.Exec(`
						INSERT INTO text_lines (work_id, edition_id, scene, paragraph_num, content,
							content_type, word_count, sonnet_number, stanza)
						VALUES (?, ?, ?, ?, ?, 'verse', ?, ?, ?)`,
						work.ID, editionID, snum, line.LineNumber, line.Text,
						countWords(line.Text), snum, line.Stanza)
				}
			}
			tx.Commit()
			totalImported += sortOrder
			fmt.Printf("  Sonnets: %d lines across %d sonnets\n", sortOrder, len(data.Sonnets))
		}

		// Lover's Complaint
		if work, ok := works["loverscomplaint"]; ok && len(data.LoversComplaint) > 0 {
			database.Exec("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", work.ID, editionID)
			tx, _ := database.Begin()
			for _, line := range data.LoversComplaint {
				tx.Exec(`
					INSERT INTO text_lines (work_id, edition_id, paragraph_num, content, content_type, word_count, stanza)
					VALUES (?, ?, ?, ?, 'verse', ?, ?)`,
					work.ID, editionID, line.LineNumber, line.Text, countWords(line.Text), line.Stanza)
			}
			tx.Commit()
			totalImported += len(data.LoversComplaint)
			fmt.Printf("  A Lover's Complaint: %d lines\n", len(data.LoversComplaint))
		}
	}

	// === Folger URLs ===
	updated := 0
	for ossID, slug := range constants.FolgerSlugs {
		url := fmt.Sprintf("https://www.folger.edu/explore/shakespeares-works/%s/", slug)
		result, err := database.Exec("UPDATE works SET folger_url = ? WHERE oss_id = ?", url, ossID)
		if err == nil {
			if n, _ := result.RowsAffected(); n > 0 {
				updated++
			}
		}
	}
	fmt.Printf("  Folger URLs: %d works\n", updated)

	elapsed := time.Since(start).Seconds()
	db.LogImport(database, "se_poetry", "import_complete",
		fmt.Sprintf("%d lines", totalImported), totalImported, elapsed)

	fmt.Printf("  ✓ %d poetry lines in %.1fs\n", totalImported, elapsed)
	return nil
}

func loadOrDownloadFile(cachePath, url string, skipDownload bool, label string) string {
	// Try cache
	if data, err := os.ReadFile(cachePath); err == nil {
		return string(data)
	}

	if skipDownload {
		fmt.Printf("  %s — SKIPPED (no cache)\n", label)
		return ""
	}

	fmt.Printf("  Downloading %s...\n", label)
	content, err := fetch.URL(url)
	if err != nil {
		fmt.Printf("  ERROR downloading %s: %v\n", label, err)
		return ""
	}

	os.MkdirAll(filepath.Dir(cachePath), 0755)
	os.WriteFile(cachePath, []byte(content), 0644)
	return content
}
