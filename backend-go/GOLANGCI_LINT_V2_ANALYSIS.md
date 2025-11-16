# golangci-lint v2.6.1 Configuration Analysis and Recommendations

## Executive Summary

Your current `.golangci.yml` configuration is mostly correct for v2.6.1, but `govet.enable-all: true` is enabling noisy analyzers like `fieldalignment` that are not recommended for production code. This document provides the fixes and best practices.

## Key Findings

### 1. What Changed in govet Configuration (v1 → v2)

**Breaking Changes:**
- The old `check-shadowing: true` option was deprecated in v1.57.0 and removed in v2
- Must now use `linters.settings.govet.enable: [shadow]` or `disable: [shadow]` instead
- Configuration is now more explicit - you control analyzers via `enable-all`, `disable-all`, `enable`, or `disable` lists

**Available Analyzers in v2.6.1 (45 total):**
```
appends, asmdecl, assign, atomic, atomicalign, bools, buildtag, cgocall,
composites, copylocks, deepequalerrors, defers, directive, errorsas,
fieldalignment, findcall, framepointer, hostport, httpmux, httpresponse,
ifaceassert, loopclosure, lostcancel, nilfunc, nilness, printf,
reflectvaluecompare, shadow, shift, sigchanyzer, slog, sortslice, stdmethods,
stdversion, stringintconv, structtag, testinggoroutine, tests, timeformat,
unmarshal, unreachable, unsafeptr, unusedresult, unusedwrite, waitgroup
```

**Default Enabled (35 analyzers):**
```
appends, asmdecl, assign, atomic, bools, buildtag, cgocall, composites,
copylocks, defers, directive, errorsas, framepointer, hostport, httpresponse,
ifaceassert, loopclosure, lostcancel, nilfunc, printf, shift, sigchanyzer,
slog, stdmethods, stdversion, stringintconv, structtag, testinggoroutine,
tests, timeformat, unmarshal, unreachable, unsafeptr, unusedresult, waitgroup
```

**Excluded from Default (10 analyzers):**
These are deliberately disabled by default because they're noisy or opinionated:
```
atomicalign       # Memory alignment (rarely needed)
deepequalerrors   # Deep comparison of errors (usually not an issue)
fieldalignment    # Struct field ordering optimization (micro-optimization)
findcall          # Find specific function calls (too specialized)
nilness           # Nil pointer analysis (high false positive rate)
reflectvaluecompare # Reflection comparisons (edge case)
shadow            # Variable shadowing (opinionated style choice)
sortslice         # Sort slice usage (specialized)
unusedwrite       # Unused write detection (experimental)
httpmux           # HTTP multiplexer patterns (new in v2.1.0)
```

### 2. The Problem with `enable-all: true`

Your config uses `govet.enable-all: true`, which enables ALL 45 analyzers including the 10 noisy ones. This is why you're seeing numerous `fieldalignment` warnings about struct field ordering.

**Impact:**
- `fieldalignment` alone generates 30+ warnings in your codebase
- These are micro-optimizations (saving 8-16 bytes per struct)
- Not worth the code churn for most applications
- Distracts from real issues

### 3. Recommended Configuration

**Option A: Use Defaults (Recommended for Most Projects)**
```yaml
linters:
  settings:
    govet:
      # Use the 35 default analyzers - well-balanced set
      # This is the same as not configuring govet at all
```

**Option B: Explicit Control (Recommended for Your Case)**
```yaml
linters:
  settings:
    govet:
      # Start with all analyzers disabled
      disable-all: true

      # Enable specific analyzers you want
      enable:
        # Core correctness checks (highly recommended)
        - appends          # Incorrect use of append
        - assign           # Useless assignments
        - atomic           # Atomic operation misuse
        - bools            # Boolean mistakes
        - buildtag         # Build tag errors
        - composites       # Unkeyed composite literals
        - copylocks        # Copying locks
        - defers           # Defer mistakes
        - directive        # Go directive comments
        - errorsas         # errors.As usage
        - httpresponse     # HTTP response mistakes
        - loopclosure      # Loop variable capture
        - lostcancel       # Context cancel not called
        - nilfunc          # Nil function comparison
        - printf           # Printf format strings
        - shift            # Shift operation errors
        - stdmethods       # Standard method signatures
        - structtag        # Struct tag errors
        - tests            # Test function errors
        - unmarshal        # Unmarshal mistakes
        - unreachable      # Unreachable code
        - unsafeptr        # Unsafe pointer usage
        - unusedresult     # Unused results

        # Additional useful checks
        - cgocall          # CGo mistakes (if you use CGo)
        - framepointer     # Frame pointer issues
        - hostport         # Host:port formatting
        - ifaceassert      # Interface assertion mistakes
        - sigchanyzer      # Signal channel issues
        - slog             # Structured logging errors
        - stdversion       # Go version-specific issues
        - stringintconv    # String/int conversion issues
        - testinggoroutine # Testing goroutine issues
        - timeformat       # Time format issues
        - waitgroup        # WaitGroup usage errors

        # Optional: Enable if you want stricter checking
        # - shadow         # Variable shadowing (style preference)
        # - nilness        # Nil pointer checks (high false positives)
```

