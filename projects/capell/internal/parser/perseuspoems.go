// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strconv"
)

// PerseusPoem represents a parsed poem/sonnet collection from Perseus TEI XML.
type PerseusPoem struct {
	Lines []PerseusPoemLine
}

// PerseusPoemLine represents a single line from a Perseus poem.
type PerseusPoemLine struct {
	LineNumber int    // Globe/explicit line number (1-based)
	Content    string // Line text
	Section    int    // Sonnet number, stanza number, or poem number (0 = none)
}

// ParsePerseusPoem parses a Perseus TEI XML poem into lines.
// Handles three structures:
//   - Sonnets: <div1 type="sonnet" n="I"> with unnumbered <l> elements
//   - Narrative poems (Lucrece, Venus): <l n="N"> or <lb n="N" ed="G"> numbered lines
//   - Poem collections (Passionate Pilgrim): <div1 n="N" type="sequence">
func ParsePerseusPoem(xmlData []byte, workType string) (*PerseusPoem, error) {
	root, err := ParseXML(xmlData)
	if err != nil {
		return nil, err
	}

	body := root.Find("body")
	if body == nil {
		text := root.Find("text")
		if text != nil {
			body = text.Find("body")
		}
	}
	if body == nil {
		return nil, nil
	}

	poem := &PerseusPoem{}

	switch workType {
	case "sonnet_sequence":
		parseSonnets(body, poem)
	default:
		parseNarrativePoem(body, poem)
	}

	return poem, nil
}

// parseSonnets extracts sonnets from <div1 type="sonnet" n="I"> structures.
// Each sonnet becomes a section; lines are numbered sequentially within each sonnet.
func parseSonnets(body *XMLNode, poem *PerseusPoem) {
	for _, div1 := range body.Children {
		if div1.Name != "div1" {
			continue
		}
		if div1.Attr("type") != "sonnet" {
			continue
		}

		nStr := div1.Attr("n")
		sonnetNum := parseRoman(nStr)
		if sonnetNum == 0 {
			sonnetNum, _ = strconv.Atoi(nStr)
		}
		if sonnetNum == 0 {
			continue // skip dedication etc.
		}

		lineNum := 0
		collectLines(div1, &lineNum, sonnetNum, poem)
	}
}

// parseNarrativePoem extracts lines from narrative poems (Venus, Lucrece, etc.).
// Lines are numbered via <l n="N"> attributes or <lb n="N" ed="G"> tags.
func parseNarrativePoem(body *XMLNode, poem *PerseusPoem) {
	lineNum := 0
	// Walk all div1 children
	for _, div1 := range body.Children {
		if div1.Name != "div1" {
			continue
		}

		section := 0
		nStr := div1.Attr("n")
		if v, err := strconv.Atoi(nStr); err == nil {
			section = v
		}

		collectLines(div1, &lineNum, section, poem)
	}
}

// collectLines recursively finds all <l> elements within a node and
// extracts their text and line numbers.
func collectLines(node *XMLNode, lineCounter *int, section int, poem *PerseusPoem) {
	for _, child := range node.Children {
		if child.Name == "l" {
			text := cleanText(child.GetText())
			if text == "" {
				continue
			}

			// Try explicit line number from n attribute
			num := 0
			if nStr := child.Attr("n"); nStr != "" {
				num, _ = strconv.Atoi(nStr)
			}

			// For sonnets (no n attribute), use sequential numbering
			if num == 0 {
				*lineCounter++
				num = *lineCounter
			} else {
				*lineCounter = num
			}

			poem.Lines = append(poem.Lines, PerseusPoemLine{
				LineNumber: num,
				Content:    text,
				Section:    section,
			})
		} else if child.Name == "lg" || child.Name == "lg1" ||
			child.Name == "div2" || child.Name == "div3" ||
			child.Name == "p" || child.Name == "sp" {
			// Recurse into grouping elements
			collectLines(child, lineCounter, section, poem)
		}
	}
}

// parseRoman is defined in lexicon.go
