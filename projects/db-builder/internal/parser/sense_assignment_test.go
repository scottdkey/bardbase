package parser

import (
	"testing"
)

func TestAssignSenses_SingleSenseAllCitations(t *testing.T) {
	entry := &LexiconEntry{
		FullText: "Abandon, to give up, desert.",
		Senses:   []Sense{{Number: 1, Text: "to give up, desert."}},
		Citations: []Citation{
			{DisplayText: "Abandon", RawBibl: "LLL I, 1, 106"},
			{DisplayText: "desert", RawBibl: "Tp. I, 2, 3"},
		},
	}
	assignSensesToCitations(entry)

	for i, c := range entry.Citations {
		if c.SenseNumber != 1 {
			t.Errorf("citation %d: expected sense 1, got %d", i, c.SenseNumber)
		}
	}
}

func TestAssignSenses_MultipleSenses(t *testing.T) {
	entry := &LexiconEntry{
		FullText: "Abandon, 1) to give up: Hml. I, 1, 1. 2) to desert: Oth. II, 3, 4.",
		Senses: []Sense{
			{Number: 1, Text: "to give up: Hml. I, 1, 1."},
			{Number: 2, Text: "to desert: Oth. II, 3, 4."},
		},
		Citations: []Citation{
			{DisplayText: "Hml. I, 1, 1", RawBibl: "Hml. I, 1, 1"},
			{DisplayText: "Oth. II, 3, 4", RawBibl: "Oth. II, 3, 4"},
		},
	}
	assignSensesToCitations(entry)

	if entry.Citations[0].SenseNumber != 1 {
		t.Errorf("citation 0: expected sense 1, got %d", entry.Citations[0].SenseNumber)
	}
	if entry.Citations[1].SenseNumber != 2 {
		t.Errorf("citation 1: expected sense 2, got %d", entry.Citations[1].SenseNumber)
	}
}

func TestAssignSenses_CitationBeforeFirstSenseBoundary(t *testing.T) {
	// Citation text appears before the "1)" marker
	entry := &LexiconEntry{
		FullText: "Abandon, (see also Leave) 1) to give up: Hml. I, 1, 1. 2) to desert.",
		Senses: []Sense{
			{Number: 1, Text: "to give up: Hml. I, 1, 1."},
			{Number: 2, Text: "to desert."},
		},
		Citations: []Citation{
			{DisplayText: "see also Leave", RawBibl: "see also Leave"},
			{DisplayText: "Hml. I, 1, 1", RawBibl: "Hml. I, 1, 1"},
		},
	}
	assignSensesToCitations(entry)

	// Citation before first sense boundary should get sense 1 (first sense)
	if entry.Citations[0].SenseNumber != 1 {
		t.Errorf("citation before boundary: expected sense 1, got %d", entry.Citations[0].SenseNumber)
	}
	if entry.Citations[1].SenseNumber != 1 {
		t.Errorf("citation in sense 1: expected sense 1, got %d", entry.Citations[1].SenseNumber)
	}
}

func TestAssignSenses_EmptyDisplayText(t *testing.T) {
	// Citation with empty display text should default to first sense
	entry := &LexiconEntry{
		FullText: "Test, 1) first. 2) second.",
		Senses: []Sense{
			{Number: 1, Text: "first."},
			{Number: 2, Text: "second."},
		},
		Citations: []Citation{
			{DisplayText: "", RawBibl: ""},
		},
	}
	assignSensesToCitations(entry)

	if entry.Citations[0].SenseNumber != 1 {
		t.Errorf("empty display text: expected sense 1, got %d", entry.Citations[0].SenseNumber)
	}
}

func TestAssignSenses_DisplayTextNotFoundInFullText(t *testing.T) {
	// Citation display text doesn't appear in full text
	entry := &LexiconEntry{
		FullText: "Test, 1) first. 2) second.",
		Senses: []Sense{
			{Number: 1, Text: "first."},
			{Number: 2, Text: "second."},
		},
		Citations: []Citation{
			{DisplayText: "this text is nowhere in the entry", RawBibl: "xyz"},
		},
	}
	assignSensesToCitations(entry)

	// Should default to first sense
	if entry.Citations[0].SenseNumber != 1 {
		t.Errorf("unmatched text: expected sense 1, got %d", entry.Citations[0].SenseNumber)
	}
}

func TestAssignSenses_NoCitations(t *testing.T) {
	entry := &LexiconEntry{
		FullText:  "Test, 1) first. 2) second.",
		Senses:    []Sense{{Number: 1, Text: "first."}, {Number: 2, Text: "second."}},
		Citations: nil,
	}
	// Should not panic
	assignSensesToCitations(entry)
}

func TestAssignSenses_NoSenses(t *testing.T) {
	entry := &LexiconEntry{
		FullText: "Simple definition with no numbered senses.",
		Senses:   nil,
		Citations: []Citation{
			{DisplayText: "definition", RawBibl: "Hml. I, 1, 1"},
		},
	}
	assignSensesToCitations(entry)

	// No senses = len <= 1 path, all get sense 1
	if entry.Citations[0].SenseNumber != 1 {
		t.Errorf("no senses: expected sense 1, got %d", entry.Citations[0].SenseNumber)
	}
}

func TestAssignSenses_ThreeSensesMultipleCitations(t *testing.T) {
	entry := &LexiconEntry{
		FullText: "Word, 1) meaning A: ref_a1 ref_a2. 2) meaning B: ref_b1. 3) meaning C: ref_c1 ref_c2 ref_c3.",
		Senses: []Sense{
			{Number: 1, Text: "meaning A: ref_a1 ref_a2."},
			{Number: 2, Text: "meaning B: ref_b1."},
			{Number: 3, Text: "meaning C: ref_c1 ref_c2 ref_c3."},
		},
		Citations: []Citation{
			{DisplayText: "ref_a1", RawBibl: "ref_a1"},
			{DisplayText: "ref_a2", RawBibl: "ref_a2"},
			{DisplayText: "ref_b1", RawBibl: "ref_b1"},
			{DisplayText: "ref_c1", RawBibl: "ref_c1"},
			{DisplayText: "ref_c2", RawBibl: "ref_c2"},
			{DisplayText: "ref_c3", RawBibl: "ref_c3"},
		},
	}
	assignSensesToCitations(entry)

	expected := []int{1, 1, 2, 3, 3, 3}
	for i, want := range expected {
		if entry.Citations[i].SenseNumber != want {
			t.Errorf("citation %d: expected sense %d, got %d", i, want, entry.Citations[i].SenseNumber)
		}
	}
}

func TestAssignSenses_FallbackToRawBibl(t *testing.T) {
	// When DisplayText is empty, should try RawBibl
	entry := &LexiconEntry{
		FullText: "Word, 1) first: raw_ref_here. 2) second.",
		Senses: []Sense{
			{Number: 1, Text: "first: raw_ref_here."},
			{Number: 2, Text: "second."},
		},
		Citations: []Citation{
			{DisplayText: "", RawBibl: "raw_ref_here"},
		},
	}
	assignSensesToCitations(entry)

	if entry.Citations[0].SenseNumber != 1 {
		t.Errorf("rawbibl fallback: expected sense 1, got %d", entry.Citations[0].SenseNumber)
	}
}
