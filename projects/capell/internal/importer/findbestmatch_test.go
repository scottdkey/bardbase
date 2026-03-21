// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package importer

import (
	"testing"
)

// helper to make citation rows concise
func makeCitation(quoteText string, line *int) citationRow {
	return citationRow{
		ID:        1,
		EntryID:   1,
		WorkID:    1,
		Line:      line,
		QuoteText: quoteText,
		Headword:  "test", // default headword for tests
	}
}

func makeCitationWithHeadword(quoteText string, line *int, headword string) citationRow {
	return citationRow{
		ID:        1,
		EntryID:   1,
		WorkID:    1,
		Line:      line,
		QuoteText: quoteText,
		Headword:  headword,
	}
}


// === Strategy 1: Exact Quote Match ===

func TestFindBestMatch_ExactQuote(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "Now is the winter of our discontent", LineNumber: 1, EditionID: 1},
		{ID: 2, Content: "Made glorious summer by this sun of York", LineNumber: 2, EditionID: 1},
		{ID: 3, Content: "And all the clouds that lourd upon our house", LineNumber: 3, EditionID: 1},
	}
	cit := makeCitation("glorious summer by this sun", nil)

	line, matchType, confidence := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a match, got nil")
	}
	if line.ID != 2 {
		t.Errorf("expected line ID 2, got %d", line.ID)
	}
	if matchType != "exact_quote" {
		t.Errorf("expected exact_quote, got %s", matchType)
	}
	if confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", confidence)
	}
}

func TestFindBestMatch_ExactQuoteCleansDashes(t *testing.T) {
	// Schmidt uses -- for abbreviations like "abandon--ed"
	lines := []textLineRow{
		{ID: 1, Content: "He hath abandoned his physicians", LineNumber: 1, EditionID: 1},
	}
	cit := makeCitation("abandon--ed his physicians", nil)

	line, matchType, _ := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a match after cleaning dashes, got nil")
	}
	if matchType != "exact_quote" {
		t.Errorf("expected exact_quote, got %s", matchType)
	}
}

func TestFindBestMatch_ExactQuoteSkipsShortQuotes(t *testing.T) {
	// Quotes ≤3 chars should be skipped for exact matching (too ambiguous)
	lines := []textLineRow{
		{ID: 1, Content: "To be or not to be", LineNumber: 1, EditionID: 1},
	}
	cit := makeCitation("to", nil) // too short — won't trigger exact_quote

	line, matchType, _ := findBestMatch(lines, cit)

	// Should NOT match via exact_quote. Might match via fuzzy or not at all.
	if line != nil && matchType == "exact_quote" {
		t.Error("expected short quotes to be skipped for exact matching")
	}
}

func TestFindBestMatch_ExactQuoteCaseInsensitive(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "ABANDON the society of this female", LineNumber: 1, EditionID: 1},
	}
	cit := makeCitation("abandon the society", nil)

	line, matchType, _ := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected case-insensitive match")
	}
	if matchType != "exact_quote" {
		t.Errorf("expected exact_quote, got %s", matchType)
	}
}

// === Strategy 2: Line Number Match ===

func TestFindBestMatch_LineNumberExact_NoQuote(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "First line", LineNumber: 1, EditionID: 1},
		{ID: 2, Content: "Second line", LineNumber: 2, EditionID: 1},
		{ID: 3, Content: "Third line", LineNumber: 3, EditionID: 1},
	}
	cit := makeCitationWithHeadword("", intPtr(2), "Second")

	line, matchType, confidence := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a match")
	}
	if line.ID != 2 {
		t.Errorf("expected line ID 2, got %d", line.ID)
	}
	if matchType != "line_number" {
		t.Errorf("expected line_number, got %s", matchType)
	}
	if confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", confidence)
	}
}

