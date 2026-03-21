// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strings"
	"testing"
)

func intPtr(v int) *int { return &v }

func TestParsePerseusRef_ValidThreePart(t *testing.T) {
	ref := ParsePerseusRef("shak. ham 3.1.56")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "Hml." {
		t.Errorf("expected 'Hml.', got %q", ref.SchmidtAbbrev)
	}
	if ref.Act == nil || *ref.Act != 3 {
		t.Errorf("expected act 3, got %v", ref.Act)
	}
	if ref.Scene == nil || *ref.Scene != 1 {
		t.Errorf("expected scene 1, got %v", ref.Scene)
	}
	if ref.Line == nil || *ref.Line != 56 {
		t.Errorf("expected line 56, got %v", ref.Line)
	}
}

func TestParsePerseusRef_TwoPartSonnet(t *testing.T) {
	// Sonnets: two-part = sonnet_number.line
	ref := ParsePerseusRef("shak. son 1.14")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "Sonn." {
		t.Errorf("expected 'Sonn.', got %q", ref.SchmidtAbbrev)
	}
	// For sonnets, first number is scene (sonnet number), second is line
	if ref.Scene == nil || *ref.Scene != 1 {
		t.Errorf("expected scene 1, got %v", ref.Scene)
	}
	if ref.Line == nil || *ref.Line != 14 {
		t.Errorf("expected line 14, got %v", ref.Line)
	}
	if ref.Act != nil {
		t.Errorf("expected nil act for sonnet, got %v", ref.Act)
	}
}

func TestParsePerseusRef_TwoPartPlay_LowScene(t *testing.T) {
	// "5.1" → act=5, scene=1 (scene number is plausible)
	ref := ParsePerseusRef("shak. ayl 5.1")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "As" {
		t.Errorf("expected 'As', got %q", ref.SchmidtAbbrev)
	}
	if ref.Act == nil || *ref.Act != 5 {
		t.Errorf("expected act 5, got %v", ref.Act)
	}
	if ref.Scene == nil || *ref.Scene != 1 {
		t.Errorf("expected scene 1, got %v", ref.Scene)
	}
	if ref.Line != nil {
		t.Errorf("expected nil line for two-part play ref with low scene, got %v", ref.Line)
	}
}

func TestParsePerseusRef_TwoPartPlay_HighNumber(t *testing.T) {
	// "4.60" → act=4, line=60 (60 is too high to be a scene number)
	ref := ParsePerseusRef("shak. tmp 4.60")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Act == nil || *ref.Act != 4 {
		t.Errorf("expected act 4, got %v", ref.Act)
	}
	if ref.Scene != nil {
		t.Errorf("expected nil scene (60 is a line, not a scene), got %v", *ref.Scene)
	}
	if ref.Line == nil || *ref.Line != 60 {
		t.Errorf("expected line 60, got %v", ref.Line)
	}
}

func TestParsePerseusRef_TwoPartPoem(t *testing.T) {
	// Poems: two-part = section.line (section stored in Scene, matching DB structure)
	ref := ParsePerseusRef("shak. ven 1.123")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "Ven." {
		t.Errorf("expected 'Ven.', got %q", ref.SchmidtAbbrev)
	}
	if ref.Scene == nil || *ref.Scene != 1 {
		t.Errorf("expected scene 1 (section/poem number), got %v", ref.Scene)
	}
	if ref.Line == nil || *ref.Line != 123 {
		t.Errorf("expected line 123, got %v", ref.Line)
	}
	if ref.Act != nil {
		t.Errorf("expected act nil for poem, got %v", ref.Act)
	}
}

func TestParsePerseusRef_InvalidPrefix(t *testing.T) {
	ref := ParsePerseusRef("not-shakespeare")
	if ref != nil {
		t.Errorf("expected nil for non-Shakespeare ref, got %+v", ref)
	}
}

func TestParsePerseusRef_Empty(t *testing.T) {
	ref := ParsePerseusRef("")
	if ref != nil {
		t.Errorf("expected nil for empty string, got %+v", ref)
	}
}

