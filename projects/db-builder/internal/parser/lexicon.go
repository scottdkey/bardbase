package parser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/scottdkey/shakespeare_db/projects/db-builder/internal/constants"
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
// Two-part references are interpreted based on work type:
//   - Plays: act.scene (no line number)
//   - Sonnets: sonnet_number.line
//   - Poems: section.line
func ParsePerseusRef(biblN string) *PerseusRef {
	if biblN == "" || !strings.HasPrefix(biblN, "shak.") {
		return nil
	}

	rest := strings.TrimPrefix(biblN, "shak.")
	rest = strings.TrimSpace(rest)
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return nil
	}

	workCode := parts[0]
	numbers := parts[1]

	schmidtAbbrev, ok := constants.PerseusToSchmidt[workCode]
	if !ok {
		return nil
	}

	// Determine work type for two-part reference interpretation
	workType := ""
	if sw, ok := constants.SchmidtWorks[schmidtAbbrev]; ok {
		workType = sw.WorkType
	}

	ref := &PerseusRef{
		SchmidtAbbrev: schmidtAbbrev,
		Raw:           biblN,
	}

	numParts := strings.Split(numbers, ".")
	switch len(numParts) {
	case 3:
		// Always: act.scene.line
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
		switch workType {
		case "sonnet_sequence":
			// sonnet_number.line
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Scene = &v
			}
			if v, err := strconv.Atoi(numParts[1]); err == nil {
				ref.Line = &v
			}
		case "poem":
			// section.line (stanza or canto)
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Act = &v
			}
			if v, err := strconv.Atoi(numParts[1]); err == nil {
				ref.Line = &v
			}
		default:
			// Play: act.scene (line unknown from this ref alone)
			if v, err := strconv.Atoi(numParts[0]); err == nil {
				ref.Act = &v
			}
			if v, err := strconv.Atoi(numParts[1]); err == nil {
				ref.Scene = &v
			}
		}
	case 1:
		if numParts[0] != "" {
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

	// Full text (all text content, whitespace normalized)
	fullText := normalizeWhitespace(entryFree.GetText())

	// Senses
	senses := ParseSenses(fullText)

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

func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