func TestFindBestMatch_LineNumberExact_WithConfirmingQuote(t *testing.T) {
	// Quote is contained in content → exact_quote fires first (higher priority).
	// This verifies the priority ordering is correct.
	lines := []textLineRow{
		{ID: 1, Content: "To be or not to be that is the question", LineNumber: 56, EditionID: 1},
	}
	cit := makeCitation("to be or not to be", intPtr(56))

	line, matchType, confidence := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a match")
	}
	// exact_quote has priority over line_number — both would match but exact_quote runs first
	if matchType != "exact_quote" {
		t.Errorf("expected exact_quote (higher priority), got %s", matchType)
	}
	if confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", confidence)
	}
}

func TestFindBestMatch_LineNumberExact_WithMismatchingQuote(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "Whether tis nobler in the mind to suffer", LineNumber: 57, EditionID: 1},
	}
	// Quote doesn't match content, but headword "nobler" IS in the line →
	// line number match accepted.
	cit := makeCitationWithHeadword("apple banana cherry dog elephant", intPtr(57), "nobler")

	line, matchType, confidence := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a match (headword in line)")
	}
	if matchType != "line_number" {
		t.Errorf("expected line_number, got %s", matchType)
	}
	if confidence < 0.5 {
		t.Errorf("expected confidence >= 0.5 for line number match, got %f", confidence)
	}
}

func TestFindBestMatch_LineNumberNearby_WithQuote(t *testing.T) {
	// Line 56 doesn't exist, but line 55 has partially matching text and the headword.
	lines := []textLineRow{
		{ID: 1, Content: "something completely unrelated here today", LineNumber: 54, EditionID: 1},
		{ID: 2, Content: "the nobler mind suffers slings and arrows", LineNumber: 55, EditionID: 1},
		{ID: 3, Content: "another unrelated line of text here today", LineNumber: 58, EditionID: 1},
	}
	// Quote shares words and headword "nobler" is in line 55
	cit := makeCitationWithHeadword("nobler mind suffers outrageous fortune", intPtr(56), "Nobler")

	line, matchType, confidence := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a nearby match")
	}
	if line.ID != 2 {
		t.Errorf("expected line ID 2 (nearby with matching text), got %d", line.ID)
	}
	if matchType != "line_number" {
		t.Errorf("expected line_number, got %s", matchType)
	}
	if confidence < 0.5 {
		t.Errorf("expected confidence >= 0.5 for nearby match with quote, got %f", confidence)
	}
}

func TestFindBestMatch_LineNumberNearby_NoQuote(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "some line", LineNumber: 8, EditionID: 1},
		{ID: 2, Content: "another test line", LineNumber: 12, EditionID: 1},
	}
	// Line 10 doesn't exist. Line 12 contains headword "test" (delta=2).
	cit := makeCitationWithHeadword("", intPtr(10), "test")

	line, matchType, confidence := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a nearby match")
	}
	if line.ID != 2 {
		t.Errorf("expected line ID 2 (contains headword), got %d", line.ID)
	}
	if matchType != "line_number" {
		t.Errorf("expected line_number, got %s", matchType)
	}
	if confidence < 0.5 {
		t.Errorf("expected confidence >= 0.5 for headword-verified nearby match, got %f", confidence)
	}
}

func TestFindBestMatch_LineNumberNearby_OutOfRange_NoHeadword(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "some line", LineNumber: 1, EditionID: 1},
		{ID: 2, Content: "another line", LineNumber: 100, EditionID: 1},
	}
	// Line 50 — no headword in any line, no match should be returned.
	// Bad matches are worse than no match — later phases can handle it.
	cit := makeCitationWithHeadword("", intPtr(50), "assuage")

	line, _, _ := findBestMatch(lines, cit)

	if line != nil {
		t.Errorf("expected no match when headword not found in any line, got line %d", line.ID)
	}
}

// === Strategy 3: Fuzzy Text Match ===

func TestFindBestMatch_FuzzyMatch_AboveThreshold(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "the slings and arrows of outrageous fortune", LineNumber: 1, EditionID: 1},
		{ID: 2, Content: "to take arms against a sea of troubles", LineNumber: 2, EditionID: 1},
	}
	// No line number. Quote shares significant words with line 1.
	cit := makeCitation("slings arrows outrageous fortune", nil)

	line, matchType, confidence := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a fuzzy match")
	}
	if line.ID != 1 {
		t.Errorf("expected line ID 1, got %d", line.ID)
	}
	// word-set containment (1e) may now fire before fuzzy — both are valid outcomes
	if matchType != "fuzzy_text" && matchType != "exact_quote" {
		t.Errorf("expected fuzzy_text or exact_quote, got %s", matchType)
	}
	if confidence <= 0.25 {
		t.Errorf("expected confidence > 0.25, got %f", confidence)
	}
}

