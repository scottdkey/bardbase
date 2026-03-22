package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type editionLineRef struct {
	EditionID   int    `json:"edition_id"`
	EditionCode string `json:"edition_code"`
	LineNumber  *int   `json:"line_number"`
}

type citationDetail struct {
	ID               int              `json:"id"`
	EntryID          int              `json:"entry_id"`
	SenseID          *int             `json:"sense_id"`
	WorkID           *int             `json:"work_id"`
	WorkAbbrev       *string          `json:"work_abbrev"`
	WorkTitle        *string          `json:"work_title"`
	Act              *int             `json:"act"`
	Scene            *int             `json:"scene"`
	Line             *int             `json:"line"`
	QuoteText        *string          `json:"quote_text"`
	DisplayText      *string          `json:"display_text"`
	RawBibl          *string          `json:"raw_bibl"`
	MatchedLine      *string          `json:"matched_line"`
	MatchedLineNum   *int             `json:"matched_line_number"`
	MatchedCharacter *string          `json:"matched_character"`
	MatchedEditionID *int             `json:"matched_edition_id"`
	EditionLines     []editionLineRef `json:"edition_lines"`
}

type senseDetail struct {
	ID             int     `json:"id"`
	EntryID        int     `json:"entry_id"`
	SenseNumber    int     `json:"sense_number"`
	SubSense       *string `json:"sub_sense"`
	DefinitionText *string `json:"definition_text"`
}

type subEntryDetail struct {
	ID          int              `json:"id"`
	Key         string           `json:"key"`
	EntryType   *string          `json:"entry_type"`
	FullText    *string          `json:"full_text"`
	Orthography *string          `json:"orthography"`
	Senses      []senseDetail    `json:"senses"`
	Citations   []citationDetail `json:"citations"`
}

type referenceCitation struct {
	SourceName   string           `json:"source_name"`
	SourceCode   string           `json:"source_code"`
	WorkTitle    *string          `json:"work_title"`
	WorkAbbrev   *string          `json:"work_abbrev"`
	Act          *int             `json:"act"`
	Scene        *int             `json:"scene"`
	Line         *int             `json:"line"`
	EditionLines []editionLineRef `json:"edition_lines"`
}

type lexiconEntryResponse struct {
	ID          int                 `json:"id"`
	Key         string              `json:"key"`
	Orthography *string             `json:"orthography"`
	EntryType   *string             `json:"entry_type"`
	FullText    *string             `json:"full_text"`
	SubEntries  []subEntryDetail    `json:"subEntries"`
	Senses      []senseDetail       `json:"senses"`
	Citations   []citationDetail    `json:"citations"`
	References  []referenceCitation `json:"references"`
}

