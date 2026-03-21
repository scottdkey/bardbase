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
// Sub-senses (a, b, c, etc.) are stored with SubSense set to the letter.
type Sense struct {
	Number   int    // top-level sense number (1, 2, 3, ...)
	SubSense string // sub-sense letter ("a", "b", "c", ...) or "" for top-level
	Text     string
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
var subSensePattern = regexp.MustCompile(`(?:^|\s)([a-z])\)\s`)

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
		// Play references from Perseus:
		//   3-part "5.1.56" → act=5, scene=1, line=56
		//   2-part "4.60"   → ambiguous: could be act.scene OR act.line
		//   1-part "56"     → line only
		//
		// For 2-part refs, Shakespeare plays have at most ~10 scenes per act.
		// If the second number is > 15, it's almost certainly a line number,
		// not a scene number. Perseus format "shak.tmp 4.60" means act 4, line 60.
		// supplementFromDisplayText will refine further using the display text.
		const maxReasonableScene = 15
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
				if v > maxReasonableScene {
					// Too high to be a scene → treat as line number.
					ref.Line = &v
				} else {
					// Could be scene or line — store as scene for now.
					// supplementFromDisplayText will correct if display text
					// shows only 2 numbers (act, line) format.
					ref.Scene = &v
				}
			}
		case 1:
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Line = &v
			}
		}
	}

	return ref
}

// ParseSenses splits entry full text into numbered senses and sub-senses.
//
// Schmidt uses two levels:
//   - Numbered senses:    "1) first meaning  2) second meaning"
//   - Lettered sub-senses: "a) to stay  b) to remain  c) to continue"
//
// Sub-senses are children of the most recent numbered sense.
// If no numbered senses are found, returns a single sense with the full text.
func ParseSenses(fullText string) []Sense {
	senseMatches := findSenseMarkers(fullText)
	if len(senseMatches) == 0 {
		return []Sense{{Number: 1, Text: fullText}}
	}

	// Split into numbered sense chunks first.
	type senseChunk struct {
		number int
		text   string
	}
	var chunks []senseChunk
	for i, m := range senseMatches {
		start := m.textStart // first char after the "N) " pattern
		var end int
		if i+1 < len(senseMatches) {
			end = senseMatches[i+1].matchStart
		} else {
			end = len(fullText)
		}
		chunks = append(chunks, senseChunk{m.number, strings.TrimSpace(fullText[start:end])})
	}

	var senses []Sense
	for _, chunk := range chunks {
		// Look for sub-senses (a), b), c), etc.) within this sense.
		subMatches := findSubSenseMarkers(chunk.text)
		if len(subMatches) == 0 {
			// No sub-senses — single sense.
			senses = append(senses, Sense{Number: chunk.number, Text: chunk.text})
			continue
		}

		// Text before the first sub-sense is the sense's preamble.
		preamble := strings.TrimSpace(chunk.text[:subMatches[0].matchStart])
		if preamble != "" {
			senses = append(senses, Sense{Number: chunk.number, Text: preamble})
		}

		// Each sub-sense.
		for j, sub := range subMatches {
			start := sub.textStart
			var end int
			if j+1 < len(subMatches) {
				end = subMatches[j+1].matchStart
			} else {
				end = len(chunk.text)
			}
			senses = append(senses, Sense{
				Number:   chunk.number,
				SubSense: sub.letter,
				Text:     strings.TrimSpace(chunk.text[start:end]),
			})
		}
	}
	return senses
}

// senseMarker represents a found sense boundary position.
type senseMarker struct {
	number     int
	matchStart int // start of the match (including leading whitespace)
	textStart  int // start of definition text (after "N) ")
}

// subSenseMarker represents a found sub-sense boundary position.
type subSenseMarker struct {
	letter     string
	matchStart int
	textStart  int
}

// isInsideParens checks whether position pos is inside parentheses in text.
func isInsideParens(text string, pos int) bool {
	depth := 0
	for i := 0; i < pos && i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth > 0
}

