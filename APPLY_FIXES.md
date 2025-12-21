# Apply Homebrew Fixes - Copy/Paste Implementation

This file has exact code changes you need to make. Pick the method that works best for you.

---

## Method 1: Use the Fixed Script (Easiest)

The fixed script is already prepared at: `update-homebrew-formula-FIXED.sh`

```bash
# Backup current version
cp scripts/update-homebrew-formula.sh scripts/update-homebrew-formula.sh.backup

# Copy the fixed version
cp update-homebrew-formula-FIXED.sh scripts/update-homebrew-formula.sh

# Update workflow file (see next section)

# Test it
git tag v0.7.15-test-$(date +%s)
git push origin --tags
# Monitor GitHub Actions for success
```

---

## Method 2: Manual Line-by-Line Changes

### Change 1: Lines 380-382 in `scripts/update-homebrew-formula.sh`

**FIND THIS:**
```bash
    while [ $retry_count -lt $max_retries ]; do
        log_info "Fetching release information (attempt $((retry_count + 1))/$max_retries)..."

        if [ "$version" = "latest" ]; then
            release_data=$(get_latest_release) || true
        else
            release_data=$(get_specific_release "$version") || true
        fi
```

**REPLACE WITH:**
```bash
    while [ $retry_count -lt $max_retries ]; do
        log_info "Fetching release information (attempt $((retry_count + 1))/$max_retries)..."

        # DO NOT suppress errors with || true - we need to know if the call failed
        if [ "$version" = "latest" ]; then
            if release_data=$(get_latest_release); then
                : # Success
            else
                release_data=""
            fi
        else
            if release_data=$(get_specific_release "$version"); then
                : # Success
            else
                release_data=""
            fi
        fi
```

**OR (simpler with `set -euo pipefail` already active):**
```bash
        if [ "$version" = "latest" ]; then
            release_data=$(get_latest_release) 2>&1 || release_data=""
        else
            release_data=$(get_specific_release "$version") 2>&1 || release_data=""
        fi
```

---

### Change 2: Lines 376-428 in `scripts/update-homebrew-formula.sh`

**FIND THIS ENTIRE BLOCK:**
```bash
    # Fetch release information (with retry for eventual consistency)
    # GitHub CDN can take time to propagate release assets globally
    local release_data
    local retry_count=0
    local max_retries=10
    local retry_delay=30

    while [ $retry_count -lt $max_retries ]; do
        log_info "Fetching release information (attempt $((retry_count + 1))/$max_retries)..."

        if [ "$version" = "latest" ]; then
            release_data=$(get_latest_release) || true
        else
            release_data=$(get_specific_release "$version") || true
        fi

        # Debug: Show what we got from the API
        log_info "DEBUG: release_data length: ${#release_data} chars"
        if [ -n "$release_data" ]; then
            log_info "DEBUG: First 200 chars of release_data:"
            echo "$release_data" | head -c 200 | sed 's/^/  /' >&2
            log_info "DEBUG: Attempting jq validation..."
            if echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
                log_info "DEBUG: jq validation PASSED"
            else
                log_warning "DEBUG: jq validation FAILED"
                log_warning "DEBUG: jq error output:"
                echo "$release_data" | jq -e '.tag_name' 2>&1 | head -5 | sed 's/^/  /' >&2
            fi
        else
            log_warning "DEBUG: release_data is EMPTY"
        fi

        # Check if we got valid data
        if echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
            # Verify assets are actually present
            local asset_count
            asset_count=$(echo "$release_data" | jq '.assets | length' 2>/dev/null || echo "0")

            if [ "$asset_count" -gt 0 ]; then
                log_success "Found release with $asset_count assets"
                break
            else
                log_warning "Release found but no assets uploaded yet (0 assets found)"
                log_info "Build jobs may still be running. Will retry..."
            fi
        else
            log_warning "Release not found or API returned invalid response"
        fi

        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            log_warning "Release not ready yet, waiting $retry_delay seconds before retry..."
            sleep $retry_delay
        else
            log_error "Failed to fetch release with assets after $max_retries attempts"
            log_error "GitHub CDN may need more time to propagate. Try running this script again in a few minutes."
            exit 1
        fi
    done
```

**REPLACE WITH:**
```bash
    # Fetch release information (with retry for eventual consistency)
    # GitHub CDN can take time to propagate release assets globally
    local release_data=""
    local retry_count=0
    local max_retries=10

    log_info "Fetching release with up to $max_retries attempts (exponential backoff)"

    while [ $retry_count -lt $max_retries ]; do
        log_info "Release fetch attempt $((retry_count + 1))/$max_retries"

        # Attempt to fetch release - DO NOT suppress errors
        if [ "$version" = "latest" ]; then
            if release_data=$(get_latest_release); then
                : # Success
            else
                release_data=""
            fi
        else
            if release_data=$(get_specific_release "$version"); then
                : # Success
            else
                release_data=""
            fi
        fi

        # Check result
        if [ -z "$release_data" ]; then
            log_warning "Release fetch returned empty response"
        elif ! echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
            log_error "API response is invalid JSON"
            log_info "Response preview:"
            echo "$release_data" | head -c 300 | sed 's/^/  /'
        else
            # Check asset count
            local asset_count
            asset_count=$(echo "$release_data" | jq '.assets | length')

            if [ "$asset_count" -gt 0 ]; then
                log_success "Found release with $asset_count assets. Proceeding..."
                break
            else
                log_warning "Release found but no assets yet ($asset_count/$asset_count expected)"
            fi
        fi

        # Exponential backoff retry
        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            # First retry: 30s, second: 60s, third: 120s, etc. (capped at 5 min)
            local backoff_delay=$((30 * (2 ** (retry_count - 1))))
            if [ $backoff_delay -gt 300 ]; then
                backoff_delay=300
            fi

            log_warning "Retrying in $backoff_delay seconds (attempt $((retry_count + 1))/$max_retries)..."
            sleep $backoff_delay
        fi
    done

    # Check if we got valid data
    if [ -z "$release_data" ] || ! echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
        log_error "Failed to fetch valid release after $max_retries attempts"
        log_error ""
        log_error "Possible causes:"
        log_error "  1. GitHub token (GH_TOKEN/GITHUB_TOKEN) is invalid or expired"
        log_error "  2. Repository is not accessible with current token"
        log_error "  3. GitHub API rate limit exceeded - wait and try again"
        log_error "  4. Release build/upload jobs are still in progress"
        log_error ""
        log_error "To debug:"
        log_error "  gh api repos/jbeck018/howlerops/releases/latest"
        log_error "  gh release view $version --json assets"
        exit 1
    fi
```

