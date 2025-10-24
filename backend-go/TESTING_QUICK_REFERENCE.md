# Testing Quick Reference

## Quick Start

```bash
# Run all tests (interactive)
./scripts/run-all-tests.sh

# Fast feedback (unit tests only)
go test ./... -v -short

# Pre-push check
make test-ci

# Deploy verification
./scripts/smoke-tests.sh
```

## Common Commands

### Unit Tests
```bash
go test ./...                          # Run all unit tests
go test ./... -v                       # Verbose output
go test ./... -short                   # Skip slow tests
go test ./internal/auth/... -v         # Test specific package
go test ./... -run TestLogin           # Run specific test
go test ./... -v -race                 # Race detection
go test ./... -coverprofile=coverage.out  # With coverage
```

### Integration Tests
```bash
# Prerequisites: Start server first
go run cmd/server/main.go

# Then in another terminal:
go test ./test/integration/... -v              # All integration tests
go test ./test/integration/... -v -run TestAuth   # Auth tests only
go test ./test/integration/... -v -run TestSync   # Sync tests only
go test ./test/integration/... -v -run TestHealth # Health tests only
```

### Bash Scripts
```bash
./scripts/test-api.sh        # API testing with color output
./scripts/smoke-tests.sh     # Fast deployment verification
./scripts/load-test.sh       # Performance testing
./scripts/run-all-tests.sh   # Complete test suite
```

### Remote Testing
```bash
# Against staging
TEST_BASE_URL=https://api.staging.sqlstudio.io ./scripts/smoke-tests.sh

# Against production (read-only smoke tests)
TEST_BASE_URL=https://api.sqlstudio.io ./scripts/smoke-tests.sh
```

## Test File Locations

```
backend-go/
├── test/
│   └── integration/          # Integration tests
│       ├── auth_test.go      # Auth endpoint tests
│       ├── sync_test.go      # Sync endpoint tests
│       └── health_test.go    # Health/metrics tests
├── scripts/
│   ├── test-api.sh           # API test script
│   ├── smoke-tests.sh        # Smoke test suite
│   ├── load-test.sh          # Load testing
│   └── run-all-tests.sh      # Master test runner
└── internal/*/               # Unit tests (*_test.go)
```

## Environment Variables

```bash
# Test target (default: http://localhost:8500)
export TEST_BASE_URL=https://api.example.com

# Metrics endpoint (default: http://localhost:9100/metrics)
export METRICS_URL=https://metrics.example.com

# Load test parameters
export LOAD_TEST_DURATION=60      # Duration in seconds
export LOAD_TEST_CONCURRENT=10    # Concurrent requests
export LOAD_TEST_RATE=100         # Requests per second
```

## Test Output Examples

### Passing Test
```
✓ PASS Health endpoint returns 200
✓ PASS Health status is 'healthy'
```

### Failing Test
```
✗ FAIL Login with invalid password rejected
  Expected: 401, Got: 200
  Response: {"token": "..."}
```

## Coverage Commands

```bash
# Generate coverage
go test ./... -coverprofile=coverage.out

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total
```

## Debug Failed Tests

```bash
# Run single test with verbose output
go test ./internal/auth/... -v -run TestLogin

# Show full stack traces
go test ./... -v -failfast

# Disable test caching
go test ./... -count=1

# Run with race detector
go test ./... -race

# Increase timeout
go test ./... -timeout 30m
```

## CI/CD Integration

```bash
# GitHub Actions (see .github/workflows/backend-tests.yml)
- Unit tests run on every push
- Integration tests run with PostgreSQL service
- Smoke tests verify deployment
- Coverage uploaded to Codecov

# Local CI simulation
make test-ci
```

## Performance Benchmarks

### Expected Response Times
| Endpoint | Good | Warning | Critical |
|----------|------|---------|----------|
| /health | < 50ms | > 100ms | > 500ms |
| /api/auth/login | < 200ms | > 500ms | > 1s |
| /api/sync/upload | < 1s | > 2s | > 5s |

### Load Test Baselines
- Health: 1000+ req/s with < 1% errors
- Auth: 100+ req/s with < 5% errors
- Sync: 50+ req/s with < 5% errors

## Common Issues

### Server not running
```bash
Error: Cannot reach server at http://localhost:8500

Solution:
go run cmd/server/main.go &
```

### Tests timeout
```bash
Error: Test timed out

Solution:
go test ./... -timeout 10m
```

### Race condition detected
```bash
Error: DATA RACE detected

Solution:
go test ./... -race  # Investigate the output
```

### Coverage too low
```bash
Warning: Coverage below 80%

Solution:
# Add more unit tests for uncovered code
# Use coverage report to find gaps:
go tool cover -html=coverage.out
```

## Test Data

### Test User Credentials
```go
// Integration tests create users like:
email: "test{timestamp}@example.com"
username: "testuser{timestamp}"
password: "TestPassword123!"
```

### Cleanup Test Data
```bash
# Manual cleanup (if needed)
psql -c "DELETE FROM users WHERE email LIKE 'test%@example.com';"
```

## Useful Tools

```bash
# Install test dependencies
brew install jq        # JSON processing
brew install hey       # Load testing
brew install bc        # Calculations

# Install Go tools
go install gotest.tools/gotestsum@latest  # Better test output
```

## Makefile Shortcuts

```bash
make test              # Unit + Integration
make test-unit         # Unit tests only
make test-integration  # Integration tests only
make test-smoke        # Smoke tests
make test-coverage     # With coverage report
make test-race         # With race detection
make test-all          # Everything
make test-ci           # CI/CD tests
make test-quick        # Fast tests only
```

## When to Run Which Tests

### Before Commit
```bash
make test-quick  # Fast unit tests
```

### Before Push
```bash
make test-ci  # Unit + smoke tests
```

### Before Deployment
```bash
make test-all  # Everything including load tests
```

### After Deployment
```bash
TEST_BASE_URL=<production-url> ./scripts/smoke-tests.sh
```

### During Development
```bash
# Watch mode (if entr installed)
make test-watch

# Or manually:
go test ./internal/mypackage/... -v -run TestMyFeature
```

## Getting Help

```bash
# Test options
go test -help

# Makefile targets
make help

# Script usage
./scripts/test-api.sh --help        # (if implemented)
./scripts/smoke-tests.sh --help     # (if implemented)
```

## Resources

- Full documentation: [TESTING.md](TESTING.md)
- Integration tests: [test/integration/README.md](test/integration/README.md)
- GitHub Actions: [.github/workflows/backend-tests.yml](../.github/workflows/backend-tests.yml)
