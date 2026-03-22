package api

import (
	"fmt"
	"net/http"
)

type editionInfo struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type alignedEditionLine struct {
	LineNumber    *int    `json:"line_number"`
	Content       string  `json:"content"`
	ContentType   *string `json:"content_type"`
	CharacterName *string `json:"character_name"`
}

type alignedSceneRow struct {
	Editions map[int]*alignedEditionLine `json:"editions"`
}

type multiEditionScene struct {
	WorkTitle         string            `json:"work_title"`
	Act               int               `json:"act"`
	Scene             int               `json:"scene"`
	AvailableEditions []editionInfo     `json:"available_editions"`
	Rows              []alignedSceneRow `json:"rows"`
}

func (s *Server) handleScene(w http.ResponseWriter, r *http.Request) {
	workId, ok := s.resolveWorkID(r, "workId")
	if !ok {
		writeError(w, http.StatusNotFound, "work not found")
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

	result, err := s.getMultiEditionScene(workId, act, scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if result == nil {
		writeError(w, http.StatusNotFound, "scene not found")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getMultiEditionScene(workId, act, scene int) (*multiEditionScene, error) {
	var workTitle, workType string
	err := s.db.QueryRow(`SELECT title, work_type FROM works WHERE id = ?`, workId).Scan(&workTitle, &workType)
	if err != nil {
		return nil, nil
	}

	// Find available editions for this scene
	edRows, err := s.db.Query(`SELECT DISTINCT e.id, e.short_code, e.name
		FROM editions e
		JOIN text_lines tl ON tl.edition_id = e.id
		WHERE tl.work_id = ? AND tl.act = ? AND tl.scene = ?
		  AND e.id IN (1,2,3,4,5)
		ORDER BY e.id`, workId, act, scene)
	if err != nil {
		return nil, fmt.Errorf("editions: %w", err)
	}
	defer edRows.Close()

	var availEditions []editionInfo
	var editionIDs []int
	for edRows.Next() {
		var e editionInfo
		if err := edRows.Scan(&e.ID, &e.Code, &e.Name); err != nil {
			continue
		}
		availEditions = append(availEditions, e)
		editionIDs = append(editionIDs, e.ID)
	}
	if len(availEditions) == 0 {
		return nil, nil
	}

	// Load all lines for all editions
	edPlaceholders := makePlaceholders(len(editionIDs))
	edArgs := intsToArgs(editionIDs)

	charCoalesce := `COALESCE(
		c.name,
		(SELECT c2.name FROM characters c2
		 WHERE c2.work_id = tl.work_id
		   AND LOWER(c2.name) LIKE LOWER(REPLACE(REPLACE(REPLACE(tl.char_name, '.', ''), 'æ', 'ae'), 'Æ', 'Ae')) || '%'
		 LIMIT 1),
		tl.char_name
	)`

	queryArgs := append([]any{workId}, edArgs...)
	queryArgs = append(queryArgs, act, scene)

	lineRows, err := s.db.Query(fmt.Sprintf(`SELECT tl.id, tl.edition_id, tl.line_number, tl.content, tl.content_type,
		%s AS character_name
		FROM text_lines tl
		LEFT JOIN characters c ON c.id = tl.character_id
		WHERE tl.work_id = ? AND tl.edition_id IN (%s)
		  AND tl.act = ? AND tl.scene = ?
		  AND tl.id = (
		    SELECT MIN(t2.id) FROM text_lines t2
		    WHERE t2.work_id = tl.work_id AND t2.edition_id = tl.edition_id
		      AND COALESCE(t2.act, 0) = COALESCE(tl.act, 0)
		      AND COALESCE(t2.scene, 0) = COALESCE(tl.scene, 0)
		      AND t2.line_number = tl.line_number
		  )
		ORDER BY tl.edition_id, tl.line_number, tl.id`, charCoalesce, edPlaceholders), queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("lines: %w", err)
	}
	defer lineRows.Close()

	type lineRow struct {
		id            int
		editionID     int
		lineNumber    *int
		content       string
		contentType   *string
		characterName *string
	}

	linesByEdition := map[int][]lineRow{}
	lineByID := map[int]lineRow{}
	for lineRows.Next() {
		var lr lineRow
		if err := lineRows.Scan(&lr.id, &lr.editionID, &lr.lineNumber, &lr.content, &lr.contentType, &lr.characterName); err != nil {
			continue
		}
		linesByEdition[lr.editionID] = append(linesByEdition[lr.editionID], lr)
		lineByID[lr.id] = lr
	}

	// Use edition 1 (OSS) as anchor, or first available
	anchorID := editionIDs[0]
	for _, eid := range editionIDs {
		if eid == 1 {
			anchorID = 1
			break
		}
	}
	anchorLines := linesByEdition[anchorID]

	if len(availEditions) <= 1 {
		rows := make([]alignedSceneRow, len(anchorLines))
		for i, l := range anchorLines {
			rows[i] = alignedSceneRow{
				Editions: map[int]*alignedEditionLine{
					anchorID: {LineNumber: l.lineNumber, Content: l.content, ContentType: l.contentType, CharacterName: l.characterName},
				},
			}
		}
		return &multiEditionScene{WorkTitle: workTitle, Act: act, Scene: scene, AvailableEditions: availEditions, Rows: rows}, nil
	}

	// Load line_mappings between anchor and other editions
	otherIDs := make([]int, 0, len(editionIDs)-1)
	for _, eid := range editionIDs {
		if eid != anchorID {
			otherIDs = append(otherIDs, eid)
		}
	}
	otherPlaceholders := makePlaceholders(len(otherIDs))
	otherArgs := intsToArgs(otherIDs)

	mappingArgs := append([]any{workId, act, scene, anchorID}, otherArgs...)
	mappingArgs = append(mappingArgs, anchorID)
	mappingArgs = append(mappingArgs, otherArgs...)

	mappingRows, err := s.db.Query(fmt.Sprintf(`SELECT lm.edition_a_id, lm.edition_b_id, lm.line_a_id, lm.line_b_id
		FROM line_mappings lm
		WHERE lm.work_id = ? AND lm.act = ? AND lm.scene = ?
		  AND (
		    (lm.edition_a_id = ? AND lm.edition_b_id IN (%s))
		    OR (lm.edition_b_id = ? AND lm.edition_a_id IN (%s))
		  )
		ORDER BY lm.edition_b_id, lm.align_order`, otherPlaceholders, otherPlaceholders), mappingArgs...)
	if err != nil {
		return nil, fmt.Errorf("mappings: %w", err)
	}
	defer mappingRows.Close()

	// anchor line_id → map[otherEditionID]otherLineID
	anchorToOther := map[int]map[int]int{}
	var otherOnlyLineIDs []struct {
		edID   int
		lineID int
	}

	for mappingRows.Next() {
		var edA, edB int
		var lineA, lineB *int
		if err := mappingRows.Scan(&edA, &edB, &lineA, &lineB); err != nil {
			continue
		}

		var anchorLineID, otherLineID *int
		var otherEdID int
		if edA == anchorID {
			anchorLineID = lineA
			otherLineID = lineB
			otherEdID = edB
		} else {
			anchorLineID = lineB
			otherLineID = lineA
			otherEdID = edA
		}

		if anchorLineID != nil && otherLineID != nil {
			if anchorToOther[*anchorLineID] == nil {
				anchorToOther[*anchorLineID] = map[int]int{}
			}
			anchorToOther[*anchorLineID][otherEdID] = *otherLineID
		} else if anchorLineID == nil && otherLineID != nil {
			otherOnlyLineIDs = append(otherOnlyLineIDs, struct {
				edID   int
				lineID int
			}{otherEdID, *otherLineID})
		}
	}

	// Build aligned rows
	usedLines := map[int]bool{}
	rows := make([]alignedSceneRow, 0, len(anchorLines))
	for _, al := range anchorLines {
		row := alignedSceneRow{
			Editions: map[int]*alignedEditionLine{
				anchorID: {LineNumber: al.lineNumber, Content: al.content, ContentType: al.contentType, CharacterName: al.characterName},
			},
		}
		if edMap, ok := anchorToOther[al.id]; ok {
			for otherEdID, otherLineID := range edMap {
				if ol, ok := lineByID[otherLineID]; ok {
					row.Editions[otherEdID] = &alignedEditionLine{
						LineNumber: ol.lineNumber, Content: ol.content, ContentType: ol.contentType, CharacterName: ol.characterName,
					}
					usedLines[otherLineID] = true
				}
			}
		}
		rows = append(rows, row)
	}

	// Append lines only in non-anchor editions
	for _, ol := range otherOnlyLineIDs {
		if usedLines[ol.lineID] {
			continue
		}
		if l, ok := lineByID[ol.lineID]; ok {
			rows = append(rows, alignedSceneRow{
				Editions: map[int]*alignedEditionLine{
					ol.edID: {LineNumber: l.lineNumber, Content: l.content, ContentType: l.contentType, CharacterName: l.characterName},
				},
			})
		}
	}

	return &multiEditionScene{WorkTitle: workTitle, Act: act, Scene: scene, AvailableEditions: availEditions, Rows: rows}, nil
}
