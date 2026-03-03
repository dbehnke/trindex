.PHONY: build test run clean lint fmt docker-build docker-up docker-down install-hooks

# Binary name
BINARY=trindex

# Build the binary
build:
	go build -o $(BINARY) ./cmd/trindex

# Run tests
test:
	go test -v ./...

# Run the application (requires Postgres)
run: build
	./$(BINARY)

# Clean build artifacts
clean:
	rm -f $(BINARY)
	go clean

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Build Docker image
docker-build:
	docker build -t $(BINARY) .

# Start Docker Compose services
docker-up:
	docker compose up -d

# Stop Docker Compose services
docker-down:
	docker compose down

# Install git hooks
install-hooks:
	./scripts/install-hooks.sh

# Download dependencies
deps:
	go mod download
	go mod tidy

# Check for security vulnerabilities
audit:
	go list -json -m all | nancy sleuth

# Run all checks before commit
check: fmt lint test build
