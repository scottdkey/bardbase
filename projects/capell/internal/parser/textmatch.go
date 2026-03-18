// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"strings"
	"unicode"
)

// normalizeSpecialChars replaces early-modern print variants with ASCII equivalents
// before the main normalization pass removes punctuation.
func normalizeSpecialChars(s string) string {
	// Ligatures (common in early-print OCR)
	s = strings.ReplaceAll(s, "ﬁ", "fi")
	s = strings.ReplaceAll(s, "ﬂ", "fl")
	s = strings.ReplaceAll(s, "ﬀ", "ff")
	s = strings.ReplaceAll(s, "ﬃ", "ffi")
	s = strings.ReplaceAll(s, "ﬄ", "ffl")
	s = strings.ReplaceAll(s, "ﬅ", "st")
	s = strings.ReplaceAll(s, "ﬆ", "st")
	// Long s (ſ → s)
	s = strings.ReplaceAll(s, "ſ", "s")
	s = strings.ReplaceAll(s, "ſ", "s") // U+017F long s
	// Modifier-letter apostrophe and curved quotes → plain apostrophe
	s = strings.ReplaceAll(s, "\u02BC", "'")
	s = strings.ReplaceAll(s, "\u02BB", "'")
	s = strings.ReplaceAll(s, "\u2019", "'")
	s = strings.ReplaceAll(s, "\u2018", "'")
	// Early-modern I/J and U/V interchangeability (lowercase after ToLower pass)
	// Handled after lowercasing in the main loop below.
	return s
}

// NormalizeForMatch lowercases text, removes punctuation, and normalizes whitespace.
// Also normalises early-modern print variants (ligatures, long-s, u/v interchange,
// and -ie endings) so that First Folio and Q1 quarto spellings match modern editions.
// Applied identically to both sides of a comparison, so normalized forms need not
// be "correct" modern English — only consistent.
func NormalizeForMatch(s string) string {
	s = normalizeSpecialChars(s)
	s = strings.ToLower(s)
	// After lowercasing, map archaic letter variants used in early-modern printing.
	s = strings.ReplaceAll(s, "vv", "w") // vv → w (rare but present in OCR)
	// u/v interchange: early-modern printing used 'v' word-initially (even for vowel
	// sound) and 'u' medially (even for consonant sound).  Normalise all 'v' to 'u'
	// on both sides so that  have↔haue, give↔giue, upon↔vpon, love↔loue all match.
	s = strings.ReplaceAll(s, "v", "u")
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	// Word-final -ie → -y: early-modern '-ie' endings (beautie, mercie, pittie, trie)
	// normalise to the modern '-y' form.  Applied to both sides, so modern 'mercy' and
	// FF 'mercie' both produce 'mercy'.  Skip words ≤3 chars: 'die', 'lie', 'pie' are
	// identical in both periods and should not be transformed.
	words := strings.Fields(result.String())
	for i, w := range words {
		if len(w) > 3 && strings.HasSuffix(w, "ie") {
			words[i] = w[:len(w)-2] + "y"
		}
	}
	return strings.Join(words, " ")
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

// ContainsWordPrefix checks if any word in text starts with the given prefix
// (after normalization). Requires the prefix to be at least 4 characters to
// avoid false positives. This handles inflected forms like baby→babies,
// dance→dancing, mercury→mercuries.
func ContainsWordPrefix(text, prefix string) bool {
	prefix = NormalizeForMatch(prefix)
	if len(prefix) < 4 {
		return false
	}
	for _, word := range strings.Fields(NormalizeForMatch(text)) {
		if strings.HasPrefix(word, prefix) {
			return true
		}
	}
	return false
}

// ContainsStemPrefix checks if any word in text starts with a shortened stem
// of the given word (all but the last character, after normalization).
// This catches English inflection patterns where the stem changes:
//   - "mercury" → "mercur" matches "mercuries" (y→ies)
//   - "baby" → "bab" matches "babies" (y→ies)
//   - "gape" → "gap" matches "gaping" (e→ing)
//   - "dance" → "danc" matches "dancing" (e→ing)
//
// Requires the shortened stem to be at least 3 characters.
func ContainsStemPrefix(text, word string) bool {
	word = NormalizeForMatch(word)
	if len(word) < 4 {
		return false // need at least 4 chars to produce a 3-char stem
	}
	stem := word[:len(word)-1]
	for _, w := range strings.Fields(NormalizeForMatch(text)) {
		if strings.HasPrefix(w, stem) && w != stem {
			return true
		}
	}
	return false
}

// AlignableLine represents a text line for sequence alignment.
// Words is a pre-computed normalized word set used by AlignSequences to avoid
// re-normalizing each line O(n×m) times when building the similarity matrix.
// If Words is nil, JaccardSimilarity falls back to computing it on the fly.
type AlignableLine struct {
	ID         int64
	Content    string
	LineNumber int
	Words      map[string]bool
}

// jaccardFromSets computes the Jaccard index from two pre-computed word sets.
// Callers should use this instead of JaccardSimilarity when word sets are available.
func jaccardFromSets(setA, setB map[string]bool) float64 {
	if len(setA) == 0 && len(setB) == 0 {
		return 1.0
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

	// For very large sequences, fall back to simple 1:1 alignment to bound
	// memory and compute. The limit is a cell count (n×m) rather than a
	// per-sequence length so that flat-vs-structured play comparisons
	// (e.g., Q1 Hamlet ~2000 lines × SE Hamlet ~3500 lines = 7M cells) can
	// still use Needleman-Wunsch while truly enormous tasks fall back.
	// 8M cells × 8 bytes × 3 matrices (sim, dp, trace) ≈ 192 MB per task.
	if int64(n)*int64(m) > 8_000_000 {
		return simpleAlign(linesA, linesB)
	}

	gapPenalty := -0.1

	// Pre-compute similarity matrix.
	// Use pre-computed word sets when available (set by buildLineCache) to avoid
	// re-normalizing each line O(n×m) times — one normalize per line instead.
	sim := make([][]float64, n)
	for i := 0; i < n; i++ {
		sim[i] = make([]float64, m)
		for j := 0; j < m; j++ {
			if linesA[i].Words != nil && linesB[j].Words != nil {
				sim[i][j] = jaccardFromSets(linesA[i].Words, linesB[j].Words)
			} else {
				sim[i][j] = JaccardSimilarity(linesA[i].Content, linesB[j].Content)
			}
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
			// Subtract gapPenalty magnitude so that 0-similarity pairs score the
		// same as a gap rather than always winning the diagonal. Lines with
		// Jaccard > |gapPenalty| (0.1) are preferred for alignment; lines below
		// that threshold are treated no better than a gap insertion.
		matchScore := dp[i-1][j-1] + sim[i-1][j-1] + gapPenalty
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
// Uses pre-computed word sets (AlignableLine.Words) when available to avoid
// re-normalizing every line on each call.
func simpleAlign(linesA, linesB []AlignableLine) []AlignedPair {
	n := len(linesA)
	m := len(linesB)
	maxLen := max(n, m)

	pairs := make([]AlignedPair, 0, maxLen)
	for k := 0; k < maxLen; k++ {
		var pair AlignedPair
		if k < n && k < m {
			a := linesA[k]
			b := linesB[k]
			var s float64
			if a.Words != nil && b.Words != nil {
				s = jaccardFromSets(a.Words, b.Words)
			} else {
				s = JaccardSimilarity(a.Content, b.Content)
			}
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
