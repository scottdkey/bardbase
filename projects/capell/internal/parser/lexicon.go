// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
)

// LexiconEntry represents a parsed Schmidt lexicon XML entry.
type LexiconEntry struct {
	Key         string
	Letter      string
	EntryType   string
	Orthography string
	FullText    string
	RawXML      string
	SourceFile  string
	Senses      []Sense
	Citations   []Citation
}

// Sense represents a numbered sense/definition within a lexicon entry.
type Sense struct {
	Number int
	Text   string
}

// Citation represents a work reference within a lexicon entry.
type Citation struct {
	WorkAbbrev  string
	Act         *int
	Scene       *int
	Line        *int
	SenseNumber int // which sense this citation belongs to (0 = unassigned)
	PerseusRef  string
	QuoteText   string
	DisplayText string
	RawBibl     string
}

// PerseusRef holds parsed components from a Perseus bibl reference.
type PerseusRef struct {
	SchmidtAbbrev string
	Act           *int
	Scene         *int
	Line          *int
	Raw           string
}

var sensePattern = regexp.MustCompile(`(?:^|\s)(\d+)\)\s`)

// ParsePerseusRef parses a Perseus bibl n= attribute into structured components.
// Returns nil if the reference is not a valid Shakespeare reference.
//
// Number interpretation depends on work type and part count:
//
//	Plays:            3-part → act.scene.line,   2-part → act.scene
//	Sonnets:          3-part → _.sonnet.line,    2-part → sonnet.line
//	Poems:            3-part → _._.line,          2-part → section.line, 1-part → line
//	Poem collections: 2-part → poem_number.line  (e.g., Passionate Pilgrim)
func ParsePerseusRef(biblN string) *PerseusRef {
	if biblN == "" || !strings.HasPrefix(biblN, "shak.") {
		return nil
	}

	rest := strings.TrimPrefix(biblN, "shak.")
	rest = strings.TrimSpace(rest)
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return nil
	}

	workCode := parts[0]

	schmidtAbbrev, ok := constants.PerseusToSchmidt[workCode]
	if !ok {
		return nil
	}

	// Determine work type for reference interpretation.
	workType := ""
	if sw, ok := constants.SchmidtWorks[schmidtAbbrev]; ok {
		workType = sw.WorkType
	}

	ref := &PerseusRef{
		SchmidtAbbrev: schmidtAbbrev,
		Raw:           biblN,
	}

	// No number part — still return the ref so work_id can be resolved.
	// Handles cases like "shak. luc" (work-only reference, no location).
	if len(parts) < 2 {
		return ref
	}

	numbers := parts[1]

	// Skip duplicated work codes (e.g., "shak. ven ven" → parts[1]="ven").
	// If the "numbers" part is not numeric, try the next part.
	if _, err := strconv.Atoi(strings.Split(numbers, ".")[0]); err != nil {
		if len(parts) >= 3 {
			numbers = parts[2]
		} else {
			return ref // work-only, no valid location
		}
	}

	numParts := strings.Split(numbers, ".")
	switch workType {
	case "sonnet_sequence":
		// 3-part: ignore first (volume), use second=sonnet, third=line
		// 2-part: sonnet.line
		// 1-part: line only
		switch len(numParts) {
		case 3:
			if v, err := strconv.Atoi(numParts[1]); err == nil {
				ref.Scene = &v
			}
			if v, err := strconv.Atoi(numParts[2]); err == nil {
				ref.Line = &v
			}
		case 2:
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Scene = &v
			}
			if v, err := strconv.Atoi(numParts[1]); err == nil {
				ref.Line = &v
			}
		case 1:
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Line = &v
			}
		}
	case "poem":
		// Poems use scene=section/poem_number, line=line.
		// 3-part: ignore first two sections, use third as line
		// 2-part: scene=section/poem_number, line=line
		// 1-part: line only
		switch len(numParts) {
		case 3:
			if v, err := strconv.Atoi(numParts[2]); err == nil {
				ref.Line = &v
			}
		case 2:
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Scene = &v
			}
			if v, err := strconv.Atoi(numParts[1]); err == nil {
				ref.Line = &v
			}
		case 1:
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Line = &v
			}
		}
	default:
		// Play: 3-part → act.scene.line, 2-part → act.scene, 1-part → line
		switch len(numParts) {
		case 3:
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Act = &v
			}
			if v, err := strconv.Atoi(numParts[1]); err == nil {
				ref.Scene = &v
			}
			if v, err := strconv.Atoi(numParts[2]); err == nil {
				ref.Line = &v
			}
		case 2:
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Act = &v
			}
			if v, err := strconv.Atoi(numParts[1]); err == nil {
				ref.Scene = &v
			}
		case 1:
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Line = &v
			}
		}
	}

	return ref
}

