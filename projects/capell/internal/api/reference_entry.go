package api

import "net/http"

type referenceEntryDetail struct {
	ID         int    `json:"id"`
	Headword   string `json:"headword"`
	RawText    string `json:"raw_text"`
	SourceName string `json:"source_name"`
	SourceCode string `json:"source_code"`
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

	writeJSON(w, http.StatusOK, entry)
}
