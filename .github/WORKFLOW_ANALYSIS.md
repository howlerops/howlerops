# CI/CD Workflow Analysis & Fixes

## Executive Summary

**Current State**: 0 passing workflows
**Root Causes**: Directory mismatches, obsolete configurations, syntax errors, missing dependencies
**Priority**: Fix core CI workflow first, then deployment workflows

## Workflow Inventory

### ✅ Keep & Fix (Priority)

1. **ci.yml** - Primary CI workflow
   - Status: MOSTLY GOOD
   - Issues: Minor, should mostly work
   - Action: Test and fix minor issues

2. **backend-tests.yml** - Comprehensive backend testing
   - Status: HAS ISSUES
   - Issues: PostgreSQL config, server startup
   - Action: Update database config, simplify tests

3. **deploy-cloud-run.yml** - Cloud Run deployment
   - Status: HAS CRITICAL ISSUES
   - Issues: Syntax error (duplicate `push`), wrong Dockerfile path
   - Action: Fix syntax, update paths

### ⚠️ Review & Update

4. **unit-tests.yml** - Specific unit test runner
   - Status: MAY WORK
   - Issues: Tests specific modules
   - Action: Verify module paths exist

5. **integration-tests.yml** - Integration test runner
   - Status: SHOULD WORK
   - Issues: None if integration tests exist
   - Action: Test as-is

6. **deploy-backend.yml** - Comprehensive backend deployment
   - Status: COMPLEX BUT FIXABLE
   - Issues: Dockerfile path references
   - Action: Update paths, verify secrets

### ❌ Remove or Archive

7. **ci-cd.yml** - OLD monolithic workflow
   - Status: COMPLETELY BROKEN
   - Issues: References non-existent "backend" and "frontend" directories
   - Action: DELETE or ARCHIVE

8. **deploy-production.yml** - Kubernetes deployment
   - Status: NOT APPLICABLE
   - Issues: References K8s infra that doesn't exist
   - Action: DELETE or ARCHIVE

