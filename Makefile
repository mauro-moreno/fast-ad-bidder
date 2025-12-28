.PHONY: test test-coverage test-unit test-integration test-contract test-all clean build run

# Default target
all: test build

# Run all tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=./src/... ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total
	@echo "Coverage report generated: coverage.html"

# Run tests without coverage (faster)
test:
	@go test -v ./...

# Run unit tests only
test-unit:
	@go test -v ./tests/unit/...

# Run integration tests only
test-integration:
	@go test -v ./tests/integration/...

# Run contract tests only
test-contract:
	@go test -v ./tests/contract/...

# Run all tests with verbose output
test-all:
	@go test -v -race ./...

# Build the bidder binary
build:
	@echo "Building bidder..."
	@go build -o bin/bidder cmd/bidder/main.go
	@echo "Build complete: bin/bidder"

# Run the bidder locally
run:
	@go run cmd/bidder/main.go

# Run load tests
load-test:
	@echo "Running k6 load test..."
	@k6 run tests/load/bid_endpoint.js

# Clean build artifacts
clean:
	@rm -f bin/bidder coverage.out coverage.html
	@echo "Clean complete"

# Install dependencies
deps:
	@go mod download
	@go mod tidy

# Format code
fmt:
	@go fmt ./...

# Run linter
lint:
	@golangci-lint run

# Docker build
docker-build:
	@docker build -t fast-ad-bidder:latest .

# Start services
services-up:
	@docker-compose up -d

# Stop services
services-down:
	@docker-compose down

# Show help
help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests only"
	@echo "  test-contract     - Run contract tests only"
	@echo "  build             - Build the bidder binary"
	@echo "  run               - Run the bidder locally"
	@echo "  load-test         - Run k6 load tests"
	@echo "  clean             - Clean build artifacts"
	@echo "  deps              - Install dependencies"
	@echo "  fmt               - Format code"
	@echo "  docker-build      - Build Docker image"
	@echo "  services-up       - Start Docker Compose services"
	@echo "  services-down     - Stop Docker Compose services"
