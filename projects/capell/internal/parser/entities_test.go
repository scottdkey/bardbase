// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import "testing"

func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "no entities", input: "hello world", want: "hello world"},
		{name: "right single quote", input: "it&#8217;s", want: "it\u2019s"},
		{name: "less than", input: "&#60;tag&#62;", want: "<tag>"},
		{name: "multiple entities", input: "&#65;&#66;&#67;", want: "ABC"},
		{name: "mixed content", input: "Hello &#8212; World", want: "Hello \u2014 World"},
		{name: "invalid entity preserved", input: "&#99999999;", want: "&#99999999;"},
		{name: "non-entity text", input: "no & entities here", want: "no & entities here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeHTMLEntities(tt.input)
			if got != tt.want {
				t.Errorf("DecodeHTMLEntities(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
