// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"testing"
)

func TestPlayCitation_ExactQuoteMatch(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "The Tempest", "Tp.", "comedy")

	// Text lines: Act 1, Scene 1, lines 1-5 in both editions
	lines := []struct {
		content    string
		lineNum    int
		act, scene int
	}{
		{"Boatswain!", 1, 1, 1},
		{"Here, master: what cheer?", 2, 1, 1},
		{"Good, speak to the mariners.", 3, 1, 1},
		{"Fall to't, yarely, or we run ourselves aground.", 4, 1, 1},
		{"Heigh, my hearts! cheerly, cheerly, my hearts!", 5, 1, 1},
	}

	for _, ed := range []int64{td.EdOSSID, td.EdSEID} {
		for _, l := range lines {
			td.insertTextLine(t, workID, ed, l.act, l.scene, l.lineNum, "", l.content)
		}
	}

	// Lexicon entry + sense + citation with quote text that matches line 4
	entryID := td.insertLexiconEntry(t, "Aground", "A")
	senseID := td.insertSense(t, entryID, 1, "stranded")

	td.insertCitation(t, entryID, &senseID, workID, intPtr(1), intPtr(1), intPtr(4), "we run ourselves aground")

	err := ResolveCitations(td.DB)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	// Should have 2 matches (one per edition), both exact_quote with confidence 1.0
	var count int
	td.DB.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 matches (one per edition), got %d", count)
	}

	rows, _ := td.DB.Query(`SELECT edition_id, match_type, confidence, matched_text FROM citation_matches`)
	defer rows.Close()
	for rows.Next() {
		var edID int64
		var matchType, matchedText string
		var confidence float64
		rows.Scan(&edID, &matchType, &confidence, &matchedText)

		if matchType != "exact_quote" {
			t.Errorf("edition %d: expected match_type 'exact_quote', got '%s'", edID, matchType)
		}
		if confidence != 1.0 {
			t.Errorf("edition %d: expected confidence 1.0, got %f", edID, confidence)
		}
		if matchedText != "Fall to't, yarely, or we run ourselves aground." {
			t.Errorf("edition %d: unexpected matched_text: %s", edID, matchedText)
		}
	}
}

func TestPlayCitation_LineNumberMatch(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "The Tempest", "Tp.", "comedy")

	// Text lines with no matching quote — forces line_number strategy
	td.insertTextLine(t, workID, td.EdOSSID, 2, 1, 10, "", "Abhorred slave, which any print of goodness will not take.")
	td.insertTextLine(t, workID, td.EdOSSID, 2, 1, 11, "", "Being capable of all ill!")

	// Citation with line number but NO quote text
	entryID := td.insertLexiconEntry(t, "Abhorred", "A")
	td.insertCitation(t, entryID, nil, workID, intPtr(2), intPtr(1), intPtr(10), "")

	err := ResolveCitations(td.DB)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var matchType string
	var confidence float64
	var matchedText string
	err = td.DB.QueryRow(`SELECT match_type, confidence, matched_text FROM citation_matches`).Scan(&matchType, &confidence, &matchedText)
	if err != nil {
		t.Fatalf("no match found: %v", err)
	}

	if matchType != "line_number" {
		t.Errorf("expected 'line_number', got '%s'", matchType)
	}
	if confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", confidence)
	}
	if matchedText != "Abhorred slave, which any print of goodness will not take." {
		t.Errorf("unexpected matched_text: %s", matchedText)
	}
}

func TestPlayCitation_FuzzyMatch(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "The Tempest", "Tp.", "comedy")

	// Lines in act 3 scene 2 — none with matching line number
	td.insertTextLine(t, workID, td.EdOSSID, 3, 2, 40, "", "Monster, I do smell all horse-piss")
	td.insertTextLine(t, workID, td.EdOSSID, 3, 2, 41, "", "at which my nose is in great indignation.")
	td.insertTextLine(t, workID, td.EdOSSID, 3, 2, 42, "", "So is mine. Do you hear, monster?")

	// Citation references line 99 (doesn't exist) but has quote text matching line 40
	entryID := td.insertLexiconEntry(t, "Horse-piss", "H")
	td.insertCitation(t, entryID, nil, workID, intPtr(3), intPtr(2), intPtr(99), "I do smell all horse-piss")

	err := ResolveCitations(td.DB)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var matchType string
	var confidence float64
	err = td.DB.QueryRow(`SELECT match_type, confidence FROM citation_matches`).Scan(&matchType, &confidence)
	if err != nil {
		t.Fatalf("no match found: %v", err)
	}

	if matchType != "exact_quote" {
		t.Errorf("expected 'exact_quote' (substring of line content), got '%s'", matchType)
	}
}

