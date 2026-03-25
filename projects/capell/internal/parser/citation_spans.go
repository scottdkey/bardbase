// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strings"
)

// CitationSpan marks a citation's byte range within a reference entry's raw_text.
// The API layer resolves WorkAbbrev to a work slug for frontend linking.
type CitationSpan struct {
	Start      int    // byte offset of citation start in raw_text
	End        int    // byte offset of citation end in raw_text
	WorkAbbrev string // raw abbreviation as found in text
	Act        *int
	Scene      *int
	Line       *int
}

// LocateCitationSpans dispatches to the source-specific span locator.
func LocateCitationSpans(sourceCode, rawText string) []CitationSpan {
	switch sourceCode {
	case "abbott":
		return LocateAbbottCitationSpans(rawText)
	case "onions":
		return LocateOnionsCitationSpans(rawText)
	case "bartlett":
		return LocateBartlettCitationSpans(rawText)
	case "henley_farmer":
		return LocateHenleyFarmerCitationSpans(rawText)
	default:
		return nil
	}
}

// LocateAbbottCitationSpans finds citation byte ranges in Abbott entry text.
func LocateAbbottCitationSpans(rawText string) []CitationSpan {
	var spans []CitationSpan

	// Play citations: ABBREV act. scene. line
	for _, idx := range abbottPlayRe.FindAllStringSubmatchIndex(rawText, -1) {
		// idx[0:2] = full match, idx[2:4] = abbrev, idx[4:6] = act, idx[6:8] = scene, idx[8:10] = line
		abbrev := strings.Join(strings.Fields(rawText[idx[2]:idx[3]]), " ")
		act := romanToInt(rawText[idx[4]:idx[5]])
		scene := romanToInt(rawText[idx[6]:idx[7]])
		if scene == 0 {
			scene = atoi(rawText[idx[6]:idx[7]])
		}
		line := atoi(rawText[idx[8]:idx[9]])
		if act > 0 && scene > 0 && line > 0 {
			a, s, l := act, scene, line
			spans = append(spans, CitationSpan{
				Start: idx[0], End: idx[1],
				WorkAbbrev: abbrev, Act: &a, Scene: &s, Line: &l,
			})
		}
	}

	// Poem citations: ABBREV line
	for _, idx := range abbottPoemRe.FindAllStringSubmatchIndex(rawText, -1) {
		line := atoi(rawText[idx[4]:idx[5]])
		if line > 0 {
			l := line
			spans = append(spans, CitationSpan{
				Start: idx[0], End: idx[1],
				WorkAbbrev: rawText[idx[2]:idx[3]], Line: &l,
			})
		}
	}

	return spans
}

// LocateOnionsCitationSpans finds citation byte ranges in Onions entry text.
func LocateOnionsCitationSpans(rawText string) []CitationSpan {
	var spans []CitationSpan

	// Play citations
	for _, idx := range playRe.FindAllStringSubmatchIndex(rawText, -1) {
		abbrev := rawText[idx[2]:idx[3]]
		act := romanToInt(rawText[idx[4]:idx[5]])
		scene := romanToInt(rawText[idx[6]:idx[7]])
		line := atoi(rawText[idx[8]:idx[9]])
		if act > 0 && scene > 0 && line > 0 {
			a, s, l := act, scene, line
			spans = append(spans, CitationSpan{
				Start: idx[0], End: idx[1],
				WorkAbbrev: abbrev, Act: &a, Scene: &s, Line: &l,
			})
		}
	}

	// Sonnet citations
	for _, idx := range sonnetRe.FindAllStringSubmatchIndex(rawText, -1) {
		sonnetStr := rawText[idx[2]:idx[3]]
		sonnetNum := 0
		if isRoman(sonnetStr) {
			sonnetNum = romanToInt(sonnetStr)
		} else {
			sonnetNum = atoi(sonnetStr)
		}
		line := atoi(rawText[idx[4]:idx[5]])
		if sonnetNum > 0 && line > 0 {
			s, l := sonnetNum, line
			spans = append(spans, CitationSpan{
				Start: idx[0], End: idx[1],
				WorkAbbrev: "Sonn.", Scene: &s, Line: &l,
			})
		}
	}

	// Poem citations
	for _, idx := range poemRe.FindAllStringSubmatchIndex(rawText, -1) {
		line := atoi(rawText[idx[4]:idx[5]])
		if line > 0 {
			l := line
			spans = append(spans, CitationSpan{
				Start: idx[0], End: idx[1],
				WorkAbbrev: rawText[idx[2]:idx[3]], Line: &l,
			})
		}
	}

	return spans
}

