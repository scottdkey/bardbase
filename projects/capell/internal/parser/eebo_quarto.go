// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

// QuartoLine represents a single parsed line from an EEBO-TCP early quarto TEI XML file.
// Early quartos typically lack act/scene structure; all lines are assigned act=1, scene=1.
// Text is in original spelling with long-s (ſ) normalized to s.
type QuartoLine struct {
	Act              int
	Scene            int
	Character        string
	Text             string
	IsStageDirection bool
	LineInScene      int // Sequential 1-based counter within the play
}

// ParseEEBOQuartoTEI parses a single-play EEBO-TCP TEI XML quarto into a flat list
// of QuartoLines. Early quartos (Q1 Hamlet, Q1 1H4, etc.) use a flat structure:
//
//	<div type="text">
//	  <head>Play Title</head>
//	  <stage>Enter …</stage>
//	  <sp>
//	    <speaker>Name.</speaker>
//	    <p>Prose speech…</p>    — or —
//	    <l>Verse line</l>
//	  </sp>
//	</div>
//
// Since quartos predate standard act/scene notation, all lines are assigned act=1, scene=1.
// Long-s (ſ) is normalized to s. <gap> is replaced with [?].
func ParseEEBOQuartoTEI(xmlData []byte) ([]QuartoLine, error) {
	root, err := ParseXML(xmlData)
	if err != nil {
		return nil, err
	}

	var lines []QuartoLine
	counter := 0

	// Find all top-level play/text divs — quartos vary:
	//   <div type="text"> — Q1 Hamlet (A11959)
	//   <div type="play"> — Q1 1H4 (A11966), Q1 Titus (A12017), Q1 2H6 (A68931)
	for _, div := range root.FindAll("div") {
		t := div.Attr("type")
		if t == "text" || t == "play" {
			parseQuartoDiv(div, &lines, &counter)
		}
	}

	return lines, nil
}

// parseQuartoDiv walks a div element collecting stage directions and speeches.
// Handles nested divs (some quartos have minor structural divisions).
func parseQuartoDiv(div *XMLNode, lines *[]QuartoLine, counter *int) {
	for _, child := range div.Children {
		switch child.Name {
		case "stage":
			text := normalizeFolioText(child.GetText())
			if text != "" {
				*counter++
				*lines = append(*lines, QuartoLine{
					Act:              1,
					Scene:            1,
					Text:             text,
					IsStageDirection: true,
					LineInScene:      *counter,
				})
			}
		case "sp":
			spLines := parseQuartoSpeech(child, counter)
			*lines = append(*lines, spLines...)
		case "div":
			// Recurse into nested structural divs
			parseQuartoDiv(child, lines, counter)
		}
	}
}

// parseQuartoSpeech extracts lines from a <sp> element.
// Handles <p> (prose, one line per paragraph) and <l> (verse, one line per element).
func parseQuartoSpeech(sp *XMLNode, counter *int) []QuartoLine {
	speaker := ""
	for _, child := range sp.Children {
		if child.Name == "speaker" {
			speaker = normalizeFolioText(child.GetText())
			break
		}
	}

	var lines []QuartoLine
	for _, child := range sp.Children {
		switch child.Name {
		case "p":
			text := normalizeFolioText(extractFolioText(child))
			if text != "" {
				*counter++
				lines = append(lines, QuartoLine{
					Act:         1,
					Scene:       1,
					Character:   speaker,
					Text:        text,
					LineInScene: *counter,
				})
			}
		case "l":
			text := normalizeFolioText(extractFolioText(child))
			if text != "" {
				*counter++
				lines = append(lines, QuartoLine{
					Act:         1,
					Scene:       1,
					Character:   speaker,
					Text:        text,
					LineInScene: *counter,
				})
			}
		case "lg":
			for _, lChild := range child.Children {
				if lChild.Name == "l" {
					text := normalizeFolioText(extractFolioText(lChild))
					if text != "" {
						*counter++
						lines = append(lines, QuartoLine{
							Act:         1,
							Scene:       1,
							Character:   speaker,
							Text:        text,
							LineInScene: *counter,
						})
					}
				}
			}
		case "stage":
			text := normalizeFolioText(child.GetText())
			if text != "" {
				*counter++
				lines = append(lines, QuartoLine{
					Act:              1,
					Scene:            1,
					Text:             text,
					IsStageDirection: true,
					LineInScene:      *counter,
				})
			}
		}
	}
	return lines
}
