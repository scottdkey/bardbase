package api

import "net/http"

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	limit := queryInt(r, "limit", 50)
	if limit > 100 {
		limit = 100
	}
	offset := queryInt(r, "offset", 0)

	cleaned := sanitizeFTS(q)
	if cleaned == "" {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	// Use prefix matching for autocomplete-style search
	matchExpr := cleaned + "*"

	rows, err := s.db.Query(`SELECT le.id, le.base_key AS key, le.orthography
		FROM lexicon_fts
		JOIN lexicon_entries le ON le.id = lexicon_fts.rowid
		WHERE lexicon_fts MATCH ?
		GROUP BY le.base_key
		ORDER BY rank
		LIMIT ? OFFSET ?`, matchExpr, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	defer rows.Close()

	type result struct {
		ID          int     `json:"id"`
		Key         string  `json:"key"`
		Orthography *string `json:"orthography"`
	}
	var results []result
	for rows.Next() {
		var r result
		if err := rows.Scan(&r.ID, &r.Key, &r.Orthography); err != nil {
			continue
		}
		results = append(results, r)
	}
	if results == nil {
		results = []result{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleLexiconLetters(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.Query(`SELECT letter, COUNT(*) AS count
		FROM lexicon_entries
		GROUP BY letter
		ORDER BY letter`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type letterCount struct {
		Letter string `json:"letter"`
		Count  int    `json:"count"`
	}
	var result []letterCount
	for rows.Next() {
		var lc letterCount
		if err := rows.Scan(&lc.Letter, &lc.Count); err != nil {
			continue
		}
		result = append(result, lc)
	}
	if result == nil {
		result = []letterCount{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleLexiconKeys(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.Query(`SELECT DISTINCT LOWER(base_key) FROM lexicon_entries ORDER BY 1`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			continue
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []string{}
	}
	writeJSON(w, http.StatusOK, keys)
}
