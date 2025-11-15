.PHONY: build clean test proto run dev docker

# Variables
BINARY_NAME=sql-studio-backend
DOCKER_IMAGE=sql-studio/backend-go
PROTO_DIR=../proto
API_DIR=api
BUILD_DIR=build

# Version variables (can be overridden: make release VERSION=2.0.0)
VERSION ?= 2.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags="-X main.Version=$(VERSION) -X main.Commit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE) -w -s"

# Build the application (server)
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go

# Build the CLI tool
build-cli:
	@echo "Building Howlerops CLI..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/sqlstudio cmd/sqlstudio/main.go
	@echo "✓ Built: $(BUILD_DIR)/sqlstudio"

# Build both server and CLI
build-all: build build-cli

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

# -------------------------------------------------------------------
# Local Development Commands
# -------------------------------------------------------------------

# Setup local development environment
setup-local-dev:
	@echo "==================================================================="
	@echo "Setting up Howlerops local development environment..."
	@echo "==================================================================="
	@mkdir -p ./data
	@mkdir -p ./logs
	@if [ ! -f .env.development ]; then \
		echo "Creating .env.development from example..."; \
		cp .env.example .env.development; \
		echo ""; \
		echo "✓ Created .env.development"; \
		echo "⚠ WARNING: Update .env.development if you need custom configuration"; \
	else \
		echo "✓ .env.development already exists"; \
	fi
	@echo ""
	@echo "==================================================================="
	@echo "Local environment setup complete!"
	@echo "Next steps:"
	@echo "  1. Run 'make migrate' to initialize the database"
	@echo "  2. Run 'make dev' to start the development server"
	@echo "==================================================================="

# Run database migrations
migrate: setup-local-dev
	@echo "Running database migrations..."
	@env -u LOG_FORMAT -u LOG_OUTPUT ENVIRONMENT=development go run cmd/migrate/main.go
	@echo "✓ Migrations complete"

# Run the application in development mode
dev:
	@echo "==================================================================="
	@echo "Starting Howlerops Backend in DEVELOPMENT mode..."
	@echo "==================================================================="
	@if [ ! -f .env.development ]; then \
		echo "⚠ .env.development not found. Running setup..."; \
		make setup-local-dev; \
	fi
	@if [ ! -f ./data/development.db ]; then \
		echo "⚠ Database not found. Running migrations..."; \
		make migrate; \
	fi
	@echo ""
	@echo "Starting server..."
	@echo "  - HTTP/REST API:  http://localhost:8080"
	@echo "  - gRPC API:       localhost:9090"
	@echo "  - Metrics:        http://localhost:9100/metrics"
	@echo "  - Database:       ./data/development.db (SQLite)"
	@echo "  - Environment:    development"
	@echo ""
	@echo "Press Ctrl+C to stop the server"
	@echo "==================================================================="
	@env -u LOG_FORMAT -u LOG_OUTPUT ENVIRONMENT=development go run cmd/server/main.go

# Clean local database
clean-db:
	@echo "Cleaning local database..."
	@rm -f ./data/development.db ./data/development.db-shm ./data/development.db-wal
	@echo "✓ Local database cleaned"

# Reset local development environment
reset-dev: clean-db
	@echo "Resetting local development environment..."
	@make setup-local-dev
	@make migrate
	@echo ""
	@echo "==================================================================="
	@echo "✓ Local environment reset complete!"
	@echo "Run 'make dev' to start the server"
	@echo "==================================================================="

# Test with local database
test-local: setup-local-dev
	@echo "Running tests with local database..."
	@ENVIRONMENT=development go test ./... -v

# Database shell (SQLite CLI)
db-shell:
	@if [ ! -f ./data/development.db ]; then \
		echo "⚠ Database not found. Run 'make migrate' first."; \
		exit 1; \
	fi
	@echo "Opening SQLite shell for development database..."
	@echo "Type '.help' for SQLite commands, '.exit' to quit"
	@sqlite3 ./data/development.db