// ParseSenses splits entry full text into numbered senses.
// If no numbered senses are found, returns a single sense with the full text.
func ParseSenses(fullText string) []Sense {
	matches := sensePattern.FindAllStringSubmatchIndex(fullText, -1)
	if len(matches) == 0 {
		return []Sense{{Number: 1, Text: fullText}}
	}

	var senses []Sense
	for i, match := range matches {
		num, _ := strconv.Atoi(fullText[match[2]:match[3]])
		start := match[1] // end of the "N) " pattern
		var end int
		if i+1 < len(matches) {
			end = matches[i+1][0]
		} else {
			end = len(fullText)
		}
		senses = append(senses, Sense{
			Number: num,
			Text:   strings.TrimSpace(fullText[start:end]),
		})
	}
	return senses
}

// assignSensesToCitations determines which sense each citation belongs to
// by finding the citation's display text position within the full text
// relative to sense boundaries.
func assignSensesToCitations(entry *LexiconEntry) {
	if len(entry.Citations) == 0 {
		return
	}

	// If single sense or no senses, all citations belong to sense 1
	if len(entry.Senses) <= 1 {
		for i := range entry.Citations {
			entry.Citations[i].SenseNumber = 1
		}
		return
	}

	// Find sense boundary positions in full text
	type senseBound struct {
		number int
		start  int
	}
	matches := sensePattern.FindAllStringSubmatchIndex(entry.FullText, -1)
	var bounds []senseBound
	for _, match := range matches {
		num, _ := strconv.Atoi(entry.FullText[match[2]:match[3]])
		bounds = append(bounds, senseBound{number: num, start: match[0]})
	}

	if len(bounds) == 0 {
		for i := range entry.Citations {
			entry.Citations[i].SenseNumber = 1
		}
		return
	}

	// For each citation, find its position in the full text
	for i, cit := range entry.Citations {
		searchText := cit.DisplayText
		if searchText == "" {
			searchText = cit.RawBibl
		}
		if searchText == "" {
			// Can't determine position — assign to first sense
			entry.Citations[i].SenseNumber = bounds[0].number
			continue
		}

		pos := strings.Index(entry.FullText, searchText)
		if pos == -1 {
			// Not found — assign to first sense
			entry.Citations[i].SenseNumber = bounds[0].number
			continue
		}

		// Find which sense this position falls in
		senseNum := bounds[0].number
		for _, b := range bounds {
			if pos >= b.start {
				senseNum = b.number
			}
		}
		entry.Citations[i].SenseNumber = senseNum
	}
}

// ParseEntryXML parses a single Schmidt lexicon XML file into a LexiconEntry.
// The xmlContent should be the raw XML bytes, sourceFile is the basename of the file.
func ParseEntryXML(xmlContent []byte, sourceFile string) (*LexiconEntry, error) {
	root, err := ParseXML(xmlContent)
	if err != nil {
		// Try fixing unescaped ampersands
		fixed := strings.ReplaceAll(string(xmlContent), "&", "&amp;")
		root, err = ParseXML([]byte(fixed))
		if err != nil {
			return nil, err
		}
	}

	entryFree := root.Find("entryFree")
	if entryFree == nil {
		return nil, nil
	}

	key := entryFree.Attr("key")
	entryType := entryFree.Attr("type")
	if entryType == "" {
		entryType = "main"
	}

	// Letter from div1 element or directory name
	letter := ""
	div1 := root.Find("div1")
	if div1 != nil {
		letter = div1.Attr("n")
	}
	if letter == "" {
		// Fall back to parent directory name
		letter = filepath.Base(filepath.Dir(sourceFile))
	}

	// Orthography
	orthography := key
	orthElem := entryFree.Find("orth")
	if orthElem != nil {
		orthography = orthElem.GetText()
	}

	// Full text (all text content, whitespace normalized) — used for citation position matching
	fullText := normalizeWhitespace(entryFree.GetText())

	// Definition text excludes bibl references — used for sense definitions shown to users
	defText := normalizeWhitespace(entryFree.GetTextExcluding("bibl"))

	// Senses parsed from definition text (without inline references)
	senses := ParseSenses(defText)

	// Citations from bibl elements
	var citations []Citation
	for _, bibl := range entryFree.FindAll("bibl") {
		biblN := bibl.Attr("n")
		displayText := strings.TrimSpace(bibl.GetText())

		// Find quote text from parent cit element
		var quoteText string
		for _, cit := range entryFree.FindAll("cit") {
			if cit.ContainsChild(bibl) {
				quoteElem := cit.Find("quote")
				if quoteElem != nil {
					quoteText = strings.TrimSpace(quoteElem.GetText())
				}
				break
			}
		}

		parsed := ParsePerseusRef(biblN)
		if parsed != nil {
			// Supplement incomplete Perseus refs with data from display text
			workType := ""
			if sw, ok := constants.SchmidtWorks[parsed.SchmidtAbbrev]; ok {
				workType = sw.WorkType
			}
			supplementFromDisplayText(parsed, displayText, workType)

			citations = append(citations, Citation{
				WorkAbbrev:  parsed.SchmidtAbbrev,
				Act:         parsed.Act,
				Scene:       parsed.Scene,
				Line:        parsed.Line,
				PerseusRef:  parsed.Raw,
				QuoteText:   quoteText,
				DisplayText: displayText,
				RawBibl:     displayText,
			})
		} else if displayText != "" {
			ref := ""
			if biblN != "" {
				ref = biblN
			}
			citations = append(citations, Citation{
				PerseusRef:  ref,
				QuoteText:   quoteText,
				DisplayText: displayText,
				RawBibl:     displayText,
			})
		}
	}

	entry := &LexiconEntry{
		Key:         key,
		Letter:      letter,
		EntryType:   entryType,
		Orthography: orthography,
		FullText:    fullText,
		RawXML:      string(xmlContent),
		SourceFile:  sourceFile,
		Senses:      senses,
		Citations:   citations,
	}

	// Assign citations to their senses based on text position
	assignSensesToCitations(entry)

	return entry, nil
}

