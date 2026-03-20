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
	// Early-modern I/J and U/V interchangeability are handled after
	// lowercasing in NormalizeForMatch (v→u, j→i).
	return s
}

// elisionExpander expands common verse elisions before punctuation is stripped.
// These contracted forms appear in the First Folio and early quartos; without
// expansion, "o' th' castle" → "o th castl" while "of the castle" → "of the castl"
// — different word sets despite identical meaning.
//
// Patterns are ordered longest-first so that "o' th'" wins over "o'" or "th'"
// when they appear together. Must be applied after lowercasing and v→u/j→i
// while the apostrophe is still present.
var elisionExpander = strings.NewReplacer(
	"o' th'", "of the",
	"i' th'", "in the",
	"o'th'", "of the",
	"i'th'", "in the",
	"o' t'", "of the",
	"i' t'", "in the",
	"o' ", "of ",
	"o'", "of",
	"th' ", "the ",
	"th'", "the",
	"i' ", "in ",
	"'em", "them",
	"'gainst", "against",
	"'pon", "upon",
)

// NormalizeForMatch lowercases text, removes punctuation, and normalizes whitespace.
// Also normalises early-modern print variants (ligatures, long-s, u/v interchange,
// i/j interchange, silent terminal -e, -ick endings, and -ie endings) so that
// First Folio and Q1 quarto spellings match modern editions.
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
	// i/j interchange: early-modern printing used 'i' where modern English uses 'j'
	// (Iuliet↔Juliet, ioy↔joy, iust↔just).  Normalise all 'j' to 'i' on both sides.
	s = strings.ReplaceAll(s, "j", "i")
	// Elision expansion: common contractions in verse and the First Folio.
	// Applied before punctuation removal so the apostrophe disambiguates the form
	// ("o'" = "of", "th'" = "the").  Both sides get the same expansion so the
	// normalization is consistent even when one side uses the elided form and the
	// other uses the full form.
	s = elisionExpander.Replace(s)
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	// Word-level spelling normalizations applied in order.  Each rule feeds
	// into the next so that e.g. "musicke" → strip -e → "musick" → -ick → "music".
	words := strings.Fields(result.String())
	for i, w := range words {
		// -ie → -y: beautie→beauty, mercie→mercy.  Skip ≤3 chars (die, lie, pie).
		if len(w) > 3 && strings.HasSuffix(w, "ie") {
			w = w[:len(w)-2] + "y"
			words[i] = w
		}
		// Strip silent terminal -e after a consonant: speake→speak, looke→look,
		// turne→turn, beene→been.  The consonant check avoids stripping from
		// words like "arriue" (v→u) where the -e follows a vowel.
		// Skip ≤4 chars to protect short words (done, gone, come, etc.).
		if len(w) > 4 && w[len(w)-1] == 'e' && isConsonant(w[len(w)-2]) {
			w = w[:len(w)-1]
			words[i] = w
		}
		// -ick → -ic: musick→music, tragick→tragic, publick→public.
		// Applied after terminal-e strip so "musicke" → "musick" → "music".
		// Skip ≤4 chars to preserve kick, sick, pick (both sides transform
		// consistently so correctness is maintained either way).
		if len(w) > 4 && strings.HasSuffix(w, "ick") {
			w = w[:len(w)-1]
			words[i] = w
		}
		// -cion → -tion: perfeccion→perfection, instruccion→instruction.
		if len(w) > 5 && strings.HasSuffix(w, "cion") {
			w = w[:len(w)-4] + "tion"
			words[i] = w
		}
		// -ll doubling at word end: hee→he is handled below; here handle
		// double-l patterns: wil→will, al→all, til→till, ful→full.
		// Only add trailing -l for short words (≤4) ending in a single l
		// after a vowel, to avoid over-matching.
		// Actually safer: normalize double-l to single-l on both sides:
		// well→wel, all→al, shall→shal. Applied consistently so both sides match.
		if len(w) > 2 && w[len(w)-1] == 'l' && w[len(w)-2] == 'l' {
			w = w[:len(w)-1]
			words[i] = w
		}
		// Pronoun/verb ee-doubling: hee→he, shee→she, mee→me, wee→we, bee→be.
		// These are extremely common in the First Folio. Only apply to short
		// words (≤5 chars) ending in -ee to avoid mangling words like "three",
		// "free", "agree" (which normalize consistently on both sides anyway).
		if len(w) >= 3 && len(w) <= 5 && strings.HasSuffix(w, "ee") {
			w = w[:len(w)-1]
			words[i] = w
		}
	}
	return strings.Join(words, " ")
}

