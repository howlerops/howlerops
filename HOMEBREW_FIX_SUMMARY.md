# Homebrew Update Failure - Root Cause & Fix Summary

## The Core Problem (One Sentence)

**The script suppresses errors with `|| true`, causing empty API responses to be silently treated as success, then failing validation on empty data.**

---

## What's Really Happening

```
gh api call fails in GitHub Actions
         ↓
Function tries to return an error
         ↓
But `release_data=$(get_latest_release) || true` ignores the error
         ↓
release_data becomes EMPTY
         ↓
Script tries to validate empty string
         ↓
"Release not found or API returned invalid response"
         ↓
Retries 10 times with same broken data
         ↓
Job fails
```

The actual `gh api` error is **completely hidden** by the `|| true`.

---

## Why This Happens in GitHub Actions Specifically

1. **gh CLI authentication issues**: Even with GH_TOKEN set, gh may not properly authenticate
2. **Rate limiting**: First failure triggers rate limit, subsequent retries fail immediately
3. **Token scope**: GITHUB_TOKEN in Actions has limited scope
4. **Timing**: Assets still uploading when API query happens (before the 5-minute wait)

---

## Three Required Fixes

### Fix 1: Remove Error Suppression (HIGH PRIORITY)

**Lines 380-382** in `scripts/update-homebrew-formula.sh`:

**BEFORE:**
```bash
release_data=$(get_latest_release) || true  # ❌ HIDES ERRORS
```

**AFTER:**
```bash
if release_data=$(get_latest_release); then
    : # Continue
else
    release_data=""
fi
```

Or even simpler with `set -euo pipefail` already active:
```bash
release_data=$(get_latest_release)  # Let error propagate naturally
```

**Why it matters:** Without `|| true`, if `get_latest_release()` fails, the script immediately knows and can report the actual error instead of silently continuing with empty data.

---

### Fix 2: Increase Wait Time (HIGH PRIORITY)

**Line 715** in `.github/workflows/release.yml`:

**BEFORE:**
```yaml
sleep 300  # 5 minutes - NOT ENOUGH TIME
```

**AFTER:**
```yaml
sleep 900  # 15 minutes - realistic timeline
```

