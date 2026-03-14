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

func TestParsePerseusRef_ValidTwoPart(t *testing.T) {
	ref := ParsePerseusRef("shak. son 1.14")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.SchmidtAbbrev != "Sonn." {
		t.Errorf("expected 'Sonn.', got %q", ref.SchmidtAbbrev)
	}
	if ref.Act == nil || *ref.Act != 1 {
		t.Errorf("expected act 1, got %v", ref.Act)
	}
	if ref.Line == nil || *ref.Line != 14 {
		t.Errorf("expected line 14, got %v", ref.Line)
	}
	if ref.Scene != nil {
		t.Errorf("expected nil scene, got %v", ref.Scene)
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
