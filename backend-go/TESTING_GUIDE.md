# SQL Studio Testing Guide

Complete guide for running tests in the SQL Studio backend and frontend.

## Table of Contents

- [Overview](#overview)
- [Backend Testing](#backend-testing)
- [Frontend Testing](#frontend-testing)
- [E2E Testing](#e2e-testing)
- [Coverage Reports](#coverage-reports)
- [CI/CD Integration](#cicd-integration)
- [Troubleshooting](#troubleshooting)

## Overview

SQL Studio uses a comprehensive testing strategy covering multiple layers:

- **Unit Tests**: Test individual components and functions
- **Integration Tests**: Test service layer with real database
- **HTTP Handler Tests**: Test API endpoints with mocked services
- **E2E Tests**: Test complete user flows with Playwright

### Test Coverage Goals

- Backend integration tests: >90% flow coverage
- HTTP handler tests: >85% coverage
- E2E tests: All critical user journeys
- Email service: >80% coverage

## Backend Testing

### Prerequisites

```bash
cd backend-go
go mod download
```

### Running All Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run tests in parallel
go test -p 4 ./...
```

### Running Specific Test Suites

```bash
# Organization integration tests
go test -v ./internal/organization -run TestIntegrationTestSuite

# Organization unit tests
go test -v ./internal/organization -run TestService

# Email service tests
go test -v ./internal/email

# Run a specific test
go test -v ./internal/organization -run TestFlow_CreateOrganization_UserBecomesOwner
```

### Running Tests with Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Open coverage report in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Test Organization Structure

```
backend-go/
├── internal/
│   ├── organization/
│   │   ├── integration_test.go      # Full API integration tests
│   │   ├── handler_integration_test.go  # HTTP handler tests
│   │   ├── service_test.go          # Unit tests
│   │   └── testutil/                # Test helpers
│   │       ├── fixtures.go          # Test data builders
│   │       ├── db.go                # Database helpers
│   │       ├── auth.go              # Auth helpers
│   │       ├── assert.go            # Custom assertions
│   │       └── repository.go        # SQLite test repository
│   └── email/
│       └── email_test.go            # Email service tests
└── test/
    └── integration/                 # Cross-service integration tests
```

### Test Database

Integration tests use an in-memory SQLite database that is:
- Created fresh for each test
- Automatically cleaned up after tests
- Fast and deterministic
- No external dependencies required

### Example: Running Organization Tests

```bash
# Run all organization tests
go test -v ./internal/organization

# Run integration tests only
go test -v ./internal/organization -run Integration

# Run with coverage
go test -v ./internal/organization -coverprofile=org_coverage.out

# Run specific flow test
go test -v ./internal/organization -run TestFlow_InviteMember_InvitationCreated
```

## Frontend Testing

### Prerequisites

```bash
cd frontend
npm install
```

### Running Unit Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with UI
npm run test:ui

# Run specific test file
npm test -- organization.test.ts
```

### Running Component Tests

```bash
# Run component tests only
npm run test:component

# Run integration tests only
npm run test:integration
```

### Frontend Test Coverage

```bash
# Generate coverage report
npm run test:coverage

# View coverage in browser
open coverage/index.html
```

## E2E Testing

### Prerequisites

```bash
cd frontend

# Install Playwright browsers (one-time setup)
npm run playwright:install
```

### Running E2E Tests

```bash
# Run all E2E tests (headless)
npm run test:e2e

# Run E2E tests with UI
npm run test:e2e:ui

# Run specific test file
npx playwright test e2e/organization.spec.ts

# Run tests in debug mode
npx playwright test --debug

# Run tests in headed mode (see browser)
npx playwright test --headed
```

### E2E Test Configuration

E2E tests run against the local development environment. Before running:

1. **Start Backend Server**:
   ```bash
   cd backend-go
   go run cmd/server/main.go
   ```

2. **Start Frontend Dev Server**:
   ```bash
   cd frontend
   npm run dev
   ```

3. **Run E2E Tests**:
   ```bash
   npm run test:e2e
   ```

### E2E Test Reports

```bash
# Generate HTML report
npx playwright test --reporter=html

# Open report
npx playwright show-report
```

### E2E Test Browsers

Tests run on multiple browsers:
- Chromium
- Firefox
- WebKit (Safari)

Configure in `playwright.config.ts`.

## Coverage Reports

### Backend Coverage

```bash
cd backend-go

# Generate coverage for all packages
go test ./... -coverprofile=coverage.out

# View coverage summary
go tool cover -func=coverage.out | grep total

# Generate detailed HTML report
go tool cover -html=coverage.out -o coverage.html

# Check coverage for specific package
go test ./internal/organization -coverprofile=org_coverage.out
go tool cover -func=org_coverage.out
```

### Coverage Goals

```bash
# Check if coverage meets threshold (example)
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
THRESHOLD=85
if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "Coverage $COVERAGE% is below threshold $THRESHOLD%"
    exit 1
fi
```

### Frontend Coverage

```bash
cd frontend

# Generate coverage report
npm run test:coverage

# View in browser
open coverage/index.html

# Check coverage thresholds (configured in vitest.config.ts)
npm run test:coverage -- --reporter=json --outputFile=coverage/coverage.json
```

## CI/CD Integration

### GitHub Actions Workflow

Create `.github/workflows/test.yml`:

```yaml
name: Tests

on: [push, pull_request]

jobs:
  backend-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run tests
        working-directory: backend-go
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./backend-go/coverage.out
          flags: backend

  frontend-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Node
        uses: actions/setup-node@v3
        with:
          node-version: '20'

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Run tests
        working-directory: frontend
        run: npm run test:coverage

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./frontend/coverage/coverage-final.json
          flags: frontend

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Set up Node
        uses: actions/setup-node@v3
        with:
          node-version: '20'

      - name: Install dependencies
        working-directory: frontend
        run: |
          npm ci
          npx playwright install --with-deps

      - name: Start backend
        working-directory: backend-go
        run: |
          go build -o server cmd/server/main.go
          ./server &
          sleep 5

      - name: Start frontend
        working-directory: frontend
        run: |
          npm run build
          npm run preview &
          sleep 5

      - name: Run E2E tests
        working-directory: frontend
        run: npm run test:e2e

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: frontend/playwright-report/
```

### Running Tests Locally (Pre-commit)

Create a pre-commit hook:

```bash
#!/bin/sh
# .git/hooks/pre-commit

echo "Running backend tests..."
cd backend-go && go test ./... || exit 1

echo "Running frontend tests..."
cd ../frontend && npm test || exit 1

echo "All tests passed!"
```

## Troubleshooting

### Common Issues

#### 1. Test Database Errors

**Problem**: `database is locked` or `constraint failed`

**Solution**:
```bash
# Ensure tests are not running in parallel for same package
go test -p 1 ./internal/organization
```

#### 2. E2E Tests Timing Out

**Problem**: Playwright tests timeout

**Solution**:
```typescript
// Increase timeout in playwright.config.ts
export default defineConfig({
  timeout: 60000, // 60 seconds
  expect: {
    timeout: 10000 // 10 seconds for assertions
  }
});
```

#### 3. Import Errors in Tests

**Problem**: `cannot find package`

**Solution**:
```bash
# Backend
cd backend-go && go mod tidy

# Frontend
cd frontend && npm install
```

#### 4. Coverage Report Empty

**Problem**: Coverage file is empty

**Solution**:
```bash
# Ensure tests actually ran
go test -v ./... -coverprofile=coverage.out

# Check for build errors
go build ./...
```

#### 5. E2E Tests Can't Connect to Server

**Problem**: `ECONNREFUSED` errors

**Solution**:
```bash
# Verify backend is running
curl http://localhost:8080/health

# Verify frontend is running
curl http://localhost:3000

# Check ports in use
lsof -i :8080
lsof -i :3000
```

### Test Performance

#### Slow Tests

```bash
# Find slow tests
go test -v ./... | grep -E '\s+[0-9]+\.[0-9]+s'

# Run specific package tests
go test -v ./internal/organization -run TestSpecificTest
```

#### Parallel Execution

```bash
# Run packages in parallel (default is GOMAXPROCS)
go test -p 4 ./...

# Run tests within package in parallel
# Use t.Parallel() in test functions
```

### Debugging Tests

#### Backend

```bash
# Run single test with verbose output
go test -v ./internal/organization -run TestFlow_CreateOrganization

# Use debugger (delve)
dlv test ./internal/organization -- -test.run TestFlow_CreateOrganization
```

#### Frontend

```bash
# Debug in VS Code
# Add breakpoints and run "Debug Test" in test file

# Use Vitest UI for interactive debugging
npm run test:ui
```

#### E2E

```bash
# Run in debug mode (opens browser inspector)
npx playwright test --debug

# Take screenshots on failure
npx playwright test --screenshot=only-on-failure

# Record video
npx playwright test --video=on
```

## Test Best Practices

### 1. Test Naming

```go
// Good: Descriptive and follows pattern
func TestFlow_CreateOrganization_UserBecomesOwner(t *testing.T) {}

// Bad: Unclear what is being tested
func TestOrg1(t *testing.T) {}
```

### 2. Test Independence

```go
// Good: Each test is independent
func TestCreateOrganization(t *testing.T) {
    testDB := testutil.SetupTestDB(t)
    defer testDB.Close()
    // ... test logic
}

// Bad: Tests depend on each other
var globalOrg *Organization // Don't do this!
```

### 3. Use Test Helpers

```go
// Good: Use helper functions
func (suite *IntegrationTestSuite) createTestOrganization(ownerID, name string) *organization.Organization {
    // ... reusable test setup
}

// Bad: Repeat setup in every test
```

### 4. Clear Assertions

```go
// Good: Clear assertion with message
assert.Equal(t, "Acme Corp", org.Name, "Organization name should match input")

// Acceptable: testutil custom assertions
testutil.AssertOrganizationNotNil(t, org)
```

### 5. Test Data

```go
// Good: Use builders for complex test data
org := testutil.NewOrganizationBuilder("Acme", "user1").
    WithMaxMembers(20).
    WithDescription("Test org").
    Build()

// Good: Use fixtures for common scenarios
invitation := testutil.CreateTestInvitation(orgID, email, inviter, role)
```

## Quick Reference

### Backend Commands

```bash
# Run all tests
go test ./...

# Coverage
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Specific package
go test -v ./internal/organization

# With race detection
go test -race ./...
```

### Frontend Commands

```bash
# Unit tests
npm test

# E2E tests
npm run test:e2e

# Coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

### Common Test Patterns

```go
// Table-driven test
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid name", "Acme Corp", false},
        {"too short", "AB", true},
        {"empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Additional Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Playwright Documentation](https://playwright.dev)
- [Vitest Documentation](https://vitest.dev)

## Support

For test-related issues:
1. Check this guide's troubleshooting section
2. Review test logs for specific errors
3. Check CI/CD pipeline logs
4. Create an issue with test failure details
