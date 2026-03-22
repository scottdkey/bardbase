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
	`Troi\. and Cret\.`,  // OCR: s→t
	`Rom\. and Jul\.`,
	`Rom\. and Jvl\.`,    // OCR: u→v
	`Com\. of Errors`,
	`T\. of Athens`,
	`T\. of Shrew`,
	`T\. of threw`,       // OCR: Sh→th
	`Mer\. of Venice`,
	`Her\. of Venice`,    // OCR: M→H
	`M\. N\. Dream`,
	`M\. \.V\. iirrnm`,  // OCR: N. Dream garbled
	`As Y\. Like It`,
	`L\. L\. Lost`,
	`L\. L\. L\.`,        // Alternate abbreviation
	`Mer\. Wives`,
	`Mrr\. Wives`,        // OCR: e→r
	`Mer\. Wires`,        // OCR: v→r
	`Much Ado`,
	`All's Well`,
	`T\. Andron\.`,
	`T\. A\.`,
	`T\. Night`,
	`T\. Sight`,          // OCR: N→S
	`T\. N\.`,
	`K\. John`,
	`J\. Caesar`,
	`J\. Ccesar`,         // OCR: ae→ce
	`J\. Cfesar`,         // OCR: ae→fe
	`W\. Tale`,
	// Richard plays — III before II to avoid prefix clash
	`Richard III\.`,
	`Richard II\.`,
	`Rich\.`,             // Shortened form
	// Numbered Henry plays — longer forms first
	`3 Hen\. VI\.`,
	`2 Hen\. VI\.`,
	`1 Hen\. VI\.`,
	`2 Hen\. IV\.`,
	`1 Hen\. IV\.`,
	`Hen\. VIII\.`,
	`Hen\. FIFL\.`,       // OCR: VIII→FIFL
	`Hen\. K\.`,
	`Hen\. V\.`,
	`Henry VIII\.`,
	// Short abbreviated forms
	`Macb\.`,
	`Ham\.`,
	`Hmnlr\.t`,           // OCR: Hamlet garbled
	`M\. for M\.`,
	// OCR corruption variants
	`Coriolanut`,          // OCR: s→t
	`Coriol\.`,            // Short form
	`Cymbrline`,           // OCR: missing e
	`Tult`,                // OCR: Tit.
	`Cmsar`,               // OCR: Caesar garbled
	`Temp\.`,              // Short Tempest
	// Poem abbreviations — used with line number only (2-component)
	`Sonn\.`,
	`Lucr\.`,
	`Lucrece`,
	`Ven\. and Adon\.`,
	`Ven\.`,
	`Pilgr\.`,
	`Phoen\.`,
	`Compl\.`,
	`Pass\. Pil\.`,
}

// bartlettPlayRe matches a Bartlett citation with play + act(roman) + scene + line.
// Now matches ANYWHERE in a line (not just end-of-line).
var bartlettPlayRe = func() *regexp.Regexp {
	parts := make([]string, len(bartlettAbbrevList))
	for i, a := range bartlettAbbrevList {
		parts[i] = strings.ReplaceAll(a, " ", `\s+`)
	}
	alt := strings.Join(parts, "|")
	// Match play + roman act + arabic scene + arabic line, anywhere in line.
	// Use non-greedy . before to allow mid-line matches.
	return regexp.MustCompile(
		`(?:^|[\s.])` +
			`(` + alt + `)` +
			`\s+([ivxlIVXL]+)\s+(\d+)\s+(\d+)` +
			`(?:\s|$|[.,;])`,
	)
}()

// bartlettPoemRe matches poem citations: POEM_ABBREV line_number
// These have no act/scene, just a line number.
var bartlettPoemRe = func() *regexp.Regexp {
	poems := []string{
		`Sonn\.`, `Lucr\.`, `Lucrece`, `Ven\. and Adon\.`, `Ven\.`,
		`Pilgr\.`, `Phoen\.`, `Compl\.`, `Pass\. Pil\.`,
	}
	for i, p := range poems {
		poems[i] = strings.ReplaceAll(p, " ", `\s+`)
	}
	alt := strings.Join(poems, "|")
	return regexp.MustCompile(
		`(?:^|[\s.])` +
			`(` + alt + `)` +
			`\s+(\d+)` +
			`(?:\s|$|[.,;])`,
	)
}()