func TestFindBestMatch_FuzzyMatch_BelowThreshold(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "the slings and arrows of outrageous fortune", LineNumber: 1, EditionID: 1},
	}
	// Completely different words — Jaccard should be 0 or near 0
	cit := makeCitation("banana helicopter purple dinosaur", nil)

	line, _, _ := findBestMatch(lines, cit)

	if line != nil {
		t.Error("expected no match below fuzzy threshold")
	}
}

func TestFindBestMatch_FuzzyPicksBestCandidate(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "apple banana cherry date elderberry fig", LineNumber: 1, EditionID: 1},
		{ID: 2, Content: "love is not love which alters when it alteration finds", LineNumber: 2, EditionID: 1},
		{ID: 3, Content: "or bends with the remover to remove", LineNumber: 3, EditionID: 1},
	}
	cit := makeCitation("love not love alters alteration finds", nil)

	line, matchType, _ := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a fuzzy match")
	}
	if line.ID != 2 {
		t.Errorf("expected line ID 2 (best fuzzy match), got %d", line.ID)
	}
	// word-set containment (1e) may now fire before fuzzy — both are valid outcomes
	if matchType != "fuzzy_text" && matchType != "exact_quote" {
		t.Errorf("expected fuzzy_text or exact_quote, got %s", matchType)
	}
}

// === Edge Cases ===

func TestFindBestMatch_EmptyLines(t *testing.T) {
	cit := makeCitation("some quote", intPtr(1))
	line, _, _ := findBestMatch(nil, cit)
	if line != nil {
		t.Error("expected nil for empty lines")
	}
}

func TestFindBestMatch_NoQuoteNoLine(t *testing.T) {
	lines := []textLineRow{
		{ID: 1, Content: "some content", LineNumber: 1, EditionID: 1},
	}
	cit := makeCitation("", nil)

	line, _, _ := findBestMatch(lines, cit)

	if line != nil {
		t.Error("expected no match with no quote and no line number")
	}
}

func TestFindBestMatch_ExactQuoteTakesPriority(t *testing.T) {
	// Line 2 matches by line number, but line 3 matches by exact quote
	lines := []textLineRow{
		{ID: 1, Content: "unrelated first line here", LineNumber: 1, EditionID: 1},
		{ID: 2, Content: "wrong content for this line number", LineNumber: 5, EditionID: 1},
		{ID: 3, Content: "to thine own self be true", LineNumber: 78, EditionID: 1},
	}
	cit := makeCitation("thine own self be true", intPtr(5))

	line, matchType, confidence := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a match")
	}
	// Exact quote should win over line number
	if matchType != "exact_quote" {
		t.Errorf("expected exact_quote (priority over line_number), got %s", matchType)
	}
	if line.ID != 3 {
		t.Errorf("expected line ID 3 (exact quote match), got %d", line.ID)
	}
	if confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", confidence)
	}
}

func TestFindBestMatch_LineNumberNearby_PicksClosestWithHeadword(t *testing.T) {
	// Both lines 9 and 11 are delta=1 from target 10.
	// Only line 11 has the headword, so it should be picked.
	lines := []textLineRow{
		{ID: 1, Content: "some text here", LineNumber: 9, EditionID: 1},
		{ID: 2, Content: "other test text here", LineNumber: 11, EditionID: 1},
	}
	cit := makeCitationWithHeadword("", intPtr(10), "test")

	line, _, _ := findBestMatch(lines, cit)

	if line == nil {
		t.Fatal("expected a match")
	}
	if line.ID != 2 {
		t.Errorf("expected line ID 2 (has headword), got %d", line.ID)
	}
}