# Show database tables
db-tables:
	@if [ ! -f ./data/development.db ]; then \
		echo "⚠ Database not found. Run 'make migrate' first."; \
		exit 1; \
	fi
	@echo "Tables in development database:"
	@sqlite3 ./data/development.db ".tables"

# Show database schema
db-schema:
	@if [ ! -f ./data/development.db ]; then \
		echo "⚠ Database not found. Run 'make migrate' first."; \
		exit 1; \
	fi
	@echo "Database schema:"
	@sqlite3 ./data/development.db ".schema"

# Backup local database
db-backup:
	@if [ ! -f ./data/development.db ]; then \
		echo "⚠ Database not found. Nothing to backup."; \
		exit 1; \
	fi
	@mkdir -p ./backups
	@cp ./data/development.db ./backups/development-$(shell date +%Y%m%d-%H%M%S).db
	@echo "✓ Database backed up to ./backups/"

# Restore latest database backup
db-restore:
	@LATEST=$$(ls -t ./backups/development-*.db 2>/dev/null | head -1); \
	if [ -z "$$LATEST" ]; then \
		echo "⚠ No backups found in ./backups/"; \
		exit 1; \
	fi; \
	echo "Restoring database from $$LATEST..."; \
	cp $$LATEST ./data/development.db; \
	echo "✓ Database restored"

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

# -------------------------------------------------------------------
# Release Build Commands
# -------------------------------------------------------------------

# Build for Darwin AMD64 (Intel Macs)
.PHONY: release-darwin-amd64
release-darwin-amd64:
	@echo "Building for Darwin AMD64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build \
		$(LDFLAGS) \
		-o $(BUILD_DIR)/sql-studio-darwin-amd64 \
		cmd/server/main.go
	@echo "✓ Built: $(BUILD_DIR)/sql-studio-darwin-amd64"

# Build for Darwin ARM64 (Apple Silicon)
.PHONY: release-darwin-arm64
release-darwin-arm64:
	@echo "Building for Darwin ARM64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build \
		$(LDFLAGS) \
		-o $(BUILD_DIR)/sql-studio-darwin-arm64 \
		cmd/server/main.go
	@echo "✓ Built: $(BUILD_DIR)/sql-studio-darwin-arm64"

# Build for Linux AMD64
.PHONY: release-linux-amd64
release-linux-amd64:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
		$(LDFLAGS) \
		-o $(BUILD_DIR)/sql-studio-linux-amd64 \
		cmd/server/main.go
	@echo "✓ Built: $(BUILD_DIR)/sql-studio-linux-amd64"

# Build for Linux ARM64
.PHONY: release-linux-arm64
release-linux-arm64:
	@echo "Building for Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build \
		$(LDFLAGS) \
		-o $(BUILD_DIR)/sql-studio-linux-arm64 \
		cmd/server/main.go
	@echo "✓ Built: $(BUILD_DIR)/sql-studio-linux-arm64"

# Build for Windows AMD64
.PHONY: release-windows-amd64
release-windows-amd64:
	@echo "Building for Windows AMD64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build \
		$(LDFLAGS) \
		-o $(BUILD_DIR)/sql-studio-windows-amd64.exe \
		cmd/server/main.go
	@echo "✓ Built: $(BUILD_DIR)/sql-studio-windows-amd64.exe"

# Generate checksums for all built binaries
.PHONY: checksums
checksums:
	@echo "Generating checksums..."
	@cd $(BUILD_DIR) && shasum -a 256 sql-studio-* > checksums.txt
	@echo "✓ Checksums generated: $(BUILD_DIR)/checksums.txt"
	@cat $(BUILD_DIR)/checksums.txt

