.PHONY: build install test clean build-all

BINARY_NAME=claude-langfuse
VERSION=1.0.0

build:
	go build -o bin/$(BINARY_NAME) ./cmd/claude-langfuse

install:
	go install ./cmd/claude-langfuse

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Cross-compilation for all platforms
build-all: clean
	mkdir -p bin
	# macOS
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/claude-langfuse
	GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/claude-langfuse
	# Linux
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/claude-langfuse
	GOOS=linux GOARCH=arm64 go build -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/claude-langfuse

# Run the monitor
run:
	go run ./cmd/claude-langfuse start

# Download dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build binary for current platform"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  test        - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean       - Remove build artifacts"
	@echo "  build-all   - Build for all platforms (darwin/linux, amd64/arm64)"
	@echo "  run         - Run the monitor"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  fmt         - Format code"
	@echo "  lint        - Run linter"
