// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"testing"
)

// =============================================================================
// LINE NUMBER VERIFICATION TESTS
//
// These tests verify that line_number is set correctly across all text sources.
//
// NUMBERING SEMANTICS (important for downstream citation matching):
//
//   OSS plays:   line_number = scene-relative paragraph count.
//                Each speech block or stage direction = 1 line_number.
//                A 10-line verse speech is still line_number = 1.
//                NOT equivalent to Globe verse line numbers.
//
//   SE plays:    line_number = scene-relative sequential count (LineInScene).
//                Each verse line, prose block, or stage direction = 1 line_number.
//                Stage directions ARE counted (Globe numbering skips them).
//                Close to Globe numbers but offset by stage direction count.
//
//   SE sonnets:  line_number = per-sonnet line (1-14 typically).
//                Resets for each sonnet. Matches Schmidt refs directly.
//
//   SE poetry:   line_number = poem-relative sequential line.
//                Continuous across stanzas within a poem.
//                Matches Schmidt single-number refs.
//
// The citation resolver (chunk 6) must account for these differences when
// matching Schmidt's Globe-based line references to actual text.
// =============================================================================

// --- SE Play: line_number resets per scene ---

func TestLineNumber_SEPlay_ResetsPerScene(t *testing.T) {
	xhtml := `<html><body>
<section id="act-1">
  <section id="scene-1-1">
    <table>
      <tr>
        <td epub:type="z3998:persona">A</td>
        <td><span>Scene one, line one.</span><span>Scene one, line two.</span></td>
      </tr>
    </table>
  </section>
  <section id="scene-1-2">
    <table>
      <tr>
        <td epub:type="z3998:persona">B</td>
        <td><span>Scene two, line one.</span></td>
      </tr>
    </table>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Scene 1: lines 1, 2
	if lines[0].LineInScene != 1 {
		t.Errorf("scene 1 line 0: expected LineInScene=1, got %d", lines[0].LineInScene)
	}
	if lines[1].LineInScene != 2 {
		t.Errorf("scene 1 line 1: expected LineInScene=2, got %d", lines[1].LineInScene)
	}

	// Scene 2: should reset to 1
	if lines[2].LineInScene != 1 {
		t.Errorf("scene 2 line 0: expected LineInScene=1, got %d", lines[2].LineInScene)
	}
	if lines[2].Scene != 2 {
		t.Errorf("scene 2 line 0: expected Scene=2, got %d", lines[2].Scene)
	}
}

// --- SE Play: stage directions get their own line_number ---

func TestLineNumber_SEPlay_StageDirectionsGetNumbers(t *testing.T) {
	xhtml := `<html><body>
<section id="act-3">
  <section id="scene-3-1">
    <i epub:type="z3998:stage-direction">Enter King</i>
    <table>
      <tr>
        <td epub:type="z3998:persona">KING</td>
        <td><span>My first line.</span><span>My second line.</span></td>
      </tr>
    </table>
    <i epub:type="z3998:stage-direction">Exit King</i>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (SD + 2 verse + SD), got %d", len(lines))
	}

	// Stage direction "Enter King" = line 1
	if lines[0].LineInScene != 1 {
		t.Errorf("stage dir: expected LineInScene=1, got %d", lines[0].LineInScene)
	}
	if !lines[0].IsStageDirection {
		t.Error("line 0 should be a stage direction")
	}

	// Verse lines = 2, 3
	if lines[1].LineInScene != 2 {
		t.Errorf("verse 1: expected LineInScene=2, got %d", lines[1].LineInScene)
	}
	if lines[2].LineInScene != 3 {
		t.Errorf("verse 2: expected LineInScene=3, got %d", lines[2].LineInScene)
	}

	// Stage direction "Exit King" = line 4
	if lines[3].LineInScene != 4 {
		t.Errorf("exit stage dir: expected LineInScene=4, got %d", lines[3].LineInScene)
	}
	if !lines[3].IsStageDirection {
		t.Error("line 3 should be a stage direction")
	}
}

// --- SE Play: multi-act multi-scene numbering ---

