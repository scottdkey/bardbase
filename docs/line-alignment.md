# Cross-Edition Line Alignment

**Source file**: `projects/capell/internal/importer/mappings.go`
**Output table**: `line_mappings`

The pipeline aligns every pair of editions for each work, scene by scene, so that the web UI can show two editions side-by-side with lines matched up even when one edition inserts or omits content relative to the other.

## Data Flow

```
text_lines (edition A, act X, scene Y)  ──┐
                                           ├─→ AlignScene() ─→ []AlignedPair ─→ line_mappings rows
text_lines (edition B, act X, scene Y)  ──┘
```

For each work, the importer:
1. Loads all `(act, scene)` combinations present in either edition.
2. For each scene, fetches the `AlignableLine` slice for each edition.
3. Calls `AlignScene()` to produce `[]AlignedPair`.
4. Inserts each pair as a `line_mappings` row with a sequential `align_order`.

## AlignableLine

```go
type AlignableLine struct {
    ID      int64
    Content string  // normalized: lowercased, punctuation stripped
}
```

Content is normalized before alignment so that minor spelling differences between editions don't prevent matches.

## Algorithm Selection

```
len(A) * len(B) ≤ 250,000  →  NeedlemanWunsch()
otherwise                   →  simpleAlign()
```

The quadratic memory cost of Needleman-Wunsch (O(n×m) DP matrix) becomes prohibitive for very long scenes. The threshold of 250,000 cells (~500×500 lines) covers virtually all Shakespeare scenes; only a handful of unusually long scenes fall through to `simpleAlign`.

## Needleman-Wunsch Sequence Alignment

Needleman-Wunsch is a global sequence alignment algorithm from bioinformatics, adapted here to align *lines of text* instead of DNA bases.

### Scoring

| Situation | Score |
|---|---|
| Lines match well (Jaccard ≥ 0.3) | `+similarity` (0.3–1.0) |
| Lines don't match (Jaccard < 0.3) | `-0.3` (mismatch penalty) |
| Gap in either sequence | `-0.5` (gap penalty) |

The gap penalty is deliberately low relative to match scores so the algorithm prefers to align similar lines rather than skip them.

### Similarity Metric: Jaccard on Token Sets

```
Jaccard(A, B) = |tokens(A) ∩ tokens(B)| / |tokens(A) ∪ tokens(B)|
```

Tokens are produced by splitting on whitespace and stripping punctuation. Jaccard is symmetric and requires no ordering assumption, which makes it robust to word-order differences between editions.

### DP Table Fill

```
dp[i][j] = max(
    dp[i-1][j-1] + score(A[i], B[j]),  // align A[i] with B[j]
    dp[i-1][j]   - gapPenalty,          // gap in B (skip A[i])
    dp[i][j-1]   - gapPenalty,          // gap in A (skip B[j])
)
```

The table is (n+1) × (m+1) where `n = len(A)` and `m = len(B)`.

### Traceback

After filling the table, traceback starts at `dp[n][m]` and follows the maximum-score predecessor at each cell, producing a sequence of moves:

| Move | Meaning |
|---|---|
| diagonal | A[i] aligned with B[j] |
| up | A[i] has no counterpart in B (`only_a`) |
| left | B[j] has no counterpart in A (`only_b`) |

The resulting pairs are reversed (traceback runs right-to-left) to restore chronological order.

### Match Type Assignment

Each `AlignedPair` gets a `match_type`:

| `match_type` | Condition |
|---|---|
| `aligned` | Both lines present, Jaccard ≥ 0.2 |
| `modified` | Both lines present, Jaccard < 0.2 |
| `only_a` | Line exists only in edition A |
| `only_b` | Line exists only in edition B |

And a `similarity` score (0.0–1.0, Jaccard value, or 0.0 for `only_*`).

## Simple Positional Alignment (Fallback)

For scenes where `len(A) * len(B) > 250,000`, the algorithm falls back to a simple 1:1 positional alignment:

```
position k: pair A[k] with B[k]   (if both exist)
            A[k] alone             (if k ≥ len(B))
            B[k] alone             (if k ≥ len(A))
```

Match type is still assigned via Jaccard (≥ 0.2 → `aligned`, else → `modified`).

## Database Output

Each `AlignedPair` becomes one `line_mappings` row:

| Column | Value |
|---|---|
| `work_id` | Current work |
| `act`, `scene` | Current act/scene |
| `align_order` | Sequential 1, 2, 3… within this scene+edition-pair |
| `edition_a_id`, `edition_b_id` | The two editions being compared |
| `line_a_id` | `text_lines.id` for edition A (NULL for `only_b`) |
| `line_b_id` | `text_lines.id` for edition B (NULL for `only_a`) |
| `match_type` | `aligned`, `modified`, `only_a`, `only_b` |
| `similarity` | Jaccard score |

## Example Query

```sql
-- Side-by-side comparison: Hamlet Act 3 Scene 1 (Globe vs Standard Ebooks)
SELECT lm.align_order, lm.match_type, lm.similarity,
       a.content AS globe_text,
       b.content AS se_text
FROM line_mappings lm
LEFT JOIN text_lines a ON a.id = lm.line_a_id
LEFT JOIN text_lines b ON b.id = lm.line_b_id
WHERE lm.work_id = (SELECT id FROM works WHERE oss_id = 'hamlet')
  AND lm.act = 3 AND lm.scene = 1
  AND lm.edition_a_id = (SELECT id FROM editions WHERE short_code = 'globe_moby')
  AND lm.edition_b_id = (SELECT id FROM editions WHERE short_code = 'se_modern')
ORDER BY lm.align_order;
```
