# Testing Documentation

## Overview

Howlerops backend has a comprehensive testing strategy that includes:

1. **Unit Tests** - Test individual components in isolation
2. **Integration Tests** - Test API endpoints end-to-end
3. **Smoke Tests** - Fast critical path testing for deployments
4. **Load Tests** - Performance and stress testing
5. **Manual Testing** - API testing scripts for exploratory testing

## Test Strategy

### Testing Pyramid

```
      /\
     /  \      E2E Tests (Smoke Tests)
    /    \     - Fast, critical paths only
   /------\
  /        \   Integration Tests
 /          \  - API endpoint testing
/------------\
              Unit Tests
              - Component isolation
              - Mock dependencies
```

### Coverage Goals

- **Unit Tests**: 80%+ code coverage
- **Integration Tests**: All API endpoints
- **Smoke Tests**: Critical user flows
- **Load Tests**: Performance benchmarks

## Running Tests

### Prerequisites

```bash
# Required
- Go 1.24+
- Running backend server (for integration/smoke/load tests)

# Optional (for better test experience)
- jq (JSON processing)
- hey, ab, or wrk (load testing)
- bc (calculations in scripts)
```

### Unit Tests

Unit tests are co-located with the source code (`*_test.go` files).

```bash
# Run all unit tests
go test ./... -v

# Run tests with coverage
go test ./... -v -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run tests in a specific package
go test ./internal/auth/... -v

# Run a specific test
go test ./internal/auth/... -v -run TestLogin

# Run with race detection
go test ./... -v -race
```

### Integration Tests

Integration tests are in `test/integration/`.

```bash
# Start the server first
go run cmd/server/main.go

# In another terminal, run integration tests
cd backend-go
go test ./test/integration/... -v

# Test against remote server
export TEST_BASE_URL=https://api.sqlstudio.io
go test ./test/integration/... -v

# Run specific test suite
go test ./test/integration/... -v -run TestAuth
go test ./test/integration/... -v -run TestSync
go test ./test/integration/... -v -run TestHealth

# Skip slow tests
go test ./test/integration/... -v -short
```

#### Integration Test Suites

1. **Auth Tests** (`auth_test.go`)
   - User signup and validation
   - Login with valid/invalid credentials
   - Token refresh
   - Protected endpoint access
   - Logout
   - Rate limiting

2. **Sync Tests** (`sync_test.go`)
   - Upload changes
   - Download changes
   - List conflicts
   - Resolve conflicts
   - Authentication requirements
   - Large payload handling

3. **Health Tests** (`health_test.go`)
   - Health check endpoint
   - Response time verification
   - Reliability under concurrent load
   - Metrics endpoint
   - CORS configuration
   - Service discovery

### Smoke Tests

Fast, critical path tests for deployment verification. Should complete in < 30 seconds.

```bash
# Run smoke tests locally
./scripts/smoke-tests.sh

# Run against production
TEST_BASE_URL=https://api.sqlstudio.io ./scripts/smoke-tests.sh

# Use in CI/CD
./scripts/smoke-tests.sh || exit 1
```

**What Smoke Tests Check:**
- Service is reachable
- Health check passes
- Database connectivity
- Response time is acceptable
- Basic auth flow works
- Protected endpoints require auth
- CORS is configured

### API Testing Script

Interactive script for manual API testing with color-coded output.

```bash
# Run locally
./scripts/test-api.sh

# Test production
TEST_BASE_URL=https://api.sqlstudio.io ./scripts/test-api.sh
```

**Tests Performed:**
- Health endpoint
- User signup (valid/invalid)
- User login (valid/invalid)
- Token refresh
- Protected endpoints with/without auth
- Sync upload/download
- Logout
- CORS headers

### Load Tests

Performance and stress testing to identify bottlenecks.

```bash
# Run with default settings (60s, 10 concurrent, 100 req/s)
./scripts/load-test.sh

# Customize test parameters
export LOAD_TEST_DURATION=120      # 2 minutes
export LOAD_TEST_CONCURRENT=20     # 20 concurrent requests
export LOAD_TEST_RATE=200          # 200 requests/second
./scripts/load-test.sh

# Test production
TEST_BASE_URL=https://api.sqlstudio.io ./scripts/load-test.sh
```

