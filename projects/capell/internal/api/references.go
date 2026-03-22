package api

import "net/http"

type lineReference struct {
	EntryID    int     `json:"entry_id"`
	EntryKey   string  `json:"entry_key"`
	Source     string  `json:"source"`
	SourceCode string  `json:"source_code"`
	SenseID    *int    `json:"sense_id"`
	Definition *string `json:"definition"`
	QuoteText  *string `json:"quote_text"`
	Line       int     `json:"line"`
}

func (s *Server) handleSceneReferences(w http.ResponseWriter, r *http.Request) {
	workId, ok := pathInt(r, "workId")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid workId")
		return
	}
	act, ok := pathInt(r, "act")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid act")
		return
	}
	scene, ok := pathInt(r, "scene")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid scene")
		return
	}

	result := make(map[int][]lineReference)

	// 1. Schmidt lexicon citations
	schmidtRows, err := s.db.Query(`
		SELECT lc.line, le.id, le.base_key, lc.sense_id, ls.definition_text, lc.quote_text
		FROM lexicon_citations lc
		JOIN lexicon_entries le ON le.id = lc.entry_id
		LEFT JOIN lexicon_senses ls ON ls.id = lc.sense_id
		WHERE lc.work_id = ? AND lc.act = ? AND lc.scene = ?
		  AND lc.line IS NOT NULL
		ORDER BY lc.line, le.base_key`, workId, act, scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer schmidtRows.Close()

	for schmidtRows.Next() {
		var ref lineReference
		if err := schmidtRows.Scan(&ref.Line, &ref.EntryID, &ref.EntryKey, &ref.SenseID, &ref.Definition, &ref.QuoteText); err != nil {
			continue
		}
		ref.Source = "Schmidt Shakespeare Lexicon"
		ref.SourceCode = "schmidt"
		result[ref.Line] = append(result[ref.Line], ref)
	}

	// 2. Reference works (Onions, Abbott, Bartlett, Henley & Farmer, etc.)
	refRows, err := s.db.Query(`
		SELECT rc.line, re.id, re.headword, s.name, s.short_code, re.raw_text
		FROM reference_citations rc
		JOIN reference_entries re ON re.id = rc.entry_id
		JOIN sources s ON s.id = rc.source_id
		WHERE rc.work_id = ? AND rc.act = ? AND rc.scene = ?
		  AND rc.line IS NOT NULL
		GROUP BY rc.line, re.id
		ORDER BY rc.line, s.short_code, re.headword`, workId, act, scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer refRows.Close()

	for refRows.Next() {
		var line int
		var entryID int
		var headword, sourceName, sourceCode, rawText string
		if err := refRows.Scan(&line, &entryID, &headword, &sourceName, &sourceCode, &rawText); err != nil {
			continue
		}
		// Truncate raw_text for popover display
		def := rawText
		if len(def) > 300 {
			def = def[:300] + "…"
		}
		ref := lineReference{
			EntryID:    entryID,
			EntryKey:   headword,
			Source:     sourceName,
			SourceCode: sourceCode,
			Definition: &def,
			Line:       line,
		}
		result[line] = append(result[line], ref)
	}

	writeJSON(w, http.StatusOK, result)
}
