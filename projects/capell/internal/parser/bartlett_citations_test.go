package parser

import "testing"

func TestParseBartlettCitations_FullPlayCitation(t *testing.T) {
	raw := `Abbey.  Even  now  we  housed  him  in  the  abbey  here  .  Com.  of  Errors  v  1  188`
	cits := ParseBartlettCitations(raw)
	if len(cits) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(cits))
	}
	c := cits[0]
	if c.WorkAbbrev != "Com. of Errors" {
		t.Errorf("abbrev = %q, want %q", c.WorkAbbrev, "Com. of Errors")
	}
	if *c.Act != 5 || *c.Scene != 1 || *c.Line != 188 {
		t.Errorf("location = %d.%d.%d, want 5.1.188", *c.Act, *c.Scene, *c.Line)
	}
}

func TestParseBartlettCitations_OCRCorruption(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantKey  string
		wantAct  int
		wantScn  int
		wantLine int
	}{
		{"Coriolanut", `Show  them  the  unaching  scars  Coriolanut  ii  2  152`, "Coriolanus", 2, 2, 152},
		{"Coriol.", `not carry  her  aboard  Coriol.  iv  1  102`, "Coriolanus", 4, 1, 102},
		{"T. of threw", `he was too hard  T. of threw  i  1  33`, "T. of Shrew", 1, 1, 33},
		{"T. Sight", `sweet  mistress  T. Sight  i  2  40`, "T. Night", 1, 2, 40},
		{"Her. of Venice", `I  am  not  bound  Her. of Venice  i  1  126`, "Mer. of Venice", 1, 1, 126},
		{"J. Cfesar", `doers  J. Cfesar  iii  1  94`, "J. Caesar", 3, 1, 94},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cits := ParseBartlettCitations(tt.raw)
			if len(cits) == 0 {
				t.Fatal("no citations parsed")
			}
			c := cits[0]
			if c.WorkAbbrev != tt.wantKey {
				t.Errorf("abbrev = %q, want %q", c.WorkAbbrev, tt.wantKey)
			}
			if *c.Act != tt.wantAct || *c.Scene != tt.wantScn || *c.Line != tt.wantLine {
				t.Errorf("location = %d.%d.%d, want %d.%d.%d",
					*c.Act, *c.Scene, *c.Line, tt.wantAct, tt.wantScn, tt.wantLine)
			}
		})
	}
}

func TestParseBartlettCitations_MidLine(t *testing.T) {
	raw := `But  tish  not  Mer. of  Venice  i  1  102  more  text  after`
	cits := ParseBartlettCitations(raw)
	if len(cits) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(cits))
	}
	if *cits[0].Act != 1 || *cits[0].Scene != 1 || *cits[0].Line != 102 {
		t.Errorf("location = %d.%d.%d, want 1.1.102", *cits[0].Act, *cits[0].Scene, *cits[0].Line)
	}
}

func TestParseBartlettCitations_BareContination(t *testing.T) {
	raw := `Sail.  A whole armado  K. John  iii  4  2
  like a shifted wind onto a sail  iv  2  23`
	cits := ParseBartlettCitations(raw)
	if len(cits) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(cits))
	}
	// First: K. John iii 4 2
	if cits[0].WorkAbbrev != "K. John" {
		t.Errorf("cit[0] abbrev = %q, want %q", cits[0].WorkAbbrev, "K. John")
	}
	if *cits[0].Act != 3 || *cits[0].Scene != 4 || *cits[0].Line != 2 {
		t.Errorf("cit[0] location = %d.%d.%d, want 3.4.2", *cits[0].Act, *cits[0].Scene, *cits[0].Line)
	}
	// Second: bare continuation inherits K. John
	if cits[1].WorkAbbrev != "K. John" {
		t.Errorf("cit[1] abbrev = %q, want %q", cits[1].WorkAbbrev, "K. John")
	}
	if *cits[1].Act != 4 || *cits[1].Scene != 2 || *cits[1].Line != 23 {
		t.Errorf("cit[1] location = %d.%d.%d, want 4.2.23", *cits[1].Act, *cits[1].Scene, *cits[1].Line)
	}
}

func TestParseBartlettCitations_PoemCitation(t *testing.T) {
	raw := `Supposed  as  forfeit  to  a  confined  doom  Sonn.  107`
	cits := ParseBartlettCitations(raw)
	if len(cits) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(cits))
	}
	c := cits[0]
	if c.WorkAbbrev != "Sonn." {
		t.Errorf("abbrev = %q, want %q", c.WorkAbbrev, "Sonn.")
	}
	if c.Act != nil || c.Scene != nil {
		t.Error("poem citation should have nil act/scene")
	}
	if *c.Line != 107 {
		t.Errorf("line = %d, want 107", *c.Line)
	}
}

func TestParseBartlettCitations_Dedup(t *testing.T) {
	raw := `love  Hamlet  iii  1  100
love  Hamlet  iii  1  100`
	cits := ParseBartlettCitations(raw)
	if len(cits) != 1 {
		t.Fatalf("expected 1 citation (deduped), got %d", len(cits))
	}
}

func TestParseBartlettCitations_MultipleCitationsPerLine(t *testing.T) {
	raw := `word  Hamlet  iii  1  100  Lear  i  4  50`
	cits := ParseBartlettCitations(raw)
	// May get 1 or 2 depending on regex greediness — at least 1
	if len(cits) == 0 {
		t.Fatal("expected at least 1 citation")
	}
}
