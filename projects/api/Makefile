# api — Go HTTP API server (part of the capell module)
#
# Usage (from repo root):
#   make api build        # Compile the API binary → build/bardbase-api
#   make api run          # Run the API server locally
#   make api docker       # Build Docker production image

.PHONY: build run docker

build:
	@$(MAKE) -C ../capell api-build

run:
	@$(MAKE) -C ../capell api-run

docker:
	@$(MAKE) -C ../capell api-docker
