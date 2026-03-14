// Command build constructs the Shakespeare database from source data.
//
// Usage:
//
//	go run ./cmd/build                        # Full build
//	go run ./cmd/build -skip-download         # Skip SE downloads (use cache)
//	go run ./cmd/build -output build          # Custom output directory
//	go run ./cmd/build -step oss              # Run only one step
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scottdkey/shakespeare_db/internal/db"
	"github.com/scottdkey/shakespeare_db/internal/importer"
)

func main() {
	output := flag.String("output", "build", "Output directory")
	skipDownload := flag.Bool("skip-download", false, "Skip Standard Ebooks downloads (use cache only)")
	step := flag.String("step", "", "Run only one step: oss, lexicon, se, poetry, fts")
	flag.Parse()

	// Resolve paths relative to repo root
	repoRoot := findRepoRoot()
	sourcesDir := filepath.Join(repoRoot, "sources")
	outputDir := filepath.Join(repoRoot, *output)
	dbPath := filepath.Join(outputDir, "shakespeare.db")
	cacheDir := filepath.Join(sourcesDir, "se")

	ossSQLPath := filepath.Join(sourcesDir, "oss", "oss-db-full.sql")
	lexiconDir := filepath.Join(sourcesDir, "lexicon", "entries")

	fmt.Println("Shakespeare Database Builder")
	fmt.Printf("  Output:  %s\n", dbPath)
	fmt.Printf("  Sources: %s\n", sourcesDir)
	fmt.Println()

	// Remove existing DB for clean build (unless running single step)
	if *step == "" {
		if err := db.RemoveIfExists(dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Open database
	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create schema
	fmt.Println("Creating schema...")
	if err := db.CreateSchema(database); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Define steps
	type buildStep struct {
		name string
		fn   func() error
	}

	steps := []buildStep{
		{"oss", func() error { return importer.ImportOSS(database, ossSQLPath) }},
		{"lexicon", func() error { return importer.ImportLexicon(database, lexiconDir) }},
		{"se", func() error { return importer.ImportSEPlays(database, cacheDir, *skipDownload) }},
		{"poetry", func() error { return importer.ImportSEPoetry(database, cacheDir, *skipDownload) }},
		{"fts", func() error { return importer.BuildFTS(database) }},
	}

	if *step != "" {
		// Run single step
		found := false
		for _, s := range steps {
			if s.name == *step {
				if err := s.fn(); err != nil {
					fmt.Fprintf(os.Stderr, "Error in step %s: %v\n", s.name, err)
					os.Exit(1)
				}
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Unknown step: %s (valid: oss, lexicon, se, poetry, fts)\n", *step)
			os.Exit(1)
		}
	} else {
		// Run all steps
		for _, s := range steps {
			if err := s.fn(); err != nil {
				fmt.Fprintf(os.Stderr, "Error in step %s: %v\n", s.name, err)
				os.Exit(1)
			}
		}
	}

	importer.PrintSummary(database, dbPath)
}

// findRepoRoot walks up from the executable/working directory to find go.mod.
func findRepoRoot() string {
	// Try working directory first
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fall back to working directory
	wd, _ := os.Getwd()
	return wd
}
