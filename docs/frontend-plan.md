# Frontend Implementation Plan

## Implemented

### Texts / Ebook Reader
- **Home page** (`/`) — work browser grouped by tragedies, comedies, histories, poetry
- **Scene reader** (`/text/{slug}/{act}/{scene}`) — multi-edition aligned text
- **Edition toggles** — header dropdown to show/hide editions, preferences saved in localStorage
- **Swipe + arrow navigation** — prev/next scene with floating arrows
- **Reading position** — persisted per work in localStorage, restored on return
- **TOC panel** — slide-out panel with full act/scene list
- **Reference toggles** — header dropdown to enable/disable reference sources, preferences saved in localStorage
- **Auto-hide header** — header hides on scroll down, reappears on scroll up
- **Speaker names** — character names displayed between dialogue sections
- **Slug URLs** — `/text/hamlet/1/4` with numeric ID redirect
- **Lexicon vs reading flow** — `?hw=` param = reference mode (no position save, no nav)

### Word References
- **Scene references** — pre-loaded from DB (exact work/act/scene/line matches only)
- **Multi-source** — Schmidt, Onions, Abbott, Bartlett, Henley & Farmer
- **Cross-edition mapping** — references resolve across all editions via aligned rows
- **Interactive words** — bold + accent color background for referenced words
- **Popover** — hover (desktop) or tap (mobile) shows definitions grouped by source
- **Navigation** — click to go to full entry page

### References Browser
- **References page** (`/references`) — tabbed by source (Schmidt, Onions, Abbott, Bartlett)
- **Search + filter** — full-text search + filter by work dropdown
- **Infinite scroll** — loads 50 at a time
- **URL state** — active tab, search query, work filter persisted in URL params

### Lexicon
- **Entry page** (`/lexicon/entry/{id}`) — full entry with senses, citations, references
- **Citation links** — click to open scene in reference mode with highlighted line
- **Reference entry page** (`/reference/{id}`) — Onions, Abbott, etc. entries

### Corrections
- **GitHub issues** — corrections page shows issues from repo labeled "correction"
- **Flag system** — flag entries/citations from entry pages, opens GitHub issue
- **Filter tabs** — all/open/closed with counts

### Themes
- **Dark mode** — deep dark with teal accents
- **Light mode** — warm parchment (easy on eyes for reading)
- **Toggle** — persisted in localStorage

### PWA
- **Offline** — workbox caching (CacheFirst for stable data, StaleWhileRevalidate for search)
- **Installable** — manifest with icons

---

## Not Yet Implemented

### Sides Mode
- Character selector → show only one character's lines with cue lines
- Verse/prose awareness in sides display

### Productions
- Collaborative script management for theater productions
- Roles (director, stage manager, actor)
- Script modifications (cuts, substitutions from other editions)

### User Accounts & Notes
- Authentication (OAuth)
- Personal notes on lines
- Public/private note visibility

### Enhanced Text Features
- Verse/prose detection and typography
- Line numbers every 5th line for verse
- Prose reflow within content column

---

## Tech Notes

- Go API at `projects/capell/` — stdlib `net/http`, modernc.org/sqlite (pure Go)
- SvelteKit at `projects/web/` — Svelte 5 runes, adapter-cloudflare (prod) / adapter-node (dev)
- Docker Compose for local dev with hot reload
- API auth via shared `API_KEY` (Bearer token)
- All work endpoints accept slugs or numeric IDs
