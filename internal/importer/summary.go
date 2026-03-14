package importer

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/scottdkey/shakespeare_db/internal/db"
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
