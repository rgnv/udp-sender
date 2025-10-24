.PHONY: build test test-root coverage coverage-root lint clean run deps help

# Version can be overridden: make build VERSION=v1.0.0
VERSION ?= dev

# Build the application
build:
	go build -ldflags="-X main.Version=$(VERSION)" -o udp-sender .

# Run tests (without root - some tests will skip)
test:
	go test -v ./...

# Run tests with root privileges
test-root:
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "This target requires root privileges. Run: sudo make test-root"; \
		exit 1; \
	fi
	go test -v -count=1 ./...

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with coverage as root (bypasses test cache)
coverage-root:
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "This target requires root privileges. Run: sudo make coverage-root"; \
		exit 1; \
	fi
	go test -v -race -count=1 -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f udp-sender coverage.out coverage.html

# Run the application (requires root)
run: build
	@echo "Note: This requires root privileges"
	@echo "Run: sudo make run"
	./udp-sender

# Install dependencies
deps:
	go mod download
	go mod verify

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application (set VERSION to override version)"
	@echo "                   Example: make build VERSION=v1.0.0"
	@echo "  test           - Run tests (without root, some will skip)"
	@echo "  test-root      - Run all tests with root privileges (bypasses cache)"
	@echo "  coverage       - Run tests with coverage report"
	@echo "  coverage-root  - Run tests with coverage as root (includes raw socket tests)"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Build and run the application (requires root)"
	@echo "  deps           - Download and verify dependencies"
	@echo "  help           - Show this help message"