---

### Change 3: Line 715 in `.github/workflows/release.yml`

**FIND THIS:**
```yaml
      - name: Wait for GitHub release to stabilize
        run: |
          echo "Waiting 5 minutes for all build jobs to complete and release API to stabilize..."
          echo "This ensures all assets are uploaded and the release is fully indexed."
          echo "Build jobs can take 3-5 minutes to complete."
          sleep 300
```

**REPLACE WITH:**
```yaml
      - name: Wait for GitHub release to stabilize
        run: |
          echo "Waiting 15 minutes for all build jobs to complete and release API to stabilize..."
          echo "Timeline:"
          echo "  - Build jobs (parallel): 3-5 minutes"
          echo "  - Asset uploads: 2-5 minutes"
          echo "  - CDN propagation: 1-2 minutes"
          echo "  - Total safety window: 15 minutes"
          echo ""

          for i in {1..30}; do
            remaining=$((30 - i))
            if [ $((i % 6)) -eq 0 ]; then
              echo "Progress: $i/30 intervals (${remaining} intervals remaining)"
            fi
            sleep 30
          done

          echo "Release should now be fully propagated. Starting Homebrew update..."
```

**OR (simpler):**
```yaml
      - name: Wait for GitHub release to stabilize
        run: |
          echo "Waiting 15 minutes for all build jobs to complete and release API to stabilize..."
          sleep 900
```

---

## Verify Your Changes

After making changes, run these checks:

### Check 1: Script Syntax

```bash
bash -n scripts/update-homebrew-formula.sh
# Should output nothing if syntax is correct
```

### Check 2: Key Changes Present

```bash
# Should find 'if release_data=$(get_latest_release)' (without || true)
grep -n 'if release_data.*get_latest_release' scripts/update-homebrew-formula.sh

# Should find exponential backoff calculation
grep -n 'backoff_delay.*2 \*\*' scripts/update-homebrew-formula.sh

# Should NOT find '|| true' after get_latest_release or get_specific_release
grep -n 'get_latest_release.*|| true\|get_specific_release.*|| true' scripts/update-homebrew-formula.sh
# (This should return empty - no matches)
```

### Check 3: Workflow File

```bash
# Should show sleep 900 (not 300)
grep -n 'sleep 9' .github/workflows/release.yml

# Should NOT show sleep 300 for the Homebrew wait
grep -B5 'Update Homebrew' .github/workflows/release.yml | grep sleep
```

---

## Test the Fix

```bash
# Create a test release to verify the fix works
git tag v0.7.15-test-$(date +%s)
git push origin --tags

# Wait for workflow to start and monitor it:
# 1. Go to GitHub Actions
# 2. Find the Release workflow
# 3. Watch the "Update Homebrew Formula" job
# 4. Look for these good signs:
#    - "Found release with XX assets"
#    - "Homebrew formula update completed successfully!"
#
# If it still fails, check for these bad signs:
#    - "Release not found or API returned invalid response"
#    - "release_data is EMPTY"
#
# If it fails, run these debug commands:
gh api repos/jbeck018/howlerops/releases/latest | jq '.tag_name'
gh release view v0.7.15-test-XXXXX --json assets
```

---

## Rollback If Needed

If something goes wrong:

```bash
# Restore backup
cp scripts/update-homebrew-formula.sh.backup scripts/update-homebrew-formula.sh

# Revert workflow
git checkout .github/workflows/release.yml
```

---

## Summary of Changes

| File | Lines | Change | Why |
|------|-------|--------|-----|
| `scripts/update-homebrew-formula.sh` | 380-382 | Remove `\|\| true` after API calls | Stops hiding errors |
| `scripts/update-homebrew-formula.sh` | 376-428 | Replace fixed retry with exponential backoff | Better handling of timing issues |
| `.github/workflows/release.yml` | 715 | Change `sleep 300` to `sleep 900` | Realistic timeline for asset propagation |

---

## After Applying Fixes

Commit your changes:

```bash
git add scripts/update-homebrew-formula.sh .github/workflows/release.yml
git commit -m "fix: improve Homebrew formula update reliability

- Remove silent error suppression that hides API failures
- Add exponential backoff retry strategy (30s, 60s, 120s, ...)
- Increase wait time from 5 to 15 minutes for asset propagation
- Improve error messages with actionable debugging info

Fixes: Homebrew formula update consistently failing with 'Release not found' error"
```

Test with a new release:

```bash
git tag v0.7.15
git push origin v0.7.15
# Monitor the Homebrew job in GitHub Actions
```

