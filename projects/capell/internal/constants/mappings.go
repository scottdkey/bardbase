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

// AbbottAbbrevs maps Abbott 1877 Shakespearian Grammar abbreviations to Schmidt
// abbreviations (works.schmidt_abbrev) where they differ. Pass-through abbrevs
// are absent — callers fall through to the raw abbreviation.
// Loaded from projects/data/abbott_abbrevs.json.
var AbbottAbbrevs map[string]string

// BartlettAbbrevs maps Bartlett 1896 concordance play name abbreviations to
// Schmidt abbreviations (works.schmidt_abbrev) where they differ. Pass-through
// abbrevs are absent — callers fall through to the raw abbreviation.
// Loaded from projects/data/bartlett_abbrevs.json.
var BartlettAbbrevs map[string]string

// HenleyFarmerAbbrevs maps Henley & Farmer slang dictionary play name forms to
// Schmidt abbreviations (works.schmidt_abbrev). Pass-through abbrevs are absent.
// Loaded from projects/data/henley_farmer_abbrevs.json.
var HenleyFarmerAbbrevs map[string]string

// CitationCorrection is a manual correction for an unmatched lexicon citation.
// Loaded from projects/data/citation_corrections.json.
//
// Location fields identify the correct text_line by semantic coordinates
// (stable across database rebuilds) rather than raw row IDs:
//   - WorkID: works.id of the correct play/poem (may differ from citation's work_id for wrong-play errors)
//   - Edition: editions.short_code (e.g. "se_modern", "oss_globe", "perseus_globe", "first_folio")
//   - Act, Scene: nullable; omit for poetry without act/scene structure
//   - LineNumber: required when Edition is set
//
// An entry with no Edition (empty string) documents why no match is possible.
type CitationCorrection struct {
	CitationID int64   `json:"citation_id"`
	Reason     string  `json:"reason"`
	WorkID     *int64  `json:"work_id,omitempty"`
	Edition    string  `json:"edition,omitempty"`
	Act        *int    `json:"act,omitempty"`
	Scene      *int    `json:"scene,omitempty"`
	LineNumber *int    `json:"line_number,omitempty"`
	Confidence float64 `json:"confidence"`
	Notes      string  `json:"notes"`
}

// CitationCorrections holds manual corrections for citations that can't be
// matched automatically. Loaded from projects/data/citation_corrections.json.
var CitationCorrections []CitationCorrection
