# sources — Original Shakespeare texts (READ-ONLY)
#
# This project contains unmodified source material from:
#   - OSS/Moby Shakespeare (Public Domain)
#   - Standard Ebooks (CC0)
#   - Perseus Schmidt Lexicon (CC BY-SA 3.0)
#
# ⚠️  NOTHING in this directory should ever be modified by any automated process.
# These are original source files. If you need derived data, put it in projects/data/.
#
# Usage (from repo root):
#   make sources verify    # Verify all source files exist and are non-empty
#   make sources list      # List all source files
#   make sources stats     # Count files per source

.PHONY: verify list stats test clean

# Verify all expected source files exist and are non-empty
verify:
	@echo "Verifying source files..."
	@test -f oss/oss-db-full.sql || (echo "MISSING: oss/oss-db-full.sql" && exit 1)
	@test -f lexicon/entry_list.json || (echo "MISSING: lexicon/entry_list.json" && exit 1)
	@test -d lexicon/entries || (echo "MISSING: lexicon/entries/" && exit 1)
	@count=$$(find lexicon/entries -name '*.xml' | wc -l); \
		if [ $$count -lt 1000 ]; then \
			echo "WARN: Only $$count lexicon XML files (expected ~20,000)"; \
		else \
			echo "  ✓ lexicon/entries: $$count XML files"; \
		fi
	@se_count=$$(find se -name '*.json' 2>/dev/null | wc -l); \
		echo "  ✓ se/: $$se_count JSON files"
	@echo "All sources verified."

# List every source file
list:
	@find . -type f -not -name 'Makefile' -not -name 'README.md' -not -name 'SOURCES.md' | sort

# Show file counts per source directory
stats:
	@echo "Source file statistics:"
	@echo "  oss:               $$(find oss -type f 2>/dev/null | wc -l) files"
	@echo "  lexicon:           $$(find lexicon -type f 2>/dev/null | wc -l) files"
	@echo "  se:                $$(find se -type f 2>/dev/null | wc -l) files"
	@echo "  perseus-plays:     $$(find perseus-plays -type f 2>/dev/null | wc -l) files"
	@echo "  eebo-tcp:          $$(find eebo-tcp -type f 2>/dev/null | wc -l) files"
	@echo "  folger:            $$(find folger -type f 2>/dev/null | wc -l) files"
	@echo "  bowdler:           $$(find bowdler -type f 2>/dev/null | wc -l) files"
	@echo "  abbott:            $$(find abbott -type f 2>/dev/null | wc -l) files"
	@echo "  bartlett:          $$(find bartlett -type f 2>/dev/null | wc -l) files"
	@echo "  onions:            $$(find onions -type f 2>/dev/null | wc -l) files"
	@echo "  henley-farmer:     $$(find henley-farmer -type f 2>/dev/null | wc -l) files"
	@echo "  shakespeare-census:$$(find shakespeare-census -type f 2>/dev/null | wc -l) files"

# Alias for verify (so test-all works)
test: verify

# No-op — sources are read-only, nothing to clean
clean:
	@echo "sources: nothing to clean (read-only)"
