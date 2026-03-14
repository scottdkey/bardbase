# Data — Shared Reference Mappings

Curated reference data shared across all projects in the monorepo. These files map identifiers, abbreviations, and metadata between the different source editions.

## Contents

Reference JSON files will be added here as part of the DRY refactor (chunk 9b). These include:

- Work abbreviation mappings (Schmidt ↔ OSS ↔ SE ↔ Folger)
- Genre classifications
- Attribution/license definitions
- SE repository slugs

## Contract

- **db-builder** reads these files to configure its import pipeline
- **web** may read these files at build time for UI configuration
- **sources** is never modified — derived mappings belong here instead

## Validation

```bash
make data validate   # Checks all JSON files parse correctly
```
