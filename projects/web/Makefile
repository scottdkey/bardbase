# web — SvelteKit PWA
#
# Reads the Shakespeare SQLite database directly via better-sqlite3 and
# deploys to Cloudflare Pages (prerendered; no Go API server required).
#
# Usage (from repo root):
#   make web run         # Start dev server
#   make web build       # Production build
#   make web test        # Run tests
#   make web preview     # Preview production build
#   make web clean       # Remove build artifacts

.PHONY: dev run build test preview clean setup

# Start the SvelteKit development server
dev run:
	npm run dev

# Production build
build:
	npm run build

# Run tests
test:
	@if [ -f package.json ]; then \
		npm test; \
	else \
		echo "web: not yet initialized (no package.json)"; \
	fi

# Preview the production build locally
preview:
	npm run preview

# Install dependencies
setup:
	npm install

# Remove build artifacts
clean:
	rm -rf .svelte-kit/ build/ node_modules/.cache