9. **release.yml** & **release-macos.yml** - Release workflows
   - Status: UNKNOWN (didn't review)
   - Action: Review separately

## Detailed Issues & Fixes

### Critical Issue #1: deploy-cloud-run.yml Syntax Error

**Problem**: Duplicate `on.push` keys (YAML syntax error)
```yaml
on:
  push:
    tags:
      - 'v*.*.*'
  push:  # ❌ DUPLICATE KEY
    branches:
      - main
```

**Fix**: Combine into single push trigger
```yaml
on:
  push:
    branches:
      - main
    tags:
      - 'v*.*.*'
    paths:
      - 'backend-go/**'
      - '.github/workflows/deploy-cloud-run.yml'
```

### Critical Issue #2: ci-cd.yml Wrong Directories

**Problem**: References old project structure
```yaml
working-directory: ./backend  # ❌ Doesn't exist
working-directory: ./frontend # ❌ Wrong context
```

**Fix**: Remove this workflow entirely (replaced by ci.yml)

### Issue #3: backend-tests.yml Database Configuration

**Problem**: Uses PostgreSQL but project uses Turso/SQLite
```yaml
services:
  postgres:  # ❌ Not used in production
```

**Fix**: Remove PostgreSQL service, use Turso configuration
```yaml
- name: Set up test environment
  run: |
    export TURSO_URL="file:./test.db"
    export JWT_SECRET="test-secret-key-32-chars-minimum"
```

### Issue #4: Backend Dockerfile Path Inconsistency

**Problem**: Some workflows reference wrong paths
- Wrong: `./infrastructure/docker/backend.Dockerfile`
- Correct: `./backend-go/Dockerfile`

**Fix**: Update all workflows to use `./backend-go/Dockerfile`

### Issue #5: Missing Go Version Consistency

**Problem**: Different workflows use different Go versions
- ci.yml: `1.24`
- backend-tests.yml: `1.24`
- unit-tests.yml: `1.24.5`

**Fix**: Standardize on Go 1.24 across all workflows

### Issue #6: CGO Requirements

**Problem**: Some workflows don't install SQLite dev libraries
```yaml
# ❌ Missing
- name: Set up Go
  uses: actions/setup-go@v5
```

**Fix**: Add system dependencies before Go setup
```yaml
- name: Install system dependencies
  run: |
    sudo apt-get update -qq
    sudo apt-get install -y -qq gcc musl-dev sqlite3 libsqlite3-dev
```

## Recommended Action Plan

### Phase 1: Quick Wins (30 minutes)

1. **Delete broken workflows**
   ```bash
   mv .github/workflows/ci-cd.yml .github/workflows/archived/
   mv .github/workflows/deploy-production.yml .github/workflows/archived/
   ```

2. **Fix deploy-cloud-run.yml syntax**
   - Combine duplicate `push` triggers
   - This should make it parse correctly

3. **Test ci.yml**
   - Push a change and verify it runs
   - This is the most important workflow

### Phase 2: Core Fixes (1-2 hours)

4. **Fix backend-tests.yml**
   - Remove PostgreSQL service
   - Use file-based SQLite for tests
   - Simplify server startup tests

5. **Fix unit-tests.yml**
   - Verify module paths exist
   - Update if needed

6. **Test integration-tests.yml**
   - Run to verify it works
   - Fix if needed

### Phase 3: Deployment Fixes (2-3 hours)

7. **Fix deploy-cloud-run.yml**
   - Fix syntax
   - Update Dockerfile paths
   - Test manual deployment

8. **Fix deploy-backend.yml**
   - Update Dockerfile paths
   - Verify GCP secrets exist
   - Test deployment

### Phase 4: Verification

9. **Create test PR**
   - Verify workflows run
   - Check which ones pass
   - Fix remaining issues

10. **Update documentation**
    - Document which workflows do what
    - Add setup instructions for secrets
    - Create troubleshooting guide

## Quick Reference: What Each Workflow Does

| Workflow | Trigger | Purpose | Status |
|----------|---------|---------|--------|
| ci.yml | PR/Push to main | Tests, lint, build | ✅ Should work |
| backend-tests.yml | PR/Push | Comprehensive tests | ⚠️ Needs fixes |
| unit-tests.yml | PR/Push | Fast unit tests | ⚠️ Check paths |
| integration-tests.yml | Daily/Manual | Integration tests | ✅ Should work |
| deploy-cloud-run.yml | Release/Main push | Deploy to Cloud Run | ❌ Syntax error |
| deploy-backend.yml | Release | Full deployment | ⚠️ Path issues |
| ci-cd.yml | PR/Push | OLD monolithic | ❌ DELETE |
| deploy-production.yml | Release | K8s deployment | ❌ DELETE |

## Required GitHub Secrets

Ensure these are configured in GitHub repo settings:

### Essential (for CI)
- None required for basic CI

### For Cloud Run Deployment
- `GCP_PROJECT_ID` - Google Cloud project ID
- `GCP_SA_KEY` - Service account JSON key
- `TURSO_URL` - Database URL
- `TURSO_AUTH_TOKEN` - Database token
- `JWT_SECRET` - JWT signing key (32+ chars)

### Optional
- `RESEND_API_KEY` - Email service
- `RESEND_FROM_EMAIL` - From address
- `CODECOV_TOKEN` - Code coverage reporting

## Next Steps

1. Start with Phase 1 (delete broken workflows)
2. Fix deploy-cloud-run.yml syntax error
3. Test ci.yml on a new PR
4. Fix remaining issues as they appear

## Files to Create/Update

1. `.github/workflows/archived/` - Directory for old workflows
2. `.github/WORKFLOW_ANALYSIS.md` - This file
3. `.github/workflows/deploy-cloud-run.yml` - Fix syntax
4. `.github/workflows/backend-tests.yml` - Fix database config