# Build all platform binaries and create archives
.PHONY: release-all
release-all: clean release-darwin-amd64 release-darwin-arm64 release-linux-amd64 release-linux-arm64 release-windows-amd64
	@echo ""
	@echo "==================================================================="
	@echo "Building release archives for version $(VERSION)..."
	@echo "==================================================================="
	@echo ""

	@# Create .tar.gz for Unix platforms
	@echo "Creating Darwin AMD64 archive..."
	@cd $(BUILD_DIR) && tar -czf sql-studio-$(VERSION)-darwin-amd64.tar.gz sql-studio-darwin-amd64

	@echo "Creating Darwin ARM64 archive..."
	@cd $(BUILD_DIR) && tar -czf sql-studio-$(VERSION)-darwin-arm64.tar.gz sql-studio-darwin-arm64

	@echo "Creating Linux AMD64 archive..."
	@cd $(BUILD_DIR) && tar -czf sql-studio-$(VERSION)-linux-amd64.tar.gz sql-studio-linux-amd64

	@echo "Creating Linux ARM64 archive..."
	@cd $(BUILD_DIR) && tar -czf sql-studio-$(VERSION)-linux-arm64.tar.gz sql-studio-linux-arm64

	@# Create .zip for Windows
	@echo "Creating Windows AMD64 archive..."
	@cd $(BUILD_DIR) && zip -q sql-studio-$(VERSION)-windows-amd64.zip sql-studio-windows-amd64.exe

	@# Generate checksums for archives
	@echo ""
	@echo "Generating checksums for archives..."
	@cd $(BUILD_DIR) && shasum -a 256 sql-studio-$(VERSION)-*.tar.gz sql-studio-$(VERSION)-*.zip > checksums.txt

	@echo ""
	@echo "==================================================================="
	@echo "✓ Release build complete for version $(VERSION)"
	@echo "==================================================================="
	@echo ""
	@echo "Built artifacts:"
	@ls -lh $(BUILD_DIR)/sql-studio-$(VERSION)-*
	@echo ""
	@echo "Checksums:"
	@cat $(BUILD_DIR)/checksums.txt
	@echo ""
	@echo "Next steps:"
	@echo "  1. Test the binaries: ./$(BUILD_DIR)/sql-studio-<platform>"
	@echo "  2. Create git tag: git tag -a v$(VERSION) -m 'Release v$(VERSION)'"
	@echo "  3. Push tag: git push origin v$(VERSION)"
	@echo "  4. Upload artifacts to GitHub Releases"
	@echo "==================================================================="

# Quick release shortcut (just current platform)
.PHONY: release
release:
	@echo "Building release version for current platform..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(GIT_COMMIT)"
	@echo "Date:    $(BUILD_DATE)"

# Help
help:
	@echo "==================================================================="
	@echo "Howlerops Backend - Available Make Targets"
	@echo "==================================================================="
	@echo ""
	@echo "🚀 Quick Start:"
	@echo "  make dev              - One-command start (auto-setup + migrate + run)"
	@echo ""
	@echo "📦 Local Development:"
	@echo "  setup-local-dev       - Setup local dev environment (create dirs + .env)"
	@echo "  migrate               - Run database migrations"
	@echo "  dev                   - Start development server with auto-setup"
	@echo "  clean-db              - Delete local SQLite database"
	@echo "  reset-dev             - Clean database + fresh setup + migrate"
	@echo "  test-local            - Run tests with local database"
	@echo ""
	@echo "🗄️  Database Tools:"
	@echo "  db-shell              - Open SQLite CLI for local database"
	@echo "  db-tables             - List all tables in database"
	@echo "  db-schema             - Show complete database schema"
	@echo "  db-backup             - Backup database to ./backups/"
	@echo "  db-restore            - Restore latest database backup"
	@echo ""
	@echo "🏗️  Building:"
	@echo "  build                 - Build the server binary"
	@echo "  build-cli             - Build the CLI tool (version/update commands)"
	@echo "  build-all             - Build both server and CLI"
	@echo "  run                   - Run the built server binary"
	@echo "  release               - Build release version for current platform"
	@echo "  release-all           - Build release for all platforms + archives"
	@echo "  release-darwin-amd64  - Build for macOS Intel"
	@echo "  release-darwin-arm64  - Build for macOS Apple Silicon"
	@echo "  release-linux-amd64   - Build for Linux AMD64"
	@echo "  release-linux-arm64   - Build for Linux ARM64"
	@echo "  release-windows-amd64 - Build for Windows AMD64"
	@echo "  checksums             - Generate SHA256 checksums"
	@echo ""
	@echo "🧪 Testing & Quality:"
	@echo "  test                  - Run all tests"
	@echo "  test-coverage         - Run tests with coverage report"
	@echo "  bench                 - Run performance benchmarks"
	@echo "  lint                  - Run Go linter (golangci-lint)"
	@echo "  fmt                   - Format code (gofmt + goimports)"
	@echo "  check                 - Run all checks (format + lint + test)"
	@echo ""
	@echo "🔧 Code Generation:"
	@echo "  proto                 - Generate code from .proto files"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  docker                - Build Docker image"
	@echo "  docker-run            - Run Docker container"
	@echo ""
	@echo "🛠️  Utilities:"
	@echo "  clean                 - Remove build artifacts"
	@echo "  tidy                  - Run go mod tidy"
	@echo "  deps                  - Download and verify dependencies"
	@echo "  setup                 - Complete development setup (deps + proto)"
	@echo "  help                  - Show this help message"
	@echo ""
	@echo "☁️  Production Deployment:"
	@echo "  deploy-prod           - Deploy to Google Cloud Run (one command)"
	@echo "  prod-help             - Show production deployment help"
	@echo ""
	@echo "==================================================================="
	@echo "💡 Tips:"
	@echo "  - First time? Just run: make dev"
	@echo "  - Local DB at: ./data/development.db"
	@echo "  - Config at: .env.development"
	@echo "  - Logs: Set LOG_LEVEL=debug in .env.development"
	@echo "  - Production? Run: make prod-help"
	@echo "==================================================================="

