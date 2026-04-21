# capell — Go pipeline that produces the Shakespeare SQLite database
#
# Reads original texts from projects/sources/ and reference data from
# projects/data/, then builds a single SQLite database.
#
# Usage (from repo root):
#   make capell build         # Compile the builder binary
#   make capell test          # Run all tests
#   make capell run           # Full pipeline build (uses cached sources)
#   make capell run-fresh     # Full pipeline build, re-download source files
#   make capell lint          # Run go vet
#   make capell clean         # Remove build artifacts

.PHONY: build test run run-fresh lint cover setup clean release release-upload api-build api-run api-docker export-search

# Compile the CLI binary to the repo-root bin/ directory
build:
	go build -o ../../bin/shakespeare-build ./cmd/build

# Run all tests with verbose output
test:
	go test ./... -v

# Run tests with coverage report
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run the full build pipeline (output goes to repo-root build/)
# Use EXCLUDE to skip sources: make capell run EXCLUDE=folger
run: build
	../../bin/shakespeare-build -output build $(if $(EXCLUDE),$(foreach src,$(subst $(comma),$(space),$(EXCLUDE)),--exclude $(src)))

comma := ,
space := $(subst ,, )

# Re-download all source files before building
run-fresh: build
	../../bin/shakespeare-build -output build -force-download

# Run a single named step (e.g., make run-step-oss)
run-step-%: build
	../../bin/shakespeare-build -step $*

# Static analysis
lint:
	go vet ./...

# Install/tidy dependencies
setup:
	go mod tidy

# ─── Release ─────────────────────────────────────────────────────────────
#
# Build the database locally and publish as a GitHub release.
# The API container downloads the latest release on startup (entrypoint.sh).
#
# Usage:
#   make capell release                    # Build and release (excludes Folger by default)
#   make capell release EXCLUDE=           # Build and release with all sources
#   make capell release EXCLUDE=folger,wordhoard

EXCLUDE ?= folger
VERSION := v$(shell date -u +%Y.%m.%d)-$(shell git rev-parse --short HEAD)
DB := ../../build/bardbase.db

release: run release-upload

release-upload:
	@if [ ! -f "$(DB)" ]; then echo "Error: $(DB) not found. Run 'make run' first."; exit 1; fi
	@echo "Creating release $(VERSION)..."
	@DB_SIZE=$$(du -h "$(DB)" | cut -f1) && \
	TEXT_LINES=$$(sqlite3 "$(DB)" "SELECT COUNT(*) FROM text_lines") && \
	LEXICON=$$(sqlite3 "$(DB)" "SELECT COUNT(*) FROM lexicon_entries") && \
	CITATIONS=$$(sqlite3 "$(DB)" "SELECT COUNT(*) FROM lexicon_citations") && \
	WORKS=$$(sqlite3 "$(DB)" "SELECT COUNT(*) FROM works") && \
	gh release create "$(VERSION)" \
		--title "Bardbase $(VERSION)" \
		--notes "## Bardbase Database Build$$(printf '\n\n')| Metric | Count |$$(printf '\n')|--------|-------|$$(printf '\n')| **Works** | $$WORKS |$$(printf '\n')| **Text lines** | $$TEXT_LINES |$$(printf '\n')| **Lexicon entries** | $$LEXICON |$$(printf '\n')| **Citations** | $$CITATIONS |$$(printf '\n')| **Database size** | $$DB_SIZE |$$(printf '\n\n')Excluded sources: $(EXCLUDE)$$(printf '\n')Built from commit $$(git rev-parse HEAD)" \
		"$(DB)"
	@echo "✓ Released $(VERSION)"

# ─── Search DB Export ────────────────────────────────────────────────────
#
# Two-step export: the Go tool emits a D1/Turso-compatible SQL dump
# (`bardbase-search.sql`), then sqlite3 builds `bardbase-search.db` from it.
# The .db file is the canonical runtime artifact — it's what Turso imports
# and what local dev reads via `db.ts`'s file:// fallback.
#
# Requires bardbase.db (the full intermediate produced by `make capell run`).
#
# Usage:
#   make capell export-search
#
# Then:
#   turso db import build/bardbase-search.db    # production upload

