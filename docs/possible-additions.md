# Possible Additions

Potential open source works, reference data, and Shakespeare-themed naming to expand Bardbase. Items are grouped by category and rated by impact.

**Impact scale:**
- **High** — Unlocks new features or significantly enriches existing ones
- **Medium** — Adds depth or polish; valuable but not transformative
- **Low** — Nice-to-have, cosmetic, or narrow audience

**Status:**
- **Planned** — Confirmed for addition
- **Considering** — Under evaluation
- **Deferred** — Low priority or blocked on something else

---

## Open Source Texts & Editions

### Folger Shakespeare Library Digital Texts
- **Status**: **Done** — implemented in Phase 7 (`folger`)
- **URL**: https://www.folgerdigitaltexts.org/
- **License**: CC BY-NC 3.0
- **Content**: Richly encoded XML (TEIsimple) for all 37 plays. Considered the modern gold-standard scholarly edition. Includes word-by-word POS annotation (MorphAdorner) and Folger Through Line Numbers.
- **What it enables**: A high-quality "modern scholarly" edition column in cross-edition comparison. Folger's editorial decisions often differ from Globe/Standard Ebooks, making comparisons genuinely interesting.
- **Impact**: **High** — best available modern edition; strong complement to existing Globe/SE/Folio lineup.
- **License complexity**: The NC (non-commercial) clause creates friction for a freemium model. The edition is tagged with `license_tier = 'cc-by-nc'` and `source_key = 'folger'` so it can be excluded from commercial features at query time. The `--exclude folger` build flag omits it from production releases if needed.

### Internet Shakespeare Editions (ISE) — Early Quartos
- **Status**: Considering
- **URL**: https://internetshakespeare.uvic.ca/
- **License**: CC BY-SA (varies by text)
- **Content**: Diplomatic transcriptions of early quartos — Q1 Hamlet ("Bad Quarto"), Q1/Q2 King Lear, Q1 Romeo and Juliet, etc. These are textually distinct versions, not just spelling variants.
- **What it enables**: Quarto vs. Folio comparison for plays with significant textual differences. Q1 Hamlet is ~2,200 lines vs. the Folio's ~3,900 — a completely different experience. King Lear's two texts are arguably different plays.
- **Impact**: **High** — unique content no other source provides; cross-edition comparison becomes dramatically more compelling with genuinely divergent texts.
- **Note**: EEBO-TCP quartos (below) already provide CC0 diplomatic transcriptions of several early quartos. ISE would add scholarly editions with modernized annotations.

### EEBO-TCP Early Quartos
- **Status**: **Done** — implemented in Phase 8 (`eebo-quartos`)
- **URL**: https://github.com/textcreationpartnership
- **License**: CC0 / Public Domain
- **Content**: The same archive as the existing First Folio source. Contains early quartos (Q1 Hamlet, Q1/Q2 Lear, etc.) as diplomatic transcriptions. Each quarto gets its own edition record.
- **What it enables**: Quarto vs. Folio comparison for plays with significant textual differences.
- **Impact**: **High** — lowest friction path since the EEBO-TCP parser already existed in Capell.

### Project Gutenberg — Bowdler "Family Shakespeare" (1818)
- **Status**: Planned
- **URL**: https://www.gutenberg.org/ebooks/author/65
- **License**: Public Domain
- **Content**: Thomas Bowdler's famous censored edition, which removed "everything unfit to be read by a gentleman in the company of ladies." A genuine historical artifact showing Victorian editorial judgment.
- **What it enables**: A uniquely interesting comparison layer — what the Victorians cut from Shakespeare. The word "bowdlerize" comes from this edition.
- **Impact**: **Medium** — narrow but distinctive; no other digital Shakespeare platform includes this.

### MIT Complete Shakespeare
- **Status**: Planned
- **URL**: http://shakespeare.mit.edu/
- **License**: Public Domain
- **Content**: Plain-text complete works, widely used as a reference corpus.
- **What it enables**: Cross-reference and validation against a well-known baseline. Less useful as a primary reading edition.
- **Impact**: **Low** — duplicates existing coverage; useful for validation.

---

## Lexicons & Reference Works

### Abbott's Shakespearian Grammar (1877)
- **Status**: **Done** — implemented in Phase 10 (`abbott`)
- **License**: Public Domain
- **Content**: Systematic grammar of Shakespeare's English — constructions, syntax, and word-formation patterns Schmidt's Lexicon doesn't address. Organized as numbered paragraphs (§1 – §515+). Stored in `reference_entries` with citations resolved via Phase 17.
- **What it enables**: A "grammar" layer alongside the lexicon. When a user encounters "methinks" or a double negative, Abbott explains the grammatical rule. Complements Schmidt (vocabulary) with Abbott (syntax).
- **Impact**: **High** — fills a real gap. Schmidt tells you what a word means; Abbott tells you why the sentence is structured that way.

### Onions' Shakespeare Glossary (1911 edition)
- **Status**: **Done** — implemented in Phase 9 (`onions`)
- **License**: Public Domain (1911 edition)
- **Content**: Shorter, more accessible glossary than Schmidt (~10,000 entries) focused on words whose meaning has shifted since Shakespeare's time. Stored in `reference_entries` with citations resolved via Phase 17.
- **What it enables**: A "quick definition" layer — show Onions for at-a-glance meaning, link to Schmidt for full scholarly treatment. Well suited to the reader popover.
- **Impact**: **Medium** — good UX improvement but overlaps significantly with Schmidt.

