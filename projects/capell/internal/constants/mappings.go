// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package constants

// OSSToSchmidt maps OSS/Moby work IDs to Schmidt abbreviations.
// Loaded from projects/data/oss_to_schmidt.json.
var OSSToSchmidt map[string]string

// SEPlayRepos maps Standard Ebooks repo names to OSS work IDs.
// Loaded from projects/data/se_play_repos.json.
var SEPlayRepos map[string]string

// SEPoetryMap maps SE poetry article IDs to OSS work IDs.
// Loaded from projects/data/se_poetry_map.json.
var SEPoetryMap map[string]string

// FolgerSlugs maps OSS work IDs to Folger Shakespeare Library URL slugs.
// Loaded from projects/data/folger_slugs.json.
var FolgerSlugs map[string]string

// FolioPlayTitles maps normalized First Folio play head text to OSS work IDs.
// Normalization: replace long-s (ſ→s), lowercase, collapse whitespace.
// Loaded from projects/data/folio_play_titles.json.
var FolioPlayTitles map[string]string

// GenreMap maps OSS single-letter genre codes to full work type names.
// Loaded from projects/data/genre_map.json.
var GenreMap map[string]string

// OnionsAbbrevs maps Onions 1911 abbreviations to Schmidt abbreviations
// (works.schmidt_abbrev) where they differ. Pass-through abbrevs (same in both)
// are absent — callers fall through to the raw abbreviation.
// Loaded from projects/data/onions_abbrevs.json.
var OnionsAbbrevs map[string]string

// CitationCorrection is a manual correction for an unmatched lexicon citation.
// Loaded from projects/data/citation_corrections.json.
type CitationCorrection struct {
	CitationID    int64   `json:"citation_id"`
	Reason        string  `json:"reason"`
	BestLineID    *int64  `json:"best_line_id"`
	BestEditionID *int64  `json:"best_edition_id"`
	Confidence    float64 `json:"confidence"`
	Notes         string  `json:"notes"`
}

// CitationCorrections holds manual corrections for citations that can't be
// matched automatically. Loaded from projects/data/citation_corrections.json.
var CitationCorrections []CitationCorrection
