// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strconv"
	"strings"
)

// PerseusLine represents a single parsed line from a Perseus TEI XML play.
// Perseus texts use the Clark & Wright Globe edition with line numbers marked
// by <lb ed="G"> tags. First Folio line numbers (<lb ed="F1">) are also captured.
type PerseusLine struct {
	Act              int
	Scene            int
	Character        string // Speaker label as printed (e.g., "Phi.", "Ant.")
	CharID           string // Character ID from cast list (e.g., "ant-33")
	Text             string
	IsStageDirection bool
	IsVerse          bool
	LineInScene      int // Sequential line number within the scene (1-based)
	GlobeLine        int // Globe edition line number (0 if not on a numbered boundary)
}

// ParsePerseusTEI parses a Perseus Digital Library TEI XML play into a flat list
// of PerseusLines. The parser handles two verse/prose formats:
//
//   - <p> elements: prose; lines are split on Globe <lb ed="G"> breaks
//   - <l> elements: verse; each <l> is one line
//
// The TEI structure follows this hierarchy:
//
//	<div1 type="act" n="1">
//	  <div2 type="scene" n="1">
//	    <stage type="setting">...</stage>
//	    <stage type="entrance">Enter ...</stage>
//	    <sp who="char-id"><speaker>Name.</speaker>
//	      <p>Prose with <lb ed="G"/> line breaks...</p>
//	      — or —
//	      <l>Verse line one</l>
//	      <l>Verse line two</l>
//	    </sp>
//	  </div2>
//	</div1>
//
// This is a pure function: XML bytes in, structured lines out.
func ParsePerseusTEI(xmlData []byte) ([]PerseusLine, error) {
	root, err := ParseXML(xmlData)
	if err != nil {
		return nil, err
	}

	var lines []PerseusLine
	body := root.Find("body")
	if body == nil {
		// Try TEI.2 > text > body path
		text := root.Find("text")
		if text != nil {
			body = text.Find("body")
		}
	}
	if body == nil {
		return nil, nil
	}

	// Walk div1 (acts) — skip cast list but include prologues/epilogues.
	// Non-numeric act labels like "induction", "prologue", "epilogue" are
	// mapped to act 0 so they can be stored and aligned.
	for _, div1 := range body.Children {
		if div1.Name != "div1" {
			continue
		}
		if div1.Attr("type") != "act" {
			continue
		}
		nAttr := div1.Attr("n")
		if nAttr == "cast" {
			continue
		}
		act, _ := strconv.Atoi(nAttr) // non-numeric → 0

		// Walk div2 (scenes)
		for _, div2 := range div1.Children {
			if div2.Name != "div2" {
				continue
			}
			scene, _ := strconv.Atoi(div2.Attr("n"))
			counter := 0

			sceneLines := parseSceneChildren(div2, act, scene, &counter)
			lines = append(lines, sceneLines...)
		}
	}

	return lines, nil
}

// parseSceneChildren walks the direct children of a div2 (scene) node,
// extracting speeches and standalone stage directions.
func parseSceneChildren(div2 *XMLNode, act, scene int, counter *int) []PerseusLine {
	var lines []PerseusLine

	for _, child := range div2.Children {
		switch child.Name {
		case "stage":
			// Standalone stage direction (not inside a speech)
			text := cleanText(child.GetText())
			if text != "" {
				*counter++
				lines = append(lines, PerseusLine{
					Act:              act,
					Scene:            scene,
					Text:             text,
					IsStageDirection: true,
					LineInScene:      *counter,
				})
			}

		case "sp":
			// Speech — extract character info and walk <p>/<l> elements
			charID := child.Attr("who")
			speaker := ""
			if spkNode := child.Find("speaker"); spkNode != nil {
				speaker = cleanText(spkNode.GetText())
			}

			spLines := parseSpeech(child, act, scene, charID, speaker, counter)
			lines = append(lines, spLines...)
		}
	}

	return lines
}

// parseSpeech extracts lines from a <sp> element. Handles two TEI encoding styles:
//   - <p> children: prose, split on Globe <lb ed="G"> breaks
//   - <l> children: verse, each <l> is one line
//   - <stage> children: stage directions
func parseSpeech(sp *XMLNode, act, scene int, charID, speaker string, counter *int) []PerseusLine {
	var lines []PerseusLine

	for _, child := range sp.Children {
		switch child.Name {
		case "p":
			pLines := extractLinesFromP(child, act, scene, charID, speaker, counter)
			lines = append(lines, pLines...)

		case "l":
			// Verse line — each <l> element is one line.
			lLine := extractLineFromL(child, act, scene, charID, speaker, counter)
			if lLine != nil {
				lines = append(lines, *lLine)
			}

		case "stage":
			// Stage direction directly inside <sp> but outside <p>/<l>
			text := cleanText(child.GetText())
			if text != "" {
				*counter++
				lines = append(lines, PerseusLine{
					Act:              act,
					Scene:            scene,
					Text:             text,
					IsStageDirection: true,
					LineInScene:      *counter,
				})
			}
		}
	}

	return lines
}