**Load Test Scenarios:**

1. **Health Endpoint Load** - High volume GET requests
2. **Auth Login Load** - POST requests with authentication
3. **Sustained Mixed Load** - Random endpoint testing over time

**Results:**
- Saved to `./load-test-results/load_test_TIMESTAMP.txt`
- Includes success rates, response times, requests/second
- Identifies performance bottlenecks

### Installing Load Testing Tools

```bash
# macOS
brew install hey
brew install apache-bench  # for ab
brew install wrk

# Linux (Ubuntu/Debian)
sudo apt-get install apache2-utils  # for ab
wget https://github.com/rakyll/hey/releases/download/v0.1.4/hey_linux_amd64
chmod +x hey_linux_amd64 && sudo mv hey_linux_amd64 /usr/local/bin/hey
```

## Test Data Management

### Deterministic Test Data

Tests use deterministic data to ensure repeatability:

```go
// Use timestamps for unique identifiers
timestamp := time.Now().Unix()
testEmail := fmt.Sprintf("test%d@example.com", timestamp)
testUsername := fmt.Sprintf("testuser%d", timestamp)
```

### Test Data Cleanup

Currently, test data is **not automatically cleaned up**. This allows for:
- Post-test inspection
- Debugging failures
- Audit trail

To clean up test data manually:

```sql
-- In your database
DELETE FROM users WHERE email LIKE 'test%@example.com';
DELETE FROM users WHERE email LIKE 'apitest%@example.com';
DELETE FROM users WHERE email LIKE 'smoke%@test.com';
```

### Avoiding Production Pollution

**Never run tests against production with test data creation enabled.**

For production testing:
1. Use read-only tests
2. Use dedicated test accounts
3. Use feature flags to prevent data creation
4. Run smoke tests only (minimal data creation)

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Backend Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run unit tests
        run: |
          cd backend-go
          go test ./... -v -coverprofile=coverage.out

      - name: Start server
        run: |
          cd backend-go
          go run cmd/server/main.go &
          sleep 10

      - name: Run smoke tests
        run: |
          cd backend-go
          ./scripts/smoke-tests.sh

      - name: Run integration tests
        run: |
          cd backend-go
          go test ./test/integration/... -v -timeout 5m

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./backend-go/coverage.out
```

### Deployment Verification

After deploying to a new environment:

```bash
# 1. Run smoke tests
TEST_BASE_URL=https://api.staging.sqlstudio.io ./scripts/smoke-tests.sh

# 2. If smoke tests pass, run full integration tests
TEST_BASE_URL=https://api.staging.sqlstudio.io go test ./test/integration/... -v

# 3. Optional: Run load tests to verify performance
TEST_BASE_URL=https://api.staging.sqlstudio.io ./scripts/load-test.sh
```

## Manual Testing Procedures

### Testing New Features

1. **Write Tests First** (TDD)
   ```go
   func TestNewFeature(t *testing.T) {
       // Arrange
       // Act
       // Assert
   }
   ```

2. **Run Unit Tests**
   ```bash
   go test ./internal/mypackage/... -v
   ```

3. **Test Integration**
   ```bash
   go test ./test/integration/... -v -run TestMyFeature
   ```

4. **Manual API Testing**
   ```bash
   # Test the endpoint manually
   curl -X POST http://localhost:8500/api/my-feature \
     -H "Content-Type: application/json" \
     -d '{"key": "value"}'
   ```

5. **Run Smoke Tests**
   ```bash
   ./scripts/smoke-tests.sh
   ```

### Testing Authentication

```bash
# 1. Signup
curl -X POST http://localhost:8500/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "username": "testuser",
    "password": "TestPassword123!"
  }'

# 2. Login
curl -X POST http://localhost:8500/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "TestPassword123!"
  }'

# 3. Extract token from response, then test protected endpoint
TOKEN="your-token-here"
curl -X GET http://localhost:8500/api/auth/profile \
  -H "Authorization: Bearer $TOKEN"