func TestParsePerseusRef_ThreePartSonnet(t *testing.T) {
	// "shak. son 19.104.5" → ignore volume (19), scene=104, line=5
	ref := ParsePerseusRef("shak. son 19.104.5")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Scene == nil || *ref.Scene != 104 {
		t.Errorf("expected scene 104 (sonnet number), got %v", ref.Scene)
	}
	if ref.Line == nil || *ref.Line != 5 {
		t.Errorf("expected line 5, got %v", ref.Line)
	}
	if ref.Act != nil {
		t.Errorf("expected act nil for sonnet, got %v", ref.Act)
	}
}

func TestParsePerseusRef_ThreePartPoem(t *testing.T) {
	// "shak. luc 2.3.64" → ignore stanza parts, line=64
	ref := ParsePerseusRef("shak. luc 2.3.64")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Line == nil || *ref.Line != 64 {
		t.Errorf("expected line 64, got %v", ref.Line)
	}
	if ref.Act != nil {
		t.Errorf("expected act nil for poem 3-part, got %v", ref.Act)
	}
}

func TestParsePerseusRef_DuplicatedWorkCode(t *testing.T) {
	// "shak. ven ven" → work resolved, no location
	ref := ParsePerseusRef("shak. ven ven")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "Ven." {
		t.Errorf("expected 'Ven.', got %q", ref.SchmidtAbbrev)
	}
	if ref.Line != nil {
		t.Errorf("expected nil line for duplicated code, got %v", ref.Line)
	}
}

func TestParsePerseusRef_WorkOnly(t *testing.T) {
	// "shak. luc" → work resolved, no location
	ref := ParsePerseusRef("shak. luc")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "Lucr." {
		t.Errorf("expected 'Lucr.', got %q", ref.SchmidtAbbrev)
	}
}

func TestParsePerseusRef_PhoenixSingleLine(t *testing.T) {
	// "shak. pht 21" → line=21 (single-part poem ref)
	ref := ParsePerseusRef("shak. pht 21")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "Phoen." {
		t.Errorf("expected 'Phoen.', got %q", ref.SchmidtAbbrev)
	}
	if ref.Line == nil || *ref.Line != 21 {
		t.Errorf("expected line 21, got %v", ref.Line)
	}
}

func TestParsePerseusRef_PassionatePilgrim(t *testing.T) {
	// "shak. pp 2.21" → scene=2 (poem number), line=21
	ref := ParsePerseusRef("shak. pp 2.21")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Scene == nil || *ref.Scene != 2 {
		t.Errorf("expected scene 2 (poem number), got %v", ref.Scene)
	}
	if ref.Line == nil || *ref.Line != 21 {
		t.Errorf("expected line 21, got %v", ref.Line)
	}
}

func TestParsePerseusRef_UnknownWork(t *testing.T) {
	ref := ParsePerseusRef("shak. xyz 1.2.3")
	if ref != nil {
		t.Errorf("expected nil for unknown work code, got %+v", ref)
	}
}

func TestParseSenses_SingleSense(t *testing.T) {
	senses := ParseSenses("A simple definition without numbered senses.")
	if len(senses) != 1 {
		t.Fatalf("expected 1 sense, got %d", len(senses))
	}
	if senses[0].Number != 1 {
		t.Errorf("expected sense number 1, got %d", senses[0].Number)
	}
}

func TestParseSenses_MultipleSenses(t *testing.T) {
	text := "Headword. 1) First definition. 2) Second definition. 3) Third one."
	senses := ParseSenses(text)
	if len(senses) != 3 {
		t.Fatalf("expected 3 senses, got %d", len(senses))
	}
	if senses[0].Number != 1 {
		t.Errorf("sense 0: expected number 1, got %d", senses[0].Number)
	}
	if senses[1].Number != 2 {
		t.Errorf("sense 1: expected number 2, got %d", senses[1].Number)
	}
	if senses[2].Number != 3 {
		t.Errorf("sense 2: expected number 3, got %d", senses[2].Number)
	}
}

func TestParseSenses_IgnoresNumbersInsideParens(t *testing.T) {
	// "4)" inside "(cf. def. 4)" should NOT be treated as a sense boundary
	text := "1) reckoning: (== store). 2) computation: (cf. def. 4). 3) estimation: something. 4) explanation:"
	senses := ParseSenses(text)
	if len(senses) != 4 {
		t.Fatalf("expected 4 senses, got %d: %+v", len(senses), senses)
	}
	for i, s := range senses {
		if s.Number != i+1 {
			t.Errorf("sense %d: expected number %d, got %d (text: %q)", i, i+1, s.Number, s.Text)
		}
	}
	// Sense 2 should contain "(cf. def. 4)" as part of its text, not split on it
	if !strings.Contains(senses[1].Text, "cf. def. 4") {
		t.Errorf("sense 2 should contain 'cf. def. 4', got: %q", senses[1].Text)
	}
}