SEARCH_SQL := ../../build/bardbase-search.sql
SEARCH_DB := ../../build/bardbase-search.db

export-search:
	@echo "→ generating $(SEARCH_SQL)"
	@go run ./cmd/export-search -src ../../build/bardbase.db -out $(SEARCH_SQL)
	@ls -lh $(SEARCH_SQL)
	@echo "→ building $(SEARCH_DB) from SQL (fast mode: single txn, no journal)"
	@rm -f $(SEARCH_DB)
	@(echo "PRAGMA synchronous=OFF;"; \
	  echo "PRAGMA journal_mode=OFF;"; \
	  echo "PRAGMA temp_store=MEMORY;"; \
	  echo "PRAGMA cache_size=-1000000;"; \
	  cat $(SEARCH_SQL)) | sqlite3 $(SEARCH_DB)
	@sqlite3 $(SEARCH_DB) "PRAGMA journal_mode=WAL" >/dev/null
	@ls -lh $(SEARCH_DB)
	@echo "→ row counts"
	@sqlite3 $(SEARCH_DB) \
		"SELECT 'works' AS t, COUNT(*) FROM works \
		 UNION ALL SELECT 'sources', COUNT(*) FROM sources \
		 UNION ALL SELECT 'editions', COUNT(*) FROM editions \
		 UNION ALL SELECT 'attributions', COUNT(*) FROM attributions \
		 UNION ALL SELECT 'characters', COUNT(*) FROM characters \
		 UNION ALL SELECT 'text_divisions', COUNT(*) FROM text_divisions \
		 UNION ALL SELECT 'text_lines', COUNT(*) FROM text_lines \
		 UNION ALL SELECT 'line_mappings', COUNT(*) FROM line_mappings \
		 UNION ALL SELECT 'lexicon_entries', COUNT(*) FROM lexicon_entries \
		 UNION ALL SELECT 'lexicon_senses', COUNT(*) FROM lexicon_senses \
		 UNION ALL SELECT 'lexicon_citations', COUNT(*) FROM lexicon_citations \
		 UNION ALL SELECT 'citation_matches', COUNT(*) FROM citation_matches \
		 UNION ALL SELECT 'reference_entries', COUNT(*) FROM reference_entries \
		 UNION ALL SELECT 'reference_citations', COUNT(*) FROM reference_citations \
		 UNION ALL SELECT 'reference_citation_matches', COUNT(*) FROM reference_citation_matches \
		 UNION ALL SELECT 'text_fts', COUNT(*) FROM text_fts \
		 UNION ALL SELECT 'lexicon_fts', COUNT(*) FROM lexicon_fts \
		 UNION ALL SELECT 'reference_fts', COUNT(*) FROM reference_fts \
		 UNION ALL SELECT 'reference_spans', COUNT(*) FROM reference_spans \
		 UNION ALL SELECT 'lexicon_drawer', COUNT(*) FROM lexicon_drawer;"
	@echo "✓ export-search complete — $(SEARCH_SQL) and $(SEARCH_DB) ready"
	@echo "   • Turso upload: turso db import $(SEARCH_DB)"
	@echo "   • Local dev:    $(SEARCH_DB) via db.ts file:// fallback"


# ─── API Server (invoked via `make api build/run/docker` at repo root) ───

# Compile the API server binary (output to build/ alongside the database)
api-build:
	go build -o ../../build/bardbase-api ./cmd/api

# Run the API server locally (uses repo-root build/ database)
api-run: api-build
	DB_PATH=../../build/bardbase.db ../../build/bardbase-api

# Build Docker image for the API server
api-docker:
	docker build -t bardbase-api .

# ─── Cleanup ─────────────────────────────────────────────────────────────

# Remove build artifacts
clean:
	rm -rf ../../build/ ../../bin/ coverage.out coverage.html
