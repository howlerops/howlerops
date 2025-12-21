# Homebrew Script Fixes - Implementation Guide

## Quick Summary

**Three critical changes needed:**

1. Remove `|| true` error suppression that hides failures
2. Increase wait time from 5 to 15 minutes before Homebrew job starts
3. Add exponential backoff retry instead of fixed delays

---

## Fix #1: Correct Function Return Logic

### Current Problem (Lines 98-110)

```bash
response=$(gh api "repos/${GITHUB_REPO}/releases/latest")
local exit_code=$?

if [ $exit_code -ne 0 ]; then
    log_error "Failed to fetch release via gh CLI (exit code: $exit_code):"
    echo "$response" | head -20
    return 1  # ✗ This only exits the function, not the subshell
fi

log_info "Successfully fetched latest release via gh CLI"
echo "$response"
return 0
```

**The Issue:**
The `response=$(... )` creates a subshell. The `return 1` inside the subshell doesn't propagate error state to the caller properly. The assignment completes, so `response` contains either the result or error message.

### Fixed Version

```bash
get_latest_release() {
    log_info "Fetching latest release information from GitHub..."

    if command -v gh &> /dev/null; then
        log_info "Using gh CLI to fetch latest release (GH_TOKEN is ${GH_TOKEN:+set}${GH_TOKEN:-unset})"

        # Capture response AND exit code separately
        local response
        local exit_code

        response=$(gh api "repos/${GITHUB_REPO}/releases/latest" 2>&1)
        exit_code=$?

        if [ $exit_code -ne 0 ]; then
            log_error "Failed to fetch release via gh CLI (exit code: $exit_code):"
            echo "$response" | head -20
            return 1
        fi

        # Validate response is actually JSON before returning
        if ! echo "$response" | jq empty > /dev/null 2>&1; then
            log_error "Invalid JSON response from gh CLI:"
            echo "$response" | head -20
            return 1
        fi

        echo "$response"
        return 0
    else
        log_warning "gh CLI not found, falling back to curl"
    fi

    # [rest of curl fallback - unchanged]
}
```

### Fix #2: Remove Silent Error Suppression at Caller Level

### Current Problem (Lines 379-383)

```bash
while [ $retry_count -lt $max_retries ]; do
    log_info "Fetching release information (attempt $((retry_count + 1))/$max_retries)..."

    if [ "$version" = "latest" ]; then
        release_data=$(get_latest_release) || true  # ✗ THIS IS THE PROBLEM
    else
        release_data=$(get_specific_release "$version") || true  # ✗ THIS TOO
    fi
```

**The Issue:**
The `|| true` at the end makes the command always succeed, even if the function returns an error. This means `release_data` can be empty, but the script continues as if it succeeded.

### Fixed Version

```bash
while [ $retry_count -lt $max_retries ]; do
    log_info "Fetching release information (attempt $((retry_count + 1))/$max_retries)..."

    # Capture function result and exit code separately
    local fetch_exit_code

    if [ "$version" = "latest" ]; then
        release_data=$(get_latest_release)
    else
        release_data=$(get_specific_release "$version")
    fi
    fetch_exit_code=$?

    # Debug logging (keep this - it's helpful!)
    if [ -n "$release_data" ]; then
        log_info "DEBUG: Fetch succeeded, response length: ${#release_data} chars"
        log_info "DEBUG: First 200 chars:"
        echo "$release_data" | head -c 200 | sed 's/^/  /'
    else
        log_warning "DEBUG: Fetch returned empty response (exit code: $fetch_exit_code)"
    fi

    # Validate release data
    if [ -z "$release_data" ]; then
        log_warning "Release fetch returned empty response"
    elif ! echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
        log_warning "Release data is invalid JSON or missing tag_name"
    else
        # Verify assets are present
        local asset_count
        asset_count=$(echo "$release_data" | jq '.assets | length' 2>/dev/null || echo "0")

        if [ "$asset_count" -gt 0 ]; then
            log_success "Found release with $asset_count assets"
            break  # Success!
        else
            log_warning "Release found but no assets uploaded yet ($asset_count assets)"
        fi
    fi

    # Retry logic
    retry_count=$((retry_count + 1))
    if [ $retry_count -lt $max_retries ]; then
        # Exponential backoff instead of fixed delay
        local backoff_delay=$((30 * (2 ** (retry_count - 1))))
        if [ $backoff_delay -gt 300 ]; then
            backoff_delay=300  # Cap at 5 minutes
        fi
        log_warning "Retrying in $backoff_delay seconds (attempt $retry_count/$max_retries)..."
        sleep $backoff_delay
    else
        log_error "Failed to fetch release with assets after $max_retries attempts"
        log_error "GitHub CDN may need more time to propagate. Try running this script again."
        exit 1
    fi
done
```

---

## Fix #3: Increase Workflow Wait Time

### Current Problem (release.yml lines 710-715)

```yaml
- name: Wait for GitHub release to stabilize
  run: |
    echo "Waiting 5 minutes for all build jobs to complete and release API to stabilize..."
    echo "This ensures all assets are uploaded and the release is fully indexed."
    echo "Build jobs can take 3-5 minutes to complete."
    sleep 300  # ✗ Only 5 minutes
```

**The Issue:**
Five minutes is not enough time for:
- 5 concurrent build jobs (3-5 min each)
- Asset uploads (2-5 min)
- CDN propagation (1-2 min)
- Checksum generation and upload (2-3 min)

Total real time: 8-15 minutes before Homebrew job should run.

### Fixed Version

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

This way:
- Wait time is based on realistic timelines
- Provides progress feedback every 3 minutes
- Can be easily adjusted if needed

**Alternative: Even simpler**

