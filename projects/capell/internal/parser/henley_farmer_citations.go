// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"regexp"
	"strings"
)

// henleyFarmerAbbrevList is the ordered list of Henley & Farmer play name
// forms, longest-first so that longer names match before shorter prefixes.
var henleyFarmerAbbrevList = []string{
	`Much Ado [Aa]bout Nothing`,
	`Midsummer Night's Dream`,
	`Midsummer Night Dream`,
	`Two Gentlemen of Verona`,
	`Taming of the Shrew`,
	`Merry Wives of Windsor`,
	`Much Ado About Nothing`,
	`Much Ado about Nothing`,
	`Troilus and Cressida`,
	`Antony and Cleopatra`,
	`Love's Labour Lost`,
	`Loves Labour Lost`,
	`Merchant of Venice`,
	`Measure for Measure`,
	`Romeo and Juliet`,
	`Comedy of Errors`,
	`Two Gentlemen`,
	`Taming the Shrew`,
	`Timon of Athens`,
	`Titus Andronicus`,
	`Julius Caesar`,
	`As You Like It`,
	`Twelfth Night`,
	`Winter's Tale`,
	`King Richard III`,
	`King Richard II`,
	`King Henry VIII`,
	`King Henry VI`,
	`King Henry V`,
	`King Henry IV`,
	`Merry Wives`,
	`All's Well`,
	`The Tempest`,
	`3 Henry VI`,
	`2 Henry VI`,
	`1 Henry VI`,
	`2 Henry IV`,
	`1 Henry IV`,
	`Henry VIII`,
	`Richard III`,
	`Richard II`,
	`Coriolanus`,
	`Cymbeline`,
	`King John`,
	`King Lear`,
	`Henry VI`,
	`Henry IV`,
	`Henry V`,
	`Pericles`,
	`Othello`,
	`Macbeth`,
	`Hamlet`,
}

// henleyFarmerPlayAlt is the alternation string built from henleyFarmerAbbrevList.
var henleyFarmerPlayAlt = strings.Join(henleyFarmerAbbrevList, "|")

// hfCitRe1 matches: PLAY [year,] Act roman., Scene roman/arabic
// e.g. "King Lear [1605], Act ii., Scene 3"
//      "Love's Labour Lost [Act iii., Sc. i]"
var hfCitRe1 = regexp.MustCompile(
	`(` + henleyFarmerPlayAlt + `)` +
		`\s+(?:\[\d{4}\],?\s+)?` +
		`(?:\[)?` +
		`(?:[Aa]ct\s+)?` +
		`([ivxIVX]+)\.?,?\s+` +
		`(?:[Ss]cene|[Ss]c\.)\s+` +
		`([ivxIVX]+|\d+)`,
)

// hfCitRe2 matches: PLAY [roman., arabic.]
// e.g. "All's Well [ii., 2.]"
var hfCitRe2 = regexp.MustCompile(
	`(` + henleyFarmerPlayAlt + `)` +
		`\s+\[([ivxIVX]+)\.,\s+(\d+)\.?\]`,
)

// ParseHenleyFarmerCitations extracts play citations from the raw text of a
// Henley & Farmer slang dictionary entry. Since H&F does not include line
// numbers, only act and scene are captured; Line is nil in all results.
//
// Two citation forms are recognised:
//   - "PLAY [year,] Act roman., Scene roman/arabic"
//   - "PLAY [roman., arabic.]"
//
// Duplicate (abbrev, act, scene) tuples within one entry are suppressed.
func ParseHenleyFarmerCitations(rawText string) []RefCitation {
	// Use a 3-key dedup (no line component).
	seen := make(map[[3]int]bool)
	var out []RefCitation

	addCit := func(abbrev string, act, scene int) {
		k := [3]int{int(strhash(abbrev)), act, scene}
		if seen[k] {
			return
		}
		seen[k] = true
		a, s := act, scene
		out = append(out, RefCitation{WorkAbbrev: abbrev, Act: &a, Scene: &s, Line: nil})
	}

	resolveScene := func(s string) int {
		if isRoman(s) {
			return romanToInt(s)
		}
		return atoi(s)
	}

	// Pattern 1
	for _, m := range hfCitRe1.FindAllStringSubmatch(rawText, -1) {
		abbrev := m[1]
		act := romanToInt(m[2])
		scene := resolveScene(m[3])
		if act > 0 && scene > 0 {
			addCit(abbrev, act, scene)
		}
	}

	// Pattern 2
	for _, m := range hfCitRe2.FindAllStringSubmatch(rawText, -1) {
		abbrev := m[1]
		act := romanToInt(m[2])
		scene := atoi(m[3])
		if act > 0 && scene > 0 {
			addCit(abbrev, act, scene)
		}
	}

	return out
}
