# web — SvelteKit application (future)
#
# This project will contain the SvelteKit PWA that reads the Shakespeare
# SQLite database at build time and deploys to Cloudflare Pages/Workers.
#
# Usage (from repo root):
#   make web dev         # Start dev server
#   make web build       # Production build
#   make web test        # Run tests
#   make web preview     # Preview production build
#   make web clean       # Remove build artifacts

.PHONY: dev build test preview clean setup

# Start the SvelteKit development server
dev:
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