### Bartlett's Concordance to Shakespeare (1896)
- **Status**: **Done** — implemented in Phase 11 (`bartlett`)
- **License**: Public Domain
- **Content**: Every word in Shakespeare indexed to every occurrence with surrounding context. Stored in `reference_entries` with citations resolved via Phase 17.
- **What it enables**: Word frequency analysis per play, statistical features ("this word appears 47 times in comedies but only 3 times in tragedies"), word explorer.
- **Impact**: **Medium** — interesting analytical features; existing FTS + text_lines can approximate some of this.

### Henley & Farmer's Slang Dictionary (1890-1904)
- **Status**: **Done** — implemented in Phase 12 (`henley-farmer`)
- **License**: Public Domain
- **Content**: Dictionary of Elizabethan slang, cant, and colloquial language (7 volumes). Only entries citing Shakespeare are imported. Stored in `reference_entries` with citations resolved via Phase 17.
- **What it enables**: A "slang" annotation layer. Shakespeare is full of double entendres that modern readers miss. Fills gaps where Schmidt is euphemistic.
- **Impact**: **Medium** — niche but genuinely useful for understanding Shakespeare's humor. A feature few other platforms offer.

---

## Supplementary Data & Tools

### Prosodic / Scansion Data
- **Status**: Planned
- **Content**: Datasets marking iambic pentameter stress patterns, caesura positions, and metrical variations line-by-line.
- **What it enables**: A "scansion mode" in the reader (da-DUM da-DUM da-DUM da-DUM da-DUM). Useful for actors learning verse-speaking and students studying meter.
- **Impact**: **Medium** — valuable for theater/education audience; niche for casual readers.

### Shakespeare Census
- **Status**: Planned
- **URL**: https://shakespearecensus.org/
- **Content**: Bibliographic database tracking surviving early printed editions (First Folio copies, quartos) and their locations.
- **What it enables**: Edition metadata enrichment — "this First Folio text is based on one of 235 surviving copies."
- **Impact**: **Low** — informational only; enriches context but doesn't enable new features.

### WordHoard Annotated Corpus (Northwestern University)
- **Status**: Considering
- **URL**: http://wordhoard.northwestern.edu/
- **License**: GPL
- **Content**: Shakespeare corpus with lemmatization, part-of-speech tagging, and prosodic markup.
- **What it enables**: Lemmatized search (searching "die" finds "died", "dying", "dies"), POS-aware lexicon lookups.
- **Impact**: **High** — transforms search from string matching to linguistic search. Major differentiator.
- **Caveat**: GPL license — may require keeping as a separate optional module.

---

## Source Exclusion in the Build Pipeline

**Status**: **Done** — implemented via the `--exclude` flag.

Sources with licensing restrictions (e.g. Folger's CC BY-NC clause) can be excluded from builds:

```
go run ./cmd/build --exclude folger
go run ./cmd/build --exclude folger,wordhoard
```

Each edition is tagged with `source_key` and `license_tier` columns in the `editions` table, enabling downstream query-time filtering by license. The Folger edition is tagged `license_tier = 'cc-by-nc'` so it can be excluded from commercial/paid features while remaining available for free-tier read-only access.

---

## Shakespeare-Themed Naming

Names for features and components, extending the Bardbase/Capell/Variorum pattern.

### High-Value Names (core features)

| Component | Name | Reference | Why it fits |
|---|---|---|---|
| Sides mode | **Promptbook** | The stage manager's annotated script used to cue actors | Exactly what sides mode is — an actor's working script |
| Notes/annotations | **Glosses** | Marginal notes in medieval/Renaissance manuscripts | The literal term for what the feature does |
| Productions feature | **Playhouse** | The physical theater building | A production lives in a playhouse |
| Search engine | **Quarto** | The small printed editions of individual plays | Quartos were how you found a specific play's text |
| Comparison tool | **Collation** | The scholarly process of comparing variant texts | The technical term for exactly this activity |

### Medium-Value Names (supporting features)

| Component | Name | Reference | Why it fits |
|---|---|---|---|
| User accounts | **The Company** | Acting troupes (Lord Chamberlain's Men, King's Men) | Users join a "company" to collaborate |
| Director role | **Master of Revels** | Royal official who licensed plays | The authority figure who approves the production |
| Issue reporting | **Errata** | Traditional publishing correction notices | Reports errors in the text |
| API/backend | **Heminges** | John Heminges, co-compiler of the First Folio | The service that assembles and serves the data |
| Build/CI pipeline | **Compositor** | Workers who physically typeset the Folio | Assembles the final output from raw materials |
| Auth tokens | **Warrant** | Royal warrants authorized acting companies | Grants permission to operate |

### Low-Value Names (cosmetic / easter eggs)

| Component | Name | Reference |
|---|---|---|
| Dark mode | **Blackfriars** | Shakespeare's indoor (candlelit) theater |
| Light mode | **Globe** | The open-air theater |
| Mobile/PWA | **Groundling** | Audience members who stood in the pit |
| Cache layer | **Stationers** | The Stationers' Company controlled book distribution |

---

## UI Copy & Easter Eggs

Shakespeare quotes as UI microcopy — adds personality with zero engineering cost.

| State | Quote | Source |
|---|---|---|
| 404 page | "Exit, pursued by a bear." | *The Winter's Tale* III.3 |
| Empty search | "Nothing will come of nothing." | *King Lear* I.1 |
| Loading | "The readiness is all." | *Hamlet* V.2 |
| Error | "Though this be madness, yet there is method in't." | *Hamlet* II.2 |
| Rate limited | "How poor are they that have not patience!" | *Othello* II.3 |
| Offline | "Now is the winter of our discontent." | *Richard III* I.1 |
| First visit | "All the world's a stage." | *As You Like It* II.7 |
| Empty notes | "The rest is silence." | *Hamlet* V.2 |
| Successful save | "It is done." | *Cymbeline* I.5 |
| Account created | "We know what we are, but know not what we may be." | *Hamlet* IV.5 |
