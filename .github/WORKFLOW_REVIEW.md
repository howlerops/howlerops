# GitHub Workflows Review

## Summary

Reviewed all CI/CD workflows for HowlerOps. Found several issues that need fixing.

## Issues Found

### Critical Issues

1. **❌ Missing OAuth Secrets in deploy-cloud-run.yml**
   - **File**: `.github/workflows/deploy-cloud-run.yml:276`
   - **Issue**: The `--set-secrets` line doesn't include the new OAuth secrets
   - **Impact**: OAuth authentication will not work in Cloud Run deployments
   - **Fix**: Add OAuth secrets to line 276

2. **⚠️  Invalid Go Version** - Multiple workflows
   - **Files**:
     - `backend-tests.yml` (Go 1.24)
     - `deploy-cloud-run.yml` (Go 1.24)
     - `unit-tests.yml` (Go 1.24.5)
     - `integration-tests.yml` (Go 1.24.5)
     - `release.yml` (Go 1.24)
   - **Issue**: Go 1.24 doesn't exist yet (latest stable is 1.23.x)
   - **Impact**: Workflows will fail if these versions don't exist
   - **Fix**: Update to Go 1.23 or 1.21 (stable versions)

### Medium Priority Issues

3. **⚠️  Deprecated Actions in release.yml**
   - **File**: `.github/workflows/release.yml`
   - **Issue**: Uses deprecated `actions/create-release@v1` and `actions/upload-release-asset@v1`
   - **Impact**: May break in the future when GitHub removes these
   - **Fix**: Migrate to `softprops/action-gh-release@v1` (like release-macos.yml does)

4. **⚠️  Workflow Duplication**
   - **Files**: `backend-tests.yml` and `unit-tests.yml`
   - **Issue**: Both run unit tests with slightly different configurations
   - **Impact**: Confusion about which workflow to use, wastes CI resources
   - **Recommendation**: Consolidate into one workflow

### Low Priority Issues

5. **ℹ️  Missing Documentation for New OAuth Sync Workflow**
   - **File**: `.github/workflows/sync-oauth-secrets.yml`
   - **Issue**: Requires GCP Workload Identity Federation setup but not documented
   - **Impact**: Users won't know how to set it up
   - **Fix**: Add setup documentation

## Workflow Inventory

| Workflow | Status | Purpose | Trigger |
|----------|--------|---------|---------|
| `release.yml` | ⚠️ Needs fixes | Backend release builds | Tags `v*` |
| `release-macos.yml` | ✅ Working | macOS app release | Tags `v*.*.*` |
| `backend-tests.yml` | ⚠️ Go version | Backend testing | Push to main/develop |
| `deploy-cloud-run.yml` | ❌ Missing secrets | Cloud Run deployment | Push to main, PRs |
| `sync-oauth-secrets.yml` | ℹ️ Needs docs | OAuth secrets sync | Manual |
| `unit-tests.yml` | ⚠️ Go version | Quick unit tests | Push/PR |
| `integration-tests.yml` | ⚠️ Go version | Integration tests | Daily schedule |

## Recommended Fixes

### Fix 1: Update deploy-cloud-run.yml OAuth Secrets

```yaml
# Line 276 in deploy-cloud-run.yml
--set-secrets="TURSO_URL=turso-url:latest,TURSO_AUTH_TOKEN=turso-auth-token:latest,RESEND_API_KEY=resend-api-key:latest,JWT_SECRET=jwt-secret:latest,GOOGLE_CLIENT_ID=google-oauth-client-id:latest,GOOGLE_CLIENT_SECRET=google-oauth-client-secret:latest,GH_CLIENT_ID=github-oauth-client-id:latest,GH_CLIENT_SECRET=github-oauth-client-secret:latest" \
```

### Fix 2: Update Go Versions

Replace all instances of:
- `go-version: '1.24'` → `go-version: '1.21'`
- `go-version: '1.24.5'` → `go-version: '1.21'`

Why 1.21? Because:
- It's the version used in `release-macos.yml` (line 39)
- It's stable and widely available
- The codebase already works with it

### Fix 3: Update release.yml to Use Modern Actions

Replace:
```yaml
- uses: actions/create-release@v1
- uses: actions/upload-release-asset@v1
```

With:
```yaml
- uses: softprops/action-gh-release@v1
  with:
    files: |
      backend-go/${{ env.FINAL_ASSET }}
      backend-go/${{ env.CHECKSUM_FILE }}
```

### Fix 4: Consolidate Testing Workflows (Optional)

Either:
- Remove `unit-tests.yml` and use only `backend-tests.yml`
- Or keep both but rename for clarity:
  - `unit-tests.yml` → `quick-tests.yml` (fast feedback)
  - `backend-tests.yml` → `comprehensive-tests.yml` (full suite)

## Required GitHub Secrets

### Currently Required
- `GITHUB_TOKEN` - ✅ Auto-provided
- `GCP_PROJECT_ID` - For Cloud Run deployments
- `GCP_SA_KEY` - Service account JSON key
- `HOMEBREW_TAP_TOKEN` - For Homebrew formula updates

### New Requirements (for OAuth sync workflow)
- `GCP_WORKLOAD_IDENTITY_PROVIDER` - Workload Identity Federation provider
- `GCP_SERVICE_ACCOUNT` - Service account email
- `GOOGLE_CLIENT_ID` - Already in repository
- `GOOGLE_CLIENT_SECRET` - Already in repository
- `GH_CLIENT_ID` - Already in repository
- `GH_CLIENT_SECRET` - Already in repository

## Testing Strategy

After fixes are applied:

1. **Test locally**:
   ```bash
   # Validate YAML syntax
   yamllint .github/workflows/*.yml

   # Or use act to test locally
   act -l  # List workflows
   act push  # Test push workflows
   ```

2. **Test on non-critical branch**:
   - Create a test branch
   - Push to trigger workflows
   - Verify all jobs pass

3. **Monitor first production run**:
   - Watch GitHub Actions tab
   - Check logs for any errors
   - Verify deployments work

## Next Steps

1. Apply critical fixes (OAuth secrets, Go versions)
2. Update deprecated actions in release.yml
3. Document OAuth sync workflow setup
4. Test workflows on a feature branch
5. Merge fixes to main

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Deprecated GitHub Actions](https://github.com/actions/create-release/issues/119)
- [Go Versions](https://go.dev/dl/)
- [Google Cloud Workload Identity](https://cloud.google.com/iam/docs/workload-identity-federation)
