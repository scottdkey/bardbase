// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strings"
	"unicode"
)

// NormalizeForMatch lowercases text, removes punctuation, and normalizes whitespace.
// Used for fuzzy text comparison between editions and citation matching.
func NormalizeForMatch(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(result.String()), " ")
}

// WordSet splits normalized text into a set of unique words.
func WordSet(s string) map[string]bool {
	words := strings.Fields(NormalizeForMatch(s))
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[w] = true
	}
	return set
}

// JaccardSimilarity computes the Jaccard index between two strings
// based on their word sets. Returns 0.0-1.0 (1.0 = identical word sets).
func JaccardSimilarity(a, b string) float64 {
	setA := WordSet(a)
	setB := WordSet(b)

	if len(setA) == 0 && len(setB) == 0 {
		return 1.0 // both empty = identical
	}
	if len(setA) == 0 || len(setB) == 0 {
		return 0.0
	}

	intersection := 0
	for w := range setA {
		if setB[w] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

// ContainsNormalized checks if the normalized text contains the normalized substring.
func ContainsNormalized(text, substring string) bool {
	if substring == "" {
		return false
	}
	return strings.Contains(NormalizeForMatch(text), NormalizeForMatch(substring))
}

// AlignableLine represents a text line for sequence alignment.
type AlignableLine struct {
	ID         int64
	Content    string
	LineNumber int
}

// AlignedPair represents a matched or unmatched line pair from two aligned sequences.
type AlignedPair struct {
	LineA      *AlignableLine // nil if only in B
	LineB      *AlignableLine // nil if only in A
	MatchType  string         // "aligned", "modified", "only_a", "only_b"
	Similarity float64
}

// AlignSequences performs Needleman-Wunsch global alignment on two sequences of text lines.
// Returns aligned pairs in display order with gap handling for insertions/deletions.
// Uses Jaccard word similarity as the scoring function.
func AlignSequences(linesA, linesB []AlignableLine) []AlignedPair {
	n := len(linesA)
	m := len(linesB)

	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		pairs := make([]AlignedPair, m)
		for j := 0; j < m; j++ {
			b := linesB[j]
			pairs[j] = AlignedPair{LineB: &b, MatchType: "only_b"}
		}
		return pairs
	}
	if m == 0 {
		pairs := make([]AlignedPair, n)
		for i := 0; i < n; i++ {
			a := linesA[i]
			pairs[i] = AlignedPair{LineA: &a, MatchType: "only_a"}
		}
		return pairs
	}

	// For very large scenes, fall back to simple 1:1 alignment
	if n > 500 || m > 500 {
		return simpleAlign(linesA, linesB)
	}

	gapPenalty := -0.1

	// Pre-compute similarity matrix
	sim := make([][]float64, n)
	for i := 0; i < n; i++ {
		sim[i] = make([]float64, m)
		for j := 0; j < m; j++ {
			sim[i][j] = JaccardSimilarity(linesA[i].Content, linesB[j].Content)
		}
	}

	// DP table
	dp := make([][]float64, n+1)
	for i := range dp {
		dp[i] = make([]float64, m+1)
	}
	for i := 1; i <= n; i++ {
		dp[i][0] = float64(i) * gapPenalty
	}
	for j := 1; j <= m; j++ {
		dp[0][j] = float64(j) * gapPenalty
	}

	// Traceback direction: 0=diagonal(match), 1=up(gap in B), 2=left(gap in A)
	trace := make([][]int, n+1)
	for i := range trace {
		trace[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		trace[i][0] = 1
	}
	for j := 1; j <= m; j++ {
		trace[0][j] = 2
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			matchScore := dp[i-1][j-1] + sim[i-1][j-1]
			gapA := dp[i-1][j] + gapPenalty
			gapB := dp[i][j-1] + gapPenalty

			if matchScore >= gapA && matchScore >= gapB {
				dp[i][j] = matchScore
				trace[i][j] = 0
			} else if gapA >= gapB {
				dp[i][j] = gapA
				trace[i][j] = 1
			} else {
				dp[i][j] = gapB
				trace[i][j] = 2
			}
		}
	}

	// Traceback (builds in reverse order)
	var pairs []AlignedPair
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && trace[i][j] == 0 {
			a := linesA[i-1]
			b := linesB[j-1]
			s := sim[i-1][j-1]
			mt := "aligned"
			if s < 0.2 {
				mt = "modified"
			}
			pairs = append(pairs, AlignedPair{LineA: &a, LineB: &b, MatchType: mt, Similarity: s})
			i--
			j--
		} else if i > 0 && (j == 0 || trace[i][j] == 1) {
			a := linesA[i-1]
			pairs = append(pairs, AlignedPair{LineA: &a, MatchType: "only_a"})
			i--
		} else {
			b := linesB[j-1]
			pairs = append(pairs, AlignedPair{LineB: &b, MatchType: "only_b"})
			j--
		}
	}

	// Reverse to get display order
	for left, right := 0, len(pairs)-1; left < right; left, right = left+1, right-1 {
		pairs[left], pairs[right] = pairs[right], pairs[left]
	}

	return pairs
}

// simpleAlign performs 1:1 positional alignment for very large sequences.
func simpleAlign(linesA, linesB []AlignableLine) []AlignedPair {
	n := len(linesA)
	m := len(linesB)
	maxLen := n
	if m > maxLen {
		maxLen = m
	}

	pairs := make([]AlignedPair, 0, maxLen)
	for k := 0; k < maxLen; k++ {
		var pair AlignedPair
		if k < n && k < m {
			a := linesA[k]
			b := linesB[k]
			s := JaccardSimilarity(a.Content, b.Content)
			mt := "aligned"
			if s < 0.2 {
				mt = "modified"
			}
			pair = AlignedPair{LineA: &a, LineB: &b, MatchType: mt, Similarity: s}
		} else if k < n {
			a := linesA[k]
			pair = AlignedPair{LineA: &a, MatchType: "only_a"}
		} else {
			b := linesB[k]
			pair = AlignedPair{LineB: &b, MatchType: "only_b"}
		}
		pairs = append(pairs, pair)
	}
	return pairs
}
