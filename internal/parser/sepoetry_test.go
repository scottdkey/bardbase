package parser

import (
	"testing"
)

func TestParseSEPoetry_BasicPoem(t *testing.T) {
	xhtml := `<html><body>
<article id="venus-and-adonis">
  <header><h2>Venus and Adonis</h2></header>
  <p>
    <span>Even as the sun with purple-colour'd face</span>
    <span>Had ta'en his last leave of the weeping morn,</span>
  </p>
  <p>
    <span>Rose-cheek'd Adonis hied him to the chase;</span>
  </p>
</article>
</body></html>`

	poems := ParseSEPoetry(xhtml)
	lines, ok := poems["venus-and-adonis"]
	if !ok {
		t.Fatal("expected 'venus-and-adonis' article")
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].LineNumber != 1 {
		t.Errorf("line 0: expected LineNumber 1, got %d", lines[0].LineNumber)
	}
	if lines[0].Stanza != 1 {
		t.Errorf("line 0: expected Stanza 1, got %d", lines[0].Stanza)
	}
	if lines[2].Stanza != 2 {
		t.Errorf("line 2: expected Stanza 2, got %d", lines[2].Stanza)
	}
}

func TestParseSEPoetry_HeaderIgnored(t *testing.T) {
	xhtml := `<html><body>
<article id="test-poem">
  <header><h2>Title Text</h2></header>
  <p><span>Actual verse.</span></p>
</article>
</body></html>`

	poems := ParseSEPoetry(xhtml)
	lines := poems["test-poem"]
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "Actual verse." {
		t.Errorf("expected 'Actual verse.', got %q", lines[0].Text)
	}
}

func TestParseSEPoetry_DedicationIgnored(t *testing.T) {
	xhtml := `<html><body>
<article id="test-poem">
  <section id="dedication">
    <p><span>To the Earl of Something</span></p>
  </section>
  <p><span>Real verse line.</span></p>
</article>
</body></html>`

	poems := ParseSEPoetry(xhtml)
	lines := poems["test-poem"]
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (dedication excluded), got %d", len(lines))
	}
}

func TestParseSEPoetry_EmptyInput(t *testing.T) {
	poems := ParseSEPoetry("")
	if len(poems) != 0 {
		t.Errorf("expected empty map for empty input, got %d entries", len(poems))
	}
}

func TestParseSESonnets_BasicSonnet(t *testing.T) {
	xhtml := `<html><body>
<article id="sonnet-18">
  <header><h2>XVIII</h2></header>
  <p>
    <span>Shall I compare thee to a summer's day?</span>
    <span>Thou art more lovely and more temperate.</span>
  </p>
</article>
</body></html>`

	data := ParseSESonnets(xhtml)
	if data == nil {
		t.Fatal("expected non-nil data")
	}
	lines, ok := data.Sonnets[18]
	if !ok {
		t.Fatal("expected sonnet 18")
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].Text != "Shall I compare thee to a summer's day?" {
		t.Errorf("unexpected text: %q", lines[0].Text)
	}
}

func TestParseSESonnets_MultipleSonnets(t *testing.T) {
	xhtml := `<html><body>
<article id="sonnet-1">
  <p><span>From fairest creatures we desire increase,</span></p>
</article>
<article id="sonnet-2">
  <p><span>When forty winters shall besiege thy brow,</span></p>
</article>
</body></html>`

	data := ParseSESonnets(xhtml)
	if len(data.Sonnets) != 2 {
		t.Fatalf("expected 2 sonnets, got %d", len(data.Sonnets))
	}
	if _, ok := data.Sonnets[1]; !ok {
		t.Error("expected sonnet 1")
	}
	if _, ok := data.Sonnets[2]; !ok {
		t.Error("expected sonnet 2")
	}
}

func TestParseSESonnets_LoversComplaint(t *testing.T) {
	xhtml := `<html><body>
<article id="a-lovers-complaint">
  <p><span>From off a hill whose concave womb reworded</span></p>
  <p><span>A plaintful story from a sistering vale,</span></p>
</article>
</body></html>`

	data := ParseSESonnets(xhtml)
	if len(data.LoversComplaint) != 2 {
		t.Fatalf("expected 2 lines in Lover's Complaint, got %d", len(data.LoversComplaint))
	}
}

func TestParseSESonnets_EmptyInput(t *testing.T) {
	data := ParseSESonnets("")
	if data == nil {
		t.Fatal("expected non-nil data")
	}
	if len(data.Sonnets) != 0 {
		t.Errorf("expected 0 sonnets, got %d", len(data.Sonnets))
	}
}