func TestParseSenses_IgnoresNonSequentialNumbers(t *testing.T) {
	// Random "4)" without preceding 3) should be ignored
	text := "1) first definition. 2) second (see 4) for more). 3) third."
	senses := ParseSenses(text)
	if len(senses) != 3 {
		t.Fatalf("expected 3 senses, got %d: %+v", len(senses), senses)
	}
}

func TestParseSenses_SubSensesInsideParens(t *testing.T) {
	// Letters inside parens should not be treated as sub-senses
	text := "1) definition (see a) for details). a) first sub. b) second sub."
	senses := ParseSenses(text)
	// Should have: preamble "definition (see a) for details).", sub a), sub b)
	found := 0
	for _, s := range senses {
		if s.SubSense != "" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("expected 2 sub-senses, got %d: %+v", found, senses)
	}
}

func TestParseEntryXML_BasicEntry(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2>
  <text>
    <body>
      <div1 n="A" type="alphabetic letter">
        <entryFree key="Abandon" type="main">
          <orth>Abandon</orth>,
          <sense>to give up, desert.
            <cit>
              <quote>Abandon the society of this female</quote>
              <bibl n="shak. lll 1.1.106">LLL I, 1, 106</bibl>
            </cit>
          </sense>
        </entryFree>
      </div1>
    </body>
  </text>
</TEI.2>`

	entry, err := ParseEntryXML([]byte(xml), "abandon.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}

	if entry.Key != "Abandon" {
		t.Errorf("key: expected 'Abandon', got %q", entry.Key)
	}
	if entry.Letter != "A" {
		t.Errorf("letter: expected 'A', got %q", entry.Letter)
	}
	if entry.Orthography != "Abandon" {
		t.Errorf("orthography: expected 'Abandon', got %q", entry.Orthography)
	}
	if entry.EntryType != "main" {
		t.Errorf("entry_type: expected 'main', got %q", entry.EntryType)
	}
	if len(entry.Citations) == 0 {
		t.Fatal("expected at least one citation")
	}
	if entry.Citations[0].WorkAbbrev != "LLL" {
		t.Errorf("citation work: expected 'LLL', got %q", entry.Citations[0].WorkAbbrev)
	}
}

func TestParseEntryXML_SenseAssignment(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2><text><body><div1 n="T" type="alphabetic letter">
<entryFree key="Test" type="main"><orth>Test</orth>, 1) first meaning: <cit><quote>first quote</quote> <bibl n="shak. ham 1.1.1">Hml. I, 1, 1</bibl></cit>. 2) second meaning: <cit><quote>second quote</quote> <bibl n="shak. oth 2.3.4">Oth. II, 3, 4</bibl></cit>.
</entryFree></div1></body></text></TEI.2>`

	entry, err := ParseEntryXML([]byte(xml), "test.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}

	if len(entry.Senses) != 2 {
		t.Fatalf("expected 2 senses, got %d", len(entry.Senses))
	}
	if len(entry.Citations) < 2 {
		t.Fatalf("expected at least 2 citations, got %d", len(entry.Citations))
	}

	// First citation should be assigned to sense 1
	if entry.Citations[0].SenseNumber != 1 {
		t.Errorf("citation 0: expected sense 1, got %d", entry.Citations[0].SenseNumber)
	}
	// Second citation should be assigned to sense 2
	if entry.Citations[1].SenseNumber != 2 {
		t.Errorf("citation 1: expected sense 2, got %d", entry.Citations[1].SenseNumber)
	}
}

func TestParseEntryXML_NilForMissingEntryFree(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2><text><body><div1 n="X"></div1></body></text></TEI.2>`

	entry, err := ParseEntryXML([]byte(xml), "empty.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Error("expected nil entry for XML without entryFree")
	}
}

func TestSupplementFromDisplayText_PlayMissingLine(t *testing.T) {
	// Perseus ref "shak. 2h6 1.2" gives act=1, scene=2, no line
	// Display text "H6B I, 2, 15" has the line number
	ref := ParsePerseusRef("shak. 2h6 1.2")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Line != nil {
		t.Fatalf("expected nil line before supplement, got %d", *ref.Line)
	}

	supplementFromDisplayText(ref, "H6B I, 2, 15", "")
	if ref.Line == nil || *ref.Line != 15 {
		t.Errorf("expected line 15 after supplement, got %v", ref.Line)
	}
	// Act and scene should remain unchanged
	if ref.Act == nil || *ref.Act != 1 {
		t.Errorf("expected act 1, got %v", ref.Act)
	}
	if ref.Scene == nil || *ref.Scene != 2 {
		t.Errorf("expected scene 2, got %v", ref.Scene)
	}
}

func TestSupplementFromDisplayText_TwoPartPlayIsActLine(t *testing.T) {
	// Perseus ref "shak. tmp 4.56" gives act=4, scene=56, no line
	// Display text "Tp. IV, 56" has only 2 numbers → act=4, line=56 (not scene)
	ref := ParsePerseusRef("shak. tmp 4.56")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}

	supplementFromDisplayText(ref, "Tp. IV, 56", "")
	if ref.Act == nil || *ref.Act != 4 {
		t.Errorf("expected act 4, got %v", ref.Act)
	}
	if ref.Scene != nil {
		t.Errorf("expected nil scene (was line, not scene), got %v", *ref.Scene)
	}
	if ref.Line == nil || *ref.Line != 56 {
		t.Errorf("expected line 56, got %v", ref.Line)
	}
}

func TestSupplementFromDisplayText_AlreadyComplete(t *testing.T) {
	// Complete ref should not be modified
	ref := ParsePerseusRef("shak. ham 3.1.56")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	origLine := *ref.Line
	supplementFromDisplayText(ref, "Hml. III, 1, 56", "")
	if *ref.Line != origLine {
		t.Errorf("expected line unchanged at %d, got %d", origLine, *ref.Line)
	}
}

func TestSupplementFromDisplayText_NilRef(t *testing.T) {
	// Should not panic on nil
	supplementFromDisplayText(nil, "H6B I, 2, 15", "")
}

func TestParseEntryXML_SupplementsIncompleteRef(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2><text><body><div1 n="A" type="alphabetic letter">
<entryFree key="Abase" type="main"><orth>Abase,</orth> to lower: <cit><quote>a. our sight so low,</quote> <bibl n="shak. 2h6 1.2">H6B I, 2, 15</bibl></cit>.
</entryFree></div1></body></text></TEI.2>`

	entry, err := ParseEntryXML([]byte(xml), "abase.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if len(entry.Citations) == 0 {
		t.Fatal("expected at least one citation")
	}
	c := entry.Citations[0]
	if c.Act == nil || *c.Act != 1 {
		t.Errorf("expected act 1, got %v", c.Act)
	}
	if c.Scene == nil || *c.Scene != 2 {
		t.Errorf("expected scene 2, got %v", c.Scene)
	}
	if c.Line == nil || *c.Line != 15 {
		t.Errorf("expected line 15 (supplemented from display text), got %v", c.Line)
	}
}

func TestParseEntryXML_DefinitionStripsReferences(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI.2><text><body><div1 n="A" type="alphabetic letter">
<entryFree key="A-front" type="main"><orth>A-front,</orth> in front, directly opposed: <bibl n="shak. 1h4 2.4">H4A II, 4, 222</bibl>.
</entryFree></div1></body></text></TEI.2>`

	entry, err := ParseEntryXML([]byte(xml), "a-front.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if len(entry.Senses) == 0 {
		t.Fatal("expected at least one sense")
	}
	// Definition should NOT contain "H4A II, 4, 222"
	def := entry.Senses[0].Text
	if strings.Contains(def, "H4A") {
		t.Errorf("definition should not contain bibl text, got: %q", def)
	}
	// But should contain the actual definition
	if !strings.Contains(def, "in front, directly opposed") {
		t.Errorf("definition should contain definition text, got: %q", def)
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello   world", "hello world"},
		{"  leading", "leading"},
		{"trailing  ", "trailing"},
		{"multiple\n\nnewlines\n", "multiple newlines"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeWhitespace(tt.input)
		if got != tt.want {
			t.Errorf("normalizeWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
