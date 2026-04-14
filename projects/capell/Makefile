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

.PHONY: build test run run-fresh lint cover setup clean release release-upload api-build api-run api-docker

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
# Builds bardbase-search.sql for import into Cloudflare D1.
# Requires bardbase.db to exist (run `make capell run` first).
#
# Usage:
#   make capell export-search

export-search:
	go run ./cmd/export-search -src ../../build/bardbase.db -out ../../build/bardbase-search.sql

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
