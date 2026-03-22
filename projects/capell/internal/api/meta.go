package api

import (
	"net/http"
	"strings"
	"unicode"
)

// slugify converts a title like "Henry IV, Part I" to "henry-iv-part-i"
func slugify(title string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	var stats struct {
		WorkCount      int `json:"work_count"`
		CharacterCount int `json:"character_count"`
		LineCount      int `json:"line_count"`
		LexiconCount   int `json:"lexicon_count"`
	}
	err := s.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM works)          AS work_count,
		(SELECT COUNT(*) FROM characters)     AS character_count,
		(SELECT COUNT(*) FROM text_lines)     AS line_count,
		(SELECT COUNT(*) FROM lexicon_entries) AS lexicon_count`).
		Scan(&stats.WorkCount, &stats.CharacterCount, &stats.LineCount, &stats.LexiconCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleAttributions(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.Query(`SELECT s.name AS source_name,
		a.attribution_html,
		a.license_notice_text,
		COALESCE(a.display_priority, 0) AS display_priority,
		CASE WHEN a.required = 1 THEN 1 ELSE 0 END AS required
		FROM attributions a
		JOIN sources s ON s.id = a.source_id
		WHERE a.display_format = 'footer'
		  AND (
		    s.id IN (
		      SELECT DISTINCT e.source_id FROM editions e
		      WHERE EXISTS (SELECT 1 FROM text_lines tl WHERE tl.edition_id = e.id)
		    )
		    OR (s.short_code = 'perseus_schmidt'
		        AND EXISTS (SELECT 1 FROM lexicon_entries LIMIT 1))
		  )
		ORDER BY a.display_priority DESC, s.name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type attribution struct {
		SourceName       string  `json:"source_name"`
		AttributionHTML  string  `json:"attribution_html"`
		LicenseNotice    *string `json:"license_notice_text"`
		DisplayPriority  int     `json:"display_priority"`
		Required         bool    `json:"required"`
	}
	var result []attribution
	for rows.Next() {
		var a attribution
		var reqInt int
		if err := rows.Scan(&a.SourceName, &a.AttributionHTML, &a.LicenseNotice, &a.DisplayPriority, &reqInt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		a.Required = reqInt == 1
		result = append(result, a)
	}
	if result == nil {
		result = []attribution{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleWorks(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.Query(`SELECT id, title, work_type, date_composed FROM works ORDER BY title`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type work struct {
		ID           int     `json:"id"`
		Title        string  `json:"title"`
		Slug         string  `json:"slug"`
		WorkType     string  `json:"work_type"`
		DateComposed *string `json:"date_composed"`
	}

	var plays, poetry []work
	for rows.Next() {
		var w work
		if err := rows.Scan(&w.ID, &w.Title, &w.WorkType, &w.DateComposed); err != nil {
			continue
		}
		w.Slug = slugify(w.Title)
		switch w.WorkType {
		case "comedy", "tragedy", "history":
			plays = append(plays, w)
		case "poem", "sonnet_sequence", "apocrypha":
			poetry = append(poetry, w)
		// Skip biblical_reference, classical_reference, lexicon_appendix
		}
	}
	if plays == nil {
		plays = []work{}
	}
	if poetry == nil {
		poetry = []work{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plays": plays, "poetry": poetry})
}

func (s *Server) handleWorkTOC(w http.ResponseWriter, r *http.Request) {
	workId, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid work id")
		return
	}

	rows, err := s.db.Query(`SELECT act, scene, description, line_count
		FROM text_divisions
		WHERE work_id = ? AND edition_id = (
			SELECT MIN(edition_id) FROM text_divisions WHERE work_id = ?
		)
		ORDER BY act, scene`, workId, workId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type division struct {
		Act         int     `json:"act"`
		Scene       int     `json:"scene"`
		Description *string `json:"description"`
		LineCount   int     `json:"line_count"`
	}
	var result []division
	for rows.Next() {
		var d division
		if err := rows.Scan(&d.Act, &d.Scene, &d.Description, &d.LineCount); err != nil {
			continue
		}
		result = append(result, d)
	}
	if result == nil {
		result = []division{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleEditions(w http.ResponseWriter, r *http.Request) {
	workId, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid work id")
		return
	}

	rows, err := s.db.Query(`SELECT e.id, e.name, e.short_code, e.year, s.name AS source_name
		FROM editions e
		JOIN sources s ON s.id = e.source_id
		WHERE e.id IN (SELECT DISTINCT edition_id FROM text_lines WHERE work_id = ?)`, workId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type edition struct {
		ID         int     `json:"id"`
		Name       string  `json:"name"`
		ShortCode  string  `json:"short_code"`
		Year       *int    `json:"year"`
		SourceName string  `json:"source_name"`
	}
	var result []edition
	for rows.Next() {
		var e edition
		if err := rows.Scan(&e.ID, &e.Name, &e.ShortCode, &e.Year, &e.SourceName); err != nil {
			continue
		}
		result = append(result, e)
	}
	if result == nil {
		result = []edition{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleWorkBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	rows, err := s.db.Query(`SELECT id, title FROM works`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			continue
		}
		if slugify(title) == slug {
			writeJSON(w, http.StatusOK, map[string]any{"id": id, "title": title, "slug": slug})
			return
		}
	}
	writeError(w, http.StatusNotFound, "work not found")
}
