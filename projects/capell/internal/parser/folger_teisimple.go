// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strings"
)

// FolgerWord holds word-level linguistic data extracted from a single <w> element
// in a Folger TEIsimple file. Every word token in the Folger edition is tagged by
// MorphAdorner with a lemma and a POS tag.
type FolgerWord struct {
	Word  string // surface form (text content of <w>)
	Lemma string // dictionary headword (lemma="" attribute)
	POS   string // MorphAdorner tag (ana="" attribute, leading '#' stripped)
}

// FolgerLine represents a single parsed line from a Folger Shakespeare TEIsimple XML file.
// Folger TEIsimple uses per-word XML tagging (<w>, <c>, <pc>) with Folger Through Line
// Numbers (FTLNs) encoded in n="act.scene.line" attributes on <l> and <lb> elements.
type FolgerLine struct {
	Act              int
	Scene            int
	LineNumber       int        // Folger act.scene.line number (verse-based, from n attribute)
	Character        string     // From <speaker> text, uppercased in source
	Text             string     // Reconstructed text from <w>, <c>, <pc> children
	IsStageDirection bool
	IsVerse          bool
	// Stage direction metadata — non-empty only when IsStageDirection is true.
	// StageType is the value of <stage type="…"> (e.g. "entrance", "exit").
	// StageWho is the raw who="…" attribute: space-separated XML ID references
	// like "#Theseus_MND #Hippolyta_MND". The leading '#' is preserved here;
	// callers that need bare IDs should strip it.
	StageType string
	StageWho  string
	// Words holds per-word annotations from <w lemma="…" ana="…"> elements.
	// Populated for all line types (speech and stage directions). The slice
	// is in document order and aligns with the words in Text.
	// Words is nil when the line has no <w> elements (e.g. some stage directions).
	Words []FolgerWord
}

// ParseFolgerTEIsimple parses a Folger Shakespeare TEIsimple XML file into a flat list
// of FolgerLines. The parser walks:
//
//	<div type="act" n="1">
//	  <div type="scene" n="1">
//	    <stage type="entrance" who="#Theseus_MND #Hippolyta_MND">…word-tagged…</stage>
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
// Word-level annotations (lemma, POS from MorphAdorner) are captured in FolgerLine.Words.
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
					StageType:        child.Attr("type"),
					StageWho:         child.Attr("who"),
					Words:            extractFolgerWords(child),
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
					Words:      extractFolgerWords(child),
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
					Act:        act,
					Scene:      scene,
					LineNumber: lineNum,
					Character:  character,
					Text:       text,
					Words:      extractFolgerWords(child),
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
					StageType:        child.Attr("type"),
					StageWho:         child.Attr("who"),
					Words:            extractFolgerWords(child),
				})
			}
		}
	}
	return lines
}

// parseFolgerProse splits a <p> element on <lb n="act.scene.line"/> markers.
// Each <lb> begins a new line; text (from <w>, <c>, <pc> siblings) accumulates
// until the next <lb> or end of <p>.
//
// Word annotations are accumulated per prose line: words between two <lb> markers
// belong to the line started by the first <lb>.
func parseFolgerProse(p *XMLNode, act, scene int, character string) []FolgerLine {
	var lines []FolgerLine
	currentLineNum := 0
	var textBuf strings.Builder
	var wordBuf []FolgerWord

	flush := func() {
		text := cleanText(textBuf.String())
		if text != "" && currentLineNum > 0 {
			lines = append(lines, FolgerLine{
				Act:        act,
				Scene:      scene,
				LineNumber: currentLineNum,
				Character:  character,
				Text:       text,
				Words:      wordBuf,
			})
		}
		textBuf.Reset()
		wordBuf = nil
	}

	for _, child := range p.Children {
		switch child.Name {
		case "lb":
			flush()
			currentLineNum = parseFolgerLineAttr(child.Attr("n"))
		case "w":
			textBuf.WriteString(child.Text)
			wordBuf = append(wordBuf, FolgerWord{
				Word:  child.Text,
				Lemma: child.Attr("lemma"),
				POS:   strings.TrimPrefix(child.Attr("ana"), "#"),
			})
		case "c", "pc":
			textBuf.WriteString(child.Text)
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
					StageType:        child.Attr("type"),
					StageWho:         child.Attr("who"),
					Words:            extractFolgerWords(child),
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

// extractFolgerWords extracts word-level annotations from all <w> child elements
// of node. Returns nil if there are no <w> elements.
func extractFolgerWords(node *XMLNode) []FolgerWord {
	var words []FolgerWord
	for _, child := range node.Children {
		if child.Name == "w" {
			words = append(words, FolgerWord{
				Word:  child.Text,
				Lemma: child.Attr("lemma"),
				POS:   strings.TrimPrefix(child.Attr("ana"), "#"),
			})
		}
	}
	return words
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
