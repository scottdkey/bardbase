package parser

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// PoemLine represents a single line from a Standard Ebooks poetry file.
type PoemLine struct {
	Text       string
	LineNumber int
	Stanza     int
}

// SonnetData holds the parsed result of a sonnets XHTML file.
type SonnetData struct {
	Sonnets          map[int][]PoemLine // sonnet number → lines
	LoversComplaint  []PoemLine
}

// ParseSEPoetry parses a Standard Ebooks poetry XHTML file.
// Returns a map of article ID → lines.
func ParseSEPoetry(xhtml string) map[string][]PoemLine {
	tokenizer := html.NewTokenizer(strings.NewReader(xhtml))
	p := &poetryParser{
		tokenizer: tokenizer,
		poems:     make(map[string][]PoemLine),
	}
	p.parse()
	return p.poems
}

// ParseSESonnets parses a Standard Ebooks sonnets XHTML file.
// Returns structured sonnet data including individual sonnets and A Lover's Complaint.
func ParseSESonnets(xhtml string) *SonnetData {
	tokenizer := html.NewTokenizer(strings.NewReader(xhtml))
	p := &sonnetParser{
		tokenizer: tokenizer,
		sonnets:   make(map[int][]PoemLine),
	}
	p.parse()
	return &SonnetData{
		Sonnets:         p.sonnets,
		LoversComplaint: p.loversComplaint,
	}
}

// ---- Poetry parser ----

type poetryParser struct {
	tokenizer *html.Tokenizer
	poems     map[string][]PoemLine

	currentArticle string
	buf            strings.Builder
	inSpan         bool
	inHeader       bool
	inDedication   bool
	lineCounter    int
	stanzaCounter  int
	tagStack       []string
}

func (p *poetryParser) parse() {
	for {
		tt := p.tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return

		case html.StartTagToken:
			tn, hasAttr := p.tokenizer.TagName()
			tag := string(tn)
			attrs := readAttrsGeneric(p.tokenizer, hasAttr)
			p.tagStack = append(p.tagStack, tag)
			p.handleStart(tag, attrs)

		case html.EndTagToken:
			tn, _ := p.tokenizer.TagName()
			tag := string(tn)
			p.handleEnd(tag)
			if len(p.tagStack) > 0 && p.tagStack[len(p.tagStack)-1] == tag {
				p.tagStack = p.tagStack[:len(p.tagStack)-1]
			}

		case html.TextToken:
			if p.inSpan {
				p.buf.WriteString(string(p.tokenizer.Text()))
			}
		}
	}
}

func (p *poetryParser) handleStart(tag string, attrs map[string]string) {
	epubType := attrs["epub:type"]
	id := attrs["id"]

	switch tag {
	case "article":
		p.currentArticle = id
		p.poems[id] = nil
		p.lineCounter = 0
		p.stanzaCounter = 0

	case "section":
		if strings.Contains(id, "dedication") || strings.Contains(epubType, "dedication") {
			p.inDedication = true
		}

	case "h2", "h3", "h4", "header", "hgroup":
		p.inHeader = true

	case "span":
		if p.currentArticle != "" && !p.inHeader && !p.inDedication {
			p.inSpan = true
			p.buf.Reset()
		}

	case "p":
		if p.currentArticle != "" && !p.inHeader && !p.inDedication {
			p.stanzaCounter++
		}
	}
}

func (p *poetryParser) handleEnd(tag string) {
	switch tag {
	case "span":
		if p.inSpan {
			text := strings.TrimSpace(p.buf.String())
			if text != "" && p.currentArticle != "" {
				p.lineCounter++
				p.poems[p.currentArticle] = append(p.poems[p.currentArticle], PoemLine{
					Text:       text,
					LineNumber: p.lineCounter,
					Stanza:     p.stanzaCounter,
				})
			}
			p.inSpan = false
			p.buf.Reset()
		}

	case "article":
		p.currentArticle = ""
		p.inDedication = false

	case "section":
		if p.inDedication {
			p.inDedication = false
		}

	case "h2", "h3", "h4", "header", "hgroup":
		p.inHeader = false
	}
}

