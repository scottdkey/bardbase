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

.PHONY: build test run run-fresh lint cover setup clean

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

# Re-download all source files before building
run-fresh:
	go run ./cmd/build -output build -force-download

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