func TestPlayCitation_MultipleEditionsGetMatches(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "Hamlet", "Hml.", "tragedy")

	// Same line in both editions (slightly different text)
	td.insertTextLine(t, workID, td.EdOSSID, 3, 1, 56, "", "To be, or not to be, that is the question:")
	td.insertTextLine(t, workID, td.EdSEID, 3, 1, 56, "", "To be, or not to be: that is the question:")

	entryID := td.insertLexiconEntry(t, "Question", "Q")
	td.insertCitation(t, entryID, nil, workID, intPtr(3), intPtr(1), intPtr(56), "that is the question")

	err := ResolveCitations(td.DB)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var count int
	td.DB.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 matches (one per edition), got %d", count)
	}

	var editions []int64
	rows, _ := td.DB.Query("SELECT DISTINCT edition_id FROM citation_matches ORDER BY edition_id")
	defer rows.Close()
	for rows.Next() {
		var eid int64
		rows.Scan(&eid)
		editions = append(editions, eid)
	}

	if len(editions) != 2 {
		t.Errorf("expected matches in 2 editions, got %d", len(editions))
	}
	if len(editions) == 2 && (editions[0] != td.EdOSSID || editions[1] != td.EdSEID) {
		t.Errorf("expected editions [%d, %d], got %v", td.EdOSSID, td.EdSEID, editions)
	}
}

func TestPlayCitation_NoMatchReturnsNothing(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "The Tempest", "Tp.", "comedy")

	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 1, "", "Boatswain!")

	// Citation references act 5 scene 3 (doesn't exist in our data)
	entryID := td.insertLexiconEntry(t, "Nonexistent", "N")
	td.insertCitation(t, entryID, nil, workID, intPtr(5), intPtr(3), intPtr(10), "totally fake quote")

	err := ResolveCitations(td.DB)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var count int
	td.DB.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 matches for nonexistent scene, got %d", count)
	}
}

func TestPlayCitation_ActOnlyNoScene(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "The Tempest", "Tp.", "comedy")

	td.insertTextLine(t, workID, td.EdOSSID, 4, 1, 5, "", "Our revels now are ended.")
	td.insertTextLine(t, workID, td.EdOSSID, 4, 1, 6, "", "These our actors, as I foretold you, were all spirits.")

	// Citation with act=4 but scene=NULL — should search whole act
	entryID := td.insertLexiconEntry(t, "Revels", "R")
	td.insertCitation(t, entryID, nil, workID, intPtr(4), nil, intPtr(5), "Our revels now are ended")

	err := ResolveCitations(td.DB)
	if err != nil {
		t.Fatalf("ResolveCitations: %v", err)
	}

	var matchType string
	var confidence float64
	err = td.DB.QueryRow("SELECT match_type, confidence FROM citation_matches").Scan(&matchType, &confidence)
	if err != nil {
		t.Fatalf("no match: %v", err)
	}

	if matchType != "exact_quote" {
		t.Errorf("expected 'exact_quote', got '%s'", matchType)
	}
	if confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", confidence)
	}
}

func TestPlayCitation_Idempotent(t *testing.T) {
	td := newTestDB(t)
	workID := td.insertWork(t, "The Tempest", "Tp.", "comedy")

	td.insertTextLine(t, workID, td.EdOSSID, 1, 1, 1, "", "Boatswain!")

	entryID := td.insertLexiconEntry(t, "Boatswain", "B")
	td.insertCitation(t, entryID, nil, workID, intPtr(1), intPtr(1), intPtr(1), "Boatswain")

	// Run twice
	ResolveCitations(td.DB)
	ResolveCitations(td.DB)

	var count int
	td.DB.QueryRow("SELECT COUNT(*) FROM citation_matches").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 match after two runs (idempotent), got %d", count)
	}
}

