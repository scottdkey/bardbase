// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Command build constructs the Shakespeare database from source data.
//
// This is the main entry point for the capell project within the
// bardbase monorepo. It reads original source files from
// projects/sources/ and produces a SQLite database.
//
// Usage:
//
//	# From repo root via Makefile (recommended):
//	make capell build
//	make capell run
//	make capell run-cached
//
//	# Or directly from projects/capell/:
//	go run ./cmd/build                        # Full build (uses cached source files)
//	go run ./cmd/build -force-download        # Re-download SE source files
//	go run ./cmd/build -output build          # Custom output directory
//	go run ./cmd/build -step oss              # Run only one step
//
// Steps: oss, lexicon, se, poetry, perseus, folio, attributions, citations, mappings, fts
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottdkey/bardbase/projects/capell/internal/db"
	"github.com/scottdkey/bardbase/projects/capell/internal/importer"
)

func main() {
	output := flag.String("output", "build", "Output directory (relative to repo root)")
	forceDownload := flag.Bool("force-download", false, "Re-download Standard Ebooks source files (ignores cache)")
	step := flag.String("step", "", "Run only one step: oss, lexicon, se, poetry, perseus, folio, folger, eebo-quartos, onions, abbott, bartlett, henley-farmer, standalone, attributions, citations, ref-citations, mappings, fts")
	excludeStr := flag.String("exclude", "", "Comma-separated source keys to skip (e.g. folger,wordhoard)")
	flag.Parse()

	// Build exclusion set from --exclude flag.
	excludeSet := make(map[string]bool)
	if *excludeStr != "" {
		for _, k := range strings.Split(*excludeStr, ",") {
			excludeSet[strings.TrimSpace(k)] = true
		}
	}

	// Resolve paths relative to the monorepo root.
	// Sources live at projects/sources/, output goes to build/ at repo root.
	repoRoot := findRepoRoot()
	sourcesDir := filepath.Join(repoRoot, "projects", "sources")
	outputDir := filepath.Join(repoRoot, *output)
	dbPath := filepath.Join(outputDir, "bardbase.db")
	cacheDir := filepath.Join(sourcesDir, "se")

	ossSQLPath := filepath.Join(sourcesDir, "oss", "oss-db-full.sql")
	lexiconDir := filepath.Join(sourcesDir, "lexicon", "entries")

	fmt.Println("Capell Database Builder")
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
	//   6. folio       — Import First Folio 1623 (EEBO-TCP A11954, 35 plays, original spelling)
	//   7. folger      — Import Folger Shakespeare TEIsimple (37 plays, CC BY-NC 3.0)
	//   8. eebo-quartos — Import EEBO-TCP early quartos (Q1 Hamlet, Q1 1H4, etc.)
	//   9. onions       — Import Onions Shakespeare Glossary (1911, reference entries)
	//  10. abbott       — Import Abbott Shakespearian Grammar (1877, reference entries)
	//  11. bartlett     — Import Bartlett's Shakespeare Concordance (1896, reference entries)
	//  12. henley-farmer — Import Henley & Farmer Slang Dictionary (1890-1904, Shakespeare cits only)
	//  13. standalone  — Import standalone passages (biblical/classical cited by Schmidt)
	//  14. attributions — Populate attribution records for all sources
	//  15. mappings    — Build cross-edition line alignments (needed by citation propagation)
	//  16. citations   — Resolve lexicon citations to text_lines (with cross-edition propagation)
	//  17. ref-citations — Resolve reference entry citations to text_lines
	//  18. fts         — Build full-text search index
	type buildStep struct {
		name string
		fn   func() error
	}

	steps := []buildStep{
		{"oss", func() error { return importer.ImportOSS(database, ossSQLPath) }},
		{"lexicon", func() error { return importer.ImportLexicon(database, lexiconDir) }},
		{"se", func() error { return importer.ImportSEPlays(database, cacheDir, *forceDownload) }},
		{"poetry", func() error { return importer.ImportSEPoetry(database, cacheDir, *forceDownload) }},
		{"perseus", func() error { return importer.ImportPerseusPlays(database, sourcesDir) }},
		{"folio", func() error { return importer.ImportFirstFolio(database, sourcesDir) }},
		{"folger", func() error { return importer.ImportFolger(database, sourcesDir) }},
		{"eebo-quartos", func() error { return importer.ImportEEBOQuartos(database, sourcesDir) }},
		{"onions", func() error { return importer.ImportOnions(database, sourcesDir) }},
		{"abbott", func() error { return importer.ImportAbbott(database, sourcesDir) }},
		{"bartlett", func() error { return importer.ImportBartlett(database, sourcesDir) }},
		{"henley-farmer", func() error { return importer.ImportHenleyFarmer(database, sourcesDir) }},
		{"standalone", func() error { return importer.ImportStandalonePassages(database, sourcesDir) }},
		{"attributions", func() error { return importer.PopulateAttributions(database) }},
		{"mappings", func() error { return importer.BuildLineMappings(database) }},
		{"citations", func() error { return importer.ResolveCitations(database) }},
		{"ref-citations", func() error { return importer.ResolveReferenceCitations(database) }},
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
			if excludeSet[s.name] {
				fmt.Printf("  [excluded] %s\n", s.name)
				continue
			}
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
// of whether it's invoked from the repo root or from projects/capell/.
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
