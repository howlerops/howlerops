# HowlerOps Makefile

# Variables
GO := go
NPM := npm
WAILS := wails
BINARY_NAME := howlerops
BUILD_DIR := ./build/bin
FRONTEND_DIR := ./frontend
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -w -s"
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
.PHONY: help
help:
	@echo "$(COLOR_BOLD)HowlerOps Makefile$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Usage:$(COLOR_RESET)"
	@echo "  make [target]"
	@echo ""
	@echo "$(COLOR_BOLD)Build Targets:$(COLOR_RESET)"
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'

## build: Build the Wails desktop application
.PHONY: build
build: deps proto
	@echo "$(COLOR_BLUE)Building Wails desktop application...$(COLOR_RESET)"
	@$(WAILS) build -clean
	@echo "$(COLOR_GREEN)✓ Desktop application built$(COLOR_RESET)"

## build-debug: Build the application with debug symbols
.PHONY: build-debug
build-debug: deps proto
	@echo "$(COLOR_BLUE)Building debug version...$(COLOR_RESET)"
	@$(WAILS) build -debug -clean
	@echo "$(COLOR_GREEN)✓ Debug build complete$(COLOR_RESET)"

## build-mac: Build for macOS (universal binary)
.PHONY: build-mac
build-mac: deps proto
	@echo "$(COLOR_BLUE)Building macOS universal binary...$(COLOR_RESET)"
	@$(WAILS) build -platform darwin/universal -clean
	@echo "$(COLOR_GREEN)✓ macOS build complete$(COLOR_RESET)"

## build-windows: Build for Windows
.PHONY: build-windows
build-windows: deps proto
	@echo "$(COLOR_BLUE)Building Windows executable...$(COLOR_RESET)"
	@$(WAILS) build -platform windows/amd64 -clean
	@echo "$(COLOR_GREEN)✓ Windows build complete$(COLOR_RESET)"

## build-linux: Build for Linux
.PHONY: build-linux
build-linux: deps proto
	@echo "$(COLOR_BLUE)Building Linux executable...$(COLOR_RESET)"
	@$(WAILS) build -platform linux/amd64 -clean
	@echo "$(COLOR_GREEN)✓ Linux build complete$(COLOR_RESET)"

## deps: Install all dependencies
.PHONY: deps
deps: deps-go deps-frontend check-wails
	@echo "$(COLOR_GREEN)✓ All dependencies installed$(COLOR_RESET)"

## deps-go: Install Go dependencies
.PHONY: deps-go
deps-go:
	@echo "$(COLOR_BLUE)Installing Go dependencies...$(COLOR_RESET)"
	@$(GO) mod download
	@$(GO) mod tidy

## deps-frontend: Install Node dependencies
.PHONY: deps-frontend
deps-frontend:
	@echo "$(COLOR_BLUE)Installing frontend dependencies...$(COLOR_RESET)"
	@cd $(FRONTEND_DIR) && $(NPM) install

## build-frontend: Build Vite bundle for the desktop shell
.PHONY: build-frontend
build-frontend:
	@echo "$(COLOR_BLUE)Building frontend bundle...$(COLOR_RESET)"
	@cd $(FRONTEND_DIR) && $(NPM) run build
	@echo "$(COLOR_GREEN)✓ Frontend bundle ready$(COLOR_RESET)"

## check-wails: Check if Wails is installed
.PHONY: check-wails
check-wails:
	@echo "$(COLOR_BLUE)Checking Wails installation...$(COLOR_RESET)"
	@which wails > /dev/null || (echo "$(COLOR_YELLOW)Wails not found. Installing...$(COLOR_RESET)" && go install github.com/wailsapp/wails/v2/cmd/wails@latest)
	@echo "$(COLOR_GREEN)✓ Wails is installed$(COLOR_RESET)"

## check-node: Ensure Node.js version is compatible with Vite
.PHONY: check-node
check-node:
	@echo "$(COLOR_BLUE)Verifying Node.js version...$(COLOR_RESET)"
	@node -e '\
const [major, minor] = process.versions.node.split(".").map(Number); \
const valid = (major === 20 && minor >= 19) || (major === 22 && minor >= 12) || major > 22; \
if (!valid) { \
  console.error("Node.js 20.19+ or 22.12+ is required for the frontend dev server. Detected:", process.version); \
  process.exit(1); \
}' || (echo "$(COLOR_YELLOW)Please install a supported Node.js version (>=20.19 or >=22.12).$(COLOR_RESET)" && exit 1)
	@echo "$(COLOR_GREEN)✓ Node.js version compatible$(COLOR_RESET)"

