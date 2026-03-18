// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"regexp"
	"strings"
)

// RefCitation represents a single play/poem citation extracted from the raw
// text of a reference entry (Onions, Abbott, etc.).
type RefCitation struct {
	WorkAbbrev string // raw abbreviation as it appears in the source text
	Act        *int   // nil for poems and sonnets
	Scene      *int   // nil for poems; sonnet number for Sonn.
	Line       *int   // Globe line number (nil if not parseable)
}

// onionsPlayAbbrevsAlt is the alternation of all known Onions 1911 play
// abbreviations, ordered longest-first so that e.g. "Tw.N." matches before
// "Tw." and "Troil." matches before "Tit.".
//
// Dots inside abbreviations are escaped; Henry plays without dots (H5, 2H4…)
// are included at the end.
const onionsPlayAbbrevsAlt = `` +
	`Compl\.|Phoen\.|Pilgr\.|Troil\.|` +
	`Meas\.|Wint\.|Lucr\.|Sonn\.|Caes\.|` +
	`LLL\.|MND\.|Gent\.|AYL\.|All's|` +
	`Mac\.|Rom\.|Tim\.|Tit\.|Oth\.|Ant\.|Cym\.|Ham\.|Merch\.|Per\.|Err\.|Shr\.|Ven\.|Wiv\.|Tw\.N\.|Tw\.|` +
	`Cor\.|Lr\.|Tp\.|Ado|John|R2|R3|H5|H8|HS|` +
	`1H4|2H4|1H6|2H6|3H6`

// playRe matches a 3-part Globe citation: ABBREV act. scene. line
// where act and scene are Roman numerals (upper or lower case) and line is Arabic.
// OCR double-spaces are handled by \s+/\s*.
var playRe = regexp.MustCompile(
	`\b(` + onionsPlayAbbrevsAlt + `)\s+` +
		`([IVXivx]+)\.\s*` +
		`([IVXivx]+)\.\s*` +
		`(\d+)`,
)

// sonnetRe matches Onions sonnet citations: "Sonn. [roman]. line"
// e.g. "Sonn.  xlii.  7" → sonnet 42, line 7.
// Also handles Arabic sonnet numbers for OCR variants.
var sonnetRe = regexp.MustCompile(
	`\bSonn?\.\s+([IVXivxlcLCM]+|\d+)\.\s*(\d+)`,
)

// poemRe matches citations to non-sonnet poems that use only a line number:
// Lucr., Ven., Phoen., Compl., Pilgr. followed by an Arabic number.
// e.g. "Lucr.  1403", "Ven.  52"
var poemRe = regexp.MustCompile(
	`\b(Lucr\.|Ven\.|Phoen\.|Compl\.|Pilgr\.)\s+(\d+)`,
)

// ParseOnionsCitations extracts all play/poem citations from the raw OCR text
// of an Onions glossary entry.
//
// Three citation forms are recognised:
//   - Play:   ABBREV act. scene. line   (e.g. "Ham.  ii.  i.  58")
//   - Sonnet: Sonn. N. line             (e.g. "Sonn.  xlii.  7")
//   - Poem:   ABBREV line               (e.g. "Lucr.  1403")
//
// Duplicate (abbrev, act, scene, line) tuples within one entry are suppressed.
// Roman numerals are converted to integers; Arabic line numbers are parsed directly.
func ParseOnionsCitations(rawText string) []RefCitation {
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
	addSonnet := func(sonnetNum, line int) {
		k := [4]int{0, 0, sonnetNum, line}
		if seen[k] {
			return
		}
		seen[k] = true
		s, l := sonnetNum, line
		out = append(out, RefCitation{WorkAbbrev: "Sonn.", Scene: &s, Line: &l})
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
	for _, m := range playRe.FindAllStringSubmatch(rawText, -1) {
		// m[1]=abbrev m[2]=act_roman m[3]=scene_roman m[4]=line_arabic
		abbrev := m[1]
		act := romanToInt(m[2])
		scene := romanToInt(m[3])
		line := atoi(m[4])
		if act > 0 && scene > 0 && line > 0 {
			addPlay(abbrev, act, scene, line)
		}
	}

	// --- Sonnet citations ---
	for _, m := range sonnetRe.FindAllStringSubmatch(rawText, -1) {
		// m[1]=sonnet_num (roman or arabic) m[2]=line_arabic
		sonnetNum := 0
		if isRoman(m[1]) {
			sonnetNum = romanToInt(m[1])
		} else {
			sonnetNum = atoi(m[1])
		}
		line := atoi(m[2])
		if sonnetNum > 0 && line > 0 {
			addSonnet(sonnetNum, line)
		}
	}

	// --- Poem citations (Lucr., Ven., etc.) ---
	for _, m := range poemRe.FindAllStringSubmatch(rawText, -1) {
		// m[1]=abbrev m[2]=line_arabic
		line := atoi(m[2])
		if line > 0 {
			addPoem(m[1], line)
		}
	}

	return out
}

// romanToInt converts a Roman numeral string (case-insensitive) to an integer.
// Returns 0 for empty or unrecognised input.
func romanToInt(s string) int {
	s = strings.ToUpper(s)
	vals := map[byte]int{
		'I': 1, 'V': 5, 'X': 10, 'L': 50,
		'C': 100, 'D': 500, 'M': 1000,
	}
	n := 0
	for i := 0; i < len(s); i++ {
		cur, ok := vals[s[i]]
		if !ok {
			return 0
		}
		if i+1 < len(s) {
			if next, ok2 := vals[s[i+1]]; ok2 && next > cur {
				n -= cur
				continue
			}
		}
		n += cur
	}
	return n
}

// isRoman returns true if s consists entirely of Roman numeral characters.
func isRoman(s string) bool {
	s = strings.ToUpper(s)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'I', 'V', 'X', 'L', 'C', 'D', 'M':
		default:
			return false
		}
	}
	return len(s) > 0
}

// strhash is a minimal string → uint32 for deduplication keys.
func strhash(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