// bartlettBareRe matches continuation citations without a play name:
// roman_act  arabic_scene  arabic_line at end of line (with optional text before).
var bartlettBareRe = regexp.MustCompile(
	`\s([ivxlIVXL]+)\s+(\d+)\s+(\d+)\s*$`,
)

// bartlettOCRAbbrevMap maps OCR-corrupted abbreviations to their canonical forms
// so they resolve correctly through the abbreviation mapping.
var bartlettOCRAbbrevMap = map[string]string{
	"Coriolanut":     "Coriolanus",
	"Cymbrline":      "Cymbeline",
	"Tult":           "T. Andron.",
	"Cmsar":          "J. Caesar",
	"Hmnlr.t":       "Hamlet",
	"T. Sight":       "T. Night",
	"T. of threw":    "T. of Shrew",
	"Her. of Venice": "Mer. of Venice",
	"Mrr. Wives":     "Mer. Wives",
	"Mer. Wires":     "Mer. Wives",
	"Troi. and Cret.": "Troi. and Cres.",
	"Rom. and Jvl.":  "Rom. and Jul.",
	"J. Cfesar":      "J. Caesar",
	"Hen. FIFL.":     "Hen. VIII.",
	"M. .V. iirrnm":  "M. N. Dream",
	"Temp.":          "Tempest",
	"Rich.":          "Richard III.",
	"L. L. L.":       "L. L. Lost",
	"Henry VIII.":    "Hen. VIII.",
	"Coriol.":        "Coriolanus",
}

// ParseBartlettCitations extracts play citations from the raw OCR text of a
// Bartlett 1896 concordance entry (headword group).
//
// Handles three citation formats:
//   - Full: PLAY act(roman) scene(arabic) line(arabic) — anywhere in line
//   - Poem: POEM_ABBREV line(arabic) — for sonnets, Lucrece, Venus, etc.
//   - Bare: act(roman) scene(arabic) line(arabic) — continuation lines
//     that inherit the play name from the previous citation
//
// Duplicate (abbrev, act, scene, line) tuples within one entry are suppressed.
func ParseBartlettCitations(rawText string) []RefCitation {
	seen := make(map[[4]int]bool)
	var out []RefCitation
	lastAbbrev := ""

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

	// Normalize OCR abbreviation to canonical form
	normalize := func(abbrev string) string {
		collapsed := strings.Join(strings.Fields(abbrev), " ")
		if canon, ok := bartlettOCRAbbrevMap[collapsed]; ok {
			return canon
		}
		// Also try with OCR comma→period substitution
		dotted := strings.ReplaceAll(collapsed, ",", ".")
		if dotted != collapsed {
			if canon, ok := bartlettOCRAbbrevMap[dotted]; ok {
				return canon
			}
		}
		return collapsed
	}

	for _, line := range strings.Split(rawText, "\n") {
		// Try full play citation (PLAY act scene line)
		// Use FindAllStringSubmatch to catch multiple citations per line
		matches := bartlettPlayRe.FindAllStringSubmatch(line, -1)
		if len(matches) > 0 {
			for _, m := range matches {
				abbrev := normalize(m[1])
				act := romanToInt(m[2])
				scene := atoi(m[3])
				lineNum := atoi(m[4])
				if act > 0 && scene > 0 && lineNum > 0 {
					addPlay(abbrev, act, scene, lineNum)
					lastAbbrev = abbrev
				}
			}
			continue
		}

		// Try poem citation (POEM line)
		pm := bartlettPoemRe.FindStringSubmatch(line)
		if pm != nil {
			abbrev := normalize(pm[1])
			lineNum := atoi(pm[2])
			if lineNum > 0 {
				addPoem(abbrev, lineNum)
				lastAbbrev = abbrev
			}
			continue
		}

		// Try bare continuation (act scene line without play name)
		if lastAbbrev != "" {
			bm := bartlettBareRe.FindStringSubmatch(line)
			if bm != nil {
				act := romanToInt(bm[1])
				scene := atoi(bm[2])
				lineNum := atoi(bm[3])
				if act > 0 && scene > 0 && lineNum > 0 {
					addPlay(lastAbbrev, act, scene, lineNum)
				}
			}
		}
	}

	return out
}
