# Shakespeare Database

A comprehensive, multi-edition Shakespeare database built from open sources. The database is a **build artifact** — clone the repo, run the build, get a full SQLite database.

## Quick Start

```bash
git clone https://github.com/scottdkey/shakespeare_db.git
cd shakespeare_db
python3 tools/build.py
```

This produces `build/shakespeare.db` (~55 MB) containing:

| Content | Count | Source |
|---------|-------|--------|
| **Works** | 43 | All plays + poems |
| **Text lines** | ~137,000 | 2 editions (OSS/Moby + Standard Ebooks) |
| **Characters** | 1,265 | Every named character |
| **Lexicon entries** | 7,500+ | Schmidt Shakespeare Lexicon |
| **Citations** | 81,000+ | Cross-references to play passages |

Or download a pre-built database from [Releases](https://github.com/scottdkey/shakespeare_db/releases).

## Repository Structure

```
shakespeare_db/
├── sources/                    # Source data (git-tracked)
│   ├── oss/
│   │   └── oss-db-full.sql    # Open Source Shakespeare MySQL dump (16 MB)
│   └── lexicon/
│       ├── entry_list.json    # Perseus entry manifest
│       └── entries/           # Schmidt Lexicon XMLs (A-Z directories)
├── tools/
│   └── build.py               # Master build script
├── .github/workflows/
│   └── build.yml              # Auto-build on source changes → GitHub Release
├── SOURCES.md                  # Detailed source documentation
├── SCHEMA.md                   # Database schema reference
└── README.md
```

## Build Options

```bash
# Full build (downloads Standard Ebooks from the internet)
python3 tools/build.py

# Skip downloads (use cached SE data only)
python3 tools/build.py --skip-download

# Custom output directory
python3 tools/build.py --output dist/

# Run a single build step
python3 tools/build.py --step oss       # Just OSS/Moby data
python3 tools/build.py --step lexicon   # Just Schmidt lexicon
python3 tools/build.py --step se        # Just Standard Ebooks plays
python3 tools/build.py --step poetry    # Just poetry + Folger URLs
python3 tools/build.py --step fts       # Just rebuild FTS indexes
```

## Data Sources

| Source | License | Content | In Repo? |
|--------|---------|---------|----------|
| **OSS/Moby** | Public Domain | 43 works, full text, characters | ✅ `sources/oss/` |
| **Schmidt Lexicon** | CC BY-SA 3.0 | Dictionary entries + citations | ✅ `sources/lexicon/` |
| **Standard Ebooks** | CC0 1.0 | Modern-spelling verse-level text | Downloaded at build time |
| **Folger Library** | Reference only | URLs to online editions | Generated at build time |

See [SOURCES.md](SOURCES.md) for detailed provenance, licensing, and attribution requirements.

## Editions

The database tracks text from multiple editions side-by-side:

| Edition | Source | Lines | Description |
|---------|--------|-------|-------------|
| **OSS/Globe** | Open Source Shakespeare | ~35,600 | Globe-based, paragraph-level |
| **SE Modern** | Standard Ebooks | ~101,200 | Modern-spelling, verse-level |

## Schema

Key tables: `works`, `characters`, `editions`, `text_lines`, `lexicon_entries`, `lexicon_citations`.

Full-text search via SQLite FTS5 on both text content and lexicon entries.

See [SCHEMA.md](SCHEMA.md) for complete table definitions and example queries.

## GitHub Actions

Every push to `main` that changes `sources/` or `tools/build.py` automatically:
1. Builds the database from scratch
2. Creates a GitHub Release with the `.db` file attached
3. Uploads a build artifact (retained 90 days)

## Attribution

This database includes content from the **Perseus Digital Library** (CC BY-SA 3.0).
Applications using this data must include attribution:

> Alexander Schmidt, *Shakespeare Lexicon and Quotation Dictionary*.
> Provided by the Perseus Digital Library, Tufts University.
> Licensed under CC BY-SA 3.0.

The `sources` table in the database tracks all attribution requirements programmatically.

## License

Source data licenses vary by origin (see table above). The build tooling and schema are MIT.
