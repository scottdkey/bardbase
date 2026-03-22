package api

import "net/http"

type refCitation struct {
	WorkTitle *string `json:"work_title"`
	Act       *int    `json:"act"`
	Scene     *int    `json:"scene"`
	Line      *int    `json:"line"`
	WorkSlug  *string `json:"work_slug"`
}

type referenceEntryDetail struct {
	ID         int          `json:"id"`
	Headword   string       `json:"headword"`
	RawText    string       `json:"raw_text"`
	SourceName string       `json:"source_name"`
	SourceCode string       `json:"source_code"`
	Citations  []refCitation `json:"citations"`
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

	writeJSON(w, http.StatusOK, entry)
}