// LocateBartlettCitationSpans finds citation byte ranges in Bartlett entry text.
// Bartlett entries are multi-line; bare continuation lines inherit the play name
// from the preceding citation.
func LocateBartlettCitationSpans(rawText string) []CitationSpan {
	var spans []CitationSpan
	lastAbbrev := ""

	// Bartlett processes line-by-line for bare continuations.
	offset := 0
	for _, line := range strings.Split(rawText, "\n") {
		// Full play citation
		if indices := bartlettPlayRe.FindAllStringSubmatchIndex(line, -1); len(indices) > 0 {
			for _, idx := range indices {
				abbrev := strings.Join(strings.Fields(line[idx[2]:idx[3]]), " ")
				if canon, ok := bartlettOCRAbbrevMap[abbrev]; ok {
					abbrev = canon
				}
				act := romanToInt(line[idx[4]:idx[5]])
				scene := atoi(line[idx[6]:idx[7]])
				lineNum := atoi(line[idx[8]:idx[9]])
				if act > 0 && scene > 0 && lineNum > 0 {
					a, s, l := act, scene, lineNum
					spans = append(spans, CitationSpan{
						Start: offset + idx[0], End: offset + idx[1],
						WorkAbbrev: abbrev, Act: &a, Scene: &s, Line: &l,
					})
					lastAbbrev = abbrev
				}
			}
			offset += len(line) + 1 // +1 for \n
			continue
		}

		// Poem citation
		if idx := bartlettPoemRe.FindStringSubmatchIndex(line); idx != nil {
			abbrev := strings.Join(strings.Fields(line[idx[2]:idx[3]]), " ")
			if canon, ok := bartlettOCRAbbrevMap[abbrev]; ok {
				abbrev = canon
			}
			lineNum := atoi(line[idx[4]:idx[5]])
			if lineNum > 0 {
				l := lineNum
				spans = append(spans, CitationSpan{
					Start: offset + idx[0], End: offset + idx[1],
					WorkAbbrev: abbrev, Line: &l,
				})
				lastAbbrev = abbrev
			}
			offset += len(line) + 1
			continue
		}

		// Bare continuation
		if lastAbbrev != "" {
			if idx := bartlettBareRe.FindStringSubmatchIndex(line); idx != nil {
				act := romanToInt(line[idx[2]:idx[3]])
				scene := atoi(line[idx[4]:idx[5]])
				lineNum := atoi(line[idx[6]:idx[7]])
				if act > 0 && scene > 0 && lineNum > 0 {
					a, s, l := act, scene, lineNum
					spans = append(spans, CitationSpan{
						Start: offset + idx[0], End: offset + idx[1],
						WorkAbbrev: lastAbbrev, Act: &a, Scene: &s, Line: &l,
					})
				}
			}
		}

		offset += len(line) + 1
	}

	return spans
}

// LocateHenleyFarmerCitationSpans finds citation byte ranges in Henley & Farmer text.
// H&F citations have act and scene but no line number.
func LocateHenleyFarmerCitationSpans(rawText string) []CitationSpan {
	var spans []CitationSpan

	resolveScene := func(s string) int {
		if isRoman(s) {
			return romanToInt(s)
		}
		return atoi(s)
	}

	// Pattern 1: PLAY [year,] Act roman., Scene roman/arabic
	for _, idx := range hfCitRe1.FindAllStringSubmatchIndex(rawText, -1) {
		abbrev := rawText[idx[2]:idx[3]]
		act := romanToInt(rawText[idx[4]:idx[5]])
		scene := resolveScene(rawText[idx[6]:idx[7]])
		if act > 0 && scene > 0 {
			a, s := act, scene
			spans = append(spans, CitationSpan{
				Start: idx[0], End: idx[1],
				WorkAbbrev: abbrev, Act: &a, Scene: &s,
			})
		}
	}

	// Pattern 2: PLAY [roman., arabic.]
	for _, idx := range hfCitRe2.FindAllStringSubmatchIndex(rawText, -1) {
		abbrev := rawText[idx[2]:idx[3]]
		act := romanToInt(rawText[idx[4]:idx[5]])
		scene := atoi(rawText[idx[6]:idx[7]])
		if act > 0 && scene > 0 {
			a, s := act, scene
			spans = append(spans, CitationSpan{
				Start: idx[0], End: idx[1],
				WorkAbbrev: abbrev, Act: &a, Scene: &s,
			})
		}
	}

	return spans
}
