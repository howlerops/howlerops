# Howlerops Backend - Quick Reference

## Build & Run

### Development
```bash
# Setup (first time only)
make setup-local

# Run with hot reload
make dev

# Build
go build -o ./bin/server ./cmd/server/main.go

# Run
ENVIRONMENT=development ./bin/server
```

### Production
```bash
# Build
go build -o ./bin/server ./cmd/server/main.go

# Run
ENVIRONMENT=production ./bin/server
```

## Testing

```bash
# Run local integration test
./scripts/test-local.sh

# Run unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/auth/...
```

## Environment Variables

### Required (Production)
```bash
ENVIRONMENT=production
TURSO_URL=libsql://your-db.turso.io
TURSO_AUTH_TOKEN=eyJhbGc...
JWT_SECRET=<strong-random-secret>
```

### Optional
```bash
RESEND_API_KEY=re_...              # Email service (uses mock if not set)
PORT=8080                           # HTTP port
GRPC_PORT=50051                     # gRPC port
METRICS_PORT=9090                   # Metrics port
LOG_LEVEL=info                      # debug|info|warn|error
```

## API Endpoints

### Health
```bash
GET /health
```

### Sync (Auth Required)
```bash
POST /api/sync/upload
GET  /api/sync/download?device_id=<id>&since=<timestamp>
GET  /api/sync/conflicts
POST /api/sync/conflicts/:id/resolve
```

### Auth
```bash
POST /api/auth/register
POST /api/auth/login
POST /api/auth/refresh
POST /api/auth/logout
GET  /api/auth/verify-email
POST /api/auth/reset-password
```

### Metrics
```bash
GET :9090/metrics    # Prometheus format
```

## Common Tasks

### Create Migration
```bash
# Add SQL to pkg/storage/turso/client.go getEmbeddedSchema()
# Or create new file in migrations/ directory
```

### Add New Endpoint
```bash
# 1. Define in proto (if using gRPC)
# 2. Implement handler in internal/server/
# 3. Register route in http.go or grpc.go
# 4. Update tests
```

### Update Configuration
```bash
# 1. Edit configs/config.yaml
# 2. Update internal/config/config.go
# 3. Update .env.example
```

## Database

### Local (Development)
```bash
# Location
./data/development.db

# Inspect
sqlite3 ./data/development.db

# Schema
sqlite3 ./data/development.db ".schema"

# Query
sqlite3 ./data/development.db "SELECT * FROM users;"

# Reset
rm ./data/development.db
```

### Turso (Production)
```bash
# Connect
turso db shell your-database

# Create database
turso db create your-database

# Get connection URL
turso db show your-database

# Get auth token
turso db tokens create your-database
```

## Logs

### Development
```bash
# Stdout (pretty formatted)
# No configuration needed
```

### Production
```bash
# JSON format to stdout
# Captured by platform (Fly.io, Cloud Run, etc.)

# Fly.io
fly logs

# Cloud Run
gcloud run logs read backend --limit 100

# Docker
docker logs <container-id>
```

## Deployment

### Fly.io
```bash
# Setup
fly launch

# Set secrets
fly secrets set TURSO_URL=... TURSO_AUTH_TOKEN=... JWT_SECRET=...

# Deploy
fly deploy

# Logs
fly logs

# Status
fly status

# Scale
fly scale count 2
fly scale vm shared-cpu-1x
```

### Google Cloud Run
```bash
# Build
gcloud builds submit --tag gcr.io/PROJECT/backend

# Deploy
gcloud run deploy backend \
  --image gcr.io/PROJECT/backend \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars ENVIRONMENT=production \
  --set-secrets TURSO_URL=turso_url:latest,TURSO_AUTH_TOKEN=turso_token:latest

# Logs
gcloud run logs read backend --limit 100

# Traffic
gcloud run services update-traffic backend --to-latest
```

## Troubleshooting

### Server won't start

**Check logs:**
```bash
# Look for specific error
# Common issues:
# - TURSO_URL not set
# - TURSO_AUTH_TOKEN invalid
# - Port already in use
# - Database file permissions
```

**Check environment:**
```bash
echo $ENVIRONMENT
echo $TURSO_URL
echo $JWT_SECRET
```

**Check ports:**
```bash
lsof -i :8080    # HTTP
lsof -i :50051   # gRPC
lsof -i :9090    # Metrics
```

### Database errors

**Connection failed:**
```bash
# Local: Check file exists and has write permissions
ls -la ./data/development.db

# Turso: Check URL and token
turso db show your-database
turso db tokens list your-database
```

**Schema errors:**
```bash
# Check schema version
sqlite3 ./data/development.db "SELECT * FROM sqlite_master WHERE type='table';"

# Reset local database
rm ./data/development.db
ENVIRONMENT=development go run cmd/server/main.go
```

### Build errors

**Missing dependencies:**
```bash
go mod tidy
go mod download
```

