# Recommended Additional Linters for Production Go

## Current Minimal Set (Excellent Foundation)

Your current configuration uses 5 essential linters:
```yaml
linters:
  enable:
    - govet       # Correctness checks (34 analyzers)
    - ineffassign # Unused assignments
    - errcheck    # Unchecked errors
    - gosec       # Security issues
    - nilerr      # Nil error returns
```

This is a solid, minimal set that catches the most critical issues without excessive noise.

## Recommended Additions (Priority Order)

### Tier 1: High-Value, Low-Noise (Strongly Recommended)

These linters provide significant value with minimal false positives:

```yaml
linters:
  enable:
    # Current linters...
    - govet
    - ineffassign
    - errcheck
    - gosec
    - nilerr

    # Tier 1 additions
    - staticcheck   # Comprehensive static analysis - the gold standard
    - unused        # Detects unused code (functions, variables, constants)
    - gofmt         # Code formatting consistency
    - goimports     # Import statement organization
```

**Why these?**
- `staticcheck`: The most comprehensive Go static analyzer - catches bugs, performance issues, and style problems
- `unused`: Helps keep codebase clean by finding dead code
- `gofmt`: Ensures consistent formatting (should always pass if you use gofmt)
- `goimports`: Keeps imports organized and removes unused ones

### Tier 2: Additional Quality Checks (Consider for CI)

These add value but may require some configuration tuning:

```yaml
linters:
  enable:
    # Tier 1 + Tier 2
    - misspell      # Spelling errors in comments and strings
    - unconvert     # Unnecessary type conversions
    - unparam       # Unused function parameters
    - wastedassign  # Wasted assignments
    - prealloc      # Slice preallocation opportunities
```

**Why these?**
- `misspell`: Catches typos in documentation and error messages
- `unconvert`: Finds redundant type conversions (code smell)
- `unparam`: Identifies unused parameters (potential API cleanup)
- `wastedassign`: Catches assignments that are immediately overwritten
- `prealloc`: Suggests preallocating slices for better performance

### Tier 3: Opinionated/Style Checks (Optional)

These enforce specific coding styles - use only if your team agrees:

```yaml
linters:
  enable:
    # Tier 1 + Tier 2 + Tier 3
    - revive        # Fast, flexible linter (replacement for golint)
    - gocyclo       # Cyclomatic complexity
    - dupl          # Code duplication detection
    - goconst       # Repeated strings that could be constants
    - gocognit      # Cognitive complexity
```

**Why these?**
- `revive`: Configurable linter for Go style and conventions
- `gocyclo`: Warns about overly complex functions
- `dupl`: Finds duplicated code blocks
- `goconst`: Suggests extracting repeated strings to constants
- `gocognit`: Measures how hard code is to understand

## Recommended Progressive Adoption

### Step 1: Add Tier 1 (Immediate - No Downsides)

```yaml
linters:
  enable:
    - govet
    - ineffassign
    - errcheck
    - gosec
    - nilerr
    - staticcheck    # Add this first - it's the most valuable
    - unused         # Add this second - keeps code clean
    - gofmt          # Should already pass if you use gofmt
    - goimports      # Should already pass if you use goimports
```

**Expected impact:**
- ~5-10 new issues from staticcheck (real bugs and improvements)
- ~2-5 new issues from unused (dead code cleanup)
- 0 issues from gofmt/goimports if you already format

### Step 2: Add Tier 2 in CI Only (Low Risk)

Create `.golangci-ci.yml` for CI with stricter checks:

```yaml
# .golangci-ci.yml (for CI only)
version: "2"

run:
  timeout: 5m

linters:
  enable:
    # All Tier 1 linters
    - govet
    - ineffassign
    - errcheck
    - gosec
    - nilerr
    - staticcheck
    - unused
    - gofmt
    - goimports

    # Add Tier 2 in CI
    - misspell
    - unconvert
    - unparam
    - wastedassign

  settings:
    govet:
      disable:
        - fieldalignment
        - shadow
```

**CI command:**
```bash
golangci-lint run --config .golangci-ci.yml
```

### Step 3: Evaluate Tier 3 (Team Decision)

These are more opinionated. Try them on a branch first:

```bash
# Test a linter before committing to it
golangci-lint run --enable=revive
golangci-lint run --enable=gocyclo --max-complexity=15
golangci-lint run --enable=dupl
```