**Timeline breakdown:**
- Concurrent build jobs: 3-5 minutes (they run in parallel)
- Asset uploads: 2-5 minutes (GitHub's CDN can be slow)
- CDN propagation: 1-2 minutes (API sees assets but they're not everywhere yet)
- Safety margin: 2-3 minutes
- **Minimum realistic:** 8-10 minutes before Homebrew job should run

5 minutes is too aggressive and likely catches assets mid-upload.

---

### Fix 3: Exponential Backoff (MEDIUM PRIORITY)

**Lines 376-428** in `scripts/update-homebrew-formula.sh`:

**BEFORE:**
```bash
while [ $retry_count -lt $max_retries ]; do
    release_data=$(get_latest_release) || true  # ❌
    # ... validation ...
    sleep 30  # Fixed 30-second delay each time
done
```

**AFTER:**
```bash
while [ $retry_count -lt $max_retries ]; do
    if release_data=$(get_latest_release); then
        # Validate...
        if [ valid ]; then break; fi
    else
        release_data=""
    fi

    # Exponential backoff: 30s, 60s, 120s, 240s (capped at 300s)
    retry_count=$((retry_count + 1))
    if [ $retry_count -lt $max_retries ]; then
        backoff_delay=$((30 * (2 ** (retry_count - 1))))
        [ $backoff_delay -gt 300 ] && backoff_delay=300
        sleep $backoff_delay
    fi
done
```

**Why it matters:** If assets are slowly uploading, exponential backoff gives the system more time to finish while still being aggressive early on.

---

## Detailed Explanation: Why `|| true` Is Dangerous Here

The `|| true` operator is placed at the **wrong level**:

```bash
# WRONG - Error suppressed at call site
release_data=$(get_latest_release) || true
# If get_latest_release fails:
#   - Function returns 1 (from subshell)
#   - Subshell exits
#   - || true makes the command succeed anyway
#   - release_data is EMPTY or INCOMPLETE
#   - Script continues as if nothing happened!

# RIGHT - Error handled inside function or propagated properly
release_data=$(get_latest_release)
# If get_latest_release fails:
#   - Function returns 1
#   - set -e causes script to exit with error
#   - OR we can check: if ! release_data=$(...); then handle_error; fi
```

The function `get_latest_release()` has good error handling internally (lines 98-106), but the calling code nullifies it with `|| true`.

---

## Files to Change

### 1. `/scripts/update-homebrew-formula.sh` (Primary fix)

**Changes needed:**
- Line 380-382: Remove `|| true`, add proper error handling
- Lines 99, 158: Verify JSON validation in functions (already present, good)
- Lines 376-428: Replace fixed-delay retry with exponential backoff

**Use the provided** `update-homebrew-formula-FIXED.sh` **as reference**

### 2. `.github/workflows/release.yml` (Timing fix)

**Changes needed:**
- Line 715: Change `sleep 300` to `sleep 900`

---

## How to Apply Fixes

### Quick Apply (Copy-paste the fixed script)

```bash
# Backup current version
cp scripts/update-homebrew-formula.sh scripts/update-homebrew-formula.sh.backup

# Use the fixed version
cp update-homebrew-formula-FIXED.sh scripts/update-homebrew-formula.sh

# Update workflow
# Edit .github/workflows/release.yml line 715: sleep 300 → sleep 900

# Test with a release tag
git tag v0.7.15-test
git push origin v0.7.15-test
```

### Manual Apply (If you want to make specific changes)

See `HOMEBREW_FIXES.md` for step-by-step instructions with code samples.

---

## What Success Looks Like

After applying fixes, the GitHub Actions logs should show:

```
[INFO] Release fetch attempt 1/10
[SUCCESS] Found release with 12 assets. Proceeding...
[INFO] Found release: v0.7.15 (version: 0.7.15)
[SUCCESS] Found universal macOS desktop asset: howlerops-darwin-universal.tar.gz
[INFO] macOS Universal Binary URL: https://github.com/...
[SUCCESS] Universal Binary SHA256: abc123def456...
[SUCCESS] Homebrew formula update completed successfully!
```

NOT this:
```
[INFO] Release fetch attempt 1/10
[WARNING] DEBUG: release_data is EMPTY
[WARNING] Release not found or API returned invalid response
[INFO] Release fetch attempt 2/10
[WARNING] DEBUG: release_data is EMPTY
[INFO] Release fetch attempt 3/10
...
[ERROR] Failed to fetch release with assets after 10 attempts
```

---

## Testing

After applying the fixes:

```bash
# Create a test release
git tag v0.7.15-test-$(date +%s)
git push origin --tags

# Monitor the Homebrew job in GitHub Actions
# Check that it says:
# - "Found release with XX assets"
# - NOT "Release not found"
```

If it still fails, check the actual `gh api` error in the logs:
```bash
gh api repos/jbeck018/howlerops/releases/latest
gh release view v0.7.15-test --json assets
```

---

## Prevention

To prevent this in the future:

1. **Never use `|| true` to suppress errors silently**
   - Use it only for explicitly non-critical operations
   - Example: `grep "$pattern" file.txt || true` (OK - grep not found is expected)
   - Example: `build_optional_feature || true` (OK - optional feature)
   - NOT: `critical_api_call || true` (BAD - hides real errors)

2. **Always validate critical data before using it**
   - Check if response is empty
   - Validate JSON structure
   - Count expected items (assets, etc.)

3. **Test timing-sensitive workflows with delays**
   - If you use `sleep`, test with real-world timings
   - Add progress logging at each wait point
   - Make delays configurable for testing

4. **Log the actual error, not just symptoms**
   - Current: "Release not found or API returned invalid response"
   - Better: "API call failed: Certificate expired" or "API response was empty"

---

## Summary Checklist

- [ ] Review `HOMEBREW_ISSUE_ANALYSIS.md` for detailed explanation
- [ ] Review `HOMEBREW_FIXES.md` for implementation details
- [ ] Apply one of these:
  - [ ] Use the provided `update-homebrew-formula-FIXED.sh` directly, OR
  - [ ] Manually apply changes from `HOMEBREW_FIXES.md`
- [ ] Update `.github/workflows/release.yml` line 715: `sleep 300` → `sleep 900`
- [ ] Test with a release tag
- [ ] Verify Homebrew job succeeds
- [ ] Commit the fixed script

---

## Additional Resources

- `HOMEBREW_ISSUE_ANALYSIS.md` - Full technical analysis
- `HOMEBREW_FIXES.md` - Step-by-step implementation guide
- `update-homebrew-formula-FIXED.sh` - Ready-to-use fixed script

---

## Questions This Answers

**Q: Why is the release not found even though it exists?**
A: The `release_data` variable is empty because the API call failed, and the `|| true` operator hides that failure.

**Q: Why does it fail on all 10 retries?**
A: Because the core issue (error suppression) happens on every retry. The script never gets valid data.

**Q: Why does this happen specifically in GitHub Actions?**
A: The `gh CLI` in GitHub Actions may have authentication or token scope issues that don't happen locally.

**Q: Why is 5 minutes not enough?**
A: Realistic timeline for building, uploading, and CDN propagation is 8-15 minutes.

**Q: Why didn't removing `2>&1` fix it?**
A: Removing `2>&1` made stderr visible in logs (good), but the `|| true` still suppresses the exit code.

**Q: What should I do right now?**
A: Apply the three fixes in order: (1) Remove `|| true`, (2) Increase wait to 15 min, (3) Add exponential backoff.

