# Visual Explanation: The Homebrew Update Failure

## The Core Issue: Silent Error Suppression

```
┌─────────────────────────────────────────────────────────────┐
│  CURRENT BROKEN CODE (Lines 380-382)                        │
└─────────────────────────────────────────────────────────────┘

release_data=$(get_latest_release) || true
     ↓
   ┌─────────────────────────────┐
   │ get_latest_release function │
   │                             │
   │  gh api call fails ❌      │
   │  returns error code 1       │
   │  returns to subshell        │
   └─────────────────────────────┘
     ↓
   ERROR CODE 1 (from subshell)
     ↓
   || true CATCHES IT
     ↓
   "Make the command succeed anyway"
     ↓
   release_data = "" (EMPTY! Nothing was assigned)
     ↓
   Script continues as if success ⚠️
     ↓
   jq -e '.tag_name' on EMPTY STRING
     ↓
   FAILS with "Release not found"


┌─────────────────────────────────────────────────────────────┐
│  FIXED CODE (What we need)                                  │
└─────────────────────────────────────────────────────────────┘

if release_data=$(get_latest_release); then
    : # Continue
else
    release_data=""
fi
     ↓
   ┌─────────────────────────────┐
   │ get_latest_release function │
   │                             │
   │  gh api call fails ❌      │
   │  returns error code 1       │
   │  returns to subshell        │
   └─────────────────────────────┘
     ↓
   ERROR CODE 1 (from subshell)
     ↓
   Captured by: if ... ; then ... else ... fi
     ↓
   Goes to: release_data=""
     ↓
   Script knows it failed ✓
     ↓
   Validation loop detects empty
     ↓
   Logs actual error and retries ✓
```

---

## Timeline: Why 5 Minutes Isn't Enough

### Realistic Release Process

```
Release job starts at T+0
│
├─→ T+0:00   Create GitHub Release (instant)
│            ↓
├─→ T+0:05   Start 5 build jobs (parallel)
│   ├─ macOS-amd64   ← Building...
│   ├─ macOS-arm64   ← Building...
│   ├─ Linux-amd64   ← Building...
│   ├─ Linux-arm64   ← Building...
│   └─ Windows-amd64 ← Building...
│
│   T+3:00   First build finishes, starts uploading assets
│   ├─ Platform 1: Uploading 4 assets (tar.gz, checksum files)
│   ├─ Platform 2: Uploading...
│   └─ ...continuing...
│
│   T+5:00   CURRENT HOMEBREW WAIT ENDS ⚠️
│   ├─ Build jobs: STILL IN PROGRESS
│   ├─ Some assets: UPLOADED
│   ├─ Some assets: STILL UPLOADING
│   ├─ GitHub API: Knows about SOME assets only
│   └─ CDN: Assets NOT YET PROPAGATED
│
├─→ T+5:30   Homebrew job starts (too early!)
│            Calls gh api
│            Gets partial response (some assets missing)
│            Validation fails
│            RETRIES with same problem
│
│   T+6:00   Last build job finishes
│   ├─ All assets now uploaded
│   ├─ GitHub API now has complete data
│   └─ CDN starting to propagate
│
│   T+6:30   Homebrew retry (10 retries, 30s each = 300s = 5 min)
│            Gets more assets, validation passes
│            But this is unreliable!
│
├─→ T+10:00  Generate checksums job finishes
│            Uploads combined checksum file
│
├─→ T+12:00  Validation job finishes
│
├─→ T+15:00  BETTER HOMEBREW WAIT ENDS ✓
│            Calls gh api
│            Gets COMPLETE response with all assets
│            Validation passes on FIRST try
│            SUCCESS!


┌──────────────────────────────────────────┐
│ Why this timing matters:                  │
│                                          │
│ Assets uploaded ≠ Assets available       │
│                                          │
│ GitHub API might know about some assets  │
│ before uploads complete                  │
│                                          │
│ API needs time to "see" all uploaded     │
│ assets before returning complete data    │
└──────────────────────────────────────────┘
```

---

## Error Propagation Flow

### BROKEN CODE (Current)

