# data — Shared reference data for the Heminge monorepo
#
# Contains JSON mapping files that both db-builder and web read from.
# These are NOT original sources — they are derived/curated reference data
# maintained by this project.
#
# Usage (from repo root):
#   make data validate    # Validate all JSON files parse correctly
#   make data list        # List all data files
#   make data clean       # No-op (data files are version-controlled)

.PHONY: validate list test clean

# Validate every JSON file in this directory parses correctly
validate:
	@echo "Validating reference data..."
	@errors=0; \
	for f in $$(find . -name '*.json' -not -path '*/\.*'); do \
		if python3 -c "import json; json.load(open('$$f'))" 2>/dev/null; then \
			echo "  ✓ $$f"; \
		else \
			echo "  ✗ $$f (invalid JSON)"; \
			errors=$$((errors + 1)); \
		fi; \
	done; \
	if [ $$errors -gt 0 ]; then \
		echo "$$errors file(s) failed validation."; \
		exit 1; \
	fi; \
	echo "All data files valid."

# List all data files
list:
	@find . -type f -not -name 'Makefile' -not -name 'README.md' -not -path '*/\.*' | sort

# Alias for validate (so test-all works)
test: validate

# No-op — data files are version-controlled
clean:
	@echo "data: nothing to clean"
