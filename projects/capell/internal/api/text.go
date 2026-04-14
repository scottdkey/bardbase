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

type characterInfo struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	SpeechCount int     `json:"speech_count"`
}

type multiEditionScene struct {
	WorkTitle         string            `json:"work_title"`
	Act               int               `json:"act"`
	Scene             int               `json:"scene"`
	AvailableEditions []editionInfo     `json:"available_editions"`
	Characters        []characterInfo   `json:"characters"`
	Rows              []alignedSceneRow `json:"rows"`
}

type lineRow struct {
	id            int
	editionID     int
	lineNumber    *int
	content       string
	contentType   *string
	characterName *string
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
		ORDER BY tl.edition_id, tl.line_number, tl.id`, charCoalesce, edPlaceholders), queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("lines: %w", err)
	}
	defer lineRows.Close()

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
		characters := s.loadCharacters(workId)
		return &multiEditionScene{WorkTitle: workTitle, Act: act, Scene: scene, AvailableEditions: availEditions, Characters: characters, Rows: rows}, nil
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

	// Gap lines: lines in non-anchor editions with no anchor counterpart.
	// Keyed by the anchor line ID they should appear after (0 = before all anchor lines).
	type gapEntry struct {
		edID   int
		lineID int
	}
	gapsAfterAnchor := map[int][]gapEntry{}
	lastAnchorByEdition := map[int]int{} // otherEdID → last anchor line ID seen in this alignment

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
			lastAnchorByEdition[otherEdID] = *anchorLineID
		} else if anchorLineID == nil && otherLineID != nil {
			afterID := lastAnchorByEdition[otherEdID]
			gapsAfterAnchor[afterID] = append(gapsAfterAnchor[afterID], gapEntry{otherEdID, *otherLineID})
		} else if anchorLineID != nil {
			lastAnchorByEdition[otherEdID] = *anchorLineID
		}
	}

	// Build aligned rows, interleaving gap lines at correct positions
	usedLines := map[int]bool{}
	rows := make([]alignedSceneRow, 0, len(anchorLines))

	// Gap lines that precede all anchor lines
	for _, g := range gapsAfterAnchor[0] {
		if l, ok := lineByID[g.lineID]; ok {
			rows = append(rows, alignedSceneRow{
				Editions: map[int]*alignedEditionLine{
					g.edID: {LineNumber: l.lineNumber, Content: l.content, ContentType: l.contentType, CharacterName: l.characterName},
				},
			})
			usedLines[g.lineID] = true
		}
	}

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

		// Insert gap lines that belong after this anchor line
		for _, g := range gapsAfterAnchor[al.id] {
			if usedLines[g.lineID] {
				continue
			}
			if l, ok := lineByID[g.lineID]; ok {
				rows = append(rows, alignedSceneRow{
					Editions: map[int]*alignedEditionLine{
						g.edID: {LineNumber: l.lineNumber, Content: l.content, ContentType: l.contentType, CharacterName: l.characterName},
					},
				})
				usedLines[g.lineID] = true
			}
		}
	}

	// Check for editions with work-level mappings (act=0, scene=0) that aren't
	// already present. This handles structurally divergent editions like the First
	// Folio which stores all lines in act-level scenes. The work-level alignment
	// maps anchor lines to Folio lines regardless of scene structure.
	rows, availEditions = s.mergeWorkLevelEditions(workId, anchorID, anchorLines, rows, availEditions, charCoalesce)

	characters := s.loadCharacters(workId)
	return &multiEditionScene{WorkTitle: workTitle, Act: act, Scene: scene, AvailableEditions: availEditions, Characters: characters, Rows: rows}, nil
}

func (s *Server) mergeWorkLevelEditions(
	workId, anchorID int,
	anchorLines []lineRow,
	rows []alignedSceneRow,
	availEditions []editionInfo,
	charCoalesce string,
) ([]alignedSceneRow, []editionInfo) {
	// Find editions that have work-level mappings but aren't already in availEditions.
	editionsWithContent := map[int]bool{}
	for _, row := range rows {
		for edID := range row.Editions {
			if edID != anchorID {
				editionsWithContent[edID] = true
			}
		}
	}

	wlRows, err := s.db.Query(`SELECT DISTINCT
		CASE WHEN lm.edition_a_id = ? THEN lm.edition_b_id ELSE lm.edition_a_id END AS other_ed_id,
		e.short_code, e.name
		FROM line_mappings lm
		JOIN editions e ON e.id = CASE WHEN lm.edition_a_id = ? THEN lm.edition_b_id ELSE lm.edition_a_id END
		WHERE lm.work_id = ? AND lm.act = 0 AND lm.scene = 0
		  AND (lm.edition_a_id = ? OR lm.edition_b_id = ?)
		  AND e.id NOT IN (0, 10, 11)`,
		anchorID, anchorID, workId, anchorID, anchorID)
	if err != nil {
		return rows, availEditions
	}
	defer wlRows.Close()

	var wlEditions []editionInfo
	for wlRows.Next() {
		var e editionInfo
		if err := wlRows.Scan(&e.ID, &e.Code, &e.Name); err != nil {
			continue
		}
		if !editionsWithContent[e.ID] {
			wlEditions = append(wlEditions, e)
		}
	}

	if len(wlEditions) == 0 {
		return rows, availEditions
	}

	// Build anchor line ID → row index map
	anchorIDToRow := map[int]int{}
	for _, al := range anchorLines {
		for i, row := range rows {
			if ed, ok := row.Editions[anchorID]; ok && ed.Content == al.content {
				anchorIDToRow[al.id] = i
				break
			}
		}
	}

	for _, wlEd := range wlEditions {
		// Load work-level mappings between anchor and this edition
		mapRows, err := s.db.Query(`SELECT lm.line_a_id, lm.line_b_id
			FROM line_mappings lm
			WHERE lm.work_id = ? AND lm.act = 0 AND lm.scene = 0
			  AND ((lm.edition_a_id = ? AND lm.edition_b_id = ?)
			    OR (lm.edition_a_id = ? AND lm.edition_b_id = ?))`,
			workId, anchorID, wlEd.ID, wlEd.ID, anchorID)
		if err != nil {
			continue
		}

		// Collect the other-edition line IDs that map to our anchor lines
		otherLineIDs := map[int]bool{}
		anchorToOther := map[int]int{}
		for mapRows.Next() {
			var lineA, lineB *int
			if err := mapRows.Scan(&lineA, &lineB); err != nil {
				continue
			}
			var anchorLID, otherLID *int
			if lineA != nil {
				if _, ok := anchorIDToRow[*lineA]; ok {
					anchorLID = lineA
					otherLID = lineB
				} else {
					anchorLID = lineB
					otherLID = lineA
				}
			} else {
				anchorLID = lineB
				otherLID = lineA
			}
			if anchorLID != nil && otherLID != nil {
				anchorToOther[*anchorLID] = *otherLID
				otherLineIDs[*otherLID] = true
			}
		}
		mapRows.Close()

		if len(otherLineIDs) == 0 {
			continue
		}

		// Load the actual text lines for this edition
		otherIDs := make([]int, 0, len(otherLineIDs))
		for id := range otherLineIDs {
			otherIDs = append(otherIDs, id)
		}
		olPlaceholders := makePlaceholders(len(otherIDs))
		olArgs := intsToArgs(otherIDs)

		olRows, err := s.db.Query(fmt.Sprintf(`SELECT tl.id, tl.line_number, tl.content, tl.content_type,
			%s AS character_name
			FROM text_lines tl
			LEFT JOIN characters c ON c.id = tl.character_id
			WHERE tl.id IN (%s)`, charCoalesce, olPlaceholders), olArgs...)
		if err != nil {
			continue
		}
		otherLines := map[int]lineRow{}
		for olRows.Next() {
			var lr lineRow
			lr.editionID = wlEd.ID
			if err := olRows.Scan(&lr.id, &lr.lineNumber, &lr.content, &lr.contentType, &lr.characterName); err != nil {
				continue
			}
			otherLines[lr.id] = lr
		}
		olRows.Close()

		// Merge into existing rows
		merged := false
		for _, al := range anchorLines {
			otherLID, ok := anchorToOther[al.id]
			if !ok {
				continue
			}
			ol, ok := otherLines[otherLID]
			if !ok {
				continue
			}
			rowIdx, ok := anchorIDToRow[al.id]
			if !ok {
				continue
			}
			rows[rowIdx].Editions[wlEd.ID] = &alignedEditionLine{
				LineNumber: ol.lineNumber, Content: ol.content,
				ContentType: ol.contentType, CharacterName: ol.characterName,
			}
			merged = true
		}

		if merged {
			// Only append if not already in availEditions
			alreadyPresent := false
			for _, e := range availEditions {
				if e.ID == wlEd.ID {
					alreadyPresent = true
					break
				}
			}
			if !alreadyPresent {
				availEditions = append(availEditions, wlEd)
			}
		}

	}

	return rows, availEditions
}

func (s *Server) loadCharacters(workId int) []characterInfo {
	rows, err := s.db.Query(`SELECT name, description, COALESCE(speech_count, 0)
		FROM characters WHERE work_id = ? ORDER BY name`, workId)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var chars []characterInfo
	for rows.Next() {
		var c characterInfo
		if err := rows.Scan(&c.Name, &c.Description, &c.SpeechCount); err != nil {
			continue
		}
		chars = append(chars, c)
	}
	return chars
}
