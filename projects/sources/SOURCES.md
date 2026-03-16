# Shakespeare Text Sources

Comprehensive catalog of every known openly-licensed, machine-readable Shakespeare text source. This document tracks licensing, attribution requirements, data format, and import status.

---

## Active Sources (Free & Open — Using Now)

### 1. Open Source Shakespeare / Moby Shakespeare
- **URL**: https://www.opensourceshakespeare.org/
- **License**: Public Domain
- **Attribution**: Courtesy only (not legally required)
- **Format**: MySQL SQL dump (`projects/sources/oss/oss-db-full.sql`)
- **Content**: 38 works — full text with act/scene/paragraph structure, character attributions, stage directions, word forms. Globe-edition-based modern text.
- **Missing works**: `allswell`, `henryv`, `loverscomplaint`, `midsummer`, `winterstale` and a few others may be present under different IDs — need full audit
- **Edition type**: Modern (Globe-based), some works sourced from Gutenberg
- **Import priority**: P0 — IMMEDIATE (data already in repo)
- **Status**: ✅ Imported — Go pipeline parses the SQL dump and imports all works, characters, and text lines

### 2. Perseus Digital Library — Schmidt Shakespeare Lexicon
- **URL**: http://www.perseus.tufts.edu/hopper/
- **Text ID**: `Perseus:text:1999.03.0079`
- **License**: CC BY-SA 3.0 US
- **Attribution**: **REQUIRED** — must credit Perseus Digital Library and Alexander Schmidt
- **CC BY-SA implications**: Any derived work incorporating Perseus content must be shared under a compatible license. This affects the app's overall licensing.
- **Format**: TEI XML, one file per lexicon entry (`projects/sources/lexicon/entries/[A-Z]/*.xml`)
- **Content**: 20,097 lexicon entries (dictionary of Shakespeare's language) with ~200,000+ citations referencing specific plays, acts, scenes, and lines
- **Edition type**: Schmidt's references use Globe edition numbering
- **Import priority**: P1
- **Status**: ✅ Scraping complete — 20,070 of 20,097 entries downloaded. Go pipeline parses XML, extracts senses, citations, and resolves references to text lines.
- **Attribution text**: "Text provided by Perseus Digital Library, http://www.perseus.tufts.edu. Original work: Shakespeare Lexicon and Quotation Dictionary by Alexander Schmidt, 3rd edition, 1902."

### 3. Standard Ebooks
- **URL**: https://standardebooks.org/
- **License**: CC0 (Creative Commons Zero — public domain dedication)
- **Attribution**: None required (but appreciated)
- **Format**: EPUB → XHTML → JSON (37 play JSONs + sonnets XHTML + poetry XHTML in `projects/sources/se/`)
- **Content**: 37 Shakespeare plays (individually formatted, proofread), 154 sonnets, 5 poems (Venus & Adonis, Rape of Lucrece, Passionate Pilgrim, Phoenix and the Turtle, A Lover's Complaint)
- **Edition type**: Modern, highest quality
- **Import priority**: P0 (data already in repo)
- **Status**: ✅ Imported — Go pipeline imports all 37 plays as a second edition alongside OSS, plus all sonnets and poems. Cross-edition line mappings generated.
- **Notes**: CC0 means zero licensing concerns. Highest quality modern-text source.

### 4. Perseus Digital Library — Shakespeare Play Texts
- **URL**: http://www.perseus.tufts.edu/hopper/
- **License**: CC BY-SA 3.0 US
- **Attribution**: **REQUIRED** (same as lexicon)
- **Format**: TEI XML with dual line numbering (Globe + First Folio)
- **Content**: Complete Shakespeare plays as used by Schmidt's Lexicon
- **Edition type**: Globe text with First Folio line number cross-references
- **Import priority**: P2
- **Status**: ✅ Imported — 43 XML files downloaded to `projects/sources/perseus-plays/`. Go pipeline (step 5: perseus) parses TEI and inserts as "Perseus Globe Edition" alongside OSS and SE.
- **Notes**: These are the direct text references for lexicon citations. Dual line numbering enables cross-edition mapping.

### 5. EEBO-TCP (Early English Books Online — Text Creation Partnership)
- **URL**: https://github.com/textcreationpartnership
- **License**: CC0 1.0 Universal (public domain, released January 1, 2015)
- **Attribution**: Courtesy credit appreciated but not required
- **Format**: TEI P5 XML, diplomatic transcriptions in original spelling
- **Content**:
  - **First Folio** (1623): TCP ID `A11954`, STC 22273 — 35 plays, original spelling. Downloaded to `projects/sources/eebo-tcp/A11954.xml`
  - **Various Quartos**: Available on GitHub (A11959 Hamlet, A12044 Sonnets 1609, A12034 Poems, A11966 1H4, A12040 Lucrece, A12017 Titus, A68931 2H6). Downloaded to `projects/sources/eebo-tcp/`.
- **Edition type**: Original spelling diplomatic transcriptions (long-s normalized to s on import)
- **Import priority**: P2 — First Folio done; quartos next
- **Status**: ✅ First Folio imported — parser (`parser/folio.go`) and importer (`importer/folio.go`) complete. Added as step 6 in build pipeline. Edition short code: `first_folio`. Quarto files downloaded but not yet parsed/imported.
- **Notes**: First Folio covers 35 of 36 F1 plays (all except Troilus, which had a complex print history and was omitted from the TCP transcription). Long-s (ſ→s) is normalized at import time. Original spelling is preserved otherwise.

---

## Reference Only (NO Download)

### 6. Folger Shakespeare Library
- **URL**: https://www.folger.edu/explore/shakespeares-works/
- **License**: Proprietary — free to view on their website
- **Attribution**: N/A (reference links only)
- **Format**: N/A
- **Content**: Authoritative modern scholarly editions of all plays
- **What we store**: URLs only — link to specific acts/scenes/lines on folger.edu
- **RULE**: Content must NEVER be downloaded or stored locally. Only store reference URLs.
- **Import priority**: P1 (just URLs, minimal work)
- **Status**: ⚠️ Partial — `folger_url` column exists on `works` table, some URLs populated

---

## Needs License Verification

These sources have potentially usable Shakespeare texts but their redistribution terms need to be verified before any content is downloaded or included.

### 8. Internet Shakespeare Editions (ISE)
- **URL**: https://internetshakespeare.uvic.ca/
- **Stated terms**: "Free for personal and educational use"
- **Redistribution**: UNCLEAR — need to verify if bundling in an app is permitted
- **Format**: HTML, some TEI XML
- **Content**: Best scholarly editions available online — multiple versions per play, First Folio diplomatic transcriptions, individual Quarto editions
- **Edition type**: Multiple per play (modern, F1 diplomatic, Q1, Q2, etc.)
- **Value**: Extremely high — best per-play scholarly editions online
- **Action needed**: Email ISE team to ask about redistribution in an open-source app with attribution
- **Notes**: If licensing allows, this becomes the single best source for multi-edition comparison

### 9. Shakespeare Quartos Archive
- **URL**: https://shakespearequartosarchive.org/
- **Partners**: Bodleian Library + Folger Shakespeare Library
- **Stated terms**: Unknown — joint academic project
- **Format**: TEI XML diplomatic transcriptions
- **Content**: 32 early Quarto editions of Shakespeare plays
- **Edition type**: Diplomatic transcriptions of specific physical copies
- **Value**: Very high — best Quarto source
- **Action needed**: Check terms of use on site; contact project leads if unclear

### 10. Wikisource
- **URL**: https://en.wikisource.org/wiki/Author:William_Shakespeare
- **License**: CC BY-SA 3.0 (same as Wikipedia)
- **Attribution**: **REQUIRED** if used
- **Format**: Wikitext / HTML
- **Content**: First Folio, various Quartos, modern editions — crowd-sourced transcriptions
- **Edition type**: Multiple (F1, Quartos, modern)
- **Quality**: Variable — crowd-sourced, may have transcription errors
- **Import priority**: P4 (backup source, verify against authoritative texts)
- **Notes**: Good backup/gap-filler but quality needs validation against better sources

### 11. HathiTrust Digital Library
- **URL**: https://www.hathitrust.org/
- **License**: Varies — pre-1927 publications should be public domain in the US
- **Format**: OCR text (variable quality), page images
- **Content**: Historical printed editions including potentially **Second Folio** (1632) scans
- **Edition type**: Various historical printings
- **Value**: Potentially the only source for Second Folio digital text
- **Action needed**: Search for F2 transcriptions; verify PD status of specific items
- **Notes**: OCR quality from 17th-century typefaces is often poor. May need manual correction.

### 12. WordHoard (Northwestern University)
- **URL**: http://wordhoard.northwestern.edu/
- **Stated terms**: "Free for academic use"
- **Redistribution**: UNCLEAR — "academic use" may not cover app distribution
- **Format**: Custom XML with full linguistic annotation
- **Content**: Complete works, every word POS-tagged and lemmatized
- **Value**: Enormous for search functionality — pre-built linguistic index
- **Action needed**: Contact Northwestern to ask about redistribution terms

### 13. Oxford Text Archive (OTA)
- **URL**: https://ota.bodleian.ox.ac.uk/
- **License**: Varies per deposit — each text has its own terms
- **Format**: Various (TEI XML, plain text)
- **Content**: Scholarly text deposits, may include unique Shakespeare editions
- **Action needed**: Search archive for Shakespeare texts; check individual deposit licenses

---

## Known Commercial / Unavailable (DO NOT USE)

These are authoritative editions but are commercially published and cannot be freely redistributed:

- **Arden Shakespeare** (Bloomsbury) — commercial
- **Norton Shakespeare** (W.W. Norton) — commercial
- **Riverside Shakespeare** (Cengage) — commercial
- **Oxford Shakespeare** (Oxford University Press) — commercial
- **New Cambridge Shakespeare** (Cambridge UP) — commercial
- **RSC Shakespeare** (Macmillan) — commercial
- **Pelican Shakespeare** (Penguin) — commercial

---

## The Second Folio Problem

The Second Folio (1632) is historically significant but presents a challenge:
- **No known complete open digital transcription exists**
- HathiTrust may have OCR from scans, but 17th-century typeface OCR is unreliable
- **Best approach**: Use First Folio text as base, apply known scholarly emendation lists documenting F1→F2 changes
- **Priority**: P4 — important for completeness but won't block the project

---

## Attribution Requirements Summary

| Source | License | Attribution in App? | Notes |
|--------|---------|-------------------|-------|
| OSS/Moby | Public Domain | Courtesy credit | "Based on the Moby Shakespeare" |
| Standard Ebooks | CC0 | None required | Truly free — no restrictions |
| Perseus Lexicon | CC BY-SA 3.0 | **YES — Required** | Full credit + compatible license |
| Perseus Plays | CC BY-SA 3.0 | **YES — Required** | Same as lexicon |
| EEBO-TCP | Public Domain | Courtesy credit | "Transcription by Text Creation Partnership" |
| Folger | N/A (links only) | N/A | "Visit Folger Shakespeare Library" with URL |
| Wikisource | CC BY-SA 3.0 | **YES — Required** | If used |

### CC BY-SA 3.0 Implications

Perseus (and Wikisource if used) both use CC BY-SA 3.0. This means:
1. **Attribution** is required in the app (credits page, about section)
2. **ShareAlike** — if you create a derivative work, it must be shared under CC BY-SA 3.0 or a compatible license
3. This does NOT mean the whole app must be CC BY-SA — only the portions derived from CC BY-SA sources
4. The database as a whole can have its own license, with CC BY-SA portions clearly marked
5. **Practical approach**: Include a credits/attribution page in the app that lists all sources with their licenses

---

## Current Source Inventory

### What we have on disk (`projects/sources/`)

| Directory | Files | Description |
|-----------|-------|-------------|
| `oss/` | 1 file (SQL dump) | OSS/Moby complete MySQL dump |
| `se/` | 39 files (37 JSONs + 2 XHTMLs) | All 37 plays + sonnets + poetry |
| `lexicon/entries/[A-Z]/` | 20,070 XML files | Schmidt Lexicon (nearly complete) |
| `lexicon/entry_list.json` | Entry index | Scraper tracking file |
| `perseus-plays/` | 43 XML files | Perseus Globe edition TEI XML |
| `eebo-tcp/` | 8 XML files | First Folio (A11954) + 7 quartos |

### What we still need

| Source | Priority | Effort | Blocker |
|--------|----------|--------|---------|
| EEBO-TCP Quartos (import) | P3 | Medium — parse 7 individual play XML files | Need parser + work matching |

---

## Import Phases

| Phase | Source | Priority | Status |
|-------|--------|----------|--------|
| 0 | OSS SQL dump → SQLite | P0 | ✅ Complete |
| 0b | Standard Ebooks → SQLite | P0 | ✅ Complete |
| 1 | Schmidt Lexicon XMLs → SQLite | P1 | ✅ Complete (20,070 entries) |
| 1b | Folger reference URLs | P1 | ⚠️ Partial |
| 2 | Perseus play texts → SQLite | P2 | ✅ Complete (43 files, step 5, `perseus_globe` edition) |
| 2b | EEBO-TCP First Folio → SQLite | P2 | ✅ Complete (35 plays, step 6, `first_folio` edition) |
| 3 | EEBO-TCP Quartos → SQLite | P3 | ⚠️ Files downloaded, import not yet built |
| 5 | Verified sources from "Needs Verification" list | P4+ | ❌ Pending outreach |
| 6 | Second Folio (if source found) | P4 | ❌ Research needed |
