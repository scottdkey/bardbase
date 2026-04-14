package api

import (
	"fmt"
	"net/http"
	"strings"
)

type referenceSearchResult struct {
	ID         int    `json:"id"`
	Headword   string `json:"headword"`
	RawText    string `json:"raw_text"`
	SourceCode string `json:"source_code"`
	SourceName string `json:"source_name"`
}

func (s *Server) handleReferenceSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sourceCode := r.URL.Query().Get("source")
	workIDParam := r.URL.Query().Get("work_id")
	limit := queryInt(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}
	offset := queryInt(r, "offset", 0)

	includeSchmidt := sourceCode == "" || sourceCode == "schmidt"
	includeRefs := sourceCode == "" || (sourceCode != "schmidt")

	var unions []string
	var allArgs []any

	// Schmidt lexicon entries
	if includeSchmidt {
		sq := `SELECT le.id, le.base_key AS headword,
			COALESCE(le.full_text, '') AS raw_text,
			'schmidt' AS source_code,
			'Schmidt Shakespeare Lexicon' AS source_name
			FROM `

		if q != "" {
			cleaned := sanitizeFTS(q)
			if cleaned != "" {
				sq += `lexicon_fts JOIN lexicon_entries le ON le.id = lexicon_fts.rowid
					WHERE lexicon_fts MATCH ?`
				allArgs = append(allArgs, cleaned+"*")
			} else {
				sq += `lexicon_entries le WHERE 1=1`
			}
		} else {
			sq += `lexicon_entries le WHERE 1=1`
		}

		if workIDParam != "" {
			sq += ` AND le.id IN (SELECT DISTINCT entry_id FROM lexicon_citations WHERE work_id = ?)`
			allArgs = append(allArgs, workIDParam)
		}

		// Deduplicate by base_key
		sq += ` GROUP BY le.base_key`
		unions = append(unions, sq)
	}

	// Reference entries (onions, abbott, bartlett, henley_farmer)
	if includeRefs {
		rq := `SELECT re.id, re.headword,
			re.raw_text,
			s.short_code AS source_code,
			s.name AS source_name
			FROM `

		if q != "" && sourceCode != "schmidt" {
			cleaned := sanitizeFTS(q)
			if cleaned != "" {
				rq += `reference_fts JOIN reference_entries re ON re.id = reference_fts.rowid
					JOIN sources s ON s.id = re.source_id
					WHERE reference_fts MATCH ?`
				allArgs = append(allArgs, cleaned+"*")
			} else {
				rq += `reference_entries re JOIN sources s ON s.id = re.source_id WHERE 1=1`
			}
		} else {
			rq += `reference_entries re JOIN sources s ON s.id = re.source_id WHERE 1=1`
		}

		if sourceCode != "" && sourceCode != "schmidt" {
			rq += ` AND s.short_code = ?`
			allArgs = append(allArgs, sourceCode)
		}

		if workIDParam != "" {
			rq += ` AND re.id IN (SELECT DISTINCT entry_id FROM reference_citations WHERE work_id = ?)`
			allArgs = append(allArgs, workIDParam)
		}

		unions = append(unions, rq)
	}

	query := strings.Join(unions, " UNION ALL ") + " ORDER BY headword LIMIT ? OFFSET ?"
	allArgs = append(allArgs, limit, offset)

	rows, err := s.db.Query(query, allArgs...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("query failed: %v", err))
		return
	}
	defer rows.Close()

	var results []referenceSearchResult
	for rows.Next() {
		var r referenceSearchResult
		if err := rows.Scan(&r.ID, &r.Headword, &r.RawText, &r.SourceCode, &r.SourceName); err != nil {
			continue
		}
		if len(r.RawText) > 200 {
			r.RawText = r.RawText[:200] + "…"
		}
		results = append(results, r)
	}
	if results == nil {
		results = []referenceSearchResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

// handleReferenceIndex returns all reference entry IDs as a flat array.
// Used by the SvelteKit entries() function to enumerate prerender paths.
func (s *Server) handleReferenceIndex(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.Query(`SELECT id FROM reference_entries ORDER BY id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []int{}
	}
	writeJSON(w, http.StatusOK, ids)
}

func (s *Server) handleReferenceSources(w http.ResponseWriter, _ *http.Request) {
	type source struct {
		Code  string `json:"code"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	var result []source

	// Schmidt count
	var schmidtCount int
	if err := s.db.QueryRow(`SELECT COUNT(DISTINCT base_key) FROM lexicon_entries`).Scan(&schmidtCount); err == nil {
		result = append(result, source{Code: "schmidt", Name: "Schmidt Shakespeare Lexicon", Count: schmidtCount})
	}

	// Reference sources
	rows, err := s.db.Query(`
		SELECT s.short_code, s.name, COUNT(re.id) as entry_count
		FROM sources s
		JOIN reference_entries re ON re.source_id = s.id
		GROUP BY s.id
		ORDER BY s.name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var src source
		if err := rows.Scan(&src.Code, &src.Name, &src.Count); err != nil {
			continue
		}
		result = append(result, src)
	}
	if result == nil {
		result = []source{}
	}
	writeJSON(w, http.StatusOK, result)
}
