# Workflow Issues Analysis

## Executive Summary

Analysis of GitHub Actions workflows revealed multiple issues preventing successful CI/CD pipeline execution:

- **CI Workflow**: golangci-lint version mismatch, build failures
- **Backend Tests**: Database test timeouts, integration test server issues, security scan format errors
- **Deploy to Cloud Run**: Test job running wrong tests, GCP authentication issues

## Detailed Analysis

### 1. CI Workflow (`ci.yml`)

#### Issue 1.1: golangci-lint Version Mismatch
**Status**: 🔴 CRITICAL
**Error**: `the Go language version (go1.23) used to build golangci-lint is lower than the targeted Go version (1.24.0)`

**Root Cause**:
- Project uses Go 1.24
- golangci-lint v1.61 was built with Go 1.23
- golangci-lint must be built with same or higher Go version

**Solution**:
- Upgrade to golangci-lint v1.62+ (built with Go 1.24)
- Alternative: Downgrade go.mod to Go 1.23 (NOT RECOMMENDED)

#### Issue 1.2: Invalid golangci-lint-action Parameters
**Status**: 🟡 WARNING
**Error**: `Unexpected input(s) 'skip-pkg-cache', 'skip-build-cache'`

**Root Cause**:
- golangci-lint-action@v6 doesn't support these parameters
- Parameters were valid in older versions

**Solution**:
- Remove invalid parameters:
  - `skip-pkg-cache`
  - `skip-build-cache`

#### Issue 1.3: Build Job ARM64 Timeout
**Status**: 🟡 WARNING
**Error**: Build times out after 600s for linux/arm64

**Root Cause**:
- Cross-compilation with CGO for ARM64 is slow
- May be hitting resource limits

**Solution**:
- Disable ARM64 builds temporarily OR
- Increase timeout OR
- Disable CGO for cross-platform builds

---

### 2. Backend Tests Workflow (`backend-tests.yml`)

#### Issue 2.1: Database Test Timeouts
**Status**: 🔴 CRITICAL
**Error**: `FAIL github.com/sql-studio/backend-go/pkg/database 600.059s`

**Root Cause**:
- MongoDB connection tests hang for 10 minutes
- Tests try to connect to MongoDB that doesn't exist in CI
- No timeout configured for individual tests

**Solution**:
- Skip MongoDB tests in CI (add build tags)
- Configure test timeout: `go test -timeout 5m`
- Use in-memory database for tests

#### Issue 2.2: Integration Tests Server Not Running
**Status**: 🔴 CRITICAL
**Error**: `dial tcp [::1]:8500: connect: connection refused`

**Root Cause**:
- Integration tests expect server on localhost:8500
- Server not started before tests run
- Environment variables not properly set

**Solution**:
- Start server in background before integration tests
- Wait for server to be ready (health check)
- Set required environment variables:
  - `TURSO_URL=file:./test.db`
  - `JWT_SECRET=test-secret`

#### Issue 2.3: Security Scan SARIF Format Error
**Status**: 🟡 WARNING
**Error**: `instance.runs[0].results[X].fixes[0].artifactChanges is not of a type(s) array`

**Root Cause**:
- gosec generates invalid SARIF format
- GitHub's SARIF validator rejects it

**Solution**:
- Set `continue-on-error: true` for security upload
- Use newer version of gosec
- Skip SARIF upload, just log results

#### Issue 2.4: Lint Job Same golangci-lint Issue
**Status**: 🔴 CRITICAL
**Solution**: Same as CI workflow Issue 1.1

---

### 3. Deploy to Cloud Run Workflow (`deploy-cloud-run.yml`)

#### Issue 3.1: Test Job Runs Wrong Tests
**Status**: 🟡 WARNING
**Current**: `go test -race -coverprofile=coverage.out -covermode=atomic ./...`
**Problem**: Runs ALL tests including integration tests (which fail)

**Solution**:
- Only run unit tests: `go test ./internal/... ./pkg/... ./cmd/...`
- Exclude integration tests: `-skip Integration`
- OR use build tags: `-tags=!integration`

#### Issue 3.2: GCP Authentication Not Configured
**Status**: 🟡 EXPECTED (for staging/prod)
**Required Secrets**:
- `GCP_PROJECT_ID`
- `GCP_SA_KEY`
- `TURSO_URL` (in GCP Secret Manager)
- `TURSO_AUTH_TOKEN` (in GCP Secret Manager)
- `JWT_SECRET` (in GCP Secret Manager)
- `RESEND_API_KEY` (in GCP Secret Manager)

**Note**: This is expected - deployment should work once secrets are configured

---

## Priority Fixes

### High Priority (Blocks all workflows):
1. ✅ Fix Go Unit Tests workflow (DONE)
2. 🔴 Fix golangci-lint version in CI and Backend Tests
3. 🔴 Fix database test timeouts in Backend Tests
4. 🔴 Fix integration test server startup

### Medium Priority (CI improvements):
5. 🟡 Fix Deploy workflow test job
6. 🟡 Remove invalid golangci-lint parameters
7. 🟡 Handle security scan errors gracefully

### Low Priority (Can defer):
8. ⚪ Configure GCP secrets for deployment
9. ⚪ Optimize ARM64 build performance

---

## Recommended Fixes (In Order)

### Fix 1: Update golangci-lint Version
**Files**: `.github/workflows/ci.yml`, `.github/workflows/backend-tests.yml`
**Change**: `GOLANGCI_LINT_VERSION: 'v1.61'` → `GOLANGCI_LINT_VERSION: 'v1.62'`

### Fix 2: Remove Invalid Parameters
**File**: `.github/workflows/ci.yml`
**Remove**:
```yaml
skip-pkg-cache: false
skip-build-cache: false
```

### Fix 3: Fix Database Test Timeouts
**File**: `.github/workflows/backend-tests.yml`
**Change**: `go test ./... -v -coverprofile=coverage.out -race`
**To**: `go test ./... -v -coverprofile=coverage.out -race -timeout=5m`

**Alternative**: Skip MongoDB tests with build tag

### Fix 4: Fix Integration Tests
**File**: `.github/workflows/backend-tests.yml`
**Current**: Environment variables set in separate `run` command
**Fix**: Set as proper `env:` block OR start server properly

### Fix 5: Fix Deploy Test Job
**File**: `.github/workflows/deploy-cloud-run.yml`
**Change**: Exclude integration tests from pre-deployment test

---

## Success Metrics

After fixes:
- ✅ Go Unit Tests: PASSING (achieved)
- 🎯 CI workflow: Should pass lint and build jobs
- 🎯 Backend Tests: Should pass unit tests, skip integration
- 🎯 Deploy: Should build and push image (deploy will wait for secrets)

---

## Testing Plan

1. **Fix golangci-lint**: Push and check CI/Backend Tests lint jobs
2. **Fix database tests**: Push and check Backend Tests unit-tests job
3. **Fix integration tests**: Verify or skip them
4. **Fix deploy tests**: Check deploy workflow test job

---

## Notes

- Go version 1.24 is correct (verified locally)
- Scripts exist: `smoke-tests.sh`, `deploy-cloudrun.sh`
- Integration test directory exists: `test/integration/`
- Server main.go exists: `cmd/server/main.go`