// displayTextNumbers extracts location numbers from Schmidt display text.
// Display text format examples: "H6B I, 2, 15", "Tp. IV, 56", "Ven. 654"
// Handles both Roman (I-X) and Arabic numerals, comma-separated.
var displayTextNumbers = regexp.MustCompile(`([IVXLC]+|\d+)(?:\s*,\s*(\d+))?(?:\s*,\s*(\d+))?\s*\.?\s*$`)

// supplementFromDisplayText fills in missing location data (act/scene/line) on a
// PerseusRef by parsing the display text from the <bibl> element. This handles
// cases where the Perseus n= attribute is incomplete (e.g. "shak. 2h6 1.2") but
// the display text has the full reference (e.g. "H6B I, 2, 15").
func supplementFromDisplayText(ref *PerseusRef, displayText string, workType string) {
	if ref == nil || displayText == "" {
		return
	}

	m := displayTextNumbers.FindStringSubmatch(displayText)
	if m == nil {
		return
	}

	// Collect the matched numbers (first may be Roman numeral)
	var nums []int
	for _, s := range m[1:] {
		if s == "" {
			break
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			// Try Roman numeral
			v = parseRoman(s)
			if v == 0 {
				break
			}
		}
		nums = append(nums, v)
	}

	if len(nums) == 0 {
		return
	}

	switch workType {
	case "sonnet_sequence":
		// Display: "Sonn. 112, 9" → sonnet=112, line=9
		if len(nums) >= 2 && ref.Scene != nil && ref.Line == nil {
			ref.Line = &nums[1]
		}
	case "poem":
		// Display: "Lucr. 452" → line=452
		if len(nums) >= 1 && ref.Line == nil {
			ref.Line = &nums[len(nums)-1]
		}
	default:
		// Play: display "H6B I, 2, 15" → act=1, scene=2, line=15
		if len(nums) == 3 {
			if ref.Act == nil {
				ref.Act = &nums[0]
			}
			if ref.Scene == nil {
				ref.Scene = &nums[1]
			}
			if ref.Line == nil {
				ref.Line = &nums[2]
			}
		} else if len(nums) == 2 {
			// Display has 2 numbers (e.g. "Tp. IV, 56" = act, line).
			// If Perseus gave us act+scene from a 2-part ref like "4.56",
			// the "scene" is actually a line number. Correct it.
			if ref.Act != nil && ref.Scene != nil && ref.Line == nil {
				line := *ref.Scene
				ref.Scene = nil
				ref.Line = &line
			}
		}
	}
}

var romanValues = map[byte]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100}

func parseRoman(s string) int {
	total := 0
	for i := 0; i < len(s); i++ {
		v := romanValues[s[i]]
		if v == 0 {
			return 0
		}
		if i+1 < len(s) && romanValues[s[i+1]] > v {
			total -= v
		} else {
			total += v
		}
	}
	return total
}

func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
