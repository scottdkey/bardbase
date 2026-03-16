// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalTEI wraps scene XML in a valid TEI document shell.
func minimalTEI(sceneXML string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<TEI.2><text lang="en"><body>
<div1 type="act" n="1">
` + sceneXML + `
</div1>
</body></text></TEI.2>`
}

func TestParsePerseusTEI_BasicSpeech(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="ham-1"><speaker>Ham.</speaker>
    <p><lb ed="F1" n="1" />To be, or not to be,
    <lb ed="G" /><lb ed="F1" n="2" />that is the question:
    <lb ed="G" /></p>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %+v", len(lines), lines)
	}

	assertPerseusLine(t, lines[0], 1, 1, "Ham.", "ham-1", "To be, or not to be,", false, 1, 0)
	assertPerseusLine(t, lines[1], 1, 1, "Ham.", "ham-1", "that is the question:", false, 2, 0)
}

func TestParsePerseusTEI_VerseLinesElement(t *testing.T) {
	// Tests the <l> verse format used in King John, Richard III, etc.
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="k.-john."><speaker>K. John.</speaker>
    <l>Now, say Chatillon, what would France with us?
    <lb ed="G" /><lb n="6" ed="F1" /></l>
  </sp>
  <sp who="chat."><speaker>Chat.</speaker>
    <l>Thus, after greeting, speaks the King <lb n="7" ed="F1" />of France
    <lb ed="G" /><lb n="8" ed="F1" /></l>
    <l>In my behaviour to the majesty,
    <lb ed="G" /><lb n="9" ed="F1" /></l>
    <l>The borrow'd majesty, of England here.
    <lb n="10" ed="G" /><lb n="10" ed="F1" /></l>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %+v", len(lines), lines)
	}

	assertPerseusLine(t, lines[0], 1, 1, "K. John.", "k.-john.", "Now, say Chatillon, what would France with us?", false, 1, 0)
	assertPerseusLine(t, lines[1], 1, 1, "Chat.", "chat.", "Thus, after greeting, speaks the King of France", false, 2, 0)
	assertPerseusLine(t, lines[2], 1, 1, "Chat.", "chat.", "In my behaviour to the majesty,", false, 3, 0)
	// Fourth line has Globe number 10
	assertPerseusLine(t, lines[3], 1, 1, "Chat.", "chat.", "The borrow'd majesty, of England here.", false, 4, 10)
}

func TestParsePerseusTEI_MixedProseAndVerse(t *testing.T) {
	// Some plays mix <p> and <l> within the same speech
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="a-1"><speaker>A.</speaker>
    <p><lb ed="F1" n="1" />Some prose line here.
    <lb ed="G" /></p>
    <l>Then a verse line follows.
    <lb ed="G" /></l>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %+v", len(lines), lines)
	}

	assertPerseusLine(t, lines[0], 1, 1, "A.", "a-1", "Some prose line here.", false, 1, 0)
	assertPerseusLine(t, lines[1], 1, 1, "A.", "a-1", "Then a verse line follows.", false, 2, 0)
}

func TestParsePerseusTEI_GlobeLineNumbers(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="2">
  <sp who="x-1"><speaker>X.</speaker>
    <p><lb ed="F1" n="1" />First line of scene
    <lb ed="G" n="10" /><lb ed="F1" n="2" />Tenth line explicitly numbered
    <lb ed="G" /></p>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if lines[0].GlobeLine != 0 {
		t.Errorf("line 1 GlobeLine: expected 0, got %d", lines[0].GlobeLine)
	}
	if lines[1].GlobeLine != 10 {
		t.Errorf("line 2 GlobeLine: expected 10, got %d", lines[1].GlobeLine)
	}
}

func TestParsePerseusTEI_StandaloneStageDirections(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <stage type="setting">A dark forest.</stage>
  <stage type="entrance">Enter HAMLET.</stage>
  <sp who="ham-1"><speaker>Ham.</speaker>
    <p><lb ed="F1" n="1" />To be, or not to be.
    <lb ed="G" /></p>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %+v", len(lines), lines)
	}

	assertPerseusLine(t, lines[0], 1, 1, "", "", "A dark forest.", true, 1, 0)
	assertPerseusLine(t, lines[1], 1, 1, "", "", "Enter HAMLET.", true, 2, 0)
	assertPerseusLine(t, lines[2], 1, 1, "Ham.", "ham-1", "To be, or not to be.", false, 3, 0)
}

func TestParsePerseusTEI_InlineStageDirections(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="phi-1"><speaker>Phi.</speaker>
    <p><lb ed="F1" n="1" />To cool a gipsy's lust.
    <lb ed="F1" n="2" /><stage>Flourish.</stage> <stage type="entrance">Enter ANTONY and CLEOPATRA.</stage>
    <lb ed="G" n="10" /><lb ed="F1" n="3" />Look, where they come:
    <lb ed="G" /></p>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %+v", len(lines), lines)
	}

	assertPerseusLine(t, lines[0], 1, 1, "Phi.", "phi-1", "To cool a gipsy's lust.", false, 1, 0)
	assertPerseusLine(t, lines[1], 1, 1, "", "", "Flourish.", true, 2, 0)
	assertPerseusLine(t, lines[2], 1, 1, "", "", "Enter ANTONY and CLEOPATRA.", true, 3, 0)
	assertPerseusLine(t, lines[3], 1, 1, "Phi.", "phi-1", "Look, where they come:", false, 4, 10)
}

func TestParsePerseusTEI_MultipleSpeechesInScene(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="a-1"><speaker>A.</speaker>
    <p><lb ed="F1" n="1" />First speaker talks.
    <lb ed="G" /></p>
  </sp>
  <sp who="b-2"><speaker>B.</speaker>
    <p><lb ed="F1" n="2" />Second speaker responds.
    <lb ed="G" /></p>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	assertPerseusLine(t, lines[0], 1, 1, "A.", "a-1", "First speaker talks.", false, 1, 0)
	assertPerseusLine(t, lines[1], 1, 1, "B.", "b-2", "Second speaker responds.", false, 2, 0)
}

func TestParsePerseusTEI_MultipleActsAndScenes(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<TEI.2><text lang="en"><body>
<div1 type="act" n="1">
  <div2 type="scene" n="1">
    <sp who="a-1"><speaker>A.</speaker>
      <p><lb ed="F1" n="1" />Act one scene one.
      <lb ed="G" /></p>
    </sp>
  </div2>
  <div2 type="scene" n="2">
    <sp who="b-1"><speaker>B.</speaker>
      <p><lb ed="F1" n="1" />Act one scene two.
      <lb ed="G" /></p>
    </sp>
  </div2>
</div1>
<div1 type="act" n="2">
  <div2 type="scene" n="1">
    <sp who="c-1"><speaker>C.</speaker>
      <p><lb ed="F1" n="1" />Act two scene one.
      <lb ed="G" /></p>
    </sp>
  </div2>
</div1>
</body></text></TEI.2>`

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	assertPerseusLine(t, lines[0], 1, 1, "A.", "a-1", "Act one scene one.", false, 1, 0)
	assertPerseusLine(t, lines[1], 1, 2, "B.", "b-1", "Act one scene two.", false, 1, 0)
	assertPerseusLine(t, lines[2], 2, 1, "C.", "c-1", "Act two scene one.", false, 1, 0)

	if lines[1].LineInScene != 1 {
		t.Errorf("scene 2 should restart at line 1, got %d", lines[1].LineInScene)
	}
}