// ---- Sonnet parser ----

type sonnetParser struct {
	tokenizer *html.Tokenizer
	sonnets   map[int][]PoemLine
	loversComplaint []PoemLine

	currentArticle    string
	currentSonnetNum  int
	isLoversComplaint bool
	buf               strings.Builder
	inSpan            bool
	inHeader          bool
	lineCounter       int
	stanzaCounter     int
	tagStack          []string
}

func (p *sonnetParser) parse() {
	for {
		tt := p.tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return

		case html.StartTagToken:
			tn, hasAttr := p.tokenizer.TagName()
			tag := string(tn)
			attrs := readAttrsGeneric(p.tokenizer, hasAttr)
			p.tagStack = append(p.tagStack, tag)
			p.handleStart(tag, attrs)

		case html.EndTagToken:
			tn, _ := p.tokenizer.TagName()
			tag := string(tn)
			p.handleEnd(tag)
			if len(p.tagStack) > 0 && p.tagStack[len(p.tagStack)-1] == tag {
				p.tagStack = p.tagStack[:len(p.tagStack)-1]
			}

		case html.TextToken:
			if p.inSpan {
				p.buf.WriteString(string(p.tokenizer.Text()))
			}
		}
	}
}

func (p *sonnetParser) handleStart(tag string, attrs map[string]string) {
	id := attrs["id"]

	switch tag {
	case "article":
		p.currentArticle = id
		p.lineCounter = 0
		p.stanzaCounter = 0
		p.currentSonnetNum = 0
		p.isLoversComplaint = false

		if strings.HasPrefix(id, "sonnet-") {
			parts := strings.Split(id, "-")
			if len(parts) >= 2 {
				if num, err := strconv.Atoi(parts[1]); err == nil {
					p.currentSonnetNum = num
					p.sonnets[num] = nil
				}
			}
		} else if strings.Contains(strings.ToLower(id), "lover") ||
			strings.Contains(strings.ToLower(id), "complaint") {
			p.isLoversComplaint = true
		}

	case "h2", "h3", "h4", "header", "hgroup":
		p.inHeader = true

	case "span":
		if p.currentArticle != "" && !p.inHeader {
			p.inSpan = true
			p.buf.Reset()
		}

	case "p":
		if p.currentArticle != "" && !p.inHeader {
			p.stanzaCounter++
		}
	}
}

func (p *sonnetParser) handleEnd(tag string) {
	switch tag {
	case "span":
		if p.inSpan {
			text := strings.TrimSpace(p.buf.String())
			if text != "" {
				p.lineCounter++
				line := PoemLine{
					Text:       text,
					LineNumber: p.lineCounter,
					Stanza:     p.stanzaCounter,
				}
				if p.isLoversComplaint {
					p.loversComplaint = append(p.loversComplaint, line)
				} else if p.currentSonnetNum > 0 {
					p.sonnets[p.currentSonnetNum] = append(p.sonnets[p.currentSonnetNum], line)
				}
			}
			p.inSpan = false
			p.buf.Reset()
		}

	case "article":
		p.currentArticle = ""
		p.currentSonnetNum = 0
		p.isLoversComplaint = false

	case "h2", "h3", "h4", "header", "hgroup":
		p.inHeader = false
	}
}

// readAttrsGeneric reads all attributes from the tokenizer into a map.
func readAttrsGeneric(tokenizer *html.Tokenizer, hasAttr bool) map[string]string {
	attrs := make(map[string]string)
	if hasAttr {
		for {
			key, val, more := tokenizer.TagAttr()
			attrs[string(key)] = string(val)
			if !more {
				break
			}
		}
	}
	return attrs
}