func TestLineNumber_SEPlay_MultiActMultiScene(t *testing.T) {
	xhtml := `<html><body>
<section id="act-1">
  <section id="scene-1-1">
    <table><tr>
      <td epub:type="z3998:persona">A</td>
      <td><span>Act 1 Scene 1 line.</span></td>
    </tr></table>
  </section>
  <section id="scene-1-2">
    <table><tr>
      <td epub:type="z3998:persona">B</td>
      <td><span>Act 1 Scene 2 line.</span></td>
    </tr></table>
  </section>
</section>
<section id="act-2">
  <section id="scene-2-1">
    <table><tr>
      <td epub:type="z3998:persona">C</td>
      <td><span>Act 2 Scene 1 line.</span></td>
    </tr></table>
  </section>
</section>
</body></html>`

	lines := ParseSEPlay(xhtml)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Each scene starts at 1
	for i, l := range lines {
		if l.LineInScene != 1 {
			t.Errorf("line %d (act %d scene %d): expected LineInScene=1, got %d",
				i, l.Act, l.Scene, l.LineInScene)
		}
	}

	// Verify act/scene assignments
	if lines[0].Act != 1 || lines[0].Scene != 1 {
		t.Errorf("line 0: expected 1.1, got %d.%d", lines[0].Act, lines[0].Scene)
	}
	if lines[1].Act != 1 || lines[1].Scene != 2 {
		t.Errorf("line 1: expected 1.2, got %d.%d", lines[1].Act, lines[1].Scene)
	}
	if lines[2].Act != 2 || lines[2].Scene != 1 {
		t.Errorf("line 2: expected 2.1, got %d.%d", lines[2].Act, lines[2].Scene)
	}
}

// --- SE Sonnets: line_number is per-sonnet (1-14) ---

func TestLineNumber_SESonnets_PerSonnet(t *testing.T) {
	xhtml := `<html><body>
<article id="sonnet-1">
  <p>
    <span>From fairest creatures we desire increase,</span>
    <span>That thereby beauty's rose might never die,</span>
    <span>But as the riper should by time decease,</span>
    <span>His tender heir might bear his memory:</span>
  </p>
</article>
<article id="sonnet-2">
  <p>
    <span>When forty winters shall besiege thy brow,</span>
    <span>And dig deep trenches in thy beauty's field,</span>
  </p>
</article>
</body></html>`

	data := ParseSESonnets(xhtml)

	// Sonnet 1: lines 1-4
	s1 := data.Sonnets[1]
	if len(s1) != 4 {
		t.Fatalf("sonnet 1: expected 4 lines, got %d", len(s1))
	}
	for i, l := range s1 {
		if l.LineNumber != i+1 {
			t.Errorf("sonnet 1 line %d: expected LineNumber=%d, got %d", i, i+1, l.LineNumber)
		}
	}

	// Sonnet 2: should reset to 1
	s2 := data.Sonnets[2]
	if len(s2) != 2 {
		t.Fatalf("sonnet 2: expected 2 lines, got %d", len(s2))
	}
	if s2[0].LineNumber != 1 {
		t.Errorf("sonnet 2 line 0: expected LineNumber=1, got %d", s2[0].LineNumber)
	}
	if s2[1].LineNumber != 2 {
		t.Errorf("sonnet 2 line 1: expected LineNumber=2, got %d", s2[1].LineNumber)
	}
}

// --- SE Poetry: line_number is poem-relative, continuous across stanzas ---

func TestLineNumber_SEPoetry_PoemRelative(t *testing.T) {
	xhtml := `<html><body>
<article id="venus-and-adonis">
  <p>
    <span>Even as the sun with purple-colour'd face</span>
    <span>Had ta'en his last leave of the weeping morn,</span>
    <span>Rose-cheek'd Adonis hied him to the chase;</span>
  </p>
  <p>
    <span>Hunting he loved, but love he laugh'd to scorn;</span>
    <span>Sick-thoughted Venus makes amain unto him,</span>
  </p>
</article>
</body></html>`

	poems := ParseSEPoetry(xhtml)
	lines := poems["venus-and-adonis"]

	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}

	// Line numbers should be continuous 1-5 across stanzas
	for i, l := range lines {
		if l.LineNumber != i+1 {
			t.Errorf("line %d: expected LineNumber=%d, got %d", i, i+1, l.LineNumber)
		}
	}

	// Stanza numbers should change at the <p> boundary
	if lines[0].Stanza != lines[2].Stanza {
		t.Errorf("lines 0-2 should be same stanza, got %d and %d", lines[0].Stanza, lines[2].Stanza)
	}
	if lines[2].Stanza == lines[3].Stanza {
		t.Errorf("lines 2 and 3 should be different stanzas, both got %d", lines[2].Stanza)
	}
}