```yaml
- name: Wait for GitHub release to stabilize
  run: sleep 900  # 15 minutes
```

---

## Fix #4: Improve Error Messages

The current debug logging (lines 385-400) is good but confusing. Make it clearer:

### Current (Confusing)

```bash
log_info "DEBUG: release_data length: ${#release_data} chars"
if [ -n "$release_data" ]; then
    log_info "DEBUG: First 200 chars of release_data:"
    echo "$release_data" | head -c 200 | sed 's/^/  /' >&2
    log_info "DEBUG: Attempting jq validation..."
    if echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
        log_info "DEBUG: jq validation PASSED"
    else
        log_warning "DEBUG: jq validation FAILED"
```

### Improved (Clear)

```bash
if [ -z "$release_data" ]; then
    log_error "API call returned empty response - function may have failed silently"
    log_error "Check that GH_TOKEN or GITHUB_TOKEN is set and valid"
elif ! echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
    log_error "API response is not valid JSON or missing required fields"
    log_info "Response preview:"
    echo "$release_data" | head -c 200 | sed 's/^/  /'
else
    local asset_count
    asset_count=$(echo "$release_data" | jq '.assets | length')
    if [ "$asset_count" -eq 0 ]; then
        log_warning "Release exists but has no assets yet"
        log_info "Asset count: $asset_count"
    else
        log_success "Found valid release with $asset_count assets"
    fi
fi
```

---

## Complete Fixed Script (Key Sections)

Here's the script with all three fixes integrated. I'm showing the main retry loop:

```bash
# Fetch release information (with retry for eventual consistency)
local release_data=""
local retry_count=0
local max_retries=10

log_info "Starting release fetch with up to $max_retries attempts (exponential backoff)"

while [ $retry_count -lt $max_retries ]; do
    log_info "Release fetch attempt $((retry_count + 1))/$max_retries"

    # Attempt to fetch release - DO NOT suppress errors
    if [ "$version" = "latest" ]; then
        release_data=$(get_latest_release)
    else
        release_data=$(get_specific_release "$version")
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
            log_success "Release found with $asset_count assets. Proceeding..."
            break
        else
            log_warning "Release found but no assets yet (0/$asset_count expected)"
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

        log_warning "Will retry in $backoff_delay seconds (attempt $((retry_count + 1))/$max_retries)"
        sleep $backoff_delay
    fi
done

# Check if we got valid data
if [ -z "$release_data" ] || ! echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
    log_error "Failed to fetch valid release after $max_retries attempts"
    log_error "Possible causes:"
    log_error "  1. GitHub token (GH_TOKEN/GITHUB_TOKEN) is invalid or expired"
    log_error "  2. Repository is not accessible with current token"
    log_error "  3. GitHub API rate limit exceeded - wait and try again"
    log_error "  4. Release build/upload jobs are still in progress"
    log_error ""
    log_error "To debug: run 'gh api repos/jbeck018/howlerops/releases/latest'"
    exit 1
fi

log_success "Successfully fetched release with assets"

# Continue with rest of script...
```

---

## Implementation Checklist

- [ ] Fix `get_latest_release()` function (lines 92-143)
  - Add JSON validation before returning
  - Properly capture exit code from gh CLI

- [ ] Fix `get_specific_release()` function (lines 145-202)
  - Same changes as above

- [ ] Remove `|| true` from lines 380-382
  - Change to proper error handling

- [ ] Replace retry loop (lines 376-428)
  - Add exponential backoff
  - Improve error messages
  - Keep debug logging but make it clearer

- [ ] Update release.yml line 715
  - Change `sleep 300` to `sleep 900`
  - Or add progress feedback loop

- [ ] Test with a release tag
  - Monitor the Homebrew job logs
  - Verify release_data is not empty
  - Check that assets are detected

---

## Testing Instructions

After making changes:

1. Create a test release:
   ```bash
   git tag v0.7.15-test-$(date +%s)
   git push origin --tags
   ```

2. Monitor in GitHub Actions:
   - Check "Update Homebrew Formula" job
   - Look for "DEBUG" messages showing release_data is NOT empty
   - Verify assets are detected

3. Expected output should show:
   ```
   [INFO] Release fetch attempt 1/10
   [INFO] DEBUG: Fetch succeeded, response length: XXXX chars
   [SUCCESS] Found release with 12 assets. Proceeding...
   ```

4. NOT:
   ```
   [INFO] Release fetch attempt 1/10
   [WARNING] DEBUG: release_data is EMPTY
   [WARNING] Release not found or API returned invalid response
   [INFO] Release fetch attempt 2/10
   ```

---

## Why These Fixes Work

1. **Fix #1**: Validates JSON before returning from functions, catches malformed responses
2. **Fix #2**: Allows real errors to bubble up instead of being silently ignored
3. **Fix #3**: Gives the GitHub API time to fully propagate all assets before attempting download
4. **Fix #4**: Makes error messages actionable instead of cryptic

The combination of these fixes eliminates the race condition and reveals the actual error (if any) rather than hiding it with `|| true`.

---

## If Still Failing After These Fixes

If the Homebrew job still fails after implementing all fixes, check:

1. **GH_TOKEN is valid**:
   ```bash
   gh auth status  # Should show authenticated
   ```

2. **Token has correct scopes**:
   - Should have `repo` scope (for public repo)
   - Should have `write:packages` if using package registry

3. **Release assets are actually uploaded**:
   ```bash
   gh release view v0.7.15-test --json assets
   ```

4. **No rate limiting**:
   ```bash
   gh api rate_limit  # Check remaining requests
   ```

5. **Consider using GitHub webhook** instead of polling (see Analysis document)