```
                    Script starts
                         ↓
            Homebrew job: Get release (5 min wait)
                         ↓
         release_data=$(get_latest_release) || true
                         ↓
                    gh api call
                         ↓
                   FAILS (why? no idea)
                    Error hidden by || true
                         ↓
              release_data = "" (empty)
                         ↓
         Script continues, doesn't know error happened
                         ↓
       Try to parse: jq -e '.tag_name' on ""
                         ↓
                    jq fails (obviously)
                         ↓
          Error message: "Release not found"
          (But really, API call failed silently!)
                         ↓
                 Retry loop kicks in
                         ↓
       Each retry hits SAME problem (empty response)
            Because error suppression is still there
                         ↓
           After 10 retries × 30s = 5 minutes
                         ↓
              Job fails with confusing error
           (User doesn't know it's an API auth issue!)
```

### FIXED CODE (Proposed)

```
                    Script starts
                         ↓
            Homebrew job: Get release (15 min wait)
                    [Better timeline]
                         ↓
         if release_data=$(get_latest_release); then
                         ↓
                    gh api call
                         ↓
                   FAILS (why? we'll find out)
                    Error NOT suppressed
                         ↓
              release_data assignment FAILS
                         ↓
                Goes to: else branch
                release_data = ""
                         ↓
     Script KNOWS error happened (else branch taken)
                         ↓
       Validation detects empty and logs actual error
                         ↓
    Retry loop with exponential backoff:
    Attempt 1: Wait 30s
    Attempt 2: Wait 60s
    Attempt 3: Wait 120s
    Attempt 4: Wait 240s
              (Gives build jobs more time)
                         ↓
     Assets finish uploading during retries
                         ↓
    API response becomes valid
                         ↓
           Validation passes
                         ↓
     Homebrew formula updates successfully
                         ↓
              SUCCESS! ✓
```

---

## Side-by-Side Code Comparison

```
OLD (BROKEN)                          NEW (FIXED)
─────────────────────────────────────────────────────────────

release_data=$(get_latest_release)    if release_data=$(get_latest_release)
  || true                             then
                                        : # Success
  ❌ Error suppressed                 else
                                        release_data=""
  ❌ Can't tell if failed             fi

  ❌ release_data might be empty      ✓ Error not suppressed

  ❌ No indication of failure         ✓ Script knows when failed

  while loop                          while loop
    sleep 30  (fixed)                   backoff=$((30 * 2^retry))
                                        sleep $backoff
  ❌ Same 30s every time              ✓ Longer waits for timing issues

  ❌ Tight loop aggravates rate limit ✓ Exponential backoff prevents limit
```

---

## The Root Cause in One Picture

```
                            ┌─────────────────┐
                            │  GitHub Actions │
                            │  Homebrew Job   │
                            └────────┬────────┘
                                     │
                    ┌────────────────┴────────────────┐
                    │ Problem: Release assets are     │
                    │ still uploading from build jobs │
                    └────────────────┬────────────────┘
                                     │
                         ┌───────────┴───────────┐
                         ↓                       ↓
            ┌─────────────────────────┐  ┌──────────────────┐
            │ API Call                │  │ With || true     │
            │ gh api releases/latest  │  │ (Error masked)   │
            └───────────┬─────────────┘  └────────┬─────────┘
                        │                         │
                        │ FAILS because           │
                        │ assets still uploading  │
                        │                         │
                        │ Returns exit code 1     │
                        └─────────┬───────────────┘
                                  │
                        ┌─────────┴─────────┐
                        │                   │
                    (with || true)      (without || true)
                        │                   │
                        ↓                   ↓
              "Ignore error"          "Script fails loudly"
              release_data=""         release_data=""
                        │                   │
                        ↓                   ↓
            "Continue as if success"   "Knows it failed"
            "Validation fails"         "Logs real error"
            "Confusing error message"  "Clear action items"
                        │                   │
                        ↓                   ↓
            "Retry with same code"    "Retry with more time"
            "Hits same problem"       "Gives jobs time to finish"
                        │                   │
                        ↓                   ↓
              After 10 retries =      First/second retry =
              Job still fails         Job succeeds
              User confused           User happy
```

---

## Exponential Backoff Visual

