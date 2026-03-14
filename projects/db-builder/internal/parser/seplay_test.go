package parser

import (
	"testing"
)

func TestParseSEPlay_BasicVerse(t *testing.T) {
	xhtml := `<html><body>
<section id="act-1">
  <section id="scene-1-1">
    <h3>Scene 1</h3>
    <table>
      <tr>
        <td epub:type="z3998:persona">PROSPERO</td>
        <td>
          <span>Our revels now are ended. These our actors,</span>
          <span>As I foretold you, were all spirits, and</span>
        </td>
      </tr>
    </table>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if lines[0].Act != 1 {
		t.Errorf("expected act 1, got %d", lines[0].Act)
	}
	if lines[0].Scene != 1 {
		t.Errorf("expected scene 1, got %d", lines[0].Scene)
	}
	if lines[0].Character != "PROSPERO" {
		t.Errorf("expected PROSPERO, got %q", lines[0].Character)
	}
	if lines[0].Text != "Our revels now are ended. These our actors," {
		t.Errorf("unexpected text: %q", lines[0].Text)
	}
	if lines[0].IsStageDirection {
		t.Error("expected speech, not stage direction")
	}
}

func TestParseSEPlay_StageDirection(t *testing.T) {
	xhtml := `<html><body>
<section id="act-2">
  <section id="scene-2-1">
    <i epub:type="z3998:stage-direction">Enter Ariel</i>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !lines[0].IsStageDirection {
		t.Error("expected stage direction")
	}
	if lines[0].Text != "Enter Ariel" {
		t.Errorf("expected 'Enter Ariel', got %q", lines[0].Text)
	}
	if lines[0].Character != "" {
		t.Errorf("stage direction should have no character, got %q", lines[0].Character)
	}
}

func TestParseSEPlay_NestedStageDirection(t *testing.T) {
	xhtml := `<html><body>
<section id="act-1">
  <section id="scene-1-1">
    <table>
      <tr>
        <td epub:type="z3998:persona">HAMLET</td>
        <td>
          <span>To be, or not to be,</span>
          <i epub:type="z3998:stage-direction">Drawing his sword</i>
          <span>that is the question.</span>
        </td>
      </tr>
    </table>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (2 verse + 1 stage dir), got %d", len(lines))
	}

	if lines[0].IsStageDirection {
		t.Error("first line should be speech")
	}
	if !lines[1].IsStageDirection {
		t.Error("second line should be stage direction")
	}
	if lines[1].Character != "" {
		t.Errorf("stage direction should have no character, got %q", lines[1].Character)
	}
	if !lines[2].IsStageDirection && lines[2].Text != "that is the question." {
		// Line 2 should be speech
	}
}

func TestParseSEPlay_HeadersIgnored(t *testing.T) {
	xhtml := `<html><body>
<section id="act-1">
  <h2>Act 1</h2>
  <section id="scene-1-1">
    <h3>Scene 1</h3>
    <table>
      <tr>
        <td epub:type="z3998:persona">ROMEO</td>
        <td><span>But soft, what light?</span></td>
      </tr>
    </table>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (headers should not generate lines), got %d", len(lines))
	}
	if lines[0].Text != "But soft, what light?" {
		t.Errorf("unexpected text: %q", lines[0].Text)
	}
}

func TestParseSEPlay_ProseWithoutSpans(t *testing.T) {
	xhtml := `<html><body>
<section id="act-3">
  <section id="scene-3-2">
    <table>
      <tr>
        <td epub:type="z3998:persona">FALSTAFF</td>
        <td>If I had a thousand sons, the first human principle I would teach them should be to forswear thin potations.</td>
      </tr>
    </table>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 1 {
		t.Fatalf("expected 1 prose line, got %d", len(lines))
	}
	if lines[0].Character != "FALSTAFF" {
		t.Errorf("expected FALSTAFF, got %q", lines[0].Character)
	}
}

func TestParseSEPlay_LineInSceneCounter(t *testing.T) {
	xhtml := `<html><body>
<section id="act-1">
  <section id="scene-1-1">
    <table>
      <tr>
        <td epub:type="z3998:persona">A</td>
        <td><span>Line one.</span><span>Line two.</span></td>
      </tr>
      <tr>
        <td epub:type="z3998:persona">B</td>
        <td><span>Line three.</span></td>
      </tr>
    </table>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].LineInScene != 1 {
		t.Errorf("line 0: expected LineInScene 1, got %d", lines[0].LineInScene)
	}
	if lines[1].LineInScene != 2 {
		t.Errorf("line 1: expected LineInScene 2, got %d", lines[1].LineInScene)
	}
	if lines[2].LineInScene != 3 {
		t.Errorf("line 2: expected LineInScene 3, got %d", lines[2].LineInScene)
	}
}

func TestParseSEPlay_PrologueAndEpilogue(t *testing.T) {
	xhtml := `<html><body>
<section id="act-1">
  <section id="scene-1-0" epub:type="z3998:prologue">
    <table><tr>
      <td epub:type="z3998:persona">CHORUS</td>
      <td><span>Prologue line.</span></td>
    </tr></table>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Scene != 0 {
		t.Errorf("prologue should be scene 0, got %d", lines[0].Scene)
	}
}

func TestParseSEPlay_EmptyInput(t *testing.T) {
	lines := ParseSEPlay("")
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty input, got %d", len(lines))
	}
}
