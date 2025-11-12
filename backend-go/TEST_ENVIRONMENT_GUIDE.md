# Test Environment Guide

This document explains how tests are organized and how to run them in different environments.

## Test Categories

### 1. Unit Tests (Always Run)
These tests don't require external services:
- Pure business logic tests
- Mock-based tests
- In-memory database tests

**Run with:** `go test -short ./...`

### 2. Integration Tests (Require External Services)
These tests require external services to be running:

#### Database Tests
- **MongoDB tests**: Require MongoDB on `localhost:27017`
- **ClickHouse tests**: Require ClickHouse on `localhost:9000`
- **PostgreSQL tests**: Require PostgreSQL (configured via env vars)
- **MySQL tests**: Require MySQL (configured via env vars)

#### API Integration Tests
- **Auth flow tests** (`test/integration/auth_test.go`): Require running server on `localhost:8500`

**Run with:** `go test ./...` (without `-short` flag)

## Environment Detection

Tests automatically detect if required services are available and skip gracefully if not:

### 1. Short Mode Detection
Use the `-short` flag to skip all integration tests:
```bash
go test -short ./...
```

### 2. Service Availability Detection
Tests automatically detect if services are available:
- **MongoDB**: Checks if port 27017 is reachable (2s timeout)
- **ClickHouse**: Checks if port 9000 is reachable (2s timeout)
- **Server**: Checks if configured server URL is reachable (2s timeout)

### 3. Environment Variables
Explicitly skip specific test categories:

```bash
# Skip MongoDB tests
SKIP_MONGODB_TESTS=1 go test ./...

# Skip ClickHouse tests
SKIP_CLICKHOUSE_TESTS=1 go test ./...

# Skip all external database tests
SKIP_MONGODB_TESTS=1 SKIP_CLICKHOUSE_TESTS=1 go test ./...
```

## CI/CD Configuration

### Current CI Setup (`.github/workflows/test-coverage.yml`)

The CI workflow provides:
- PostgreSQL service on `localhost:5432`
- MySQL service on `localhost:3306`

MongoDB and ClickHouse are NOT provided, so tests requiring them are automatically skipped.

### Running Different Test Modes in CI

**Unit tests only (fast):**
```yaml
- name: Run unit tests
  run: go test -short -v ./...
```

**Integration tests with provided services:**
```yaml
- name: Run integration tests
  run: go test -v ./...
  env:
    POSTGRES_HOST: localhost
    MYSQL_HOST: localhost
```

### Adding More Services to CI

To enable MongoDB tests in CI, add this service to your workflow:

```yaml
services:
  mongodb:
    image: mongo:7
    ports:
      - 27017:27017
    options: >-
      --health-cmd "mongosh --eval 'db.runCommand({ping:1})'"
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
```

To enable ClickHouse tests in CI, add this service:

```yaml
services:
  clickhouse:
    image: clickhouse/clickhouse-server:latest
    ports:
      - 9000:9000
      - 8123:8123
    options: >-
      --health-cmd "clickhouse-client --query 'SELECT 1'"
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
```

## Local Development

### Setup for Full Test Coverage

1. **Install required services** (macOS with Homebrew):
```bash
# MongoDB
brew install mongodb-community
brew services start mongodb-community

# ClickHouse
brew install clickhouse
brew services start clickhouse

# PostgreSQL
brew install postgresql
brew services start postgresql

# MySQL
brew install mysql
brew services start mysql
```

2. **Run all tests**:
```bash
go test -v ./...
```

### Running Specific Test Suites

```bash
# Only MongoDB tests
go test -v ./pkg/database -run MongoDB

# Only ClickHouse tests
go test -v ./pkg/database -run ClickHouse

# Only integration tests
go test -v ./test/integration/...
```

## Test Skip Messages

When tests are skipped, you'll see clear messages:

```
--- SKIP: TestMongoDBDatabase_Connect (0.00s)
    test_helpers_test.go:71: Skipping MongoDB test: MongoDB not available on localhost:27017 (connection timeout)

--- SKIP: TestAuthFlow (0.01s)
    auth_test.go:59: Skipping integration test: server not available at http://localhost:8500 (connection refused or timeout)
```

These messages help identify which services need to be started for full test coverage.

## Troubleshooting

### Tests timeout waiting for MongoDB/ClickHouse

**Problem**: Tests hang for 30+ seconds before timing out.

**Solution**: The new code detects unavailable services in 2 seconds and skips immediately.

### Integration tests fail with connection refused

**Problem**: `test/integration/auth_test.go` fails with "dial tcp [::1]:8500: connect: connection refused"

**Solution**:
- Start the server: `go run cmd/server/main.go`
- Or skip integration tests: `go test -short ./...`
- Or set custom server URL: `TEST_BASE_URL=http://localhost:8080 go test ./test/integration/...`

### CI tests timeout

**Problem**: CI takes too long (5+ minutes) due to database connection timeouts.

**Solution**: Always use `-short` flag in CI for unit tests:
```yaml
- name: Run unit tests
  run: go test -short -v ./...
```

## Best Practices

1. **Always use `-short` for quick feedback**: `go test -short ./...`
2. **Run full test suite before committing**: `go test ./...`
3. **Add service requirement comments** to test files that need external services
4. **Use descriptive skip messages** that explain what service is needed
5. **Keep connection timeouts short** (2 seconds) to fail fast

## Reference: Test Helper Functions

Located in `pkg/database/test_helpers_test.go`:

- `requireMongoDB(t)` - Skip test if MongoDB not available
- `requireClickHouse(t)` - Skip test if ClickHouse not available
- `isServiceAvailable(address, timeout)` - Check if service is reachable
- `shouldSkipMongoDBTests()` - Check if MongoDB tests should be skipped
- `shouldSkipClickHouseTests()` - Check if ClickHouse tests should be skipped

For integration tests (`test/integration/auth_test.go`):

- `requireServer(t, baseURL)` - Skip test if server not available
- `isServerAvailable(baseURL)` - Check if test server is reachable