// findSenseMarkers finds valid numbered sense boundaries (1), 2), 3), etc.)
// in the text. Filters out false positives inside parentheses and enforces
// sequential numbering: sense N+1 can only appear after sense N.
func findSenseMarkers(text string) []senseMarker {
	allMatches := sensePattern.FindAllStringSubmatchIndex(text, -1)
	if len(allMatches) == 0 {
		return nil
	}

	var markers []senseMarker
	expectedNext := 1

	for _, match := range allMatches {
		num, _ := strconv.Atoi(text[match[2]:match[3]])

		// Must be the next expected sense number
		if num != expectedNext {
			continue
		}

		// Must not be inside parentheses
		if isInsideParens(text, match[2]) {
			continue
		}

		markers = append(markers, senseMarker{
			number:     num,
			matchStart: match[0],
			textStart:  match[1],
		})
		expectedNext = num + 1
	}

	// Need at least sense 1 for the result to be valid
	if len(markers) == 0 || markers[0].number != 1 {
		return nil
	}
	return markers
}

// findSubSenseMarkers finds valid lettered sub-sense boundaries (a), b), c), etc.)
// in the text. Filters out false positives inside parentheses and enforces
// sequential lettering.
func findSubSenseMarkers(text string) []subSenseMarker {
	allMatches := subSensePattern.FindAllStringSubmatchIndex(text, -1)
	if len(allMatches) == 0 {
		return nil
	}

	var markers []subSenseMarker
	expectedNext := byte('a')

	for _, match := range allMatches {
		letter := text[match[2]:match[3]]

		// Must be the next expected letter
		if len(letter) != 1 || letter[0] != expectedNext {
			continue
		}

		// Must not be inside parentheses
		if isInsideParens(text, match[2]) {
			continue
		}

		markers = append(markers, subSenseMarker{
			letter:     letter,
			matchStart: match[0],
			textStart:  match[1],
		})
		expectedNext++
	}

	// Need at least sub-sense "a" for results to be valid
	if len(markers) == 0 || markers[0].letter != "a" {
		return nil
	}
	return markers
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

	// Definition text excludes elements that are displayed separately in the UI:
	//   - "orth": the headword (shown in the entry header)
	//   - "cit":  full citation blocks (quote + bibl, shown in references section)
	//   - "bibl": standalone references without a <cit> wrapper
	defText := normalizeWhitespace(entryFree.GetTextExcluding("orth", "cit", "bibl"))

	// Clean up artifacts from citation stripping: sequences of periods/commas
	// left behind where inline references were removed (e.g. "a name: . ."
	// from two consecutive <bibl> elements with ". " tails).
	defText = cleanDefinitionText(defText)

	// Expand Schmidt's headword abbreviations ("--ed", "--ing", etc.) in
	// the definition text so the stored definitions are human-readable.
	defText = expandAbbreviations(defText, key)

	// Senses parsed from definition text (without inline references)
	senses := ParseSenses(defText)

	// Citations from bibl elements
	var citations []Citation
	for _, bibl := range entryFree.FindAll("bibl") {
		biblN := bibl.Attr("n")
		displayText := strings.TrimSpace(bibl.GetText())

		// Find quote text from parent cit element.
		// Expand headword abbreviations ("--ed" → "abandoned", etc.)
		// so the stored quote is human-readable and searchable.
		var quoteText string
		for _, cit := range entryFree.FindAll("cit") {
			if cit.ContainsChild(bibl) {
				quoteElem := cit.Find("quote")
				if quoteElem != nil {
					quoteText = strings.TrimSpace(quoteElem.GetText())
					quoteText = expandAbbreviations(quoteText, key)
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
		if len(nums) >= 2 {
			if ref.Scene == nil {
				ref.Scene = &nums[0]
			}
			if ref.Line == nil {
				ref.Line = &nums[1]
			}
		} else if len(nums) == 1 {
			// Single number: could be sonnet number or line — prefer sonnet if scene missing
			if ref.Scene == nil {
				ref.Scene = &nums[0]
			} else if ref.Line == nil {
				ref.Line = &nums[0]
			}
		}
	case "poem":
		// Display: "Lucr. 452" → line=452
		if len(nums) >= 1 && ref.Line == nil {
			ref.Line = &nums[len(nums)-1]
		}
	default:
		// Play display text patterns:
		//   3 numbers: "H6B I, 2, 15" → act=1, scene=2, line=15
		//   2 numbers: "Tp. IV, 56"   → act=4, line=56 (Schmidt's format: act, line)
		//   1 number:  rare, usually just a line
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
			// Schmidt's 2-number play format is always "Act, Line" (never Act, Scene).
			// If we have act+scene but no line, the "scene" is actually the line.
			if ref.Act != nil && ref.Scene != nil && ref.Line == nil {
				line := *ref.Scene
				ref.Scene = nil
				ref.Line = &line
			}
			// If we only have act (no scene, no line), second number is the line.
			if ref.Act != nil && ref.Scene == nil && ref.Line == nil {
				ref.Line = &nums[1]
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

// expandAbbreviations expands Schmidt's headword abbreviations in text.
//
// Schmidt systematically abbreviates the headword being defined:
//   - "--ed" → headword + "ed" (e.g., for "abandon": "--ed" → "abandoned")
//   - "--ing" → headword + "ing"
//   - "--s" → headword + "s"
//   - "--ies" → headword stem + "ies" (for words ending in y: "ability" → "abilities")
//
// The headword is the lexicon entry key with trailing sense numbers stripped
// (e.g., "Ability" not "Ability1").
func expandAbbreviations(text, headword string) string {
	if headword == "" || !strings.Contains(text, "--") {
		return text
	}
	hw := strings.ToLower(headword)

	// Find each "--" and expand with the headword + suffix.
	var b strings.Builder
	i := 0
	for i < len(text) {
		dashIdx := strings.Index(text[i:], "--")
		if dashIdx < 0 {
			b.WriteString(text[i:])
			break
		}
		dashIdx += i

		// Write everything before the "--".
		b.WriteString(text[i:dashIdx])

		// Collect the suffix after "--" (letters until non-letter).
		suffixStart := dashIdx + 2
		suffixEnd := suffixStart
		for suffixEnd < len(text) {
			c := text[suffixEnd]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '\'' {
				suffixEnd++
			} else {
				break
			}
		}
		suffix := text[suffixStart:suffixEnd]

		if suffix == "" {
			// Bare "--" with no suffix → just the headword.
			b.WriteString(hw)
		} else {
			// Handle stem changes for common English inflection:
			// ability + ies → abilities (drop trailing y, add ies)
			// abide + s → abides
			stem := hw
			suffixLower := strings.ToLower(suffix)
			if strings.HasSuffix(suffixLower, "ies") && strings.HasSuffix(stem, "y") {
				stem = stem[:len(stem)-1] // drop y, suffix provides "ies"
			} else if strings.HasSuffix(suffixLower, "ied") && strings.HasSuffix(stem, "y") {
				stem = stem[:len(stem)-1]
			} else if (suffixLower == "ed" || suffixLower == "ing" || suffixLower == "er" || suffixLower == "est") && strings.HasSuffix(stem, "e") {
				stem = stem[:len(stem)-1] // abate + ed → abated (not abateed)
			}
			b.WriteString(stem)
			b.WriteString(suffix)
		}

		i = suffixEnd
	}
	return b.String()
}

// cleanDefinitionText removes artifacts left after stripping <cit> and <bibl>
// elements from definition text. When citations are removed, their surrounding
// punctuation (periods, commas between consecutive refs) remains as orphaned
// tokens like ". . ." or trailing ". ." at end of definitions.
//
// Rules:
//   - Collapse runs of orphaned periods/commas: ". . ." → nothing
//   - Preserve colons (they introduce subcategories like "With from:")
//   - Clean trailing punctuation artifacts at end of text
//   - Clean punctuation before sense markers: ". . 2)" → "2)"
var (
	// Matches sequences of orphaned periods/commas (with spaces between them).
	// These are artifacts from stripped citation references.
	// Matches: ". . .", ". .", ", .", etc. (2+ punctuation marks separated by spaces)
	orphanedPunct = regexp.MustCompile(`(?:\s*[.,]){2,}\s*`)
	// Matches a single orphaned period/comma (possibly with space) that follows
	// a colon or semicolon: "for: ." → "for:"
	colonThenDot = regexp.MustCompile(`([:;])\s*[.,]\s+`)
	// Trailing punctuation artifacts at end of string.
	trailingArtifacts = regexp.MustCompile(`[\s.,;:]+$`)
)

func cleanDefinitionText(s string) string {
	// Collapse runs of orphaned periods: ". . . ." → " "
	s = orphanedPunct.ReplaceAllString(s, " ")

	// Clean orphaned dot after colon: "for: ." → "for: "
	s = colonThenDot.ReplaceAllString(s, "$1 ")

	// Remove trailing punctuation artifacts.
	s = trailingArtifacts.ReplaceAllString(s, "")

	return normalizeWhitespace(s)
}