func TestParsePerseusTEI_SkipsCastList(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<TEI.2><text lang="en"><body>
<div1 type="act" n="cast">
  <head>DRAMATIS PERSONÆ</head>
  <castList><castItem type="role"><role id="ham-1">Hamlet</role></castItem></castList>
</div1>
<div1 type="act" n="1">
  <div2 type="scene" n="1">
    <sp who="ham-1"><speaker>Ham.</speaker>
      <p><lb ed="F1" n="1" />The only real line.
      <lb ed="G" /></p>
    </sp>
  </div2>
</div1>
</body></text></TEI.2>`

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 1 {
		t.Fatalf("expected 1 line (cast list skipped), got %d: %+v", len(lines), lines)
	}

	if lines[0].Text != "The only real line." {
		t.Errorf("unexpected text: %q", lines[0].Text)
	}
}

func TestParsePerseusTEI_RegElement(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="cl-1"><speaker>Cl.</speaker>
    <p><lb ed="F1" n="1" />I must catechize you, <reg orig="ma-donna:">madonna:</reg> answer me.
    <lb ed="G" /></p>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	if !strings.Contains(lines[0].Text, "madonna:") {
		t.Errorf("expected 'madonna:' in text, got: %q", lines[0].Text)
	}
	if strings.Contains(lines[0].Text, "ma-donna") {
		t.Errorf("should use regularized text, not orig; got: %q", lines[0].Text)
	}
}

func TestParsePerseusTEI_WhitespaceCollapse(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="a-1"><speaker>A.</speaker>
    <p><lb ed="F1" n="1" />  Multiple   spaces   and
		tabs   here  
    <lb ed="G" /></p>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	expected := "Multiple spaces and tabs here"
	if lines[0].Text != expected {
		t.Errorf("expected %q, got %q", expected, lines[0].Text)
	}
}

func TestParsePerseusTEI_EmptyInput(t *testing.T) {
	lines, err := ParsePerseusTEI([]byte(`<?xml version="1.0"?><TEI.2><text><body></body></text></TEI.2>`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestParsePerseusTEI_StageDirectionInsideSP(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="a-1"><speaker>A.</speaker>
    <p><lb ed="F1" n="1" />I speak.
    <lb ed="G" /></p>
    <stage>Dies.</stage>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %+v", len(lines), lines)
	}

	assertPerseusLine(t, lines[0], 1, 1, "A.", "a-1", "I speak.", false, 1, 0)
	assertPerseusLine(t, lines[1], 1, 1, "", "", "Dies.", true, 2, 0)
}

func TestParsePerseusTEI_MultiLineGlobe(t *testing.T) {
	xml := minimalTEI(`
<div2 type="scene" n="1">
  <sp who="phi-1"><speaker>Phi.</speaker>
    <p><lb n="4" ed="F1" />Nay, but this dotage of our general's
    <lb ed="G" /><lb n="5" ed="F1" />O'erflows the measure: those his goodly eyes,
    <lb ed="G" /><lb n="6" ed="F1" />That o'er the files and musters of the war
    <lb ed="G" /><lb n="7" ed="F1" />Have glow'd like plated Mars, <lb n="8" ed="F1" />now bend, now turn,
    <lb ed="G" /></p>
  </sp>
</div2>`)

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %+v", len(lines), lines)
	}

	assertPerseusLine(t, lines[0], 1, 1, "Phi.", "phi-1", "Nay, but this dotage of our general's", false, 1, 0)
	assertPerseusLine(t, lines[1], 1, 1, "Phi.", "phi-1", "O'erflows the measure: those his goodly eyes,", false, 2, 0)
	assertPerseusLine(t, lines[2], 1, 1, "Phi.", "phi-1", "That o'er the files and musters of the war", false, 3, 0)
	assertPerseusLine(t, lines[3], 1, 1, "Phi.", "phi-1", "Have glow'd like plated Mars, now bend, now turn,", false, 4, 0)
}

func TestParsePerseusTEI_PoetrySkipped(t *testing.T) {
	// Poetry files use div1 type="section" not type="act" — should return 0 lines
	xml := `<?xml version="1.0" encoding="utf-8"?>
<TEI.2><text lang="en"><body>
<div1 n="epigraph" type="section">
  <p>Some epigraph text</p>
</div1>
<div1 n="dedication" type="section">
  <p>Dedication text</p>
</div1>
</body></text></TEI.2>`

	lines, err := ParsePerseusTEI([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for poetry, got %d", len(lines))
	}
}

// TestParsePerseusTEI_RealFile tests against an actual Perseus XML file if available.
func TestParsePerseusTEI_RealFile(t *testing.T) {
	perseusDir := filepath.Join("..", "..", "..", "sources", "perseus-plays")
	antFile := filepath.Join(perseusDir, "1999.03.0025.xml")

	data, err := os.ReadFile(antFile)
	if err != nil {
		t.Skipf("Perseus XML not found (skipping real file test): %v", err)
	}

	lines, err := ParsePerseusTEI(data)
	if err != nil {
		t.Fatalf("failed to parse Antony and Cleopatra: %v", err)
	}

	if len(lines) < 100 {
		t.Errorf("expected at least 100 lines from Antony, got %d", len(lines))
	}

	maxAct := 0
	speeches, stageDirs, globeNumbered := 0, 0, 0
	for _, l := range lines {
		if l.Act > maxAct {
			maxAct = l.Act
		}
		if l.IsStageDirection {
			stageDirs++
		} else {
			speeches++
		}
		if l.GlobeLine > 0 {
			globeNumbered++
		}
	}

	if maxAct != 5 {
		t.Errorf("expected 5 acts, max act was %d", maxAct)
	}

	t.Logf("Antony & Cleopatra: %d total lines (%d speeches, %d stage dirs, %d Globe-numbered)",
		len(lines), speeches, stageDirs, globeNumbered)

	// Verify first speech is by Phi. (Philo)
	for _, l := range lines {
		if !l.IsStageDirection {
			if l.Character != "Phi." {
				t.Errorf("first speaker should be Phi., got %q", l.Character)
			}
			if l.CharID != "ant-33" {
				t.Errorf("first speaker charID should be ant-33, got %q", l.CharID)
			}
			break
		}
	}
}

// TestParsePerseusTEI_RealFileKingJohn tests the verse-only format (King John uses only <l>).
func TestParsePerseusTEI_RealFileKingJohn(t *testing.T) {
	perseusDir := filepath.Join("..", "..", "..", "sources", "perseus-plays")
	kjFile := filepath.Join(perseusDir, "1999.03.0033.xml")

	data, err := os.ReadFile(kjFile)
	if err != nil {
		t.Skipf("Perseus XML not found (skipping real file test): %v", err)
	}

	lines, err := ParsePerseusTEI(data)
	if err != nil {
		t.Fatalf("failed to parse King John: %v", err)
	}

	if len(lines) < 500 {
		t.Errorf("expected at least 500 lines from King John, got %d", len(lines))
	}

	speeches, stageDirs := 0, 0
	for _, l := range lines {
		if l.IsStageDirection {
			stageDirs++
		} else {
			speeches++
		}
	}

	t.Logf("King John: %d total lines (%d speeches, %d stage dirs)", len(lines), speeches, stageDirs)

	// First speech should be K. John.
	for _, l := range lines {
		if !l.IsStageDirection {
			if l.Character != "K. John." {
				t.Errorf("first speaker should be K. John., got %q", l.Character)
			}
			break
		}
	}
}

// assertPerseusLine is a test helper that verifies all fields of a PerseusLine.
func assertPerseusLine(t *testing.T, got PerseusLine, act, scene int, char, charID, text string, isSD bool, lineInScene, globeLine int) {
	t.Helper()

	if got.Act != act {
		t.Errorf("Act: expected %d, got %d (line text: %q)", act, got.Act, got.Text)
	}
	if got.Scene != scene {
		t.Errorf("Scene: expected %d, got %d (line text: %q)", scene, got.Scene, got.Text)
	}
	if got.Character != char {
		t.Errorf("Character: expected %q, got %q (line text: %q)", char, got.Character, got.Text)
	}
	if got.CharID != charID {
		t.Errorf("CharID: expected %q, got %q (line text: %q)", charID, got.CharID, got.Text)
	}
	if got.Text != text {
		t.Errorf("Text: expected %q, got %q", text, got.Text)
	}
	if got.IsStageDirection != isSD {
		t.Errorf("IsStageDirection: expected %v, got %v (line text: %q)", isSD, got.IsStageDirection, got.Text)
	}
	if got.LineInScene != lineInScene {
		t.Errorf("LineInScene: expected %d, got %d (line text: %q)", lineInScene, got.LineInScene, got.Text)
	}
	if got.GlobeLine != globeLine {
		t.Errorf("GlobeLine: expected %d, got %d (line text: %q)", globeLine, got.GlobeLine, got.Text)
	}
}