# -------------------------------------------------------------------
# Production Deployment Commands
# -------------------------------------------------------------------

# Check production readiness
.PHONY: prod-check
prod-check:
	@echo "Checking production readiness..."
	@./scripts/prod-readiness-check.sh

# Setup GCP secrets
.PHONY: setup-gcp-secrets
setup-gcp-secrets:
	@echo "Setting up GCP secrets..."
	@./scripts/setup-secrets.sh

# Deploy to Cloud Run - Production
.PHONY: deploy-prod
deploy-prod: prod-check
	@echo "Deploying to Google Cloud Run (Production)..."
	@./scripts/deploy-cloudrun.sh production

# Deploy to Cloud Run - Staging
.PHONY: deploy-staging
deploy-staging:
	@echo "Deploying to Google Cloud Run (Staging)..."
	@./scripts/deploy-cloudrun.sh staging

# Deploy to Cloud Run - Preview (for testing branches)
.PHONY: deploy-preview
deploy-preview:
	@if [ -z "$(BRANCH)" ]; then \
		echo "Error: BRANCH is required"; \
		echo "Usage: make deploy-preview BRANCH=feature-name"; \
		exit 1; \
	fi
	@echo "Deploying to Google Cloud Run (Preview: $(BRANCH))..."
	@./scripts/deploy-cloudrun.sh preview $(BRANCH)

# Full deployment pipeline (readiness check + secrets + deploy)
.PHONY: deploy-full
deploy-full: prod-check setup-gcp-secrets deploy-prod

# Verify production deployment
.PHONY: verify-prod
verify-prod:
	@if [ -z "$(SERVICE_URL)" ]; then \
		echo "Error: SERVICE_URL is required"; \
		echo "Usage: make verify-prod SERVICE_URL=https://your-service-url.run.app"; \
		exit 1; \
	fi
	@echo "Verifying production deployment..."
	@./scripts/verify-deployment.sh $(SERVICE_URL)

# Check GCP costs and usage
.PHONY: check-costs
check-costs:
	@echo "Checking GCP costs and usage..."
	@./scripts/check-costs.sh

# View production logs
.PHONY: prod-logs
prod-logs:
	@if [ -z "$(GCP_PROJECT_ID)" ]; then \
		echo "Error: GCP_PROJECT_ID is not set"; \
		echo "Usage: export GCP_PROJECT_ID=your-project-id && make prod-logs"; \
		exit 1; \
	fi
	@echo "Tailing production logs..."
	@gcloud run services logs tail sql-studio-backend --project=$(GCP_PROJECT_ID)

