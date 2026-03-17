// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strings"
)

// FolgerLine represents a single parsed line from a Folger Shakespeare TEIsimple XML file.
// Folger TEIsimple uses per-word XML tagging (<w>, <c>, <pc>) with Folger Through Line
// Numbers (FTLNs) encoded in n="act.scene.line" attributes on <l> and <lb> elements.
type FolgerLine struct {
	Act              int
	Scene            int
	LineNumber       int    // Folger act.scene.line number (verse-based, from n attribute)
	Character        string // From <speaker> text, uppercased in source
	Text             string // Reconstructed text from <w>, <c>, <pc> children
	IsStageDirection bool
	IsVerse          bool
}

// ParseFolgerTEIsimple parses a Folger Shakespeare TEIsimple XML file into a flat list
// of FolgerLines. The parser walks:
//
//	<div type="act" n="1">
//	  <div type="scene" n="1">
//	    <stage>…word-tagged…</stage>
//	    <sp who="#CharID_Play">
//	      <speaker>…word-tagged…</speaker>
//	      <l n="1.1.2">…word-tagged…</l>     — verse line
//	      <p>
//	        <lb n="1.1.5"/>…word-tagged…      — prose, split on <lb>
//	      </p>
//	    </sp>
//	  </div>
//	</div>
//
// Text is reconstructed by concatenating the .Text field of <w>, <c>, and <pc>
// child elements (ignoring element tails, which are only XML indentation whitespace).
// This avoids the "Nay , answer me" spacing issue that occurs when using GetText()
// on word-tagged content.
func ParseFolgerTEIsimple(xmlData []byte) ([]FolgerLine, error) {
	root, err := ParseXML(xmlData)
	if err != nil {
		return nil, err
	}

	var lines []FolgerLine

	for _, actDiv := range root.FindAll("div") {
		if actDiv.Attr("type") != "act" {
			continue
		}
		act := atoi(actDiv.Attr("n"))
		if act == 0 {
			continue
		}

		for _, sceneDiv := range actDiv.Children {
			if sceneDiv.Name != "div" || sceneDiv.Attr("type") != "scene" {
				continue
			}
			scene := atoi(sceneDiv.Attr("n"))
			if scene == 0 {
				scene = 1
			}

			sceneLines := parseFolgerScene(sceneDiv, act, scene)
			lines = append(lines, sceneLines...)
		}
	}

	return lines, nil
}

// parseFolgerScene walks direct children of a scene div: <stage> and <sp>.
func parseFolgerScene(sceneDiv *XMLNode, act, scene int) []FolgerLine {
	var lines []FolgerLine
	for _, child := range sceneDiv.Children {
		switch child.Name {
		case "stage":
			text := extractTEIsimpleText(child)
			text = cleanText(text)
			if text != "" {
				lines = append(lines, FolgerLine{
					Act:              act,
					Scene:            scene,
					Text:             text,
					IsStageDirection: true,
				})
			}
		case "sp":
			spLines := parseFolgerSpeech(child, act, scene)
			lines = append(lines, spLines...)
		}
	}
	return lines
}

// parseFolgerSpeech extracts lines from a <sp> element.
// Handles verse (<l>) and prose (<p> with <lb> markers) children.
func parseFolgerSpeech(sp *XMLNode, act, scene int) []FolgerLine {
	// Extract speaker from <speaker> child text.
	character := ""
	for _, child := range sp.Children {
		if child.Name == "speaker" {
			character = cleanText(extractTEIsimpleText(child))
			break
		}
	}

	var lines []FolgerLine
	for _, child := range sp.Children {
		switch child.Name {
		case "l":
			// Verse line — n="act.scene.linenum"
			text := cleanText(extractTEIsimpleText(child))
			if text != "" {
				lineNum := parseFolgerLineAttr(child.Attr("n"))
				lines = append(lines, FolgerLine{
					Act:        act,
					Scene:      scene,
					LineNumber: lineNum,
					Character:  character,
					Text:       text,
					IsVerse:    true,
				})
			}
		case "p":
			// Prose — split on <lb n="act.scene.linenum"/> markers
			proseLines := parseFolgerProse(child, act, scene, character)
			lines = append(lines, proseLines...)
		case "ab":
			// Alternative prose block (used in some editions)
			text := cleanText(extractTEIsimpleText(child))
			if text != "" {
				lineNum := parseFolgerLineAttr(child.Attr("n"))
				lines = append(lines, FolgerLine{
					Act:       act,
					Scene:     scene,
					LineNumber: lineNum,
					Character: character,
					Text:      text,
				})
			}
		case "stage":
			// Inline stage direction within a speech
			text := cleanText(extractTEIsimpleText(child))
			if text != "" {
				lines = append(lines, FolgerLine{
					Act:              act,
					Scene:            scene,
					Text:             text,
					IsStageDirection: true,
				})
			}
		}
	}
	return lines
}

// parseFolgerProse splits a <p> element on <lb n="act.scene.line"/> markers.
// Each <lb> begins a new line; text (from <w>, <c>, <pc> siblings) accumulates
// until the next <lb> or end of <p>.
func parseFolgerProse(p *XMLNode, act, scene int, character string) []FolgerLine {
	var lines []FolgerLine
	currentLineNum := 0
	var buf strings.Builder

	flush := func() {
		text := cleanText(buf.String())
		if text != "" && currentLineNum > 0 {
			lines = append(lines, FolgerLine{
				Act:       act,
				Scene:     scene,
				LineNumber: currentLineNum,
				Character: character,
				Text:      text,
			})
		}
		buf.Reset()
	}

	for _, child := range p.Children {
		switch child.Name {
		case "lb":
			flush()
			currentLineNum = parseFolgerLineAttr(child.Attr("n"))
		case "w", "c", "pc":
			buf.WriteString(child.Text)
		case "stage":
			// Inline stage direction — flush current prose line first, emit stage dir
			flush()
			text := cleanText(extractTEIsimpleText(child))
			if text != "" {
				lines = append(lines, FolgerLine{
					Act:              act,
					Scene:            scene,
					Text:             text,
					IsStageDirection: true,
				})
			}
		}
	}
	flush()

	return lines
}

// extractTEIsimpleText reconstructs readable text from a TEIsimple word-tagged element.
// It concatenates only the .Text field of <w> (word), <c> (space), and <pc> (punctuation)
// children, ignoring element tails (which are XML indentation whitespace, not content).
// Nested elements like <stage> or <lb> are skipped.
func extractTEIsimpleText(node *XMLNode) string {
	var b strings.Builder
	for _, child := range node.Children {
		switch child.Name {
		case "w", "c", "pc":
			b.WriteString(child.Text)
		}
	}
	return b.String()
}

// parseFolgerLineAttr parses a Folger n attribute of the form "act.scene.line"
// (e.g., "1.1.2") and returns the line number (third segment).
// Returns 0 if the attribute is missing, is a stage direction ("SD …"), or is malformed.
func parseFolgerLineAttr(n string) int {
	if n == "" || strings.HasPrefix(n, "SD") {
		return 0
	}
	parts := strings.SplitN(n, ".", 3)
	if len(parts) < 3 {
		return 0
	}
	return atoi(parts[2])
}
