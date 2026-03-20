// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"regexp"
	"strings"
)

// bartlettAbbrevList is the ordered list of Bartlett 1896 play abbreviations,
// longest-first so that e.g. "Richard III." matches before "Richard II." and
// "3 Hen. VI." matches before "Hen. VI.". Internal spaces become \s+ when
// compiled into a regex to tolerate OCR double-spacing.
var bartlettAbbrevList = []string{
	// Full names — no dots, match as literals
	`Macbeth`, `Hamlet`, `Othello`, `Coriolanus`, `Cymbeline`, `Pericles`, `Tempest`, `Lear`,
	// Compound abbreviations — longest first
	`T\. G\. of Ver\.`,
	`Meas\. for Meas\.`,
	`Ant\. and Cleo\.`,
	`Troi\. and Cres\.`,
	`Rom\. and Jul\.`,
	`Com\. of Errors`,
	`T\. of Athens`,
	`T\. of Shrew`,
	`Mer\. of Venice`,
	`M\. N\. Dream`,
	`As Y\. Like It`,
	`L\. L\. Lost`,
	`Mer\. Wives`,
	`Much Ado`,
	`All's Well`,
	`T\. Andron\.`,
	`T\. A\.`,
	`T\. Night`,
	`T\. N\.`,
	`K\. John`,
	`J\. Caesar`,
	`J\. Ccesar`,
	`W\. Tale`,
	// Richard plays — III before II to avoid prefix clash
	`Richard III\.`,
	`Richard II\.`,
	// Numbered Henry plays — longer forms first
	`3 Hen\. VI\.`,
	`2 Hen\. VI\.`,
	`1 Hen\. VI\.`,
	`2 Hen\. IV\.`,
	`1 Hen\. IV\.`,
	`Hen\. VIII\.`,
	`Hen\. K\.`,
	`Hen\. V\.`,
	// Short abbreviated forms
	`Macb\.`,
	`Ham\.`,
	`M\. for M\.`,
}

// bartlettPlayRe matches a 3-component Bartlett citation at end of line:
//
//	PLAY_NAME  act(roman)  scene(arabic)  line(arabic)
//
// Internal spaces in abbreviations become \s+ for OCR tolerance.
// The pattern is applied line-by-line (or with multiline mode).
var bartlettPlayRe = func() *regexp.Regexp {
	parts := make([]string, len(bartlettAbbrevList))
	for i, a := range bartlettAbbrevList {
		parts[i] = strings.ReplaceAll(a, " ", `\s+`)
	}
	alt := strings.Join(parts, "|")
	return regexp.MustCompile(
		`\b(` + alt + `)\s+` +
			`([ivxlIVXL]+)\s+` +
			`(\d+)\s+` +
			`(\d+)\s*$`,
	)
}()

// ParseBartlettCitations extracts 3-component play citations from the raw OCR
// text of a Bartlett 1896 concordance entry (headword group).
//
// Only citations with all three components (act, scene, line) are extracted.
// Scene in Bartlett is Arabic (not Roman); act is Roman.
// Duplicate (abbrev, act, scene, line) tuples within one entry are suppressed.
func ParseBartlettCitations(rawText string) []RefCitation {
	seen := make(map[[4]int]bool)
	var out []RefCitation

	addPlay := func(abbrev string, act, scene, line int) {
		k := [4]int{int(strhash(abbrev)), act, scene, line}
		if seen[k] {
			return
		}
		seen[k] = true
		a, s, l := act, scene, line
		out = append(out, RefCitation{WorkAbbrev: abbrev, Act: &a, Scene: &s, Line: &l})
	}

	for _, line := range strings.Split(rawText, "\n") {
		m := bartlettPlayRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// m[1]=abbrev, m[2]=act_roman, m[3]=scene_arabic, m[4]=line_arabic
		// Collapse OCR double-spaces within the abbreviation.
		abbrev := strings.Join(strings.Fields(m[1]), " ")
		act := romanToInt(m[2])
		scene := atoi(m[3])
		lineNum := atoi(m[4])
		if act > 0 && scene > 0 && lineNum > 0 {
			addPlay(abbrev, act, scene, lineNum)
		}
	}

	return out
}