**Import conflicts:**
```bash
# Use aliases for conflicting names
import appsync "github.com/sql-studio/backend-go/internal/sync"
```

**Type errors:**
```bash
# Check pkg/storage/types.go
# Ensure all required fields exist
```

## Monitoring

### Metrics
```bash
# Access Prometheus endpoint
curl http://localhost:9090/metrics

# Key metrics:
# - go_goroutines
# - go_memstats_*
# - http_requests_total
# - Custom application metrics
```

### Health Checks
```bash
# Basic health
curl http://localhost:8080/health

# With details (if implemented)
curl http://localhost:8080/health/detailed
```

### Database Health
```bash
# Check connection count
sqlite3 ./data/development.db "PRAGMA database_list;"

# Check table sizes
sqlite3 ./data/development.db "SELECT name, COUNT(*) FROM sqlite_master JOIN pragma_table_info(name) GROUP BY name;"
```

## Security

### Generate Strong Secrets
```bash
# JWT Secret
openssl rand -base64 32

# API Key
openssl rand -hex 32

# UUID
uuidgen
```

### Check Security
```bash
# Check for exposed secrets
git secrets --scan

# Check dependencies
go list -json -m all | nancy sleuth

# Static analysis
gosec ./...
staticcheck ./...
```

## Performance

### Profiling
```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=.

# Memory profile
go test -memprofile=mem.prof -bench=.

# Analyze
go tool pprof cpu.prof
go tool pprof mem.prof
```

### Benchmarking
```bash
# Run benchmarks
go test -bench=. ./...

# With memory stats
go test -bench=. -benchmem ./...

# Specific benchmark
go test -bench=BenchmarkAuth ./internal/auth/...
```

## Development Workflow

### 1. Make Changes
```bash
# Edit code
vim internal/auth/service.go
```

### 2. Test Locally
```bash
# Run tests
go test ./internal/auth/...

# Run server
make dev
```

### 3. Build
```bash
# Build
go build -o ./bin/server ./cmd/server/main.go

# Verify
./bin/server --version
```

### 4. Commit
```bash
git add .
git commit -m "feat: add new auth feature"
git push
```

### 5. Deploy
```bash
# Staging
fly deploy --config fly.staging.toml

# Production
fly deploy --config fly.toml
```

## Useful Commands

### Database
```bash
# Backup
cp ./data/development.db ./data/backup-$(date +%Y%m%d).db

# Restore
cp ./data/backup-20240123.db ./data/development.db

# Compact
sqlite3 ./data/development.db "VACUUM;"
```

### Docker
```bash
# Build
docker build -t sql-studio-backend .

# Run
docker run -p 8080:8080 -e ENVIRONMENT=production sql-studio-backend

# Shell
docker exec -it <container-id> /bin/sh

# Logs
docker logs -f <container-id>
```

### Go
```bash
# Format code
go fmt ./...
gofmt -s -w .

# Organize imports
goimports -w .

# Lint
golangci-lint run

# Vet
go vet ./...

# Tidy modules
go mod tidy

# Vendor
go mod vendor

# Update dependencies
go get -u ./...
```

## File Locations

### Configuration
```
configs/config.yaml              # Multi-environment config
.env.development                 # Development environment
.env.production.example          # Production template
```

### Source Code
```
cmd/server/main.go               # Main entry point
internal/auth/                   # Auth service
internal/sync/                   # Sync service
internal/email/                  # Email service
internal/server/                 # HTTP/gRPC servers
pkg/storage/turso/               # Turso storage
```

### Data
```
data/development.db              # Local SQLite database
logs/                           # Log files (if file output enabled)
tmp/                            # Temporary files
```

### Scripts
```
scripts/test-local.sh           # Local testing
scripts/deploy-fly.sh           # Fly.io deployment
scripts/deploy-cloudrun.sh      # Cloud Run deployment
scripts/setup-secrets.sh        # Secret management
```

## Support & Resources

### Documentation
- `README.md` - Project overview
- `ARCHITECTURE.md` - System architecture
- `API_DOCUMENTATION.md` - API reference
- `DEPLOYMENT.md` - Deployment guide
- `INTEGRATION_SUMMARY.md` - Integration details
- `TURSO_INTEGRATION_COMPLETE.md` - Turso setup

### External Docs
- [Turso Docs](https://docs.turso.tech/)
- [Go Documentation](https://golang.org/doc/)
- [gRPC Go](https://grpc.io/docs/languages/go/)
- [Fly.io Docs](https://fly.io/docs/)
- [Cloud Run Docs](https://cloud.google.com/run/docs)

### Commands Cheat Sheet
```bash
# Quick start
make setup-local && make dev

# Full build and test
go mod tidy && go build ./... && go test ./...

# Deploy to production
./scripts/deploy-fly.sh

# Check logs
fly logs --app sql-studio-backend

# Database shell
sqlite3 ./data/development.db

# Health check
curl http://localhost:8080/health
```
