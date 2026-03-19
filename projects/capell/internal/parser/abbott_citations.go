// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"regexp"
	"strings"
)

// abbottAbbrevList is the ordered list of Abbott 1877 play abbreviations.
// Ordered longest-first so that e.g. "M. W. of W." matches before "M. W." and
// "I Hen. VI." matches before "Hen. VI.". When compiled into a regex, internal
// spaces become \s+ to tolerate OCR double-spaces (e.g. "J.  C.").
var abbottAbbrevList = []string{
	// Compound abbreviations — longest first
	`M\. W\. of W\.`, `T\. G\. of V\.`, `Tr\. and Cr\.`,
	`A\. and C\.`, `Ant\. and Cl\.`,
	`R\. and J\.`, `M\. for M\.`, `M\. N\. D\.`, `T\. of Sh\.`,
	`V\. and A\.`, `L\. L\. L\.`,
	`Com\. of E\.`, `C\. of E\.`,
	`A\. Y\. L\.`, `A\. Y\.`, `M\. N\.`, `T\. N\.`, `M\. W\.`,
	`A\. W\.`, `J\. C\.`, `L\. C\.`, `P\. of T\.`,
	// Numbered Henry plays — must come before bare "Hen."
	`I Hen\. VI\.`, `I Hen\. IV\.`, `I Hen\. VIII\.`,
	`1 Hen\. VI\.`, `1 Hen\. IV\.`, `1 Hen\. VIII\.`,
	`2 Hen\. VI\.`, `2 Hen\. IV\.`, `3 Hen\. VI\.`,
	`Hen\. VIII\.`, `Hen\. VII\.`, `Hen\. VI\.`, `Hen\. V\.`, `Hen\. IV\.`,
	// Richard plays
	`Rich\. III\.`, `Rich\. II\.`, `Richard III\.`, `Richard II\.`,
	// Standard single-word abbreviated forms
	`Macb\.`, `Cymb\.`, `Wint\.`, `Troil\.`, `Mids\.`, `Merch\.`, `Temp\.`,
	`Oth\.`, `Cor\.`, `Tit\.`, `Tim\.`, `Per\.`, `Err\.`, `Shr\.`, `Gent\.`,
	`All's`, `Meas\.`,
	// Single-letter Othello form used by Abbott in footnotes
	`O\.`,
	// Full-name forms Abbott uses in body text (with/without comma)
	`Hamlet,`, `Hamlet`, `Lear,`, `Lear`, `Tempest,`, `Tempest`,
	// Remaining single abbreviations
	`Ham\.`, `Ant\.`, `Rom\.`, `Tw\.`, `Mac\.`, `Cym\.`, `Lr\.`, `Tp\.`,
	`Ado`, `John`, `R2`, `R3`, `H5`, `H8`,
	`1H4`, `2H4`, `1H6`, `2H6`, `3H6`,
}

// abbottPlayRe matches a 3-part Globe citation in Abbott's format:
//
//	ABBREV  act.  scene.  line
//
// where act uses Roman numerals, scene uses Roman or Arabic numerals, and
// internal spaces in abbreviations are replaced with \s+ to handle OCR
// double-spacing (e.g. "J.  C.  iii.  2.  119").
var abbottPlayRe = func() *regexp.Regexp {
	// Replace each space in abbreviation patterns with \s+ to tolerate OCR
	// double-spaces within compound abbreviations.
	parts := make([]string, len(abbottAbbrevList))
	for i, a := range abbottAbbrevList {
		parts[i] = strings.ReplaceAll(a, " ", `\s+`)
	}
	alt := strings.Join(parts, "|")
	return regexp.MustCompile(
		`\b(` + alt + `)[,\s]\s*` +
			`([IVXivx]+)\.\s*` +
			`([IVXivx]+|\d+)\.\s*` +
			`(\d+)`,
	)
}()

// abbottPoemRe matches citations to non-sonnet poems (line number only):
// V. and A., L. C., P. of T. followed by an Arabic line number.
// Spaces become \s+ for OCR tolerance.
var abbottPoemRe = regexp.MustCompile(
	`\b(V\.\s+and\s+A\.|L\.\s+C\.|P\.\s+of\s+T\.)\s+(\d+)`,
)

// ParseAbbottCitations extracts all play/poem citations from the raw OCR text
// of an Abbott Shakespearian Grammar paragraph.
//
// Two citation forms are recognised:
//   - Play:   ABBREV act. scene. line   (e.g. "Macb.  ii.  3.  143")
//   - Poem:   ABBREV line               (e.g. "V. and A.  604")
//
// Roman numerals are converted to integers; Arabic numerals are parsed directly.
// Duplicate (abbrev, act, scene, line) tuples within one paragraph are suppressed.
func ParseAbbottCitations(rawText string) []RefCitation {
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
	addPoem := func(abbrev string, line int) {
		k := [4]int{int(strhash(abbrev)), 0, 0, line}
		if seen[k] {
			return
		}
		seen[k] = true
		l := line
		out = append(out, RefCitation{WorkAbbrev: abbrev, Line: &l})
	}

	// --- Play citations ---
	for _, m := range abbottPlayRe.FindAllStringSubmatch(rawText, -1) {
		// m[1]=abbrev m[2]=act_roman m[3]=scene_roman_or_arabic m[4]=line_arabic
		// Collapse OCR double-spaces within the abbreviation so that
		// "J.  C." → "J. C." (matching the key in abbott_abbrevs.json).
		abbrev := strings.Join(strings.Fields(m[1]), " ")
		act := romanToInt(m[2])
		// Scene can be roman or arabic — try roman first, fall back to arabic.
		scene := romanToInt(m[3])
		if scene == 0 {
			scene = atoi(m[3])
		}
		line := atoi(m[4])
		if act > 0 && scene > 0 && line > 0 {
			addPlay(abbrev, act, scene, line)
		}
	}

	// --- Poem citations ---
	for _, m := range abbottPoemRe.FindAllStringSubmatch(rawText, -1) {
		line := atoi(m[2])
		if line > 0 {
			addPoem(m[1], line)
		}
	}

	return out
}
