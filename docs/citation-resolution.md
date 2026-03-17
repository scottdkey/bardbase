# Citation Resolution

**Source file**: `projects/capell/internal/importer/citations.go`
**Output table**: `citation_matches`

Schmidt's *Shakespeare Lexicon* contains ~200,000 citations in the form `Ham. III, 2, 47` — a work abbreviation, act, scene, and line number. The citation resolution phase links each of these to an actual `text_lines` row so that users can jump from a lexicon definition directly to the passage in the play.

## The Problem

Schmidt's line numbers follow the Globe edition. The `perseus_globe` edition is the most authoritative match because Perseus carries Globe line numbers directly from `<l n="...">` TEI attributes. Citation resolution therefore works exclusively against `perseus_globe`.

Complications:
- Line numbers in Schmidt are sometimes off by ±1–5 (different counting conventions for stage directions)
- Some citations omit act/scene (poetry, sonnets)
- Short quotes may appear in multiple lines
- Some citations have no quote at all (headword-only references)

## Resolution Pipeline

Each `lexicon_citations` row passes through the following strategies **in order**. The first strategy that produces a confident match wins; the row is inserted into `citation_matches` and the cascade stops.

### Strategy 1 — Exact Quote Match (confidence 1.0)

If `quote_text` is non-empty, search all `text_lines` rows in the correct work+act+scene for a line whose `content` contains the quoted text (case-insensitive, after normalizing special characters).

A "nearby" window of ±10 lines around the stated Schmidt line number is searched first. If not found there, the whole scene is scanned.

**Confidence**: `1.0`

### Strategy 2 — Exact Line Number (confidence 0.9)

If the act, scene, and line number from Schmidt resolve to exactly one `text_lines` row in `perseus_globe`, use it.

**Confidence**: `0.9`

### Strategy 3 — Fuzzy Text Match (confidence 0.7)

If the quote is non-empty but wasn't found by exact substring, compute **Levenshtein edit distance** between the quote and each candidate line's content. Accept the closest match if the edit distance is within a threshold proportional to the quote length.

Special-character normalization is applied before comparison: smart quotes (`"`, `"`, `'`, `'`) are replaced with straight quotes; em-dashes with hyphens; etc.

**Confidence**: `0.7`

### Strategy 4 — Forward / Backward Propagation (confidence 0.5)

If act/scene/line are known but no text match was found, propagate from a previously resolved citation in the same entry:

- **Forward**: if the last resolved citation in this entry was `N` lines earlier in the same scene, assume the current citation is approximately `N` lines after that match.
- **Backward**: symmetrically, scan backward from a known later citation.

This handles clusters of citations where Schmidt's line counting drifts slightly from Perseus's.

**Confidence**: `0.5`

### Strategy 5 — Headword Fallback (confidence 0.3)

If the citation provides no act/scene (e.g. sonnet references or heavily abbreviated citations) and no quote text, attempt to find any line in the work that contains the lexicon headword.

**Confidence**: `0.3`

### Unresolved

If no strategy succeeds, the citation is left without a `citation_matches` row. The `lexicon_citations` row still exists with its raw Schmidt data; it simply has no confirmed text-line link.

## Normalization Details

Before any text comparison, both the Schmidt quote and the candidate line content are normalized:

```
lower-case
strip leading/trailing whitespace
collapse internal whitespace to single spaces
smart quotes → straight quotes  (" " ' ' → " ')
em-dash / en-dash → hyphen      (— – → -)
ellipsis → ...                  (… → ...)
```

This normalization is applied consistently across all five strategies.

## Output

```sql
-- All confirmed matches for a lexicon entry
SELECT lc.work_abbrev, lc.act, lc.scene, lc.line, lc.quote_text,
       cm.match_type, cm.confidence, cm.matched_text,
       tl.content
FROM lexicon_citations lc
JOIN citation_matches cm ON cm.citation_id = lc.id
JOIN text_lines tl ON tl.id = cm.text_line_id
WHERE lc.entry_id = ?
ORDER BY lc.id;
```

## Confidence Levels Summary

| Strategy | `match_type` | `confidence` |
|---|---|---|
| Exact quote substring | `exact` | 1.0 |
| Exact line number | `positional` | 0.9 |
| Fuzzy edit distance | `fuzzy` | 0.7 |
| Propagation from neighbor | `positional` | 0.5 |
| Headword fallback | `fuzzy` | 0.3 |
