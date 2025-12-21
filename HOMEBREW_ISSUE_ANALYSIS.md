# GitHub Actions Homebrew Update Failure Analysis

## Executive Summary

The "Update Homebrew Formula" job is failing due to **three critical issues** that interact in a dangerous way:

1. **Silent function failure propagation** (Lines 380-382): The `|| true` prevents errors from bubbling up
2. **Timing race condition** (Insufficient wait time): 5 minutes is too aggressive
3. **GitHub Actions environment issue** (Authentication/rate limiting)

The validation loop (lines 385-417) appears to show good debugging, but it's catching downstream failures from broken function calls.

---

## Problem 1: Silent Function Failure (Root Cause)

### The Issue

Lines 380-382 in the main script have a critical flaw:

```bash
if [ "$version" = "latest" ]; then
    release_data=$(get_latest_release) || true        # ❌ DANGEROUS
else
    release_data=$(get_specific_release "$version") || true  # ❌ DANGEROUS
fi
```

**What's happening:**
- `get_latest_release()` and `get_specific_release()` both call `exit 1` on failure (lines 127, 186)
- The `exit 1` is **inside the subshell** (parentheses), so it exits only that subshell
- The calling script continues, but `release_data` is **empty or partially set**
- The `|| true` swallows the error code, preventing detection

### How This Manifests

When `gh api` fails in GitHub Actions:

```bash
response=$(gh api "repos/${GITHUB_REPO}/releases/latest")  # This fails
local exit_code=$?  # Captures failure

if [ $exit_code -ne 0 ]; then
    log_error "Failed to fetch..."
    return 1  # Returns from the function
fi
```

But the calling code uses:

```bash
release_data=$(get_latest_release) || true  # The '|| true' makes this always succeed!
```

So `release_data` becomes empty, and the validation loop says "Release not found or API returned invalid response."

### Why This Happens Specifically in GitHub Actions

`gh CLI` in GitHub Actions may fail for:
- **Token authentication issues** - Even with GH_TOKEN set, gh may not find it properly
- **Rate limiting** - Immediate retry with same token triggers rate limits
- **Session issues** - GitHub Actions' gh context may be stale
- **Permission issues** - The token may not have API access to the repo

---

## Problem 2: Timing Race Condition (5 minutes insufficient)

### The Issue

Line 712-715 in the workflow:

```yaml
- name: Wait for GitHub release to stabilize
  run: |
    echo "Waiting 5 minutes for all build jobs to complete and release API to stabilize..."
    sleep 300  # ❌ Only 5 minutes
```

### Why This Isn't Enough

The workflow runs jobs in parallel:
1. **build-binaries** - Builds 5 platform variants, uploads 4 assets each (20 uploads)
2. **build-howlerops** - Builds desktop app, uploads 2 assets
3. **generate-checksums** - Downloads all assets, creates checksums file, uploads
4. **validate-release** - Downloads and validates all assets

