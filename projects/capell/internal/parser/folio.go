// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strings"
)

// FolioLine represents a single parsed line from the EEBO-TCP First Folio TEI XML (A11954).
// The First Folio (1623) is a diplomatic transcription in original spelling with
// long-s (ſ) normalized to s. Verse lines are marked with <l>; prose speeches are in <p>.
type FolioLine struct {
	PlayTitle        string // Normalized head text from <div type="play"><head>
	Act              int
	Scene            int
	Character        string // Speaker label (e.g., "Ham.", "Pros."), normalized
	Text             string // Line content, original spelling preserved, long-s normalized
	IsStageDirection bool
	IsVerse          bool
	LineInScene      int // Sequential line number within the scene (1-based)
}

// ParseFirstFolioTEI parses the EEBO-TCP First Folio TEI XML (A11954) into a flat list
// of FolioLines. The parser walks:
//
//	<div type="play">
//	  <head>THE TEMPEST.</head>
//	  <div n="1" type="act">
//	    <div n="1" type="scene">
//	      <stage>...</stage>
//	      <sp>
//	        <speaker>Name.</speaker>
//	        <p>Prose speech...</p>   — or —
//	        <l>Verse line</l>
//	      </sp>
//	    </div>
//	  </div>
//	</div>
//
// Long-s (ſ, U+017F) is normalized to s in all output text.
// <g ref="char:EOLhyphen"> (typographic line-end hyphen) is treated as empty.
// <gap> elements are replaced with [?].
func ParseFirstFolioTEI(xmlData []byte) ([]FolioLine, error) {
	root, err := ParseXML(xmlData)
	if err != nil {
		return nil, err
	}

	var lines []FolioLine

	// The First Folio XML has two <body> elements: one inside <front> (front matter,
	// no plays) and one sibling of <front> that contains all 36 plays. Using
	// root.Find("body") returns the first one via DFS, which is the wrong body.
	// Instead, collect all <div type="play"> anywhere in the document — they only
	// appear at the top level of the main body, so this is unambiguous.
	for _, div := range root.FindAll("div") {
		if div.Attr("type") == "play" {
			playLines := parsePlay(div)
			lines = append(lines, playLines...)
		}
	}

	return lines, nil
}

// parsePlay extracts the play title and delegates to act/scene parsing.
func parsePlay(playDiv *XMLNode) []FolioLine {
	// Extract and normalize play title from the first <head> child
	title := ""
	for _, child := range playDiv.Children {
		if child.Name == "head" {
			title = normalizeFolioHead(child.GetText())
			break
		}
	}

	var lines []FolioLine
	for _, child := range playDiv.Children {
		if child.Name == "div" && child.Attr("type") == "act" {
			act := atoi(child.Attr("n"))
			actLines := parseAct(child, title, act)
			lines = append(lines, actLines...)
		}
	}
	return lines
}

// parseAct walks the act div for scene children.
func parseAct(actDiv *XMLNode, playTitle string, act int) []FolioLine {
	var lines []FolioLine
	for _, child := range actDiv.Children {
		if child.Name == "div" && child.Attr("type") == "scene" {
			scene := atoi(child.Attr("n"))
			counter := 0
			sceneLines := parseSceneFolio(child, playTitle, act, scene, &counter)
			lines = append(lines, sceneLines...)
		}
	}
	return lines
}

// parseSceneFolio walks direct children of a scene div: <stage> and <sp>.
func parseSceneFolio(sceneDiv *XMLNode, playTitle string, act, scene int, counter *int) []FolioLine {
	var lines []FolioLine
	for _, child := range sceneDiv.Children {
		switch child.Name {
		case "stage":
			text := normalizeFolioText(child.GetText())
			if text != "" {
				*counter++
				lines = append(lines, FolioLine{
					PlayTitle:        playTitle,
					Act:              act,
					Scene:            scene,
					Text:             text,
					IsStageDirection: true,
					LineInScene:      *counter,
				})
			}
		case "sp":
			spLines := parseSpeechFolio(child, playTitle, act, scene, counter)
			lines = append(lines, spLines...)
		}
	}
	return lines
}

