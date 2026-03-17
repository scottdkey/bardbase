// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Command fetch-perseus downloads Shakespeare play texts from the Perseus
// Digital Library as TEI XML files. It reads work metadata from
// projects/data/schmidt_works.json and downloads each text to
// projects/sources/perseus-plays/{perseus_id}.xml.
//
// Rate limited to 1 request per second to be polite to the Perseus server.
//
// Usage:
//
//	go run ./cmd/fetch-perseus                    # Fetch all works
//	go run ./cmd/fetch-perseus -work Tp.          # Fetch single work
//	go run ./cmd/fetch-perseus -skip-existing     # Skip already downloaded
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	perseusBaseURL = "http://www.perseus.tufts.edu/hopper/dltext"
	userAgent      = "Capell-Builder/2.0 (academic research; scottdkey/bardbase)"
	rateLimit      = 1 * time.Second
	httpTimeout    = 30 * time.Second
	maxRetries     = 3
)

type schmidtWork struct {
	Title     string `json:"title"`
	PerseusID string `json:"perseus_id"`
	WorkType  string `json:"work_type"`
}

func main() {
	singleWork := flag.String("work", "", "Fetch only this Schmidt abbreviation (e.g., Tp.)")
	skipExisting := flag.Bool("skip-existing", false, "Skip files that already exist")
	flag.Parse()

	// Resolve paths
	repoRoot := findRepoRoot()
	dataFile := filepath.Join(repoRoot, "projects", "data", "schmidt_works.json")
	outputDir := filepath.Join(repoRoot, "projects", "sources", "perseus-plays")

	// Read work metadata
	data, err := os.ReadFile(dataFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", dataFile, err)
		os.Exit(1)
	}

	var works map[string]schmidtWork
	if err := json.Unmarshal(data, &works); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Deduplicate by Perseus ID (aliases share IDs)
	type workEntry struct {
		Abbrev    string
		Title     string
		PerseusID string
		WorkType  string
	}
	seen := make(map[string]bool)
	var entries []workEntry
	for abbrev, w := range works {
		if seen[w.PerseusID] {
			continue
		}
		seen[w.PerseusID] = true
		entries = append(entries, workEntry{
			Abbrev:    abbrev,
			Title:     w.Title,
			PerseusID: w.PerseusID,
			WorkType:  w.WorkType,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PerseusID < entries[j].PerseusID
	})

	// Filter if single work requested
	if *singleWork != "" {
		var filtered []workEntry
		for _, e := range entries {
			if e.Abbrev == *singleWork {
				filtered = append(filtered, e)
				break
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "Work %q not found\n", *singleWork)
			os.Exit(1)
		}
		entries = filtered
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Perseus Text Fetcher\n")
	fmt.Printf("  Output: %s\n", outputDir)
	fmt.Printf("  Works:  %d\n", len(entries))
	fmt.Printf("  Rate:   1 req/sec\n\n")

	client := &http.Client{Timeout: httpTimeout}
	fetched := 0
	skipped := 0
	errors := 0

	for i, e := range entries {
		outPath := filepath.Join(outputDir, e.PerseusID+".xml")

		// Skip if already exists
		if *skipExisting {
			if _, err := os.Stat(outPath); err == nil {
				fmt.Printf("  [%d/%d] SKIP %s (%s) — already exists\n",
					i+1, len(entries), e.Abbrev, e.Title)
				skipped++
				continue
			}
		}

		// Rate limit
		if i > 0 {
			time.Sleep(rateLimit)
		}

		fmt.Printf("  [%d/%d] %s (%s) → %s.xml ... ",
			i+1, len(entries), e.Abbrev, e.Title, e.PerseusID)

		// Fetch with retries
		body, err := fetchWithRetries(client, e.PerseusID, maxRetries)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			errors++
			continue
		}

		// Write to file
		if err := os.WriteFile(outPath, []byte(body), 0644); err != nil {
			fmt.Printf("WRITE ERROR: %v\n", err)
			errors++
			continue
		}

		fmt.Printf("OK (%d bytes)\n", len(body))
		fetched++
	}

	fmt.Printf("\nDone: %d fetched, %d skipped, %d errors\n", fetched, skipped, errors)
}

func fetchWithRetries(client *http.Client, perseusID string, retries int) (string, error) {
	u := fmt.Sprintf("%s?doc=%s", perseusBaseURL,
		url.QueryEscape("Perseus:text:"+perseusID))

	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			time.Sleep(2 * time.Second)
			continue
		}

		return string(body), nil
	}

	return "", fmt.Errorf("failed after %d attempts: %w", retries, lastErr)
}

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
	wd, _ := os.Getwd()
	return wd
}