// isConsonant returns true for lowercase ASCII consonants.
// After v→u and j→i normalisation, 'v' and 'j' never appear in normalized text.
func isConsonant(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return false
	default:
		return b >= 'a' && b <= 'z'
	}
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
	ID          int64
	Content     string
	LineNumber  int
	Words       map[string]bool
	ContentType string // "speech", "stage_direction", "verse", etc.
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

// AlignOptions configures per-pair tuning for the alignment algorithm.
type AlignOptions struct {
	// LineNumberAffinity adds a similarity bonus when both lines have the same
	// (or very close) line numbers. Use for edition pairs that share a common
	// numbering scheme (e.g. oss_globe↔perseus_globe). A value of 0.15 means
	// an exact line-number match adds 0.15 to the Jaccard score; a ±1-3 line
	// offset adds half that. Zero disables the bonus entirely.
	LineNumberAffinity float64
}

// alignedThreshold returns the minimum Jaccard similarity for a pair to be
// classified as "aligned" rather than "modified".
//
//   - Stage directions use a flat 0.1 threshold — they often differ in wording
//     between editions even when they describe the same action (e.g. "Enter Hamlet."
//     vs "Enter HAMLET, Prince of Denmark.").
//   - Short dialogue lines (1-3 words) use 0.1 — a single spelling variant in a
//     two-word line drops Jaccard to 0.33, even when the lines clearly correspond.
//   - Medium lines (4 words) use 0.15.
//   - Longer lines use 0.2.
func alignedThreshold(a, b *AlignableLine) float64 {
	if a.ContentType == "stage_direction" || b.ContentType == "stage_direction" {
		return 0.1
	}
	minWords := len(a.Words)
	if len(b.Words) < minWords {
		minWords = len(b.Words)
	}
	if minWords <= 2 {
		return 0.1
	}
	if minWords <= 4 {
		return 0.15
	}
	return 0.2
}

