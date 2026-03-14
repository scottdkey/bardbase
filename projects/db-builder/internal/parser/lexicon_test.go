package parser

import (
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

func TestParsePerseusRef_TwoPartPlay(t *testing.T) {
	// Plays: two-part = act.scene (no line number)
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
		t.Errorf("expected nil line for two-part play ref, got %v", ref.Line)
	}
}

func TestParsePerseusRef_TwoPartPoem(t *testing.T) {
	// Poems: two-part = section.line
	ref := ParsePerseusRef("shak. ven 1.123")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "Ven." {
		t.Errorf("expected 'Ven.', got %q", ref.SchmidtAbbrev)
	}
	if ref.Act == nil || *ref.Act != 1 {
		t.Errorf("expected act 1 (section), got %v", ref.Act)
	}
	if ref.Line == nil || *ref.Line != 123 {
		t.Errorf("expected line 123, got %v", ref.Line)
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
