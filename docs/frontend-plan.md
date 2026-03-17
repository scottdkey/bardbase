# Frontend Implementation Plan

## Current State

The Variorum frontend has a basic scaffold built with SvelteKit (SSG + sql.js WASM for offline search):

- **Home** (`/`): Stats dashboard + work list
- **Works** (`/works`): Plays vs. poetry listing
- **Work detail** (`/works/[id]`): Simple text dump — shows first edition only, no switching, no scene nav
- **Lexicon** (`/lexicon`): Letter buttons (non-functional) + first 100 "A" entries with truncated definitions
- **Search** (`/search`): Full-text search via sql.js (functional)

The database has rich data that the frontend doesn't use yet: lexicon senses, ~200K citations, citation_matches (resolved to text lines), and line_mappings (cross-edition alignment).

---

## Phase 1: Lexicon Dictionary UI

**Goal:** A browsable, readable Shakespeare dictionary.

### 1.1 Working letter navigation

- Letter buttons become links: `/lexicon?letter=B`
- Server load function reads `letter` from URL search params
- Active letter gets visual highlight
- Show entry count badge on each letter

### 1.2 Pagination

- 20K entries can't load at once — paginate within each letter
- Cursor-based pagination using `(letter, id)` index
- "Load more" button or infinite scroll
- Target: ~50 entries per page

### 1.3 Lexicon entry detail page

- New route: `/lexicon/[id]`
- Add query: `getLexiconEntry(db, id)` — full entry with all fields
- Add query: `getLexiconSenses(db, entryId)` — all senses for an entry
- Display: headword, orthography, part of speech, each numbered sense with its definition

### 1.4 Combine duplicate headwords

- Multiple entries can share the same `key` (e.g. "Run" as verb vs. noun)
- Entry list groups them: show once with a count indicator
- Detail page: `/lexicon/[id]` shows one entry, but links to siblings with same key
- Add query: `getLexiconEntriesByKey(db, key)` — all entries sharing a headword

### 1.5 Citations under each sense

- Add query: `getCitationsBySense(db, senseId)` — returns citations with work_abbrev, act, scene, line, quote_text
- Display under each sense definition as a citation list
- Format: work abbreviation, location (II, 1, 23), quote in italics
- Citations are display-only for now (linking comes in Phase 4)

---

## Phase 2: Ebook Reader

**Goal:** A clean, comfortable reading experience for any edition.

### 2.1 Edition switcher

- Dropdown/select populated from `getEditionsByWork()`
- Changing edition reloads text via URL param: `/works/5?edition=2`
- Server load reads `edition` from search params, falls back to first available
- Show edition name + year + source in the dropdown

### 2.2 Act/Scene navigation

- Collapsible sidebar or sticky top bar with act/scene links
- Derived from the text lines already loaded (group by act/scene)
- Clicking jumps to the scene anchor in the page
- Current scene highlighted as you scroll (IntersectionObserver)

### 2.3 Verse/prose detection and clean typography

The DB has a `line_type` column (`verse` or `prose`) on `text_lines` but the current query doesn't select it and the frontend doesn't use it.

- Add `line_type` to the `getTextLines` query and expose as `is_verse: boolean` in the `TextLineWithCharacter` type
- **Verse lines** (`is_verse = true`): preserve line breaks, one line per row, slight left indent for continuation lines
- **Prose lines** (`is_verse = false`): reflow naturally within the content column, group consecutive prose lines from the same speaker into a single paragraph
- Stage directions: italic, muted color, centered or indented
- Speaker names: small caps or bold, set apart from dialogue
- Line numbers: subtle, right-aligned, shown every 5th line for verse (hover to see others); hidden for reflowed prose
- Comfortable reading width (~65ch), generous line-height (1.6-1.8)

### 2.4 Scene-aware URL

- URL updates as you scroll: `/works/5?edition=2&act=3&scene=1`
- Deep-linking: opening that URL scrolls to the right scene
- Browser back/forward navigates between scenes visited

---

## Phase 3: Cross-Edition Comparison

**Goal:** Read a passage in one edition, view the same passage in any other.

### 3.1 Line mapping queries

- Add query: `getLineMappings(db, workId, editionA, editionB, act, scene)` — returns aligned line pairs from `line_mappings` table
- Returns: `{ align_order, line_a: TextLine | null, line_b: TextLine | null }` (nulls for gaps)

### 3.2 Side-by-side view

- New route or mode: `/works/5/compare?a=1&b=3&act=2&scene=1`
- Two-column layout with aligned lines (gaps shown as blank rows)
- Edition names as column headers
- Same act/scene navigation as the reader
- Responsive: stacks vertically on mobile

### 3.3 Quick edition toggle from reader

- "Compare with..." button in the reader toolbar
- Selecting a second edition enters side-by-side mode
- Position preserved via line mappings — if you're on Act 3, Scene 2, line 45 in Globe, it finds the corresponding line in First Folio

---

## Phase 4: Lexicon–Reader Integration

**Goal:** Seamlessly move between the dictionary and the texts.

### 4.1 Citation links (lexicon → reader)

- Each citation in the lexicon becomes a link
- Uses `citation_matches` to resolve to a specific `text_line_id` + `edition_id`
- Link format: `/works/{work_id}?edition={edition_id}&line={line_id}`
- Reader highlights the target line on arrival

### 4.2 Reader lexicon lookup (reader → lexicon)

- Click/tap a word in the reader text
- Slide-out panel or popover shows matching lexicon entries
- Query: search `lexicon_entries` by key matching the clicked word
- Shows: headword, top senses, link to full entry page
- Dismiss by clicking elsewhere or pressing Escape

### 4.3 Line citation indicators

- Subtle dot or icon on lines that have lexicon citations (from `citation_matches`)
- Hover/click shows which lexicon entries reference this line
- Count badge: "3 lexicon entries cite this line"
- Each links to the corresponding lexicon entry

---

## Dependency Graph

```
Phase 1 (Lexicon)          Phase 2 (Reader)
      │                         │
      └────────┬────────────────┘
               │
         Phase 3 (Comparison)
               │
         Phase 4 (Integration)
```

Phases 1 and 2 are independent — can be built in parallel.
Phase 3 requires the reader (Phase 2).
Phase 4 requires both the lexicon (Phase 1) and reader (Phase 2).

---

## Tech Notes

- All new queries go in `queries.ts` with typed returns
- New generated types may be needed for `lexicon_senses`, `lexicon_citations`, `citation_matches`, `line_mappings` — check `generated/db.ts`
- SSG: all pages pre-rendered at build time; dynamic params (letter, edition) use search params or route params
- Keep sql.js search separate — it handles FTS only, not these structured queries
- Attribution display rules from `attributions` table must be respected when showing edition content