## dev: Start Wails development mode with hot reload
.PHONY: dev
dev: check-node check-wails deps proto init-local-db
	@echo "$(COLOR_BLUE)Starting frontend dev server and Wails (hot reload)...$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)The application will open automatically$(COLOR_RESET)"
	@bash -c '\
		set -euo pipefail; \
		cd $(FRONTEND_DIR); \
		$(NPM) run dev -- --host 127.0.0.1 & \
		FRONTEND_PID=$$!; \
		trap '\''kill "$$FRONTEND_PID" 2>/dev/null || true'\'' EXIT INT TERM; \
		echo \"$(COLOR_BLUE)Waiting for Vite dev server...$(COLOR_RESET)\"; \
		sleep 2; \
		cd ..; \
		$(WAILS) dev -s -frontenddevserverurl http://localhost:5173 || true; \
		kill "$$FRONTEND_PID" 2>/dev/null || true'

## dev-browser: Start development mode in browser with hot reload
.PHONY: dev-browser
dev-browser: check-node check-wails deps proto
	@echo "$(COLOR_BLUE)Starting frontend dev server and Wails in browser (hot reload)...$(COLOR_RESET)"
	@bash -c '\
		set -euo pipefail; \
		cd $(FRONTEND_DIR); \
		$(NPM) run dev -- --host 127.0.0.1 & \
		FRONTEND_PID=$$!; \
		trap '\''kill "$$FRONTEND_PID" 2>/dev/null || true'\'' EXIT INT TERM; \
		echo \"$(COLOR_BLUE)Waiting for Vite dev server...$(COLOR_RESET)\"; \
		sleep 2; \
		cd ..; \
		$(WAILS) dev -browser -s -frontenddevserverurl http://localhost:5173 || true; \
		kill "$$FRONTEND_PID" 2>/dev/null || true'

## proto: Generate protobuf files
.PHONY: proto
proto: proto-clean
	@echo "$(COLOR_BLUE)Generating protobuf files...$(COLOR_RESET)"
	@mkdir -p $(FRONTEND_DIR)/src/generated
	@cd $(FRONTEND_DIR) && $(NPM) run proto:build || echo "$(COLOR_YELLOW)Warning: Proto generation had issues$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)✓ Protobuf files generated$(COLOR_RESET)"

## proto-clean: Clean generated protobuf files
.PHONY: proto-clean
proto-clean:
	@echo "$(COLOR_BLUE)Cleaning protobuf files...$(COLOR_RESET)"
	@rm -rf $(FRONTEND_DIR)/src/generated

## test: Run all tests
.PHONY: test
test: test-go test-frontend
	@echo "$(COLOR_GREEN)✓ All tests passed$(COLOR_RESET)"

## test-go: Run Go tests
.PHONY: test-go
test-go:
	@echo "$(COLOR_BLUE)Running Go tests...$(COLOR_RESET)"
	@$(GO) test -v -cover ./...

## test-frontend: Run frontend tests
.PHONY: test-frontend
test-frontend:
	@echo "$(COLOR_BLUE)Running frontend tests...$(COLOR_RESET)"
	@cd $(FRONTEND_DIR) && $(NPM) run lint && $(NPM) run typecheck && $(NPM) run test:run

## test-coverage: Run tests with coverage report
.PHONY: test-coverage
test-coverage:
	@echo "$(COLOR_BLUE)Generating coverage report...$(COLOR_RESET)"
	@$(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)✓ Coverage report generated at coverage.html$(COLOR_RESET)"

## lint: Run all linters
.PHONY: lint
lint: lint-go lint-frontend
	@echo "$(COLOR_GREEN)✓ All linting passed$(COLOR_RESET)"

## lint-go: Run Go linters
.PHONY: lint-go
lint-go:
	@echo "$(COLOR_BLUE)Running Go linters...$(COLOR_RESET)"
	@golangci-lint run || go fmt ./...

## lint-frontend: Run frontend linters
.PHONY: lint-frontend
lint-frontend:
	@echo "$(COLOR_BLUE)Running frontend linters...$(COLOR_RESET)"
	@cd $(FRONTEND_DIR) && $(NPM) run lint

## fmt: Format all code
.PHONY: fmt
fmt: fmt-go fmt-frontend
	@echo "$(COLOR_GREEN)✓ All code formatted$(COLOR_RESET)"

## fmt-go: Format Go code
.PHONY: fmt-go
fmt-go:
	@echo "$(COLOR_BLUE)Formatting Go code...$(COLOR_RESET)"
	@$(GO) fmt ./...
	@goimports -w . 2>/dev/null || true

## fmt-frontend: Format frontend code
.PHONY: fmt-frontend
fmt-frontend:
	@echo "$(COLOR_BLUE)Formatting frontend code...$(COLOR_RESET)"
	@cd $(FRONTEND_DIR) && $(NPM) run format

