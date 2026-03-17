# Data — Shared Reference Mappings

Curated reference data shared across all projects in the monorepo. These files map identifiers, abbreviations, and metadata between the different source editions.

## Contents

| File | Description |
|------|-------------|
| `oss_to_schmidt.json` | OSS work IDs → Schmidt abbreviations (43 entries) |
| `schmidt_works.json` | Schmidt abbreviations → titles, Perseus IDs, work types (70 entries incl. aliases) |
| `se_play_repos.json` | Standard Ebooks repo names → OSS work IDs (37 plays) |
| `se_poetry_map.json` | SE poetry article IDs → OSS work IDs (4 poems) |
| `folio_play_titles.json` | Normalized First Folio play head text → OSS work IDs (35 plays) |
| `folger_slugs.json` | OSS work IDs → Folger Shakespeare Library URL slugs (35 plays) |
| `perseus_to_schmidt.json` | Perseus short work codes → Schmidt abbreviations (41 entries) |
| `genre_map.json` | Single-letter genre codes → full type names |
| `attributions.json` | Attribution display rules per source (required fields, format, links) |

## Contract

- **capell** loads all files at init time via auto-discovery of the repo root
- **web** may read these files at build time for UI configuration
- **sources** is never modified — derived mappings belong here instead

## Validation

```bash
make data validate   # checks all JSON files parse correctly
```
