# golangci-lint v2.6.1 Configuration - Quick Summary

## What Was Fixed

**Problem:** Configuration used `govet.enable-all: true` which enabled 45 analyzers including 10 noisy/experimental ones, generating 30+ micro-optimization warnings.

**Solution:** Changed to use default 35 analyzers and explicitly disable 2 noisy ones:
```yaml
govet:
  disable:
    - fieldalignment  # Struct field ordering micro-optimization
    - shadow         # Variable shadowing - too opinionated
```

## Results

✅ **Before:** 30+ fieldalignment warnings about struct field ordering
✅ **After:** 0 fieldalignment warnings
✅ **Enabled:** 34 production-ready analyzers (down from 45)
✅ **Build Status:** Clean - no govet errors

## What Changed Between v1 and v2

### Breaking Changes
1. `check-shadowing: true` option removed → Use `enable: [shadow]` or `disable: [shadow]`
2. Configuration is more explicit - must use `enable-all`, `disable-all`, `enable`, or `disable`

### Available Analyzers (45 total)
- **Default Enabled (35):** Production-ready correctness checks
- **Disabled by Default (10):** Noisy, experimental, or opinionated checks

### Disabled Analyzers (10)
These are deliberately excluded from defaults:
```
atomicalign       # Memory alignment (rarely needed)
deepequalerrors   # Deep comparison of errors
fieldalignment    # Struct field ordering (micro-optimization)
findcall          # Find specific function calls
nilness           # Nil pointer analysis (high false positives)
reflectvaluecompare # Reflection comparisons
shadow            # Variable shadowing (style preference)
sortslice         # Sort slice usage
unusedwrite       # Unused write detection
httpmux           # HTTP multiplexer patterns (new in v2.1.0)
```

## Current Configuration Status

**Enabled Linters (5):**
- `govet` - 34 analyzers enabled
- `ineffassign` - Unused assignments
- `errcheck` - Unchecked errors
- `gosec` - Security issues
- `nilerr` - Nil error returns

**Enabled govet Analyzers (34):**
```
appends, asmdecl, assign, atomic, bools, buildtag, cgocall, composites,
copylocks, defers, directive, errorsas, framepointer, hostport, httpresponse,
ifaceassert, lostcancel, nilfunc, printf, shift, sigchanyzer, slog, stdmethods,
stdversion, stringintconv, structtag, testinggoroutine, tests, timeformat,
unmarshal, unreachable, unsafeptr, unusedresult, waitgroup
```

**Disabled govet Analyzers (2):**
- `fieldalignment` - Struct field ordering micro-optimization
- `shadow` - Variable shadowing (too opinionated)

## Best Practices

### 1. Use Defaults
The 35 default analyzers are well-tested and production-ready. Don't use `enable-all: true`.

### 2. Debug Configuration
```bash
# See what's enabled
GL_DEBUG=govet golangci-lint run --enable=govet
```

### 3. Minimal Linter Set
Your current set is excellent for production:
- govet (correctness)
- ineffassign (unused assignments)
- errcheck (unchecked errors)
- gosec (security)
- nilerr (nil error returns)

### 4. Optional Additions
Consider adding for better coverage:
- `staticcheck` - Comprehensive static analysis
- `unused` - Unused code detection
- `misspell` - Spelling errors
- `gofmt` - Code formatting
- `goimports` - Import organization

## Migration from v1

If you have v1 configs to migrate:
```bash
golangci-lint migrate --config .golangci.yml
```

## References

- Full Analysis: See `GOLANGCI_LINT_V2_ANALYSIS.md`
- [Official v2 Docs](https://golangci-lint.run/docs/configuration/)
- [Migration Guide](https://golangci-lint.run/docs/product/migration-guide/)
- [v2 Announcement](https://ldez.github.io/blog/2025/03/23/golangci-lint-v2/)

## Quick Commands

```bash
# Run linter
golangci-lint run

# Run with auto-fix
golangci-lint run --fix

# See what analyzers are enabled
GL_DEBUG=govet golangci-lint run --enable=govet

# Check specific path
golangci-lint run ./internal/...

# Generate JSON report
golangci-lint run --out-format=json > report.json
```