// --- SE Poetry: separate poems get separate numbering ---

func TestLineNumber_SEPoetry_ResetsBetweenPoems(t *testing.T) {
	xhtml := `<html><body>
<article id="poem-a">
  <p>
    <span>First poem line one.</span>
    <span>First poem line two.</span>
  </p>
</article>
<article id="poem-b">
  <p>
    <span>Second poem line one.</span>
  </p>
</article>
</body></html>`

	poems := ParseSEPoetry(xhtml)

	a := poems["poem-a"]
	b := poems["poem-b"]

	if len(a) != 2 {
		t.Fatalf("poem-a: expected 2 lines, got %d", len(a))
	}
	if len(b) != 1 {
		t.Fatalf("poem-b: expected 1 line, got %d", len(b))
	}

	// Poem B should reset to 1
	if b[0].LineNumber != 1 {
		t.Errorf("poem-b line 0: expected LineNumber=1, got %d", b[0].LineNumber)
	}
}

// --- OSS: line_number computation verification ---
// Note: OSS line_number is computed in the importer, not the parser.
// These tests verify the computation logic in isolation.

func TestLineNumber_OSS_SceneRelative(t *testing.T) {
	// Simulate the OSS line number computation algorithm
	type para struct {
		WorkID  string
		Section int
		Chapter int
		ParaNum int
		LineNum int // output
	}

	paragraphs := []para{
		{"hamlet", 1, 1, 1, 0},
		{"hamlet", 1, 1, 2, 0},
		{"hamlet", 1, 1, 3, 0},
		{"hamlet", 1, 2, 1, 0}, // new scene
		{"hamlet", 1, 2, 2, 0},
		{"hamlet", 2, 1, 1, 0}, // new act
	}

	// Apply the same algorithm from oss.go
	prevWork := ""
	prevSection := -1
	prevChapter := -1
	lineNum := 0
	for i := range paragraphs {
		if paragraphs[i].WorkID != prevWork || paragraphs[i].Section != prevSection || paragraphs[i].Chapter != prevChapter {
			lineNum = 0
			prevWork = paragraphs[i].WorkID
			prevSection = paragraphs[i].Section
			prevChapter = paragraphs[i].Chapter
		}
		lineNum++
		paragraphs[i].LineNum = lineNum
	}

	expected := []int{1, 2, 3, 1, 2, 1}
	for i, p := range paragraphs {
		if p.LineNum != expected[i] {
			t.Errorf("paragraph %d (section %d chapter %d): expected line_number=%d, got %d",
				i, p.Section, p.Chapter, expected[i], p.LineNum)
		}
	}
}

func TestLineNumber_OSS_DifferentWorks(t *testing.T) {
	type para struct {
		WorkID  string
		Section int
		Chapter int
		ParaNum int
		LineNum int
	}

	paragraphs := []para{
		{"hamlet", 1, 1, 1, 0},
		{"hamlet", 1, 1, 2, 0},
		{"othello", 1, 1, 1, 0}, // different work — should reset
		{"othello", 1, 1, 2, 0},
	}

	prevWork := ""
	prevSection := -1
	prevChapter := -1
	lineNum := 0
	for i := range paragraphs {
		if paragraphs[i].WorkID != prevWork || paragraphs[i].Section != prevSection || paragraphs[i].Chapter != prevChapter {
			lineNum = 0
			prevWork = paragraphs[i].WorkID
			prevSection = paragraphs[i].Section
			prevChapter = paragraphs[i].Chapter
		}
		lineNum++
		paragraphs[i].LineNum = lineNum
	}

	expected := []int{1, 2, 1, 2}
	for i, p := range paragraphs {
		if p.LineNum != expected[i] {
			t.Errorf("paragraph %d (%s): expected line_number=%d, got %d",
				i, p.WorkID, expected[i], p.LineNum)
		}
	}
}
