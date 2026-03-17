// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package constants

// SchmidtWork maps a Schmidt abbreviation to its full title, Perseus text ID, and work type.
type SchmidtWork struct {
	Title     string
	PerseusID string
	WorkType  string
}

// SchmidtWorks maps Schmidt abbreviations (with and without periods) to work metadata.
// Loaded from projects/data/schmidt_works.json.
var SchmidtWorks map[string]SchmidtWork

// PerseusToSchmidt maps Perseus short work codes to Schmidt abbreviations.
// Loaded from projects/data/perseus_to_schmidt.json.
var PerseusToSchmidt map[string]string

// Attributions holds the attribution rules for all sources.
// Loaded from projects/data/attributions.json.
var Attributions []AttributionDef
