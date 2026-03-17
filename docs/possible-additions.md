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
- **Status**: Planned
- **URL**: https://www.folgerdigitaltexts.org/
- **License**: CC BY-NC 3.0
- **Content**: Richly encoded XML (TEI) for all 37 plays. Considered the modern gold-standard scholarly edition. Detailed markup for stage directions, verse/prose, speech prefixes, and editorial notes.
- **What it enables**: A high-quality "modern scholarly" edition column in cross-edition comparison. Folger's editorial decisions often differ from Globe/Standard Ebooks, making comparisons genuinely interesting.
- **Impact**: **High** — best available modern edition; strong complement to existing Globe/SE/Folio lineup.
- **License complexity**: The NC (non-commercial) clause creates friction for a freemium model. Three approaches:
  1. **Source isolation** *(recommended)*: Tag each edition in the DB with its license tier. Paid features (Productions, script editing) are restricted to CC0/public domain editions — Standard Ebooks, EEBO-TCP, Perseus. Folger is read-only, free-tier content only. A user creating a paid Production simply picks a non-NC base edition; Folger is never the working script.
  2. **Contact Folger**: Their Digital Texts team is research-friendly and has granted commercial carve-outs before. A freemium model where texts are always publicly readable is a reasonable ask.
  3. **Skip Folger**: Standard Ebooks is already CC0 and high quality. If the licensing overhead isn't worth it, use ISE quartos (CC BY-SA, commercially usable with attribution) as the premium scholarly edition instead.
- **Build note**: See [Source Exclusion](#source-exclusion-in-the-build-pipeline) below — Folger is the primary motivator for per-source build flags.

### Internet Shakespeare Editions (ISE) — Early Quartos
- **Status**: Planned
- **URL**: https://internetshakespeare.uvic.ca/
- **License**: CC BY-SA (varies by text)
- **Content**: Diplomatic transcriptions of early quartos — Q1 Hamlet ("Bad Quarto"), Q1/Q2 King Lear, Q1 Romeo and Juliet, etc. These are textually distinct versions, not just spelling variants.
- **What it enables**: Quarto vs. Folio comparison for plays with significant textual differences. Q1 Hamlet is ~2,200 lines vs. the Folio's ~3,900 — a completely different experience. King Lear's two texts are arguably different plays.
- **Impact**: **High** — unique content no other source provides; cross-edition comparison (Phase 4) becomes dramatically more compelling with genuinely divergent texts.

### EEBO-TCP Early Quartos
- **Status**: Planned
- **URL**: https://github.com/textcreationpartnership
- **License**: CC0 / Public Domain
- **Content**: The same archive as the existing First Folio source. Contains early quartos (Q1 Hamlet, Q1/Q2 Lear, etc.) as diplomatic transcriptions.
- **What it enables**: Same as ISE quartos above.
- **Impact**: **High** — lowest friction path since the EEBO-TCP parser already exists in Capell.

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

### Abbott's Shakespearian Grammar (1870)
- **Status**: Planned
- **License**: Public Domain
- **Content**: Systematic grammar of Shakespeare's English — constructions, syntax, and word-formation patterns Schmidt's Lexicon doesn't address. Organized by numbered grammatical sections.
- **What it enables**: A "grammar" layer alongside the lexicon. When a user encounters "methinks" or a double negative, Abbott explains the grammatical rule. Complements Schmidt (vocabulary) with Abbott (syntax).
- **Impact**: **High** — fills a real gap. Schmidt tells you what a word means; Abbott tells you why the sentence is structured that way.

### Onions' Shakespeare Glossary (1911 edition)
- **Status**: Planned
- **License**: Public Domain (1911 edition)
- **Content**: Shorter, more accessible glossary than Schmidt (~10,000 entries) focused on words whose meaning has shifted since Shakespeare's time.
- **What it enables**: A "quick definition" layer — show Onions for at-a-glance meaning, link to Schmidt for full scholarly treatment. Well suited to the reader popover (Phase 5.2).
- **Impact**: **Medium** — good UX improvement but overlaps significantly with Schmidt.

### Bartlett's Concordance to Shakespeare (1894)
- **Status**: Planned
- **License**: Public Domain
- **Content**: Every word in Shakespeare indexed to every occurrence with surrounding context.
- **What it enables**: Word frequency analysis per play, statistical features ("this word appears 47 times in comedies but only 3 times in tragedies"), word explorer.
- **Impact**: **Medium** — interesting analytical features; existing FTS + text_lines can approximate some of this.

### Henley & Farmer's Slang Dictionary (1905)
- **Status**: Planned
- **License**: Public Domain
- **Content**: Dictionary of Elizabethan slang, cant, and colloquial language. Covers bawdy language and street slang that Schmidt treats delicately.
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

As the number of sources grows, some may carry licensing restrictions (e.g. Folger's NC clause) or be experimental/incomplete. The build pipeline needs a per-source opt-out mechanism so we can include Folger in development and CI builds while excluding it from production releases until the licensing question is resolved.

### Proposed approach: `--exclude` flag on `capell build`

```
go run ./cmd/build --exclude folger --exclude wordhoard
```

Each phase in the pipeline registers itself with a source key. If that key appears in the exclude list, the phase is skipped entirely and its data is not written to the database. The `editions` table gains a `source_key` column so downstream code can filter by license tier at query time.

**Schema addition:**
```sql
ALTER TABLE editions ADD COLUMN source_key TEXT;  -- e.g. 'folger', 'ise', 'eebo'
ALTER TABLE editions ADD COLUMN license_tier TEXT; -- 'cc0', 'cc-by-sa', 'cc-by-nc'
```

**Build flag in CI:**
```yaml
# workflow_dispatch input (manual build)
inputs:
  exclude_sources:
    description: 'Comma-separated source keys to exclude (e.g. folger,wordhoard)'
    required: false
    default: ''
```

This way Folger can be developed and tested locally with `--exclude` omitted, while the public release build passes `--exclude folger` until the license situation is resolved.

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
