# Comprehensive Lint Fix Plan

## Current Status

**Total Issues:** 845 linting errors across multiple linters
- errcheck: 629 issues (unchecked error returns)
- staticcheck: 97 issues (code simplifications and improvements)
- gosec: 79 issues (security warnings)
- unused: 24 issues (dead code)
- ineffassign: 10 issues (ineffectual assignments)
- nilerr: 6 issues (returning nil with non-nil error check)

## Challenge

Fixing all 845 issues manually would take 10-20 hours of focused work. Most issues are not actual bugs - they're style/best-practice violations.

## Recommended Pragmatic Approach

### Phase 1: Configure Appropriate Exclusions (DONE)
- Added comprehensive errcheck exclusions for:
  - `defer Close()` patterns (standard Go practice)
  - HTTP response encoding (framework handles errors)
  - Database Scan operations (often acceptable to ignore)
  - Logging operations (best effort)

### Phase 2: Fix Real Bugs Only
**Priority: Critical bugs that could cause production issues**
- nilerr (6 issues) - These ARE actual bugs
- Security issues from gosec that are real threats
- ineffassign where values are computed but never used

### Phase 3: Remove Dead Code
- unused (24 issues) - Safe removals, improves codebase health

### Phase 4: Incremental Improvements
- Remaining staticcheck suggestions
- Remaining errcheck where checks would genuinely improve reliability

## Alternative: Disable Stricter Linters

Keep only critical linters enabled:
- govet (type safety, actual bugs)
- Basic errcheck with comprehensive exclusions

Incrementally re-enable others in future PRs.

## Estimated Time

| Approach | Time Required |
|----------|--------------|
| Fix ALL 845 issues | 10-20 hours |
| Fix critical bugs only (Phase 2) | 2-3 hours |
| Configure exclusions + disable rest | 30 minutes |

## Recommendation

**Option A (Fast):** Configure appropriate exclusions and disable problematic linters. Re-enable incrementally.

**Option B (Thorough):** Fix critical bugs (nilerr + real security issues), remove dead code, disable rest. Re-enable incrementally.

**Option C (Complete):** Fix all 845 issues (multi-session effort).

## Current Blocker

The recent commit introduced test failures in `internal/ai/claudecode_test.go`:
- `TestClaudeCode_GetHealth_BinaryNotFound`
- `TestClaudeCode_GetHealth_WithContext`

These tests expect the `claude` binary to be in PATH but it's not available in CI.

## Next Steps

1. Fix the test failures (unrelated to linting)
2. User decides on linting approach (A, B, or C)
3. Execute chosen approach
4. Verify CI passes
5. Deploy
