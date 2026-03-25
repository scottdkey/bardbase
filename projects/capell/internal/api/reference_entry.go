package api

import (
	"net/http"
	"sort"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
)

type refCitation struct {
	WorkTitle *string `json:"work_title"`
	Act       *int    `json:"act"`
	Scene     *int    `json:"scene"`
	Line      *int    `json:"line"`
	WorkSlug  *string `json:"work_slug"`
}

type citationSpanJSON struct {
	Start    int     `json:"start"`
	End      int     `json:"end"`
	WorkSlug *string `json:"work_slug,omitempty"`
	Act      *int    `json:"act,omitempty"`
	Scene    *int    `json:"scene,omitempty"`
	Line     *int    `json:"line,omitempty"`
}

type referenceEntryDetail struct {
	ID            int                `json:"id"`
	Headword      string             `json:"headword"`
	RawText       string             `json:"raw_text"`
	SourceName    string             `json:"source_name"`
	SourceCode    string             `json:"source_code"`
	Citations     []refCitation      `json:"citations"`
	CitationSpans []citationSpanJSON `json:"citation_spans"`
}

func (s *Server) handleReferenceEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var entry referenceEntryDetail
	err := s.db.QueryRow(`
		SELECT re.id, re.headword, re.raw_text, s.name, s.short_code
		FROM reference_entries re
		JOIN sources s ON s.id = re.source_id
		WHERE re.id = ?`, id).Scan(
		&entry.ID, &entry.Headword, &entry.RawText, &entry.SourceName, &entry.SourceCode,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "entry not found")
		return
	}

	// Load citations with work titles
	rows, err := s.db.Query(`
		SELECT w.title, rc.act, rc.scene, rc.line
		FROM reference_citations rc
		LEFT JOIN works w ON w.id = rc.work_id
		WHERE rc.entry_id = ?
		ORDER BY w.title, rc.act, rc.scene, rc.line`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c refCitation
			var title *string
			if err := rows.Scan(&title, &c.Act, &c.Scene, &c.Line); err != nil {
				continue
			}
			c.WorkTitle = title
			if title != nil {
				slug := slugify(*title)
				c.WorkSlug = &slug
			}
			entry.Citations = append(entry.Citations, c)
		}
	}
	if entry.Citations == nil {
		entry.Citations = []refCitation{}
	}

	// Build citation spans with byte offsets into raw_text.
	spans := parser.LocateCitationSpans(entry.SourceCode, entry.RawText)
	abbrevMap := abbrevMapForSource(entry.SourceCode)
	entry.CitationSpans = make([]citationSpanJSON, 0, len(spans))
	for _, sp := range spans {
		cs := citationSpanJSON{
			Start: sp.Start, End: sp.End,
			Act: sp.Act, Scene: sp.Scene, Line: sp.Line,
		}
		if slug := resolveAbbrevToSlug(sp.WorkAbbrev, abbrevMap); slug != "" {
			s := slug
			cs.WorkSlug = &s
		}
		entry.CitationSpans = append(entry.CitationSpans, cs)
	}
	// Sort spans by position for frontend segment splitting.
	sort.Slice(entry.CitationSpans, func(i, j int) bool {
		return entry.CitationSpans[i].Start < entry.CitationSpans[j].Start
	})

	writeJSON(w, http.StatusOK, entry)
}

// abbrevMapForSource returns the abbreviation→Schmidt mapping for a source.
func abbrevMapForSource(sourceCode string) map[string]string {
	switch sourceCode {
	case "abbott":
		return constants.AbbottAbbrevs
	case "onions":
		return constants.OnionsAbbrevs
	case "bartlett":
		return constants.BartlettAbbrevs
	case "henley_farmer":
		return constants.HenleyFarmerAbbrevs
	default:
		return nil
	}
}

// resolveAbbrevToSlug maps a source-specific abbreviation to a work URL slug
// by going through the abbreviation→Schmidt→title→slug chain.
func resolveAbbrevToSlug(abbrev string, abbrevMap map[string]string) string {
	schmidtAbbrev := abbrev
	if abbrevMap != nil {
		if mapped, ok := abbrevMap[abbrev]; ok {
			schmidtAbbrev = mapped
		}
	}
	if work, ok := constants.SchmidtWorks[schmidtAbbrev]; ok {
		return slugify(work.Title)
	}
	return ""
}
