// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Command fetch-lexicon downloads missing Schmidt Lexicon entries from the
// Perseus Digital Library. It reads projects/sources/lexicon/entry_list.json
// to determine which entries are expected, scans the entries directory for
// existing files, and fetches any missing entries via the Perseus xmlchunk
// API endpoint.
//
// Usage:
//
//	go run ./cmd/fetch-lexicon                  # Download all missing entries
//	go run ./cmd/fetch-lexicon -dry-run         # Show what would be downloaded
//	go run ./cmd/fetch-lexicon -letter C        # Only fetch missing entries for letter C
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	perseusXMLChunkURL = "http://www.perseus.tufts.edu/hopper/xmlchunk"
	lexiconTextID      = "1999.03.0079"
	userAgent          = "Capell-Builder/2.0 (academic research; scottdkey/bardbase)"
	rateLimit          = 1 * time.Second
	httpTimeout        = 30 * time.Second
	maxRetries         = 3
)

type entryList struct {
	Total   int                    `json:"total"`
	Letters map[string]letterInfo  `json:"letters"`
}

type letterInfo struct {
	Entries []string `json:"entries"`
	Groups  int      `json:"groups"`
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Show what would be downloaded without fetching")
	letterFilter := flag.String("letter", "", "Only process this letter (e.g., C)")
	flag.Parse()

	repoRoot := findRepoRoot()
	entriesDir := filepath.Join(repoRoot, "projects", "sources", "lexicon", "entries")
	listPath := filepath.Join(repoRoot, "projects", "sources", "lexicon", "entry_list.json")

	// Load entry list
	data, err := os.ReadFile(listPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading entry list: %v\n", err)
		os.Exit(1)
	}
	var list entryList
	if err := json.Unmarshal(data, &list); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing entry list: %v\n", err)
		os.Exit(1)
	}

	// Find missing entries
	type missingEntry struct {
		Letter string
		Key    string // as in entry_list.json (+ for spaces)
	}
	var missing []missingEntry

	letters := make([]string, 0, len(list.Letters))
	for l := range list.Letters {
		letters = append(letters, l)
	}
	sort.Strings(letters)

	for _, letter := range letters {
		if *letterFilter != "" && letter != *letterFilter {
			continue
		}
		info := list.Letters[letter]
		letterDir := filepath.Join(entriesDir, letter)
		existing := scanExisting(letterDir)

		for _, key := range info.Entries {
			if !existing[key] {
				missing = append(missing, missingEntry{letter, key})
			}
		}
	}

	fmt.Printf("Perseus Lexicon Fetcher\n")
	fmt.Printf("  Entries dir: %s\n", entriesDir)
	fmt.Printf("  Missing:     %d entries\n", len(missing))
	fmt.Println()

	if len(missing) == 0 {
		fmt.Println("All entries present. Nothing to do.")
		return
	}

	if *dryRun {
		for _, m := range missing {
			fmt.Printf("  [dry-run] %s/%s\n", m.Letter, m.Key)
		}
		return
	}

	// Fetch missing entries
	client := &http.Client{Timeout: httpTimeout}
	fetched := 0
	notFound := 0
	errors := 0

	for i, m := range missing {
		if i > 0 {
			time.Sleep(rateLimit)
		}

		// Convert entry_list key to Perseus XML key (+ → space)
		xmlKey := strings.ReplaceAll(m.Key, "+", " ")

		fmt.Printf("  [%d/%d] %s/%s ... ", i+1, len(missing), m.Letter, m.Key)

		body, err := fetchEntry(client, m.Letter, xmlKey)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			errors++
			continue
		}

		// Check if we got a valid entry (contains entryFree element)
		if !strings.Contains(body, "entryFree") {
			fmt.Printf("NOT FOUND in Perseus\n")
			notFound++
			continue
		}

		// Write to file
		outPath := filepath.Join(entriesDir, m.Letter, m.Key+".xml")
		if err := os.WriteFile(outPath, []byte(body), 0644); err != nil {
			fmt.Printf("WRITE ERROR: %v\n", err)
			errors++
			continue
		}

		fmt.Printf("OK (%d bytes)\n", len(body))
		fetched++
	}

	fmt.Printf("\nDone: %d fetched, %d not in Perseus, %d errors\n", fetched, notFound, errors)
}

// fetchEntry retrieves a single lexicon entry from the Perseus xmlchunk API.
// URL format: /hopper/xmlchunk?doc=Perseus:text:1999.03.0079:alphabetic%20letter=L:entry=KEY
func fetchEntry(client *http.Client, letter, entryKey string) (string, error) {
	doc := fmt.Sprintf("Perseus:text:%s:alphabetic letter=%s:entry=%s",
		lexiconTextID, letter, entryKey)

	u := fmt.Sprintf("%s?doc=%s", perseusXMLChunkURL, url.PathEscape(doc))

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
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

	return "", fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// scanExisting returns a set of entry keys that already exist as XML files.
func scanExisting(letterDir string) map[string]bool {
	existing := make(map[string]bool)
	entries, err := os.ReadDir(letterDir)
	if err != nil {
		return existing
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".xml") {
			key := strings.TrimSuffix(e.Name(), ".xml")
			existing[key] = true
		}
	}
	return existing
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
