# Integration Tests

This directory contains integration tests for the SQL Studio backend API.

## Running Tests

### Local Testing

Run all integration tests against local server:

```bash
# Start the server first
cd backend-go
go run cmd/server/main.go

# In another terminal, run tests
cd backend-go
go test ./test/integration/... -v
```

### Testing Against Remote Server

Set the `TEST_BASE_URL` environment variable:

```bash
export TEST_BASE_URL=https://api.sqlstudio.io
go test ./test/integration/... -v
```

### Running Specific Test Suites

```bash
# Run only auth tests
go test ./test/integration/... -v -run TestAuth

# Run only sync tests
go test ./test/integration/... -v -run TestSync

# Run only health tests
go test ./test/integration/... -v -run TestHealth
```

### Generate Test Report

```bash
# Run tests with coverage
go test ./test/integration/... -v -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out
```

### Running with Test Report Output

```bash
# Install gotestsum for better output
go install gotest.tools/gotestsum@latest

# Run tests with formatted output
gotestsum --format testname ./test/integration/...
```

## Test Suites

### Auth Tests (`auth_test.go`)
- User signup
- User login
- Token refresh
- Protected endpoints
- Logout
- Input validation
- Rate limiting

### Sync Tests (`sync_test.go`)
- Upload changes
- Download changes
- List conflicts
- Resolve conflicts
- Input validation
- Large payload handling
- Authentication requirements

### Health Tests (`health_test.go`)
- Health check endpoint
- Response time
- Reliability under load
- Metrics endpoint
- Readiness probe
- Liveness probe
- CORS headers
- Service discovery

## Environment Variables

- `TEST_BASE_URL`: Base URL of the API (default: `http://localhost:8500`)
- `METRICS_URL`: URL of the metrics endpoint (default: `http://localhost:9100/metrics`)

## Test Data

Tests use deterministic test data with timestamps to avoid collisions. Test data is not automatically cleaned up to allow for inspection after test runs.

## CI/CD Integration

These tests can be run in CI/CD pipelines:

```bash
# Example GitHub Actions
go test ./test/integration/... -v -timeout 5m
```

## Notes

- Tests assume the server is already running
- Tests create test users with unique identifiers
- Some tests may be skipped if running in short mode (`go test -short`)
- Load tests are skipped in short mode