// AlignSequences performs Needleman-Wunsch global alignment on two sequences of text lines.
// Returns aligned pairs in display order with gap handling for insertions/deletions.
// Uses Jaccard word similarity as the scoring function.
//
// opts is optional; when omitted the default zero value is used (no line-number
// affinity). Passing a non-zero AlignOptions enables per-pair tuning.
func AlignSequences(linesA, linesB []AlignableLine, opts ...AlignOptions) []AlignedPair {
	var opt AlignOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	n := len(linesA)
	m := len(linesB)

	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		pairs := make([]AlignedPair, m)
		for j := range m {
			b := linesB[j]
			pairs[j] = AlignedPair{LineB: &b, MatchType: "only_b"}
		}
		return pairs
	}
	if m == 0 {
		pairs := make([]AlignedPair, n)
		for i := range n {
			a := linesA[i]
			pairs[i] = AlignedPair{LineA: &a, MatchType: "only_a"}
		}
		return pairs
	}

	// For very large sequences, fall back to simple 1:1 alignment to bound
	// memory and compute. The limit is a cell count (n×m) rather than a
	// per-sequence length so that flat-vs-structured play comparisons
	// (e.g., Q1 Hamlet ~2000 lines × SE Hamlet ~3500 lines = 7M cells, and
	// Coriolanus OSS ~4000 × First Folio ~3400 = 13.6M cells) still use
	// Needleman-Wunsch while truly enormous tasks fall back.
	// 15M cells: sim(120MB float64) + dp(120MB float64) + trace(15MB int8) ≈ 255MB per task.
	if int64(n)*int64(m) > 15_000_000 {
		return simpleAlign(linesA, linesB, opt)
	}

	gapPenalty := -0.1

	// Pre-compute similarity matrix.
	// Use pre-computed word sets when available (set by buildLineCache) to avoid
	// re-normalizing each line O(n×m) times — one normalize per line instead.
	sim := make([][]float64, n)
	for i := range n {
		sim[i] = make([]float64, m)
		for j := range m {
			if linesA[i].Words != nil && linesB[j].Words != nil {
				sim[i][j] = jaccardFromSets(linesA[i].Words, linesB[j].Words)
			} else {
				sim[i][j] = JaccardSimilarity(linesA[i].Content, linesB[j].Content)
			}
			// Line-number affinity: for edition pairs sharing a numbering scheme
			// (e.g. oss_globe↔perseus_globe), add a bonus when line numbers are
			// close. This anchors the NW traceback and prevents drift in scenes
			// where many consecutive lines share common short words.
			if opt.LineNumberAffinity > 0 &&
				linesA[i].LineNumber > 0 && linesB[j].LineNumber > 0 {
				delta := linesA[i].LineNumber - linesB[j].LineNumber
				if delta < 0 {
					delta = -delta
				}
				var bonus float64
				switch {
				case delta == 0:
					bonus = opt.LineNumberAffinity
				case delta <= 3:
					bonus = opt.LineNumberAffinity * 0.5
				}
				if bonus > 0 {
					sim[i][j] += bonus
					if sim[i][j] > 1.0 {
						sim[i][j] = 1.0
					}
				}
			}
		}
	}

	// DP table
	dp := make([][]float64, n+1)
	for i := range dp {
		dp[i] = make([]float64, m+1)
	}
	for i := range n {
		dp[i+1][0] = float64(i+1) * gapPenalty
	}
	for j := range m {
		dp[0][j+1] = float64(j+1) * gapPenalty
	}

	// Traceback direction stored as int8: 0=diagonal(match), 1=up(gap in B), 2=left(gap in A).
	// int8 saves 7 bytes/cell vs int on 64-bit — matters at 10M+ cells.
	trace := make([][]int8, n+1)
	for i := range trace {
		trace[i] = make([]int8, m+1)
	}
	for i := range n {
		trace[i+1][0] = 1
	}
	for j := range m {
		trace[0][j+1] = 2
	}

	for i := range n {
		for j := range m {
			// Subtract gapPenalty magnitude so that 0-similarity pairs score the
			// same as a gap rather than always winning the diagonal. Lines with
			// Jaccard > |gapPenalty| (0.1) are preferred for alignment; lines below
			// that threshold are treated no better than a gap insertion.
			matchScore := dp[i][j] + sim[i][j] + gapPenalty
			gapA := dp[i][j+1] + gapPenalty
			gapB := dp[i+1][j] + gapPenalty

			if matchScore >= gapA && matchScore >= gapB {
				dp[i+1][j+1] = matchScore
				trace[i+1][j+1] = 0
			} else if gapA >= gapB {
				dp[i+1][j+1] = gapA
				trace[i+1][j+1] = 1
			} else {
				dp[i+1][j+1] = gapB
				trace[i+1][j+1] = 2
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
			if s < alignedThreshold(&a, &b) {
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
func simpleAlign(linesA, linesB []AlignableLine, opt AlignOptions) []AlignedPair {
	n := len(linesA)
	m := len(linesB)
	maxLen := max(n, m)

	pairs := make([]AlignedPair, 0, maxLen)
	for k := range maxLen {
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
			// Apply line-number affinity in the simple aligner too.
			if opt.LineNumberAffinity > 0 && a.LineNumber > 0 && b.LineNumber > 0 {
				delta := a.LineNumber - b.LineNumber
				if delta < 0 {
					delta = -delta
				}
				var bonus float64
				switch {
				case delta == 0:
					bonus = opt.LineNumberAffinity
				case delta <= 3:
					bonus = opt.LineNumberAffinity * 0.5
				}
				if bonus > 0 {
					s += bonus
					if s > 1.0 {
						s = 1.0
					}
				}
			}
			mt := "aligned"
			if s < alignedThreshold(&a, &b) {
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
