// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// PlayLine represents a single parsed line from a Standard Ebooks play.
type PlayLine struct {
	Act              int
	Scene            int
	Character        string
	Text             string
	IsStageDirection bool
	LineInScene      int
}

// ParseSEPlay parses a Standard Ebooks play XHTML document into a flat list of PlayLines.
// This is a pure function: input XHTML string, output structured lines.
func ParseSEPlay(xhtml string) []PlayLine {
	tokenizer := html.NewTokenizer(strings.NewReader(xhtml))
	p := &playParser{tokenizer: tokenizer}
	return p.parse()
}

type playParser struct {
	tokenizer *html.Tokenizer

	lines            []PlayLine
	currentAct       int
	currentScene     int
	currentCharacter string
	sceneLineCounter int

	buf       strings.Builder
	tagStack  []string
	stateStack []playState

	inPersona      bool
	inStageDir     bool
	inVerseSpan    bool
	inTD           bool
	inHeader       bool
	tdIsSpeech     bool
	nestedStageDir bool

	verseLines []verseLine
}

type playState int

type verseLine struct {
	text             string
	isStageDirection bool
}

func (p *playParser) parse() []PlayLine {
	for {
		tt := p.tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return p.lines

		case html.StartTagToken:
			tn, hasAttr := p.tokenizer.TagName()
			tagName := string(tn)
			attrs := p.readAttrs(hasAttr)
			p.tagStack = append(p.tagStack, tagName)
			p.handleStart(tagName, attrs)

		case html.EndTagToken:
			tn, _ := p.tokenizer.TagName()
			tagName := string(tn)
			p.handleEnd(tagName)
			// Pop tag stack
			if len(p.tagStack) > 0 && p.tagStack[len(p.tagStack)-1] == tagName {
				p.tagStack = p.tagStack[:len(p.tagStack)-1]
			}

		case html.TextToken:
			p.handleText(string(p.tokenizer.Text()))

		case html.SelfClosingTagToken:
			tn, hasAttr := p.tokenizer.TagName()
			tagName := string(tn)
			attrs := p.readAttrs(hasAttr)
			// Handle self-closing tags like <br/>
			p.handleStart(tagName, attrs)
			p.handleEnd(tagName)
		}
	}
}

func (p *playParser) readAttrs(hasAttr bool) map[string]string {
	attrs := make(map[string]string)
	if hasAttr {
		for {
			key, val, more := p.tokenizer.TagAttr()
			attrs[string(key)] = string(val)
			if !more {
				break
			}
		}
	}
	return attrs
}

func (p *playParser) handleStart(tag string, attrs map[string]string) {
	epubType := attrs["epub:type"]
	id := attrs["id"]

	switch tag {
	case "section":
		if strings.HasPrefix(id, "scene-") {
			parts := strings.Split(id, "-")
			if len(parts) >= 3 {
				p.currentAct, _ = strconv.Atoi(parts[1])
				p.currentScene, _ = strconv.Atoi(parts[2])
			}
			p.sceneLineCounter = 0
		} else if strings.HasPrefix(id, "act-") {
			parts := strings.Split(id, "-")
			if len(parts) >= 2 {
				p.currentAct, _ = strconv.Atoi(parts[1])
			}
		}
		if strings.Contains(epubType, "prologue") {
			p.currentScene = 0
		} else if strings.Contains(epubType, "epilogue") {
			p.currentScene = 99
		}

	case "h2", "h3", "h4", "hgroup":
		p.inHeader = true

	case "td":
		p.inTD = true
		if strings.Contains(epubType, "z3998:persona") {
			p.inPersona = true
			p.buf.Reset()
		} else {
			p.tdIsSpeech = true
			p.buf.Reset()
			p.verseLines = nil
		}

	case "i":
		if strings.Contains(epubType, "z3998:stage-direction") {
			if p.tdIsSpeech {
				p.nestedStageDir = true
				p.buf.Reset()
			} else {
				p.inStageDir = true
				p.buf.Reset()
			}
		}

	case "span":
		if p.tdIsSpeech && !p.nestedStageDir {
			p.inVerseSpan = true
			p.buf.Reset()
		}
	}
}

func (p *playParser) handleEnd(tag string) {
	switch tag {
	case "td":
		if p.inPersona {
			p.currentCharacter = strings.TrimSpace(p.buf.String())
			p.inPersona = false
			p.inTD = false
		} else if p.tdIsSpeech {
			if len(p.verseLines) > 0 {
				for _, vl := range p.verseLines {
					p.sceneLineCounter++
					charName := p.currentCharacter
					if vl.isStageDirection {
						charName = ""
					}
					p.lines = append(p.lines, PlayLine{
						Act:              p.currentAct,
						Scene:            p.currentScene,
						Character:        charName,
						Text:             vl.text,
						IsStageDirection: vl.isStageDirection,
						LineInScene:      p.sceneLineCounter,
					})
				}
			} else {
				prose := strings.TrimSpace(p.buf.String())
				if prose != "" {
					p.sceneLineCounter++
					p.lines = append(p.lines, PlayLine{
						Act:              p.currentAct,
						Scene:            p.currentScene,
						Character:        p.currentCharacter,
						Text:             prose,
						IsStageDirection: false,
						LineInScene:      p.sceneLineCounter,
					})
				}
			}
			p.verseLines = nil
			p.tdIsSpeech = false
			p.inTD = false
			p.buf.Reset()
		}

	case "span":
		if p.inVerseSpan {
			text := strings.TrimSpace(p.buf.String())
			if text != "" {
				p.verseLines = append(p.verseLines, verseLine{text: text})
			}
			p.inVerseSpan = false
			p.buf.Reset()
		}

	case "i":
		if p.nestedStageDir {
			sd := strings.TrimSpace(p.buf.String())
			if sd != "" {
				p.verseLines = append(p.verseLines, verseLine{text: sd, isStageDirection: true})
			}
			p.nestedStageDir = false
			p.buf.Reset()
		} else if p.inStageDir {
			sd := strings.TrimSpace(p.buf.String())
			if sd != "" {
				p.sceneLineCounter++
				p.lines = append(p.lines, PlayLine{
					Act:              p.currentAct,
					Scene:            p.currentScene,
					Text:             sd,
					IsStageDirection: true,
					LineInScene:      p.sceneLineCounter,
				})
			}
			p.inStageDir = false
			p.buf.Reset()
		}

	case "h2", "h3", "h4", "hgroup":
		p.inHeader = false
	}
}

func (p *playParser) handleText(text string) {
	if p.inHeader {
		return
	}
	if p.inPersona || p.inStageDir || p.inVerseSpan || p.nestedStageDir {
		p.buf.WriteString(text)
	} else if p.tdIsSpeech && p.inTD {
		p.buf.WriteString(text)
	}
}