# 4. Refresh token
curl -X POST http://localhost:8500/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your-refresh-token"
  }'

# 5. Logout
curl -X POST http://localhost:8500/api/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

### Testing Sync Endpoints

```bash
# Get auth token first
TOKEN="your-token-here"

# 1. Upload changes
curl -X POST http://localhost:8500/api/sync/upload \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "test-device-123",
    "last_sync_at": "2024-01-01T00:00:00Z",
    "changes": [
      {
        "id": "change-1",
        "item_type": "saved_query",
        "item_id": "query-123",
        "action": "create",
        "data": {
          "id": "query-123",
          "name": "Test Query",
          "query": "SELECT * FROM users"
        },
        "updated_at": "2024-01-01T00:00:00Z",
        "sync_version": 1,
        "device_id": "test-device-123"
      }
    ]
  }'

# 2. Download changes
curl -X GET "http://localhost:8500/api/sync/download?device_id=test-device-123&since=2024-01-01T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN"

# 3. List conflicts
curl -X GET http://localhost:8500/api/sync/conflicts \
  -H "Authorization: Bearer $TOKEN"

# 4. Resolve conflict
curl -X POST http://localhost:8500/api/sync/conflicts/conflict-id/resolve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "strategy": "last_write_wins"
  }'
```

## Performance Benchmarks

### Expected Response Times

| Endpoint | Expected | Warning | Critical |
|----------|----------|---------|----------|
| GET /health | < 50ms | > 100ms | > 500ms |
| POST /api/auth/login | < 200ms | > 500ms | > 1000ms |
| POST /api/auth/signup | < 300ms | > 1000ms | > 2000ms |
| GET /api/sync/download | < 500ms | > 1000ms | > 2000ms |
| POST /api/sync/upload | < 1000ms | > 2000ms | > 5000ms |

### Load Test Baselines

- **Health Endpoint**: Should handle 1000+ req/s with < 1% error rate
- **Auth Endpoints**: Should handle 100+ req/s with < 5% error rate
- **Sync Endpoints**: Should handle 50+ req/s with < 5% error rate

## Troubleshooting

### Common Issues

#### Tests Fail with "Connection Refused"

**Problem**: Server is not running

**Solution**:
```bash
# Start the server
go run cmd/server/main.go

# Or check if already running
ps aux | grep server
```

#### Tests Fail with "Timeout"

**Problem**: Server is slow or database is not responding

**Solution**:
```bash
# Check server logs
tail -f logs/app.log

# Check database connectivity
# Verify DATABASE_URL or TURSO_URL is set correctly
```

#### Integration Tests Fail Randomly

**Problem**: Test data conflicts or race conditions

**Solution**:
- Use unique identifiers (timestamps)
- Run tests sequentially: `go test -p 1`
- Clean up test data between runs

#### Load Tests Show Poor Performance

**Problem**: Configuration or resource limits

**Solution**:
```bash
# Check server resources
top

# Increase connection pool size in config.yaml
database:
  max_connections: 50

# Check rate limiting settings
security:
  rate_limit_rps: 200
```

## Best Practices

1. **Always Test Locally First**
   - Run unit tests before committing
   - Run smoke tests before pushing

2. **Use Appropriate Test Level**
   - Unit tests for logic
   - Integration tests for API
   - Load tests for performance

3. **Keep Tests Fast**
   - Use `-short` flag for quick feedback
   - Parallelize when possible

4. **Clean, Readable Tests**
   - Use Arrange-Act-Assert pattern
   - Clear test names
   - Good error messages

5. **Mock External Dependencies**
   - Don't call external APIs in unit tests
   - Use test doubles for databases

6. **Test Edge Cases**
   - Empty inputs
   - Large payloads
   - Invalid tokens
   - Concurrent access

## Contributing

When adding new features:

1. Write tests first (TDD)
2. Ensure all tests pass
3. Update documentation
4. Run smoke tests
5. Check code coverage

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Assertions](https://github.com/stretchr/testify)
- [HTTP Load Testing Tools](https://github.com/rakyll/hey)
- [API Testing Best Practices](https://martinfowler.com/articles/practical-test-pyramid.html)