**Option C: Use Defaults + Disable Specific (Simplest)**
```yaml
linters:
  settings:
    govet:
      # Use defaults but explicitly document what we're NOT checking
      disable:
        - fieldalignment    # Micro-optimization, not worth it
        - shadow           # Style preference, too opinionated
```

### 4. Corrected Configuration for Your Project

Based on your minimal linter philosophy (govet, ineffassign, errcheck, gosec, nilerr), here's the recommended config:

```yaml
version: "2"

run:
  timeout: 5m
  modules-download-mode: readonly
  allow-parallel-runners: true

linters:
  enable:
    - govet
    - ineffassign
    - errcheck
    - gosec
    - nilerr

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
        - (*compress/gzip.Writer).Close
        - (*compress/gzip.Reader).Close
        - (*encoding/json.Encoder).Encode
        - (net/http.ResponseWriter).Write
        - (*net/http.Response.Body).Read
        - (*database/sql.Row).Scan
        - encoding/json.Unmarshal
        - os.Setenv
        - crypto/rand.Read
        - (*github.com/sirupsen/logrus.Entry).Log
        - (*github.com/sirupsen/logrus.Logger).Log

    govet:
      # Use default analyzers (35 well-balanced checks)
      # Explicitly disable noisy micro-optimizations
      disable:
        - fieldalignment  # Struct field ordering - micro-optimization
        - shadow         # Variable shadowing - too opinionated

    gosec:
      excludes:
        - G104  # Audit errors not checked (covered by errcheck)
        - G304  # File path from user input (intentional for file operations)
      severity: medium
      confidence: medium

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
        path: "_test\\.go"

severity:
  default: error
```

## Additional Best Practices for v2

### 1. Use Debug Mode to Inspect Configuration
```bash
# See what analyzers are enabled
GL_DEBUG=govet golangci-lint run --enable=govet

# See all available analyzers and their status
golangci-lint linters
```

### 2. Recommended Minimal Linter Set for Production

Your current set (govet, ineffassign, errcheck, gosec, nilerr) is excellent for production. Consider adding:

```yaml
linters:
  enable:
    - govet         # Go vet checks (correctness)
    - ineffassign   # Unused assignments
    - errcheck      # Unchecked errors
    - gosec         # Security issues
    - nilerr        # Nil error returns

    # Consider adding these for better coverage:
    - staticcheck   # Comprehensive static analysis (highly recommended)
    - unused        # Unused code detection
    - misspell      # Spelling errors in comments
    - gofmt         # Code formatting
    - goimports     # Import organization
```

### 3. CI/CD Configuration

For CI, use stricter settings:
```yaml
# .golangci-ci.yml (for CI only)
linters:
  settings:
    govet:
      # In CI, enable more checks
      disable:
        - fieldalignment  # Still skip micro-optimizations
```

### 4. Migration Command

If you update configurations, use the migration tool:
```bash
golangci-lint migrate --config .golangci.yml
```

## Summary of Changes Needed

**Immediate Fix:**
Replace `govet.enable-all: true` with either:
1. Nothing (use defaults), or
2. Explicit `disable: [fieldalignment, shadow]`

**Reasoning:**
- `enable-all: true` enables 10 experimental/noisy analyzers
- These create 30+ warnings for micro-optimizations
- Default 35 analyzers are well-tested and production-ready

**Impact:**
- Eliminates fieldalignment warnings
- Keeps all important correctness checks
- Aligns with your minimal linter philosophy

## Testing the New Configuration

```bash
# 1. Update .golangci.yml with recommended config
# 2. Run linter to see clean output
golangci-lint run

# 3. Verify no fieldalignment warnings
golangci-lint run 2>&1 | grep -i fieldalignment

# 4. Check what's actually enabled
GL_DEBUG=govet golangci-lint run --enable=govet 2>&1 | grep "Enabled by config"
```

## References

- [golangci-lint v2 Configuration Docs](https://golangci-lint.run/docs/configuration/)
- [golangci-lint v2 Migration Guide](https://golangci-lint.run/docs/product/migration-guide/)
- [govet Analyzer Settings](https://golangci-lint.run/docs/linters/configuration/)
- [golangci-lint v2 Announcement](https://ldez.github.io/blog/2025/03/23/golangci-lint-v2/)