func (s *Server) handleLexiconEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	entry, err := s.getLexiconEntryFull(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if entry == nil {
		writeError(w, http.StatusNotFound, "entry not found")
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) getLexiconEntryFull(id int) (*lexiconEntryResponse, error) {
	// Find the entry and its base_key
	var entryID int
	var key, baseKey string
	var orthography, entryType, fullText *string
	err := s.db.QueryRow(`SELECT id, key, base_key, orthography, entry_type, full_text
		FROM lexicon_entries WHERE id = ?`, id).
		Scan(&entryID, &key, &baseKey, &orthography, &entryType, &fullText)
	if err != nil {
		return nil, nil // not found
	}

	// Get all entries in the group
	type groupEntry struct {
		id          int
		key         string
		orthography *string
		entryType   *string
		fullText    *string
	}
	groupRows, err := s.db.Query(`SELECT id, key, orthography, entry_type, full_text
		FROM lexicon_entries WHERE base_key = ? ORDER BY sense_group, id`, baseKey)
	if err != nil {
		return nil, fmt.Errorf("group entries: %w", err)
	}
	defer groupRows.Close()

	var groupEntries []groupEntry
	var entryIDs []int
	for groupRows.Next() {
		var ge groupEntry
		if err := groupRows.Scan(&ge.id, &ge.key, &ge.orthography, &ge.entryType, &ge.fullText); err != nil {
			return nil, fmt.Errorf("scan group entry: %w", err)
		}
		groupEntries = append(groupEntries, ge)
		entryIDs = append(entryIDs, ge.id)
	}

	placeholders := makePlaceholders(len(entryIDs))
	entryArgs := intsToArgs(entryIDs)

	// Load senses
	senseRows, err := s.db.Query(fmt.Sprintf(`SELECT ls.id, ls.entry_id, ls.sense_number, ls.sub_sense, ls.definition_text
		FROM lexicon_senses ls
		JOIN lexicon_entries le ON le.id = ls.entry_id
		WHERE ls.entry_id IN (%s)
		ORDER BY le.sense_group, ls.sense_number, COALESCE(ls.sub_sense, '')`, placeholders), entryArgs...)
	if err != nil {
		return nil, fmt.Errorf("senses: %w", err)
	}
	defer senseRows.Close()

	var allSenses []senseDetail
	for senseRows.Next() {
		var s senseDetail
		if err := senseRows.Scan(&s.ID, &s.EntryID, &s.SenseNumber, &s.SubSense, &s.DefinitionText); err != nil {
			return nil, fmt.Errorf("scan sense: %w", err)
		}
		allSenses = append(allSenses, s)
	}

	// Load citations (deduplicated by location)
	citRows, err := s.db.Query(fmt.Sprintf(`SELECT MIN(lc.id) AS id, lc.entry_id, lc.sense_id, lc.work_id, lc.work_abbrev,
		w.title AS work_title,
		lc.act, lc.scene, lc.line,
		MAX(lc.quote_text) AS quote_text, lc.display_text, lc.raw_bibl,
		tl.content AS matched_line,
		tl.line_number AS matched_line_number,
		COALESCE(
		  (SELECT ch.name FROM characters ch WHERE ch.id = tl.character_id),
		  tl.char_name
		) AS matched_character,
		cm.edition_id AS matched_edition_id
		FROM lexicon_citations lc
		LEFT JOIN works w ON w.id = lc.work_id
		LEFT JOIN citation_matches cm ON cm.citation_id = lc.id
		  AND cm.id = (
		    SELECT cm2.id FROM citation_matches cm2
		    WHERE cm2.citation_id = lc.id
		    ORDER BY CASE WHEN cm2.edition_id = 3 THEN 0 ELSE 1 END, cm2.confidence DESC
		    LIMIT 1
		  )
		LEFT JOIN text_lines tl ON tl.id = cm.text_line_id
		WHERE lc.entry_id IN (%s)
		GROUP BY lc.entry_id, lc.work_id, COALESCE(lc.act, -1), COALESCE(lc.scene, -1), COALESCE(lc.line, -1)
		ORDER BY w.title, lc.act, lc.scene, lc.line`, placeholders), entryArgs...)
	if err != nil {
		return nil, fmt.Errorf("citations: %w", err)
	}
	defer citRows.Close()

	var allCitations []citationDetail
	for citRows.Next() {
		var c citationDetail
		if err := citRows.Scan(&c.ID, &c.EntryID, &c.SenseID, &c.WorkID, &c.WorkAbbrev,
			&c.WorkTitle, &c.Act, &c.Scene, &c.Line,
			&c.QuoteText, &c.DisplayText, &c.RawBibl,
			&c.MatchedLine, &c.MatchedLineNum, &c.MatchedCharacter, &c.MatchedEditionID); err != nil {
			return nil, fmt.Errorf("scan citation: %w", err)
		}
		allCitations = append(allCitations, c)
	}

	// Headword validation: check if matched_line contains the headword
	headword := strings.ToLower(regexp.MustCompile(`\d+$`).ReplaceAllString(baseKey, ""))
	hwEscaped := regexp.QuoteMeta(headword)
	hwPattern := regexp.MustCompile(`(?i)\b` + hwEscaped)

	for i := range allCitations {
		c := &allCitations[i]
		if c.MatchedLine != nil && hwPattern.MatchString(*c.MatchedLine) {
			continue
		}
		if c.WorkID == nil || c.Act == nil || c.Scene == nil {
			continue
		}
		edID := 3
		if c.MatchedEditionID != nil {
			edID = *c.MatchedEditionID
		}
		target := 0
		if c.MatchedLineNum != nil {
			target = *c.MatchedLineNum
		} else if c.Line != nil {
			target = *c.Line
		}

		nearbyRows, err := s.db.Query(`SELECT line_number, content, char_name
			FROM text_lines
			WHERE work_id = ? AND edition_id = ? AND act = ? AND scene = ?
			  AND line_number BETWEEN ? AND ?
			ORDER BY line_number`, *c.WorkID, edID, *c.Act, *c.Scene, target-10, target+10)
		if err != nil {
			continue
		}

		type nearbyLine struct {
			lineNum  int
			content  string
			charName *string
		}
		var nearby []nearbyLine
		for nearbyRows.Next() {
			var nl nearbyLine
			if err := nearbyRows.Scan(&nl.lineNum, &nl.content, &nl.charName); err != nil {
				continue
			}
			nearby = append(nearby, nl)
		}
		nearbyRows.Close()

		found := false
		for offset := 0; offset <= 10; offset++ {
			for _, delta := range []int{-offset, offset} {
				if offset == 0 && delta != 0 {
					continue
				}
				for _, nl := range nearby {
					if nl.lineNum == target+delta && hwPattern.MatchString(nl.content) {
						c.MatchedLine = &nl.content
						c.MatchedLineNum = &nl.lineNum
						c.MatchedCharacter = nl.charName
						c.Line = &nl.lineNum
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			c.MatchedLine = nil
		}
	}

	// Batch-load cross-edition line numbers
	if len(allCitations) > 0 {
		citIDs := make([]int, len(allCitations))
		for i, c := range allCitations {
			citIDs[i] = c.ID
		}
		citPlaceholders := makePlaceholders(len(citIDs))
		citArgs := intsToArgs(citIDs)

		edRows, err := s.db.Query(fmt.Sprintf(`SELECT cm.citation_id, cm.edition_id, e.short_code, tl.line_number
			FROM citation_matches cm
			JOIN text_lines tl ON tl.id = cm.text_line_id
			JOIN editions e ON e.id = cm.edition_id
			WHERE cm.citation_id IN (%s)
			  AND cm.edition_id IN (1, 2, 3, 4, 5)
			ORDER BY cm.citation_id, cm.edition_id`, citPlaceholders), citArgs...)
		if err == nil {
			defer edRows.Close()
			edMap := map[int][]editionLineRef{}
			for edRows.Next() {
				var citID, edID int
				var code string
				var lineNum *int
				if err := edRows.Scan(&citID, &edID, &code, &lineNum); err != nil {
					continue
				}
				edMap[citID] = append(edMap[citID], editionLineRef{EditionID: edID, EditionCode: code, LineNumber: lineNum})
			}
			for i := range allCitations {
				if refs, ok := edMap[allCitations[i].ID]; ok {
					allCitations[i].EditionLines = refs
				}
			}
		}
	}

	// Ensure EditionLines is never nil
	for i := range allCitations {
		if allCitations[i].EditionLines == nil {
			allCitations[i].EditionLines = []editionLineRef{}
		}
	}

	// Load reference citations
	references := s.getReferenceCitations(baseKey)

	// Group senses and citations by entry
	sensesByEntry := map[int][]senseDetail{}
	for _, s := range allSenses {
		sensesByEntry[s.EntryID] = append(sensesByEntry[s.EntryID], s)
	}
	citsByEntry := map[int][]citationDetail{}
	for _, c := range allCitations {
		citsByEntry[c.EntryID] = append(citsByEntry[c.EntryID], c)
	}

	subEntries := make([]subEntryDetail, len(groupEntries))
	for i, ge := range groupEntries {
		senses := sensesByEntry[ge.id]
		if senses == nil {
			senses = []senseDetail{}
		}
		cits := citsByEntry[ge.id]
		if cits == nil {
			cits = []citationDetail{}
		}
		subEntries[i] = subEntryDetail{
			ID:          ge.id,
			Key:         ge.key,
			EntryType:   ge.entryType,
			FullText:    ge.fullText,
			Orthography: ge.orthography,
			Senses:      senses,
			Citations:   cits,
		}
	}

	if allSenses == nil {
		allSenses = []senseDetail{}
	}
	if allCitations == nil {
		allCitations = []citationDetail{}
	}

	return &lexiconEntryResponse{
		ID:          entryID,
		Key:         baseKey,
		Orthography: orthography,
		EntryType:   entryType,
		FullText:    fullText,
		SubEntries:  subEntries,
		Senses:      allSenses,
		Citations:   allCitations,
		References:  references,
	}, nil
}

func (s *Server) getReferenceCitations(baseKey string) []referenceCitation {
	rows, err := s.db.Query(`SELECT rc.id, src.name AS source_name, src.short_code AS source_code,
		w.title AS work_title, rc.work_abbrev, rc.act, rc.scene, rc.line
		FROM reference_citations rc
		JOIN reference_entries re ON re.id = rc.entry_id
		JOIN sources src ON src.id = rc.source_id
		LEFT JOIN works w ON w.id = rc.work_id
		WHERE LOWER(re.headword) = LOWER(?)
		ORDER BY src.name, w.title, rc.act, rc.scene, rc.line`, baseKey)
	if err != nil {
		return []referenceCitation{}
	}
	defer rows.Close()

	type refRow struct {
		id         int
		sourceName string
		sourceCode string
		workTitle  *string
		workAbbrev *string
		act        *int
		scene      *int
		line       *int
	}
	var refRows []refRow
	for rows.Next() {
		var r refRow
		if err := rows.Scan(&r.id, &r.sourceName, &r.sourceCode, &r.workTitle, &r.workAbbrev, &r.act, &r.scene, &r.line); err != nil {
			continue
		}
		refRows = append(refRows, r)
	}

	if len(refRows) == 0 {
		return []referenceCitation{}
	}

	// Batch-load edition lines for reference citations
	refIDs := make([]int, len(refRows))
	for i, r := range refRows {
		refIDs[i] = r.id
	}
	refPlaceholders := makePlaceholders(len(refIDs))
	refArgs := intsToArgs(refIDs)

	edMap := map[int][]editionLineRef{}
	edRows, err := s.db.Query(fmt.Sprintf(`SELECT rcm.ref_citation_id, rcm.edition_id, e.short_code, tl.line_number
		FROM reference_citation_matches rcm
		JOIN text_lines tl ON tl.id = rcm.text_line_id
		JOIN editions e ON e.id = rcm.edition_id
		WHERE rcm.ref_citation_id IN (%s)
		  AND rcm.edition_id IN (1, 2, 3, 4, 5)
		ORDER BY rcm.ref_citation_id, rcm.edition_id`, refPlaceholders), refArgs...)
	if err == nil {
		defer edRows.Close()
		for edRows.Next() {
			var refID, edID int
			var code string
			var lineNum *int
			if err := edRows.Scan(&refID, &edID, &code, &lineNum); err != nil {
				continue
			}
			edMap[refID] = append(edMap[refID], editionLineRef{EditionID: edID, EditionCode: code, LineNumber: lineNum})
		}
	}

	result := make([]referenceCitation, len(refRows))
	for i, r := range refRows {
		eds := edMap[r.id]
		if eds == nil {
			eds = []editionLineRef{}
		}
		result[i] = referenceCitation{
			SourceName:   r.sourceName,
			SourceCode:   r.sourceCode,
			WorkTitle:    r.workTitle,
			WorkAbbrev:   r.workAbbrev,
			Act:          r.act,
			Scene:        r.scene,
			Line:         r.line,
			EditionLines: eds,
		}
	}
	return result
}

// makePlaceholders returns "?,?,?" for n items.
func makePlaceholders(n int) string {
	if n == 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// intsToArgs converts []int to []any for use with db.Query.
func intsToArgs(ids []int) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}
