// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"math"
	"testing"
)

func TestNormalizeForMatch_Punctuation(t *testing.T) {
	got := NormalizeForMatch("Hello, World! How's it going?")
	want := "hello world hows it going"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForMatch_MixedCase(t *testing.T) {
	got := NormalizeForMatch("To Be Or Not To Be")
	want := "to be or not to be"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForMatch_ExtraWhitespace(t *testing.T) {
	got := NormalizeForMatch("  multiple   spaces  and\ttabs\n\nnewlines  ")
	want := "multiple spaces and tabs newlines"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForMatch_Empty(t *testing.T) {
	got := NormalizeForMatch("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestJaccardSimilarity_Identical(t *testing.T) {
	score := JaccardSimilarity("to be or not to be", "to be or not to be")
	if score != 1.0 {
		t.Errorf("expected 1.0 for identical strings, got %f", score)
	}
}

func TestJaccardSimilarity_CompletelyDifferent(t *testing.T) {
	score := JaccardSimilarity("the cat sat", "dogs run fast")
	if score != 0.0 {
		t.Errorf("expected 0.0 for completely different strings, got %f", score)
	}
}

func TestJaccardSimilarity_PartialOverlap(t *testing.T) {
	// Words: {to, be, or, not} and {to, be, and, die} → intersection=2, union=6
	score := JaccardSimilarity("to be or not", "to be and die")
	expected := 2.0 / 6.0
	if math.Abs(score-expected) > 0.001 {
		t.Errorf("expected ~%f, got %f", expected, score)
	}
}

func TestJaccardSimilarity_BothEmpty(t *testing.T) {
	score := JaccardSimilarity("", "")
	if score != 1.0 {
		t.Errorf("expected 1.0 for both empty, got %f", score)
	}
}

func TestJaccardSimilarity_OneEmpty(t *testing.T) {
	score := JaccardSimilarity("hello world", "")
	if score != 0.0 {
		t.Errorf("expected 0.0 when one is empty, got %f", score)
	}
}

func TestJaccardSimilarity_IgnoresPunctuation(t *testing.T) {
	score := JaccardSimilarity("Hello, world!", "hello world")
	if score != 1.0 {
		t.Errorf("expected 1.0 ignoring punctuation and case, got %f", score)
	}
}

func TestContainsNormalized_Basic(t *testing.T) {
	if !ContainsNormalized("To be, or not to be, that is the question", "not to be") {
		t.Error("expected true for substring match")
	}
}

func TestContainsNormalized_CaseInsensitive(t *testing.T) {
	if !ContainsNormalized("ABANDON the society", "abandon the") {
		t.Error("expected true for case-insensitive match")
	}
}

func TestContainsNormalized_WithPunctuation(t *testing.T) {
	if !ContainsNormalized("left and --ed of his velvet friends,", "his velvet friends") {
		t.Error("expected true ignoring punctuation")
	}
}

func TestContainsNormalized_NoMatch(t *testing.T) {
	if ContainsNormalized("to be or not to be", "hamlet speaks") {
		t.Error("expected false for non-matching strings")
	}
}

func TestContainsNormalized_EmptySubstring(t *testing.T) {
	if ContainsNormalized("some text", "") {
		t.Error("expected false for empty substring")
	}
}

func TestAlignSequences_BothEmpty(t *testing.T) {
	pairs := AlignSequences(nil, nil)
	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs, got %d", len(pairs))
	}
}

func TestAlignSequences_OneEmpty(t *testing.T) {
	linesA := []AlignableLine{
		{ID: 1, Content: "hello world", LineNumber: 1},
		{ID: 2, Content: "goodbye world", LineNumber: 2},
	}
	pairs := AlignSequences(linesA, nil)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	for _, p := range pairs {
		if p.MatchType != "only_a" {
			t.Errorf("expected only_a, got %s", p.MatchType)
		}
		if p.LineA == nil {
			t.Error("expected non-nil LineA")
		}
		if p.LineB != nil {
			t.Error("expected nil LineB")
		}
	}
}

func TestAlignSequences_IdenticalLines(t *testing.T) {
	linesA := []AlignableLine{
		{ID: 1, Content: "to be or not to be", LineNumber: 1},
		{ID: 2, Content: "that is the question", LineNumber: 2},
	}
	linesB := []AlignableLine{
		{ID: 10, Content: "to be or not to be", LineNumber: 1},
		{ID: 11, Content: "that is the question", LineNumber: 2},
	}
	pairs := AlignSequences(linesA, linesB)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	for _, p := range pairs {
		if p.MatchType != "aligned" {
			t.Errorf("expected aligned, got %s", p.MatchType)
		}
		if p.Similarity != 1.0 {
			t.Errorf("expected similarity 1.0, got %f", p.Similarity)
		}
	}
}

func TestAlignSequences_WithGap(t *testing.T) {
	linesA := []AlignableLine{
		{ID: 1, Content: "first line here", LineNumber: 1},
		{ID: 2, Content: "second line here", LineNumber: 2},
		{ID: 3, Content: "third line here", LineNumber: 3},
	}
	linesB := []AlignableLine{
		{ID: 10, Content: "first line here", LineNumber: 1},
		{ID: 11, Content: "third line here", LineNumber: 2},
	}
	pairs := AlignSequences(linesA, linesB)

	// Should have 3 pairs: first matched, second only_a, third matched
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	// First pair: aligned
	if pairs[0].MatchType != "aligned" {
		t.Errorf("pair 0: expected aligned, got %s", pairs[0].MatchType)
	}
	// Middle: should be only_a (second line only in A)
	if pairs[1].MatchType != "only_a" {
		t.Errorf("pair 1: expected only_a, got %s", pairs[1].MatchType)
	}
	// Last: aligned
	if pairs[2].MatchType != "aligned" {
		t.Errorf("pair 2: expected aligned, got %s", pairs[2].MatchType)
	}
}

func TestAlignSequences_CompletelyDifferent(t *testing.T) {
	linesA := []AlignableLine{
		{ID: 1, Content: "apple banana cherry", LineNumber: 1},
	}
	linesB := []AlignableLine{
		{ID: 10, Content: "dog elephant fox", LineNumber: 1},
	}
	pairs := AlignSequences(linesA, linesB)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].MatchType != "modified" {
		t.Errorf("expected modified for completely different content, got %s", pairs[0].MatchType)
	}
}