// parseSpeechFolio extracts lines from a <sp> element.
// Handles both prose (<p>) and verse (<l>) children.
func parseSpeechFolio(sp *XMLNode, playTitle string, act, scene int, counter *int) []FolioLine {
	// Extract speaker from first <speaker> child
	speaker := ""
	for _, child := range sp.Children {
		if child.Name == "speaker" {
			speaker = normalizeFolioText(child.GetText())
			break
		}
	}

	var lines []FolioLine
	for _, child := range sp.Children {
		switch child.Name {
		case "p":
			// Prose speech — treat the whole <p> as one line
			text := normalizeFolioText(extractFolioText(child))
			if text != "" {
				*counter++
				lines = append(lines, FolioLine{
					PlayTitle:   playTitle,
					Act:         act,
					Scene:       scene,
					Character:   speaker,
					Text:        text,
					LineInScene: *counter,
				})
			}
		case "l":
			// Verse line — each <l> is one line
			text := normalizeFolioText(extractFolioText(child))
			if text != "" {
				*counter++
				lines = append(lines, FolioLine{
					PlayTitle:   playTitle,
					Act:         act,
					Scene:       scene,
					Character:   speaker,
					Text:        text,
					LineInScene: *counter,
				})
			}
		case "lg":
			// Line group — walk its <l> children
			for _, lChild := range child.Children {
				if lChild.Name == "l" {
					text := normalizeFolioText(extractFolioText(lChild))
					if text != "" {
						*counter++
						lines = append(lines, FolioLine{
							PlayTitle:   playTitle,
							Act:         act,
							Scene:       scene,
							Character:   speaker,
							Text:        text,
							LineInScene: *counter,
						})
					}
				}
			}
		case "stage":
			// Inline stage direction within a speech
			text := normalizeFolioText(child.GetText())
			if text != "" {
				*counter++
				lines = append(lines, FolioLine{
					PlayTitle:        playTitle,
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

// extractFolioText collects text content from a node, handling Folio-specific elements:
//   - <g ref="char:EOLhyphen">: typographic line-end hyphen — treated as empty
//   - <gap>: illegible text — replaced with [?]
//   - <pb>, <lb>: page/line breaks — skip, collect tail
//   - <hi>, <seg>, <reg>, and other inline elements: extract text content
func extractFolioText(node *XMLNode) string {
	var b strings.Builder
	b.WriteString(node.Text)
	for _, child := range node.Children {
		switch child.Name {
		case "g":
			// char:EOLhyphen is a typographic artifact — emit nothing for the element itself
			// (the word fragments join naturally with whitespace collapsing)
		case "gap":
			b.WriteString("[?]")
		case "pb", "lb":
			// Page/line breaks — skip element, collect tail
		case "stage":
			// Inline stage direction inside <p> or <l> — include text inline
			b.WriteString(child.GetText())
		default:
			// <hi>, <seg>, <reg>, <foreign>, <abbr>, etc. — extract text
			b.WriteString(extractFolioText(child))
		}
		b.WriteString(child.Tail)
	}
	return b.String()
}

// normalizeFolioText normalizes long-s and collapses whitespace.
func normalizeFolioText(s string) string {
	s = strings.ReplaceAll(s, "ſ", "s")
	return cleanText(s)
}

// normalizeFolioHead normalizes a play head for use as a map key.
// Applies long-s normalization, lowercasing, and whitespace collapsing.
func normalizeFolioHead(s string) string {
	s = strings.ReplaceAll(s, "ſ", "s")
	s = strings.ToLower(s)
	return cleanText(s)
}

// atoi converts a string to int, returning 0 on failure.
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
