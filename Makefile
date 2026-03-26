# Bardbase — Monorepo Root Makefile
#
# Delegates all actions to per-project Makefiles under projects/.
#
# Usage:
#   make capell test           → runs tests for the Go pipeline
#   make capell build          → compiles the builder binary
#   make web dev               → starts the SvelteKit dev server (future)
#   make sources verify        → checksums original source files
#   make data validate         → validates reference JSON files
#   make test-all              → runs tests across all projects
#   make clean-all             → cleans build artifacts in all projects

.PHONY: capell web api sources data test-all clean-all help

# All project directories that have their own Makefile
PROJECTS := capell web api sources data

# ─── Namespace Delegation ─────────────────────────────────────────────
# Captures `make <project> <action>` and forwards to projects/<project>/Makefile.
# Example: `make capell test` → `make -C projects/capell test`
capell:
	@$(MAKE) -C projects/capell $(filter-out $@,$(MAKECMDGOALS))

web:
	@$(MAKE) -C projects/web $(filter-out $@,$(MAKECMDGOALS))

api:
	@$(MAKE) -C projects/api $(filter-out $@,$(MAKECMDGOALS))

sources:
	@$(MAKE) -C projects/sources $(filter-out $@,$(MAKECMDGOALS))

data:
	@$(MAKE) -C projects/data $(filter-out $@,$(MAKECMDGOALS))

# Swallow extra goal arguments so make doesn't complain about missing targets.
# Without this, `make capell test` would error on "test" as a root target.
%:
	@:

# ─── Cross-Project Convenience ────────────────────────────────────────

# Run tests in every project that has a test target
test-all:
	@echo "Running tests across all projects..."
	@for p in $(PROJECTS); do \
		if $(MAKE) -C projects/$$p -n test >/dev/null 2>&1; then \
			echo "\n=== projects/$$p ==="; \
			$(MAKE) -C projects/$$p test || exit 1; \
		fi; \
	done
	@echo "\nAll tests passed."

# Clean build artifacts in every project
clean-all:
	@for p in $(PROJECTS); do \
		$(MAKE) -C projects/$$p clean 2>/dev/null || true; \
	done
	@echo "All projects cleaned."

# Show available commands
help:
	@echo "Bardbase Monorepo"
	@echo ""
	@echo "Usage: make <project> <action>"
	@echo ""
	@echo "Projects:"
	@echo "  capell      Go pipeline (build, test, run, release, lint, clean)"
	@echo "  api         Go HTTP API server (build, run, docker)"
	@echo "  web         SvelteKit app (run, build, test, preview, clean)"
	@echo "  sources     Original texts — READ ONLY (verify, list, stats)"
	@echo "  data        Reference JSON mappings (validate, list, clean)"
	@echo ""
	@echo "Cross-project:"
	@echo "  test-all    Run tests in all projects"
	@echo "  clean-all   Clean build artifacts in all projects"
