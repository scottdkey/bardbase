package importer

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
)

// PrintSummary prints the final database build summary.
func PrintSummary(database *sql.DB, dbPath string) {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("BUILD COMPLETE")
	fmt.Println("=" + strings.Repeat("=", 59))

	tables := []struct {
		table string
		label string
	}{
		{"works", "Works"},
		{"characters", "Characters"},
		{"text_lines", "Text lines"},
		{"text_divisions", "Text divisions"},
		{"lexicon_entries", "Lexicon entries"},
		{"lexicon_senses", "Lexicon senses"},
		{"lexicon_citations", "Lexicon citations"},
		{"citation_matches", "Citation matches"},
		{"line_mappings", "Line mappings"},
		{"attributions", "Attributions"},
		{"sources", "Sources"},
		{"editions", "Editions"},
	}

	for _, t := range tables {
		count, err := db.TableCount(database, t.table)
		if err != nil {
			fmt.Printf("  %-25s %10s\n", t.label, "ERROR")
			continue
		}
		fmt.Printf("  %-25s %10d\n", t.label, count)
	}

	fmt.Println()

	// Lines by edition
	rows, err := database.Query(`
		SELECT e.name, COUNT(*) FROM text_lines t
		JOIN editions e ON t.edition_id = e.id
		GROUP BY e.id ORDER BY e.id`)
	if err == nil {
		for rows.Next() {
			var name string
			var count int
			rows.Scan(&name, &count)
			fmt.Printf("  %-35s %10d lines\n", name, count)
		}
		rows.Close()
	}

	// Citation match stats
	fmt.Println()
	matchRows, err := database.Query(`
		SELECT match_type, COUNT(*), ROUND(AVG(confidence), 3)
		FROM citation_matches
		GROUP BY match_type ORDER BY COUNT(*) DESC`)
	if err == nil {
		fmt.Println("  Citation Matches:")
		for matchRows.Next() {
			var matchType string
			var count int
			var avgConf float64
			matchRows.Scan(&matchType, &count, &avgConf)
			fmt.Printf("    %-20s %8d  (avg confidence: %.3f)\n", matchType, count, avgConf)
		}
		matchRows.Close()
	}

	// Line mapping stats
	mapRows, err := database.Query(`
		SELECT match_type, COUNT(*), ROUND(AVG(similarity), 3)
		FROM line_mappings
		GROUP BY match_type ORDER BY COUNT(*) DESC`)
	if err == nil {
		fmt.Println("  Line Mappings:")
		for mapRows.Next() {
			var matchType string
			var count int
			var avgSim float64
			mapRows.Scan(&matchType, &count, &avgSim)
			fmt.Printf("    %-20s %8d  (avg similarity: %.3f)\n", matchType, count, avgSim)
		}
		mapRows.Close()
	}

	// Attribution summary
	fmt.Println()
	attrRows, err := database.Query(`
		SELECT s.short_code, a.required, a.display_format, a.display_priority
		FROM attributions a
		JOIN sources s ON a.source_id = s.id
		ORDER BY a.display_priority DESC`)
	if err == nil {
		fmt.Println("  Attributions:")
		for attrRows.Next() {
			var code, format string
			var required bool
			var priority int
			attrRows.Scan(&code, &required, &format, &priority)
			reqStr := "voluntary"
			if required {
				reqStr = "REQUIRED"
			}
			fmt.Printf("    %-20s %s (format=%s, priority=%d)\n", code, reqStr, format, priority)
		}
		attrRows.Close()
	}

	// File size
	info, err := os.Stat(dbPath)
	if err == nil {
		sizeMB := float64(info.Size()) / 1024.0 / 1024.0
		fmt.Printf("\n  Database: %s\n", dbPath)
		fmt.Printf("  Size: %.1f MB\n", sizeMB)
	}

	fmt.Printf("  Built: %s\n", time.Now().UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Println("=" + strings.Repeat("=", 59))
}