# Check production service status
.PHONY: prod-status
prod-status:
	@if [ -z "$(GCP_PROJECT_ID)" ]; then \
		echo "Error: GCP_PROJECT_ID is not set"; \
		echo "Usage: export GCP_PROJECT_ID=your-project-id && make prod-status"; \
		exit 1; \
	fi
	@echo "Checking production service status..."
	@gcloud run services describe sql-studio-backend \
		--project=$(GCP_PROJECT_ID) \
		--region=us-central1 \
		--format="table(status.url,status.conditions.type,status.conditions.status,metadata.labels)"

# List all Cloud Run revisions
.PHONY: prod-revisions
prod-revisions:
	@if [ -z "$(GCP_PROJECT_ID)" ]; then \
		echo "Error: GCP_PROJECT_ID is not set"; \
		echo "Usage: export GCP_PROJECT_ID=your-project-id && make prod-revisions"; \
		exit 1; \
	fi
	@echo "Listing Cloud Run revisions..."
	@gcloud run revisions list \
		--service=sql-studio-backend \
		--project=$(GCP_PROJECT_ID) \
		--region=us-central1 \
		--format="table(metadata.name,status.conditions[0].status,metadata.creationTimestamp)"

# Rollback to previous revision
.PHONY: prod-rollback
prod-rollback:
	@if [ -z "$(GCP_PROJECT_ID)" ]; then \
		echo "Error: GCP_PROJECT_ID is not set"; \
		exit 1; \
	fi
	@if [ -z "$(REVISION)" ]; then \
		echo "Error: REVISION is required"; \
		echo "Usage: make prod-rollback REVISION=sql-studio-backend-00001-abc"; \
		echo ""; \
		echo "List revisions with: make prod-revisions"; \
		exit 1; \
	fi
	@echo "Rolling back to revision: $(REVISION)"
	@gcloud run services update-traffic sql-studio-backend \
		--to-revisions=$(REVISION)=100 \
		--project=$(GCP_PROJECT_ID) \
		--region=us-central1

# Production help
.PHONY: prod-help
prod-help:
	@echo "==================================================================="
	@echo "Howlerops Backend - Production Deployment"
	@echo "==================================================================="
	@echo ""
	@echo "🚀 Quick Deploy:"
	@echo "  make deploy-prod          - Full automated deployment (recommended)"
	@echo ""
	@echo "📋 Step-by-step Deployment:"
	@echo "  1. make prod-check        - Verify production readiness"
	@echo "  2. make setup-gcp-secrets - Setup GCP secrets (one-time)"
	@echo "  3. make deploy-prod       - Deploy to Cloud Run"
	@echo "  4. make verify-prod SERVICE_URL=https://... - Verify deployment"
	@echo ""
	@echo "🔍 Monitoring:"
	@echo "  make prod-status          - Check service status"
	@echo "  make prod-logs            - Tail production logs"
	@echo "  make check-costs          - Check GCP costs and usage"
	@echo "  make prod-revisions       - List all revisions"
	@echo ""
	@echo "🔄 Rollback:"
	@echo "  make prod-rollback REVISION=sql-studio-backend-00001-abc"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  - Full checklist: PRODUCTION_CHECKLIST.md"
	@echo "  - Deployment guide: DEPLOYMENT.md"
	@echo "  - Architecture: ARCHITECTURE.md"
	@echo ""
	@echo "💡 Environment Variables Required:"
	@echo "  export GCP_PROJECT_ID=your-project-id"
	@echo "  export TURSO_URL=libsql://your-db.turso.io"
	@echo "  export TURSO_AUTH_TOKEN=your-token"
	@echo "  export JWT_SECRET=\$$(openssl rand -base64 32)"
	@echo "  export RESEND_API_KEY=re_your-key"
	@echo ""
	@echo "==================================================================="