// extractLineFromL extracts a single line from a verse <l> element.
// The <l> element may contain <lb> tags (Globe/F1 markers) and inline text.
// Text content from all child elements (excluding lb markers) is concatenated.
// Globe line number is captured from any <lb ed="G" n="..."> within.
func extractLineFromL(l *XMLNode, act, scene int, charID, speaker string, counter *int) *PerseusLine {
	// Collect all text content from the <l> element.
	// Text lives in l.Text and in Tail of child elements (after <lb> tags).
	var buf strings.Builder
	buf.WriteString(l.Text)

	globeN := 0
	for _, child := range l.Children {
		switch {
		case child.Name == "lb" && child.Attr("ed") == "G":
			// Capture Globe line number if present.
			if n := child.Attr("n"); n != "" {
				globeN, _ = strconv.Atoi(n)
			}
			buf.WriteString(child.Tail)

		case child.Name == "lb":
			// F1 or other lb — just collect tail text.
			buf.WriteString(child.Tail)

		case child.Name == "stage":
			// Inline stage direction within a verse line — include text inline.
			buf.WriteString(child.GetText())
			buf.WriteString(child.Tail)

		default:
			// Other inline elements (<reg>, <foreign>, etc.)
			buf.WriteString(child.GetText())
			buf.WriteString(child.Tail)
		}
	}

	text := cleanText(buf.String())
	if text == "" {
		return nil
	}

	*counter++
	return &PerseusLine{
		Act:         act,
		Scene:       scene,
		Character:   speaker,
		CharID:      charID,
		Text:        text,
		LineInScene: *counter,
		GlobeLine:   globeN,
	}
}

// extractLinesFromP walks the children of a <p> element, splitting text on
// Globe edition line breaks (<lb ed="G">). Text between consecutive Globe breaks
// forms one PerseusLine.
//
// The TEI format intermixes <lb> tags with text content:
//   - <lb ed="G" />       — Globe line boundary (start of new line)
//   - <lb ed="G" n="10"/> — Globe line boundary with explicit line number
//   - <lb ed="F1" n="42"/>— First Folio reference (not a line boundary; text in Tail)
//   - <stage>...</stage>  — Inline stage direction (emitted as separate line)
//   - <reg orig="x">y</reg> — Regularized spelling (use text content)
//
// Text accumulates from element Tails between Globe breaks.
func extractLinesFromP(p *XMLNode, act, scene int, charID, speaker string, counter *int) []PerseusLine {
	var lines []PerseusLine
	var buf strings.Builder
	globeN := 0

	// flushSpeech emits accumulated text as a speech line (if non-empty).
	flushSpeech := func() {
		text := cleanText(buf.String())
		if text != "" {
			*counter++
			lines = append(lines, PerseusLine{
				Act:         act,
				Scene:       scene,
				Character:   speaker,
				CharID:      charID,
				Text:        text,
				LineInScene: *counter,
				GlobeLine:   globeN,
			})
		}
		buf.Reset()
		globeN = 0
	}

	// Start with any text before the first child element.
	buf.WriteString(p.Text)

	for _, child := range p.Children {
		switch {
		case child.Name == "lb" && child.Attr("ed") == "G":
			// Globe line break — flush accumulated text as a line.
			flushSpeech()
			if n := child.Attr("n"); n != "" {
				globeN, _ = strconv.Atoi(n)
			}
			// Tail text after the Globe <lb> starts the next line.
			buf.WriteString(child.Tail)

		case child.Name == "lb":
			// F1 or other <lb> — not a line boundary. Collect tail text.
			buf.WriteString(child.Tail)

		case child.Name == "stage":
			// Inline stage direction — flush current speech, emit stage dir.
			flushSpeech()
			stageText := cleanText(child.GetText())
			if stageText != "" {
				*counter++
				lines = append(lines, PerseusLine{
					Act:              act,
					Scene:            scene,
					Text:             stageText,
					IsStageDirection: true,
					LineInScene:      *counter,
				})
			}
			// Tail text after <stage> continues the speech.
			buf.WriteString(child.Tail)

		default:
			// Other inline elements (<reg>, <l>, <foreign>, etc.) —
			// extract their text content and tail.
			buf.WriteString(child.GetText())
			buf.WriteString(child.Tail)
		}
	}

	// Flush any remaining text after the last child.
	flushSpeech()

	return lines
}

// isNumeric returns true if s is a non-empty string of digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// cleanText trims whitespace and collapses internal runs of whitespace to single spaces.
func cleanText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Collapse internal whitespace runs (newlines, tabs, multiple spaces).
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
