// Package parser contains pure parsing functions for all source data formats.
package parser

import (
	"fmt"
	"regexp"
	"strconv"
)

var htmlEntityRe = regexp.MustCompile(`&#(\d+);`)

// DecodeHTMLEntities converts HTML numeric character references (e.g. &#8217;) to their
// corresponding Unicode characters.
func DecodeHTMLEntities(text string) string {
	if text == "" {
		return text
	}
	return htmlEntityRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := htmlEntityRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		code, err := strconv.Atoi(sub[1])
		if err != nil || code < 0 || code > 0x10FFFF {
			return match
		}
		return fmt.Sprintf("%c", rune(code))
	})
}