## validate: Run all validation checks
.PHONY: validate
validate: lint test
	@echo "$(COLOR_GREEN)✓ Validation complete$(COLOR_RESET)"

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf build/bin
	@rm -rf $(FRONTEND_DIR)/dist
	@rm -rf $(FRONTEND_DIR)/src/generated
	@rm -f coverage.out coverage.html
	@find . -name "*.log" -delete
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "$(COLOR_BLUE)Building Docker image...$(COLOR_RESET)"
	@docker build -t sql-studio:$(VERSION) -t sql-studio:latest .
	@echo "$(COLOR_GREEN)✓ Docker image built$(COLOR_RESET)"

## docker-run: Run Docker container
.PHONY: docker-run
docker-run:
	@echo "$(COLOR_BLUE)Running Docker container...$(COLOR_RESET)"
	@docker run -p 8580:8580 sql-studio:latest

## docker-compose-up: Start services with docker-compose
.PHONY: docker-compose-up
docker-compose-up:
	@echo "$(COLOR_BLUE)Starting docker-compose services...$(COLOR_RESET)"
	@docker-compose up -d

## docker-compose-down: Stop services with docker-compose
.PHONY: docker-compose-down
docker-compose-down:
	@echo "$(COLOR_BLUE)Stopping docker-compose services...$(COLOR_RESET)"
	@docker-compose down

## install: Install Wails and project dependencies
.PHONY: install
install:
	@echo "$(COLOR_BLUE)Installing Wails CLI...$(COLOR_RESET)"
	@go install github.com/wailsapp/wails/v2/cmd/wails@latest
	@echo "$(COLOR_BLUE)Checking Wails doctor...$(COLOR_RESET)"
	@$(WAILS) doctor
	@echo "$(COLOR_BLUE)Installing project dependencies...$(COLOR_RESET)"
	@make deps
	@echo "$(COLOR_GREEN)✓ Installation complete$(COLOR_RESET)"

## uninstall: Uninstall the binary from system
.PHONY: uninstall
uninstall:
	@echo "$(COLOR_BLUE)Uninstalling $(BINARY_NAME)...$(COLOR_RESET)"
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(COLOR_GREEN)✓ $(BINARY_NAME) uninstalled$(COLOR_RESET)"

## release: Create a new release
.PHONY: release
release: validate build
	@echo "$(COLOR_BLUE)Creating release $(VERSION)...$(COLOR_RESET)"
	@mkdir -p releases
	@tar -czf releases/$(BINARY_NAME)-$(VERSION).tar.gz -C $(BUILD_DIR) $(BINARY_NAME)
	@echo "$(COLOR_GREEN)✓ Release created: releases/$(BINARY_NAME)-$(VERSION).tar.gz$(COLOR_RESET)"

## setup: Complete development environment setup
.PHONY: setup
setup: install setup-tools
	@echo "$(COLOR_GREEN)✓ Development environment ready$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Run 'make dev' to start development mode$(COLOR_RESET)"

## setup-tools: Install required development tools
.PHONY: setup-tools
setup-tools:
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(COLOR_GREEN)✓ Development tools installed$(COLOR_RESET)"

## run: Run the application directly (development)
.PHONY: run
run: deps proto
	@echo "$(COLOR_BLUE)Running application...$(COLOR_RESET)"
	@$(GO) run .

## doctor: Check system requirements
.PHONY: doctor
doctor:
	@echo "$(COLOR_BLUE)Checking system requirements...$(COLOR_RESET)"
	@$(WAILS) doctor

.PHONY: all
all: deps build test

# SQLite Database Management

## init-local-db: Initialize local SQLite databases
.PHONY: init-local-db
init-local-db:
	@echo "$(COLOR_BLUE)Initializing local databases...$(COLOR_RESET)"
	@bash scripts/init-local-db.sh
	@echo "$(COLOR_GREEN)✓ Databases initialized at ~/.howlerops/$(COLOR_RESET)"

## reset-local-db: Reset local databases (WARNING: deletes all local data!)
.PHONY: reset-local-db
reset-local-db:
	@bash scripts/reset-local-db.sh

## backup-local-db: Backup local databases
.PHONY: backup-local-db
backup-local-db:
	@echo "$(COLOR_BLUE)Creating backup...$(COLOR_RESET)"
	@mkdir -p ~/.howlerops/backups
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S) && \
		cp ~/.howlerops/local.db ~/.howlerops/backups/local_$$TIMESTAMP.db 2>/dev/null || true && \
		cp ~/.howlerops/vectors.db ~/.howlerops/backups/vectors_$$TIMESTAMP.db 2>/dev/null || true
	@echo "$(COLOR_GREEN)✓ Backup complete in ~/.howlerops/backups/$(COLOR_RESET)"