## Complete Example Configuration

Here's a comprehensive but balanced configuration:

```yaml
version: "2"

run:
  timeout: 5m
  modules-download-mode: readonly
  allow-parallel-runners: true

linters:
  enable:
    # Core correctness
    - govet
    - ineffassign
    - errcheck
    - gosec
    - nilerr

    # Comprehensive static analysis
    - staticcheck
    - unused

    # Code formatting
    - gofmt
    - goimports

    # Quality improvements
    - misspell
    - unconvert
    - unparam
    - wastedassign

  settings:
    errcheck:
      check-type-assertions: false
      check-blank: false
      exclude-functions:
        - (io.Closer).Close
        - (*database/sql.Rows).Close
        - (*database/sql.Stmt).Close
        - (*database/sql.DB).Close
        - (*database/sql.Tx).Rollback
        - (*net/http.Response.Body).Close
        - (*os.File).Close

    govet:
      disable:
        - fieldalignment
        - shadow

    gosec:
      excludes:
        - G104  # Covered by errcheck
        - G304  # Intentional file operations
      severity: medium
      confidence: medium

    staticcheck:
      # Enable all checks except experimental ones
      checks: ["all", "-ST1000", "-ST1003"]

    unparam:
      # Don't report issues in test files
      check-exported: false

  exclusions:
    generated: lax
    presets:
      - comments
      - common-false-positives
      - legacy
      - std-error-handling
    paths:
      - vendor
      - third_party
      - testdata
      - examples
      - ".github"
    rules:
      - linters:
          - errcheck
          - gosec
          - goconst
          - unparam
        path: "_test\\.go"

severity:
  default: error
```

## Linter Performance Impact

Estimated run times (for a medium-sized project):

| Linter | Speed | Impact |
|--------|-------|--------|
| govet | Fast | < 1s |
| ineffassign | Fast | < 1s |
| errcheck | Fast | < 1s |
| gosec | Fast | < 1s |
| nilerr | Fast | < 1s |
| **staticcheck** | **Medium** | **2-5s** |
| unused | Medium | 1-3s |
| gofmt | Fast | < 1s |
| goimports | Fast | < 1s |
| misspell | Fast | < 1s |
| unconvert | Fast | < 1s |
| unparam | Medium | 1-2s |
| wastedassign | Fast | < 1s |
| revive | Fast | 1-2s |
| gocyclo | Fast | < 1s |
| dupl | Slow | 5-10s |

**Total time with Tier 1 + Tier 2:** ~10-15 seconds (acceptable for CI)

## Common Issues and Solutions

### Issue: Too Many Warnings

**Solution:** Enable linters progressively
```bash
# Fix issues from one linter at a time
golangci-lint run --enable=staticcheck --fix
golangci-lint run --enable=unused --fix
```

### Issue: False Positives

**Solution:** Use exclusions and disable rules
```yaml
linters:
  settings:
    staticcheck:
      checks: ["all", "-SA1019"]  # Disable specific check

  exclusions:
    rules:
      - linters: [staticcheck]
        text: "SA1029"  # Disable specific warning by text
```

### Issue: Slow CI

**Solution:** Use fast linters in pre-commit, comprehensive in CI
```yaml
# .golangci-fast.yml (pre-commit)
linters:
  enable:
    - govet
    - errcheck
    - gosec
    - gofmt
```

## Recommended Workflow

1. **Local development:** Use minimal set (current config)
2. **Pre-commit hook:** Add gofmt, goimports, staticcheck
3. **CI pipeline:** Run comprehensive checks (Tier 1 + Tier 2)
4. **Weekly reports:** Run expensive linters (dupl, etc.) as reports

## Next Steps

1. **Immediate:** Keep current minimal config (it's excellent)
2. **This week:** Add staticcheck - it will find real bugs
3. **This month:** Add unused, gofmt, goimports
4. **Next quarter:** Evaluate Tier 2 linters in CI

## References

- [Awesome Go Linters](https://github.com/golangci/awesome-go-linters)
- [staticcheck Documentation](https://staticcheck.io/docs/)
- [golangci-lint Linters List](https://golangci-lint.run/docs/linters/)
- [Golden Config Example](https://gist.github.com/maratori/47a4d00457a92aa426dbd48a18776322)
