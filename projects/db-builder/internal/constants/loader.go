// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Package constants contains all reference data for the Shakespeare database build.
//
// Data is loaded from JSON files in projects/data/ at initialization time.
// The loader auto-discovers the data directory by walking up from the working
// directory to find the repository root (marked by .git), then resolving
// projects/data/ relative to it.
package constants

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	loadOnce sync.Once
	loadErr  error
	dataDir  string
)

func init() {
	EnsureLoaded()
}

// DataDir returns the resolved path to projects/data/.
// Returns empty string if not yet loaded or not found.
func DataDir() string {
	return dataDir
}

// EnsureLoaded triggers the one-time load of all JSON data files.
// Safe to call multiple times — only loads once.
// If the data directory cannot be found (e.g., running outside the repo),
// maps remain nil and an error is printed to stderr.
func EnsureLoaded() {
	loadOnce.Do(func() {
		dataDir = findDataDir()
		if dataDir == "" {
			loadErr = fmt.Errorf("constants: could not find projects/data/ directory (searched from working directory upward for .git)")
			fmt.Fprintf(os.Stderr, "WARNING: %v\n", loadErr)
			return
		}
		if err := loadAllData(dataDir); err != nil {
			loadErr = fmt.Errorf("constants: loading data from %s: %w", dataDir, err)
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", loadErr)
		}
	})
}

// LoadError returns any error encountered during data loading.
func LoadError() error {
	return loadErr
}

// findDataDir walks up from the working directory to find the repo root
// (directory containing .git), then returns the path to projects/data/.
func findDataDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			candidate := filepath.Join(dir, "projects", "data")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// loadAllData reads all JSON files from dataDir and populates package-level variables.
func loadAllData(dir string) error {
	if err := loadStringMap(filepath.Join(dir, "oss_to_schmidt.json"), &OSSToSchmidt); err != nil {
		return fmt.Errorf("oss_to_schmidt.json: %w", err)
	}
	if err := loadStringMap(filepath.Join(dir, "se_play_repos.json"), &SEPlayRepos); err != nil {
		return fmt.Errorf("se_play_repos.json: %w", err)
	}
	if err := loadStringMap(filepath.Join(dir, "se_poetry_map.json"), &SEPoetryMap); err != nil {
		return fmt.Errorf("se_poetry_map.json: %w", err)
	}
	if err := loadStringMap(filepath.Join(dir, "folger_slugs.json"), &FolgerSlugs); err != nil {
		return fmt.Errorf("folger_slugs.json: %w", err)
	}
	if err := loadStringMap(filepath.Join(dir, "folio_play_titles.json"), &FolioPlayTitles); err != nil {
		return fmt.Errorf("folio_play_titles.json: %w", err)
	}
	if err := loadStringMap(filepath.Join(dir, "genre_map.json"), &GenreMap); err != nil {
		return fmt.Errorf("genre_map.json: %w", err)
	}
	if err := loadStringMap(filepath.Join(dir, "perseus_to_schmidt.json"), &PerseusToSchmidt); err != nil {
		return fmt.Errorf("perseus_to_schmidt.json: %w", err)
	}
	if err := loadSchmidtWorks(filepath.Join(dir, "schmidt_works.json")); err != nil {
		return fmt.Errorf("schmidt_works.json: %w", err)
	}
	if err := loadAttributions(filepath.Join(dir, "attributions.json")); err != nil {
		return fmt.Errorf("attributions.json: %w", err)
	}
	return nil
}

// loadStringMap reads a JSON file containing a map[string]string.
func loadStringMap(path string, target *map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	m := make(map[string]string)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*target = m
	return nil
}

// schmidtWorkJSON is the JSON representation of a Schmidt work entry.
type schmidtWorkJSON struct {
	Title     string `json:"title"`
	PerseusID string `json:"perseus_id"`
	WorkType  string `json:"work_type"`
}

// loadSchmidtWorks reads schmidt_works.json and populates the SchmidtWorks map.
func loadSchmidtWorks(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	raw := make(map[string]schmidtWorkJSON)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	SchmidtWorks = make(map[string]SchmidtWork, len(raw))
	for abbrev, w := range raw {
		SchmidtWorks[abbrev] = SchmidtWork{
			Title:     w.Title,
			PerseusID: w.PerseusID,
			WorkType:  w.WorkType,
		}
	}
	return nil
}

// AttributionDef is the JSON representation of an attribution rule.
// Loaded from attributions.json and used by the importer to populate
// the attributions table.
type AttributionDef struct {
	SourceCode            string `json:"source_code"`
	Required              bool   `json:"required"`
	AttributionText       string `json:"attribution_text"`
	AttributionHTML       string `json:"attribution_html"`
	DisplayFormat         string `json:"display_format"`
	DisplayContext        string `json:"display_context"`
	DisplayPriority       int    `json:"display_priority"`
	RequiresLinkBack      bool   `json:"requires_link_back"`
	LinkBackURL           string `json:"link_back_url"`
	RequiresLicenseNotice bool   `json:"requires_license_notice"`
	LicenseNoticeText     string `json:"license_notice_text"`
	RequiresAuthorCredit  bool   `json:"requires_author_credit"`
	AuthorCreditText      string `json:"author_credit_text"`
	ShareAlikeRequired    bool   `json:"share_alike_required"`
	CommercialAllowed     bool   `json:"commercial_allowed"`
	ModificationAllowed   bool   `json:"modification_allowed"`
	Notes                 string `json:"notes"`
}

// loadAttributions reads attributions.json and populates the Attributions slice.
func loadAttributions(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var attrs []AttributionDef
	if err := json.Unmarshal(data, &attrs); err != nil {
		return err
	}
	Attributions = attrs
	return nil
}
