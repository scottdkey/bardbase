.PHONY: build test clean setup run

# Build the CLI binary
build:
	go build -o bin/shakespeare-build ./cmd/build

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run the build (full pipeline)
run:
	go run ./cmd/build

# Run with skip-download (use cached SE data)
run-cached:
	go run ./cmd/build -skip-download

# Run a single step
run-step-%:
	go run ./cmd/build -step $*

# Install dependencies
setup:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf build/ bin/ coverage.out coverage.html