**Realistic timeline:**
- Build jobs: 3-5 minutes (parallel, so fastest determines time)
- Asset uploads: 2-5 minutes (can be slow, especially if GitHub's CDN has issues)
- Checksum job: 1-2 minutes (waits for all builds)
- Validation job: 2-3 minutes
- **Total before Homebrew job starts: 8-15 minutes minimum**

Even with `needs: [create-release, build-binaries, build-howlerops, generate-checksums, validate-release]`, GitHub Actions' job dependency system causes the Homebrew job to start as soon as the validate job **finishes running**, not when its results are stable.

### What Happens at 5-Minute Mark

If you're at the 5-minute wait point but build/upload jobs are still in progress:
- GitHub's API returns the release object ✓
- But `assets[]` array is **incomplete or empty**
- The validation check at line 408 catches this: `asset_count=$(echo "$release_data" | jq '.assets | length')`
- But the retry loop fails repeatedly because assets still aren't uploaded

---

## Problem 3: GitHub Actions Environment - gh CLI Authentication

### The Issue

Lines 97, 156 show gh CLI authentication detection, but the execution environment may have issues:

```bash
log_info "Using gh CLI to fetch latest release (GH_TOKEN is ${GH_TOKEN:+set}${GH_TOKEN:-unset})"
response=$(gh api "repos/${GITHUB_REPO}/releases/latest")
```

### Problems in GitHub Actions Specifically

1. **Token scope issues**: GITHUB_TOKEN (auto-generated) has limited scope in Actions
   - May not have full `public_repo` access needed
   - Cannot push to external repositories (like homebrew-tap)

2. **gh CLI context confusion**: The workflow sets both:
   ```yaml
   GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
   GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
   ```
   - This can cause gh CLI to receive conflicting signals
   - gh prioritizes GH_TOKEN but may fail if it detects GITHUB_TOKEN exists

3. **Rate limiting on first call**: If gh fails once, immediate retry hits rate limits
   - The 30-second delay between retries may not be enough to recover
   - GitHub API rate limits are per-token, not per-IP

4. **SSH vs HTTPS**: The script uses HTTPS for cloning (line 273)
   - GITHUB_TOKEN works for HTTPS, but token in URL can be logged
   - The script tries to filter token at line 274: `grep -v "token"` but this is weak

---

## Secondary Issues

### Issue A: jq Validation Not Capturing Real Errors (Lines 391-396)

```bash
if echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
    log_info "DEBUG: jq validation PASSED"
else
    log_warning "DEBUG: jq validation FAILED"
fi
```

This silently succeeds even if:
- `release_data` is empty string (`jq` returns error code 5, which is suppressed)
- `release_data` is null
- `release_data` is malformed JSON

**Result:** The loop retries, but the root cause (empty release_data) is never identified.

### Issue B: Asset Count Check Too Late (Lines 405-414)

The asset count check happens AFTER successful jq validation:

```bash
if echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
    local asset_count
    asset_count=$(echo "$release_data" | jq '.assets | length' 2>/dev/null || echo "0")

    if [ "$asset_count" -gt 0 ]; then
        # Good!
    else
        # Assets not uploaded yet - retry
    fi
```

But if `release_data` is empty (from the `|| true` at line 380), jq will fail to validate `.tag_name`, and you'll hit line 416: "Release not found or API returned invalid response" without ever checking assets.

### Issue C: Curl Fallback Has No Retry (Lines 115-142)

When gh CLI fails, it falls back to curl, but:
- No rate limit detection
- No intelligent retry delay
- No detection of transient failures (timeouts, 503s)
- Treats all errors the same way

---

## Root Cause Summary

```
GitHub Actions starts Homebrew job at T+5min
    ↓
Script calls get_latest_release() which calls "gh api ..."
    ↓
gh CLI fails (token issue, rate limit, timeout, etc.)
    ↓
Function returns 1 and exits subshell
    ↓
Calling code has "|| true" which swallows exit code
    ↓
release_data variable is EMPTY (assignment inside failed subshell)
    ↓
Validation loop tries jq -e '.tag_name' on empty string
    ↓
jq fails silently (error suppressed by 2>&1 redirection)
    ↓
Loop thinks "Release not found or API returned invalid response"
    ↓
Retries 10 times with same issue
    ↓
Job fails with confusing error message
```

---

## Why v0.7.13 Changes Didn't Help

You removed `2>&1` from lines 380, 382, and lines 99, 158:

**Before (v0.7.12):**
```bash
response=$(gh api "repos/${GITHUB_REPO}/releases/latest" 2>&1) || true
```

**After (v0.7.13):**
```bash
response=$(gh api "repos/${GITHUB_REPO}/releases/latest") || true
```

This helped visibility (stderr now appears in logs), but the fundamental issue remains: **the `|| true` is in the wrong place**.

The problem is that you're suppressing the error at the **caller** level (line 380) instead of **inside the function** (line 99).

---

## Immediate Fixes (Priority Order)

### Fix 1: Remove Silent Error Suppression (Lines 380-382)

**Current:**
```bash
if [ "$version" = "latest" ]; then
    release_data=$(get_latest_release) || true
else
    release_data=$(get_specific_release "$version") || true
fi
```

**Should be:**
```bash
if [ "$version" = "latest" ]; then
    release_data=$(get_latest_release)
    if [ $? -ne 0 ]; then
        log_error "Failed to fetch latest release"
        exit 1
    fi
else
    release_data=$(get_specific_release "$version")
    if [ $? -ne 0 ]; then
        log_error "Failed to fetch release $version"
        exit 1
    fi
fi
```

Or simpler with `set -e` semantics:
```bash
if [ "$version" = "latest" ]; then
    release_data=$(get_latest_release)
else
    release_data=$(get_specific_release "$version")
fi
```

(The `set -euo pipefail` at line 27 should already cause exit on error)

### Fix 2: Fix Function Return Semantics (Lines 105, 164)

**Current (doesn't work with `set -e`):**
```bash
if [ $exit_code -ne 0 ]; then
    log_error "Failed to fetch release..."
    return 1
fi
```

**Should be:**
```bash
if [ $exit_code -ne 0 ]; then
    log_error "Failed to fetch release..."
    return 1
fi
```

Actually, the issue is the subshell. Using `$(...)` creates a subshell where `exit 1` kills only that subshell. Better:

```bash
# Use command substitution properly
response=$(gh api "repos/${GITHUB_REPO}/releases/latest" || echo "ERROR")
if [[ "$response" == "ERROR" ]]; then
    log_error "gh api call failed"
    return 1
fi
```

### Fix 3: Increase Wait Time (Line 712-715)

**Current:**
```bash
sleep 300  # 5 minutes
```

**Should be:**
```bash
sleep 600  # 10 minutes minimum
```

Better would be 15 minutes to account for slow asset uploads and CDN propagation:
```bash
sleep 900  # 15 minutes
```

### Fix 4: Intelligent Retry with Exponential Backoff (Lines 376-428)

The current retry logic uses fixed 30-second delays. This should:
1. Detect actual vs transient errors
2. Increase delay exponentially: 30s, 60s, 120s, 240s, etc.
3. Provide better logging of what each retry attempt found

---

## Long-Term Improvements

### Option A: Use Release API Polling Instead of Retries

Instead of retrying the entire function, poll a specific endpoint to detect when assets are uploaded:

```bash
# Wait until all expected assets are present
wait_for_assets() {
    local max_attempts=20
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        release_data=$(get_latest_release)
        asset_count=$(echo "$release_data" | jq '.assets | length' 2>/dev/null || echo "0")

        expected_count=12  # 5 platforms × 4 assets + checksums + desktop

        if [ "$asset_count" -ge "$expected_count" ]; then
            echo "$release_data"
            return 0
        fi

        log_info "Assets: $asset_count/$expected_count. Waiting..."
        sleep 30
        attempt=$((attempt + 1))
    done

    return 1
}
```

### Option B: Use Webhook Instead of Polling

Instead of polling, trigger the Homebrew update job via a separate workflow that's triggered when the release is complete:

```yaml
on:
  release:
    types: [published]

jobs:
  update-homebrew:
    runs-on: ubuntu-latest
    steps:
      # By the time this triggers, release is guaranteed to be complete
      - run: ./scripts/update-homebrew-formula.sh
```

This eliminates timing issues entirely.

### Option C: Integrate into Release Job

Move the Homebrew update into the `generate-checksums` job, after the checksums file is uploaded. This ensures:
- Assets are fully uploaded
- Checksums are generated
- No additional timing delays needed

---

## Testing the Fix

To validate the changes:

1. Create a test tag: `git tag v0.7.15-test`
2. Push the tag: `git push origin v0.7.15-test`
3. Monitor the Homebrew job output in GitHub Actions
4. Check that release_data is not empty in the logs
5. Verify the formula is updated in the tap repo

---

## Files to Modify

1. **scripts/update-homebrew-formula.sh**
   - Lines 380-382: Remove `|| true`
   - Lines 99, 158: Fix function return logic
   - Lines 376-428: Improve retry with exponential backoff

2. **.github/workflows/release.yml**
   - Line 715: Increase sleep from 300 to 900 seconds
   - Consider moving job to after generate-checksums

---

## Why This Wasn't Caught Sooner

- The debug logging you added (lines 385-400) is excellent but activated AFTER the problem
- The functions return non-zero exit codes, but those are suppressed by `|| true`
- The actual error from `gh api` is lost because it's in a subshell
- The validation loop's error messages are vague ("Release not found or API returned invalid response")

The fix is straightforward: **eliminate the error suppression and let actual errors bubble up**.
