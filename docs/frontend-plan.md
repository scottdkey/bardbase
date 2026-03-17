# Frontend Implementation Plan

## Current State

The Variorum frontend has a basic scaffold built with SvelteKit (SSG + sql.js WASM for offline search):

- **Home** (`/`): Lexicon dictionary with easy access to works (ebook reading)
- **Works** (`/works`): Plays vs. poetry listing
- **Work detail** (`/works/[id]`): Simple text dump — shows first edition only, no switching, no scene nav
- **Lexicon** (`/lexicon`): Letter buttons (non-functional) + first 100 "A" entries with truncated definitions
- **Search** (`/search`): Full-text search via sql.js (functional)

The database has rich data that the frontend doesn't use yet: lexicon senses, ~200K citations, citation_matches (resolved to text lines), and line_mappings (cross-edition alignment).

---

## Phase 1: Lexicon Dictionary UI

**Goal:** A browsable, readable Shakespeare dictionary as the main page.

### 1.1 Working letter navigation

- Letter buttons become links: `/lexicon?letter=B`
- Server load function reads `letter` from URL search params
- Active letter gets visual highlight
- Show entry count badge on each letter

### 1.2 Pagination

- 20K entries can't load at once — paginate within each letter
- Cursor-based pagination using `(letter, id)` index with infinite scrolling
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

### 1.6 Easy access to works

- From the lexicon page, add prominent links or buttons to access works (ebook reading)
- Perhaps a sidebar or top navigation with quick access to plays/poetry lists

---

## Phase 2: Floating Search Button

**Goal:** Search accessible from anywhere with filters.

### 2.1 Floating search button

- A floating action button (FAB) present on all pages
- Clicking opens a search modal or overlay
- Ability to set filters (e.g., by work, edition, etc.)
- Search results link to relevant pages (lexicon entries, work lines, etc.)

---

## Phase 3: Ebook Reader

**Goal:** A clean, comfortable reading experience for any edition.

### 3.1 Edition switcher

- Dropdown/select populated from `getEditionsByWork()`
- Changing edition reloads text via URL param: `/works/5?edition=2`
- Server load reads `edition` from search params, falls back to first available
- Show edition name + year + source in the dropdown

### 3.2 Act/Scene navigation

- Collapsible sidebar or sticky top bar with act/scene links
- Derived from the text lines already loaded (group by act/scene)
- Clicking jumps to the scene anchor in the page
- Current scene highlighted as you scroll (IntersectionObserver)

### 3.3 Verse/prose detection and clean typography

The DB has a `line_type` column (`verse` or `prose`) on `text_lines` but the current query doesn't select it and the frontend doesn't use it.

- Add `line_type` to the `getTextLines` query and expose as `is_verse: boolean` in the `TextLineWithCharacter` type
- **Verse lines** (`is_verse = true`): preserve line breaks, one line per row, slight left indent for continuation lines
- **Prose lines** (`is_verse = false`): reflow naturally within the content column, group consecutive prose lines from the same speaker into a single paragraph
- Stage directions: italic, muted color, centered or indented
- Speaker names: small caps or bold, set apart from dialogue
- Line numbers: subtle, right-aligned, shown every 5th line for verse (hover to see others); hidden for reflowed prose
- Comfortable reading width (~65ch), generous line-height (1.6-1.8)

### 3.4 Scene-aware URL

- URL updates as you scroll: `/works/5?edition=2&act=3&scene=1`
- Deep-linking: opening that URL scrolls to the right scene
- Browser back/forward navigates between scenes visited

---

## Phase 4: Cross-Edition Comparison

**Goal:** Read a passage in one edition, view the same passage in any other.

### 4.1 Line mapping queries

- Add query: `getLineMappings(db, workId, editionA, editionB, act, scene)` — returns aligned line pairs from `line_mappings` table
- Returns: `{ align_order, line_a: TextLine | null, line_b: TextLine | null }` (nulls for gaps)

### 4.2 Side-by-side view

- New route or mode: `/works/5/compare?a=1&b=3&act=2&scene=1`
- Two-column layout with aligned lines (gaps shown as blank rows)
- Edition names as column headers
- Same act/scene navigation as the reader
- Responsive: stacks vertically on mobile

### 4.3 Quick edition toggle from reader

- "Compare with..." button in the reader toolbar
- Selecting a second edition enters side-by-side mode
- Position preserved via line mappings — if you're on Act 3, Scene 2, line 45 in Globe, it finds the corresponding line in First Folio

---

## Phase 5: Lexicon–Reader Integration

**Goal:** Seamlessly move between the dictionary and the texts.

### 5.1 Citation links (lexicon → reader)

- Each citation in the lexicon becomes a link
- Uses `citation_matches` to resolve to a specific `text_line_id` + `edition_id`
- Link format: `/works/{work_id}?edition={edition_id}&line={line_id}`
- Reader highlights the target line on arrival

### 5.2 Reader lexicon lookup (reader → lexicon)

- Click/tap a word in the reader text
- Slide-out panel or popover shows matching lexicon entries
- Query: search `lexicon_entries` by key matching the clicked word
- Shows: headword, top senses, link to full entry page
- Dismiss by clicking elsewhere or pressing Escape

### 5.3 Line citation indicators

- Subtle dot or icon on lines that have lexicon citations (from `citation_matches`)
- Hover/click shows which lexicon entries reference this line
- Count badge: "3 lexicon entries cite this line"
- Each links to the corresponding lexicon entry

---

## Phase 6: Sides Mode

**Goal:** Actors can read just their character's lines with surrounding cue lines for context.

### 6.1 Character selector

