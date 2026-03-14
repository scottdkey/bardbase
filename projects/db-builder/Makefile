# db-builder — Go pipeline that produces the Shakespeare SQLite database
#
# Reads original texts from projects/sources/ and reference data from
# projects/data/, then builds a single SQLite database.
#
# Usage (from repo root):
#   make db-builder build         # Compile the builder binary
#   make db-builder test          # Run all tests
#   make db-builder run           # Full pipeline build
#   make db-builder run-cached    # Skip SE downloads, use cached data
#   make db-builder lint          # Run go vet
#   make db-builder clean         # Remove build artifacts

.PHONY: build test run run-cached lint cover setup clean

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
run:
	go run ./cmd/build -output build

# Run with cached SE data (skip network downloads)
run-cached:
	go run ./cmd/build -output build -skip-download

# Run a single named step (e.g., make run-step-oss)
run-step-%:
	go run ./cmd/build -step $*

# Static analysis
lint:
	go vet ./...

# Install/tidy dependencies
setup:
	go mod tidy

# Remove build artifacts
clean:
	rm -rf ../../build/ ../../bin/ coverage.out coverage.html