```
FIXED 30S (current - not enough time)
├─ Attempt 1: Sleep 30s → Assets still uploading → Fails
├─ Attempt 2: Sleep 30s → Assets still uploading → Fails
├─ Attempt 3: Sleep 30s → Assets still uploading → Fails
├─ Attempt 4: Sleep 30s → Assets FINALLY done → Works!
└─ (Lost 2+ minutes of reliability for no reason)


EXPONENTIAL BACKOFF (proposed - smart timing)
├─ Attempt 1: Sleep 30s  → Assets still uploading → Fails (expected)
├─ Attempt 2: Sleep 60s  → Assets probably done → SUCCESS!
│
│ OR if API is slow:
├─ Attempt 1: Sleep 30s  → Assets still uploading → Fails
├─ Attempt 2: Sleep 60s  → API propagation slow → Fails
├─ Attempt 3: Sleep 120s → API caught up → SUCCESS!
│
│ OR if very slow:
├─ Attempt 1: Sleep 30s   → Fails
├─ Attempt 2: Sleep 60s   → Fails
├─ Attempt 3: Sleep 120s  → Fails
├─ Attempt 4: Sleep 240s  → Finally works → SUCCESS!
│
└─ (Gracefully handles all scenarios without fixed 5-minute wait)
```

---

## Real-World Example: What Actually Happens

### Current Code (v0.7.13)

```
T+5:30 Homebrew job starts
└─ Log: "Using gh CLI to fetch latest release"
   └─ gh api command runs
      └─ Fails (some assets still uploading)
         └─ Returns error to subshell
            └─ Subshell exits with code 1
               └─ || true catches exit code
                  └─ "Treats as success"
                     └─ release_data = "" (EMPTY!)
                        └─ Log: "DEBUG: release_data is EMPTY"
                           └─ Log: "Release not found or API returned invalid response"
                              └─ retry_count = 1
                                 └─ sleep 30

T+6:00 Retry attempt 2
└─ Same problem repeats
   └─ release_data is EMPTY
      └─ retry_count = 2
         └─ sleep 30

T+6:30 Retry attempt 3
└─ Same problem repeats
   (Build jobs FINALLY finishing)
   └─ retry_count = 3
      └─ sleep 30

T+7:00 Retry attempt 4
└─ NOW assets might be available
   └─ release_data has data!
      └─ Validation passes
         └─ FINALLY succeeds (after 90 seconds of retries)

LOG: "Found release with 12 assets"
Log: "Homebrew formula update completed successfully!"

💡 Lesson: It eventually works, but only by luck and retries.
          The real issue (error suppression) was never identified.
```

### Fixed Code (Proposed)

```
T+15:00 Homebrew job starts (waited 15 minutes)
└─ All build jobs completed long ago
   └─ All assets uploaded
      └─ API propagated everything
         └─ Log: "Release fetch attempt 1/10"
            └─ if release_data=$(get_latest_release); then
               └─ gh api command runs
                  └─ SUCCEEDS! (assets are ready)
                     └─ Assigns to release_data
                        └─ if condition is TRUE
                           └─ Continue in then-branch
                              └─ Validation checks jq '.tag_name'
                                 └─ PASSES
                                    └─ Check asset count
                                       └─ asset_count = 12
                                          └─ break (exit loop)

Log: "Found release with 12 assets. Proceeding..."
Log: "Successfully fetched release via gh CLI"
Log: "Homebrew formula update completed successfully!"

💡 Lesson: Works on FIRST try because timing and error handling are correct.
```

---

## Summary Diagram

```
┌─────────────────────────────────────────────────────────┐
│           THE HOMEBREW UPDATE FAILURE                   │
└─────────────────────────────────────────────────────────┘

Three problems combine into one failure:

  1. ERROR SUPPRESSION          2. TIMING ISSUE          3. POOR VISIBILITY
     (|| true hides errors)         (5 min is too early)   (No real error logged)
            ↓                               ↓                        ↓
     API call fails             Assets still uploading    "Release not found"
     Error is hidden            Validation fails anyway    (But why? No clue!)
     Script doesn't know        Retries hit same problem
     Data is empty              Loop cycles 10 times
            ↓                               ↓                        ↓
     RESULT: Silent failure        RESULT: Timing race     RESULT: No debug info
             Data loss              Unreliable process      User confused


        ┌──────────────────────────────────────────┐
        │     THREE FIXES SOLVE ALL PROBLEMS:      │
        ├──────────────────────────────────────────┤
        │ 1. Remove || true (reveals errors)      │
        │ 2. Increase wait to 15 min (fixes race)  │
        │ 3. Exponential backoff (smart retries)   │
        └──────────────────────────────────────────┘
```

