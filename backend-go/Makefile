.PHONY: build clean test proto run dev docker

# Variables
BINARY_NAME=sql-studio-backend
DOCKER_IMAGE=sql-studio/backend-go
PROTO_DIR=../proto
API_DIR=api
BUILD_DIR=build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(API_DIR)
	@go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@mkdir -p $(API_DIR)
	@# Install required tools if not present
	@which protoc > /dev/null || (echo "protoc is required but not installed." && exit 1)
	@which protoc-gen-go > /dev/null || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@which protoc-gen-go-grpc > /dev/null || go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@which protoc-gen-grpc-gateway > /dev/null || go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	@which protoc-gen-openapiv2 > /dev/null || go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

	@# Generate Go code for each proto file
	protoc -I$(PROTO_DIR) \
		--go_out=$(API_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(API_DIR) \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(API_DIR) \
		--grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=docs \
		$(PROTO_DIR)/*.proto

# Run the application in development mode
dev: proto
	@echo "Running in development mode..."
	go run cmd/server/main.go

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

# Run Docker container
docker-run: docker
	@echo "Running Docker container..."
	docker run -p 8500:8500 -p 9500:9500 $(DOCKER_IMAGE)

# Lint the code
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

# Format the code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	goimports -w .

# Mod tidy
tidy:
	@echo "Running go mod tidy..."
	go mod tidy

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod verify

# Run all checks (format, lint, test)
check: fmt lint test

# Setup development environment
setup: deps proto
	@echo "Development environment setup complete!"

# Release build with optimizations
release:
	@echo "Building release version..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 \
		cmd/server/main.go
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 \
		cmd/server/main.go
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe \
		cmd/server/main.go

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  bench       - Run benchmarks"
	@echo "  proto       - Generate protobuf code"
	@echo "  dev         - Run in development mode"
	@echo "  run         - Run the built application"
	@echo "  docker      - Build Docker image"
	@echo "  docker-run  - Run Docker container"
	@echo "  lint        - Run linter"
	@echo "  fmt         - Format code"
	@echo "  tidy        - Run go mod tidy"
	@echo "  deps        - Install dependencies"
	@echo "  check       - Run all checks (format, lint, test)"
	@echo "  setup       - Setup development environment"
	@echo "  release     - Build release versions"
	@echo "  help        - Show this help message"