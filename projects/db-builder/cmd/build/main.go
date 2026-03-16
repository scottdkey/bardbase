// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Command build constructs the Shakespeare database from source data.
//
// This is the main entry point for the db-builder project within the
// shakespeare_db monorepo. It reads original source files from
// projects/sources/ and produces a SQLite database.
//
// Usage:
//
//	# From repo root via Makefile (recommended):
//	make db-builder build
//	make db-builder run
//	make db-builder run-cached
//
//	# Or directly from projects/db-builder/:
//	go run ./cmd/build                        # Full build
//	go run ./cmd/build -skip-download         # Skip SE downloads (use cache)
//	go run ./cmd/build -output build          # Custom output directory
//	go run ./cmd/build -step oss              # Run only one step
//
// Steps: oss, lexicon, se, poetry, perseus, attributions, citations, mappings, fts
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/db"
	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/importer"
)

func main() {
	output := flag.String("output", "build", "Output directory (relative to repo root)")
	skipDownload := flag.Bool("skip-download", false, "Skip Standard Ebooks downloads (use cache only)")
	step := flag.String("step", "", "Run only one step: oss, lexicon, se, poetry, perseus, attributions, citations, mappings, fts")
	flag.Parse()

	// Resolve paths relative to the monorepo root.
	// Sources live at projects/sources/, output goes to build/ at repo root.
	repoRoot := findRepoRoot()
	sourcesDir := filepath.Join(repoRoot, "projects", "sources")
	outputDir := filepath.Join(repoRoot, *output)
	dbPath := filepath.Join(outputDir, "shakespeare.db")
	cacheDir := filepath.Join(sourcesDir, "se")

	ossSQLPath := filepath.Join(sourcesDir, "oss", "oss-db-full.sql")
	lexiconDir := filepath.Join(sourcesDir, "lexicon", "entries")

	fmt.Println("Shakespeare Database Builder")
	fmt.Printf("  Repo:    %s\n", repoRoot)
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

	// Open database with tuned pragmas
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

	// Define the build pipeline as an ordered sequence of steps.
	// Each step is idempotent — it can be re-run independently via -step flag.
	//
	// Pipeline order:
	//   1. oss         — Import Open Source Shakespeare (base works, characters, text)
	//   2. lexicon     — Import Schmidt Lexicon (20k entries)
	//   3. se          — Import Standard Ebooks plays (modern edition)
	//   4. poetry      — Import Standard Ebooks poetry (sonnets, poems)
	//   5. perseus     — Import Perseus Globe edition plays (37 plays from TEI XML)
	//   6. attributions — Populate attribution records for all sources
	//   7. mappings    — Build cross-edition line alignments (needed by citation propagation)
	//   8. citations   — Resolve lexicon citations to text_lines (with cross-edition propagation)
	//   9. fts         — Build full-text search index
	type buildStep struct {
		name string
		fn   func() error
	}

	steps := []buildStep{
		{"oss", func() error { return importer.ImportOSS(database, ossSQLPath) }},
		{"lexicon", func() error { return importer.ImportLexicon(database, lexiconDir) }},
		{"se", func() error { return importer.ImportSEPlays(database, cacheDir, *skipDownload) }},
		{"poetry", func() error { return importer.ImportSEPoetry(database, cacheDir, *skipDownload) }},
		{"perseus", func() error { return importer.ImportPerseusPlays(database, sourcesDir) }},
		{"attributions", func() error { return importer.PopulateAttributions(database) }},
		{"mappings", func() error { return importer.BuildLineMappings(database) }},
		{"citations", func() error { return importer.ResolveCitations(database) }},
		{"fts", func() error { return importer.BuildFTS(database) }},
	}

	// Build valid step names for error message
	validSteps := make([]string, len(steps))
	for i, s := range steps {
		validSteps[i] = s.name
	}

	if *step != "" {
		// Run a single named step
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
			fmt.Fprintf(os.Stderr, "Unknown step: %s (valid: %s)\n", *step, fmt.Sprintf("%v", validSteps))
			os.Exit(1)
		}
	} else {
		// Run the full pipeline in order
		for _, s := range steps {
			if err := s.fn(); err != nil {
				fmt.Fprintf(os.Stderr, "Error in step %s: %v\n", s.name, err)
				os.Exit(1)
			}
		}

		// Post-import optimization: ANALYZE + VACUUM for smallest output
		fmt.Println()
		fmt.Println("Optimizing database...")
		if err := db.Optimize(database); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: optimization failed: %v\n", err)
			// Non-fatal — DB is still usable
		}
	}

	importer.PrintSummary(database, dbPath)
}

// findRepoRoot walks up from the working directory to find the monorepo root.
// It looks for a .git directory, which marks the top of the repository.
// This allows the builder to resolve paths to projects/sources/ regardless
// of whether it's invoked from the repo root or from projects/db-builder/.
func findRepoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fall back to working directory if no .git found
	wd, _ := os.Getwd()
	return wd
}