- Dropdown listing all characters in the current work (derived from `text_lines` speaker data)
- Selecting a character enters "sides mode"
- URL param: `/works/5?edition=2&sides=HAMLET`

### 6.2 Sides display

- Show only the selected character's lines, plus their cue lines (the 1–2 lines spoken by other characters immediately before each of the selected character's speeches)
- Cue lines displayed in a muted style (lighter text, smaller font) to distinguish from the character's own lines
- Stage directions relevant to the character still shown
- Act/scene headings preserved for context
- Clear visual grouping: cue → character's speech block, repeated throughout the play

### 6.3 Sides with verse/prose awareness

- Verse lines maintain their line breaks in sides mode
- Prose speeches reflow naturally
- Line numbers from the full text preserved so the actor can reference the original

---

## Phase 7: Issue Reporting

**Goal:** Let users submit issues directly from the site, pushed to GitHub Issues.

### 7.1 Issue submission form

- Accessible from a persistent "Report Issue" button in the site footer/toolbar
- Issue types via dropdown:
  - **Site bug** — something broken in the UI
  - **Text error** — wrong text, missing lines, encoding issues
  - **Bad match** — a citation or line mapping is incorrect
  - **Feature request** — general suggestion
- Free-text description field
- Optional screenshot upload (as base64 in the issue body or via GitHub API attachment)

### 7.2 Contextual issue filing

- When filing from the reader, auto-populate: work, edition, act/scene, line number
- When filing from the lexicon, auto-populate: entry headword, sense number, citation
- This context is formatted into the GitHub issue body as structured metadata
- User only needs to describe what's wrong

### 7.3 GitHub Issues integration

- Server endpoint that creates issues via the GitHub API (`POST /repos/{owner}/{repo}/issues`)
- Issues auto-labeled by type (e.g. `bug`, `text-error`, `bad-match`, `enhancement`)
- Rate limiting to prevent spam
- Success confirmation shown to the user with a link to the created issue

---

## Phase 7: User Accounts & Notes

**Goal:** Authenticated users can annotate texts with personal notes.

### 7.1 Authentication

- User accounts via OAuth (GitHub, Google) or email/password
- Session management with secure cookies
- User profile: display name, email, role (default: reader)

### 7.2 Personal notes

- Click any line in the reader to add a private note
- Notes stored per-user, per-line, per-edition
- Notes panel: sidebar or inline popover showing the user's note on a line
- Notes are private by default — only visible to the note author
- Ability to edit, delete, and search personal notes

### 7.3 Note visibility controls

- Each note has a visibility toggle: **private** (default) or **public**
- Public notes visible to all users reading the same passage
- Public notes shown with author attribution
- Users can retract a public note back to private at any time

---

## Phase 8: Productions

**Goal:** Collaborative script management for theater productions.

### 8.1 Create a production

- A user creates a production: name, play, base edition (e.g. Globe)
- The production gets a unique shareable link
- Creator becomes the production **director** by default

### 8.2 Production roles

- **Director** — full control: invite/remove members, manage public notes, answer questions, edit the production script
- **Stage Manager** — can edit the production script, manage public notes, answer questions
- **Actor** — can view the production script, add personal notes, ask questions, use sides mode filtered to their assigned character(s)
- **Reader** — read-only access to the production script and public notes
- Character assignment: director assigns actors to characters (enables auto-sides)

### 8.3 Production script editing

- Start from the base edition as the production's working script
- Script modifications (stored as a layer over the base text, not destructive edits):
  - **Cut lines** — mark lines as cut (shown with strikethrough, excluded from sides/performance view)
  - **Modify lines** — override a line's text with a custom version
  - **Substitute from another edition** — replace a line or passage with the equivalent from a different edition (using line_mappings for alignment)
- All modifications tracked with attribution (who changed what, when)
- "Performance view" shows only the final production script with all cuts/mods applied
- "Editorial view" shows the base text with all modifications annotated

### 8.4 Production notes & questions

- **Questions**: any member can post a question on a line/passage; director and stage manager receive them and can respond
- **Production notes** (public): director/stage manager can pin notes visible to all production members (blocking notes, staging directions, etc.)
- Distinction from personal notes: production notes belong to the production, not the individual; they persist even if the author leaves
- Question thread: simple threaded replies under each question

### 8.5 Sharing & collaboration

- Invite members by email or shareable link with role assignment
- Real-time sync not required for v1 — last-write-wins with conflict warnings is acceptable
- Production dashboard: list of all productions a user belongs to, with role and last-activity date

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
               │
         Phase 5 (Sides Mode)
               │
         Phase 6 (Issue Reporting)    ← can also start after Phase 2
               │
         Phase 7 (User Accounts)
               │
         Phase 8 (Productions)
```

- Phases 1 and 2 are independent — can be built in parallel
- Phase 3 requires the reader (Phase 2)
- Phase 4 requires both lexicon (Phase 1) and reader (Phase 2)
- Phase 5 requires the reader (Phase 2), benefits from Phase 3 line mappings
- Phase 6 can start after Phase 2 (contextual filing needs the reader), but the basic form could ship earlier
- Phase 7 requires a backend/auth layer — standalone but gates Phase 8
- Phase 8 requires Phase 7 (accounts), Phase 5 (sides), and Phase 3 (cross-edition substitution)

---

## Tech Notes

- All new queries go in `queries.ts` with typed returns
- New generated types may be needed for `lexicon_senses`, `lexicon_citations`, `citation_matches`, `line_mappings` — check `generated/db.ts`
- SSG: all pages pre-rendered at build time; dynamic params (letter, edition) use search params or route params
- Keep sql.js search separate — it handles FTS only, not these structured queries
- Attribution display rules from `attributions` table must be respected when showing edition content
