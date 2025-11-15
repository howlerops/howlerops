# Version and Auto-Update Guide

This document describes the version management and auto-update functionality for Howlerops CLI.

## Overview

Howlerops includes built-in version information and auto-update capabilities:

- **Version Command**: Display current version, commit, build date, and runtime info
- **Update Command**: Check for and install updates from GitHub releases
- **Auto-Update Check**: Optional background check on server startup (development only)

## Version Information

### Display Version

**CLI Tool:**
```bash
./sqlstudio version
```

**Server Binary:**
```bash
./sql-studio-backend --version
```

**Output:**
```
Howlerops v2.0.0
Commit: 86b8752
Built: 2025-10-23T22:46:03Z
Go: go1.24.5
OS/Arch: darwin/arm64
```

### Build-Time Variables

Version information is injected at build time using Go's `-ldflags`:

```bash
VERSION=2.0.0
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags="-X main.Version=$VERSION -X main.Commit=$GIT_COMMIT -X main.BuildDate=$BUILD_DATE"
```

The Makefile handles this automatically:

```bash
make build              # Build with version info
make build-cli          # Build CLI with version info
make release           # Build release version for current platform
make release-all       # Build for all platforms with archives
```

## Update Command

### Check for Updates

Check if a new version is available without installing:

```bash
./sqlstudio update --check
```

Output:
```
Checking for updates...
Current version: 2.0.0

New version available: 2.1.0
Current version: 2.0.0

Release Notes:
------------------------------------------------------------
## What's New
- New feature: Database migration support
- Bug fix: Fixed connection pool leak
- Performance: 30% faster query execution
------------------------------------------------------------

To install the update, run: sqlstudio update
```

### Install Update

Install the latest version:

```bash
./sqlstudio update
```

The command will:
1. Check GitHub API for latest release
2. Compare with current version
3. Display release notes
4. Prompt for confirmation
5. Download appropriate binary for your OS/Arch
6. Verify SHA256 checksum
7. Create backup of current binary
8. Replace with new binary
9. Clean up on success

**Force Update (skip confirmation):**
```bash
./sqlstudio update --force
```

### Update Process

```
Checking for updates...
Current version: 2.0.0

New version available: 2.1.0

Release Notes:
[... release notes ...]

Do you want to install this update? (y/N): y

Downloading update...
✓ Update installed successfully!
Updated from 2.0.0 to 2.1.0

Please restart the application for the changes to take effect.
```

## Auto-Update Check on Startup

The server binary automatically checks for updates on startup in **development mode only**:

```go
// In cmd/server/main.go
if !cfg.IsProduction() {
    go checkForUpdates(appLogger)
}
```

### How It Works

1. **Non-blocking**: Runs in background goroutine
2. **Rate-limited**: Only checks once per 24 hours
3. **Non-intrusive**: Shows log message if update available
4. **Safe**: 10-second timeout, fails silently
5. **Cached**: Stores last check time in `~/.sqlstudio/.last_update_check`

### Log Output

When update is available:
```
INFO New version available! Run 'sqlstudio update' to upgrade
  current_version=2.0.0
  latest_version=2.1.0
```

When no update:
```
DEBUG No updates available
```

On error:
```
DEBUG Failed to check for updates error="context deadline exceeded"
```

## GitHub Release Requirements

For auto-update to work, GitHub releases must follow this structure:

### Release Tag Format
```
v2.0.0
```

### Required Assets

For each platform, provide:
1. Binary file
2. SHA256 checksum file

**Binary naming convention:**
```
sql-studio-backend-{os}-{arch}[.exe]
```

**Checksum naming convention:**
```
sql-studio-backend-{os}-{arch}[.exe].sha256
```

### Example Release Assets

```
sql-studio-backend-darwin-amd64
sql-studio-backend-darwin-amd64.sha256
sql-studio-backend-darwin-arm64
sql-studio-backend-darwin-arm64.sha256
sql-studio-backend-linux-amd64
sql-studio-backend-linux-amd64.sha256
sql-studio-backend-linux-arm64
sql-studio-backend-linux-arm64.sha256
sql-studio-backend-windows-amd64.exe
sql-studio-backend-windows-amd64.exe.sha256
```

### Generating Checksums

The Makefile includes a `checksums` target:

```bash
make release-all    # Build all platforms
# Checksums are automatically generated in build/checksums.txt
```

**Manual generation:**
```bash
cd build
shasum -a 256 sql-studio-backend-* > checksums.txt
```

**Checksum file format:**
```
abc123def456...  sql-studio-backend-darwin-amd64
```

## Building for Release

### Single Platform

Build for current platform:
```bash
make release
```

### All Platforms

Build for all supported platforms:
```bash
make release-all VERSION=2.1.0
```

This creates:
- Binaries for all platforms
- `.tar.gz` archives for Unix
- `.zip` archive for Windows
- `checksums.txt` with SHA256 hashes

Output:
```
build/
  sql-studio-2.1.0-darwin-amd64.tar.gz
  sql-studio-2.1.0-darwin-arm64.tar.gz
  sql-studio-2.1.0-linux-amd64.tar.gz
  sql-studio-2.1.0-linux-arm64.tar.gz
  sql-studio-2.1.0-windows-amd64.zip
  checksums.txt
```

### Individual Platforms

```bash
make release-darwin-amd64   # macOS Intel
make release-darwin-arm64   # macOS Apple Silicon
make release-linux-amd64    # Linux AMD64
make release-linux-arm64    # Linux ARM64
make release-windows-amd64  # Windows AMD64
```

## Release Process

1. **Update VERSION in Makefile** (if needed)
   ```makefile
   VERSION ?= 2.1.0
   ```

2. **Build release artifacts:**
   ```bash
   make release-all VERSION=2.1.0
   ```

3. **Test binaries:**
   ```bash
   ./build/sql-studio-darwin-arm64 --version
   ```

4. **Create Git tag:**
   ```bash
   git tag -a v2.1.0 -m "Release v2.1.0"
   git push origin v2.1.0
   ```

5. **Create GitHub Release:**
   - Go to GitHub Releases
   - Create new release with tag `v2.1.0`
   - Upload all archives from `build/`
   - Add release notes
   - Publish release

6. **Verify auto-update:**
   ```bash
   ./sqlstudio update --check
   ```

## Configuration

### Config Directory

Auto-update stores metadata in:
```
~/.sqlstudio/
  .last_update_check    # Timestamp of last check
```

### Update Check Interval

Default: 24 hours (configurable in `pkg/updater/updater.go`):
```go
const UpdateCheckInterval = 24 * time.Hour
```

### HTTP Timeout

Default: 30 seconds (configurable):
```go
const DefaultTimeout = 30 * time.Second
```

## Error Handling

### Common Errors

**Network errors:**
```
Error: Failed to check for updates: Get "https://api.github.com/...": context deadline exceeded

Possible causes:
  - No internet connection
  - GitHub API rate limit exceeded
  - Network firewall blocking requests
```

**Permission errors:**
```
Error: Failed to install update: open /usr/local/bin/sqlstudio: permission denied

Possible causes:
  - Insufficient permissions (try running with sudo)
  - Binary is currently running (stop it first)
  - Installation method doesn't support auto-update (e.g., brew)
```

**Checksum mismatch:**
```
Error: checksum mismatch: expected abc123..., got def456...
```

**No binary for platform:**
```
Error: no binary found for platform linux/arm
```

## Security

### Checksum Verification

All downloads are verified with SHA256 checksums:

```go
actualChecksum := sha256.Sum256(binaryData)
if actualChecksumStr != expectedChecksum {
    return fmt.Errorf("checksum mismatch")
}
```

### Backup on Update

Before replacing binary, a backup is created:

```go
backupPath := currentExe + ".backup"
copyFile(currentExe, backupPath)
```

If update fails, the backup is restored:

```go
if err := os.Rename(tmpPath, currentExe); err != nil {
    os.Rename(backupPath, currentExe)  // Restore backup
    return err
}
```

## Package Structure

```
pkg/
├── version/
│   ├── version.go          # Version info struct and formatting
│   └── version_test.go     # Tests
└── updater/
    ├── updater.go          # Update checking and installation
    └── updater_test.go     # Tests (with mocked HTTP server)

cmd/
├── sqlstudio/
│   └── main.go             # CLI tool with version/update commands
└── server/
    └── main.go             # Server with version flag and auto-check
```

## Testing

### Run Tests

```bash
# Test version package
go test ./pkg/version -v

# Test updater package
go test ./pkg/updater -v

# Test with coverage
go test ./pkg/... -cover
```

### Manual Testing

**Test version command:**
```bash
make build-cli
./build/sqlstudio version
```

**Test update check (will fail if no internet or no newer version):**
```bash
./build/sqlstudio update --check
```

**Test with mock server:**
```bash
# Run tests which include HTTP mock server
go test ./pkg/updater -v -run TestCheckForUpdate
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Build release binaries
        run: |
          cd backend-go
          make release-all VERSION=${GITHUB_REF#refs/tags/v}

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            backend-go/build/*.tar.gz
            backend-go/build/*.zip
            backend-go/build/checksums.txt
```

## Troubleshooting

### Update check never runs

**Problem:** Server doesn't check for updates

**Solutions:**
- Ensure running in development mode (not production)
- Check logs for debug messages
- Delete `~/.sqlstudio/.last_update_check` to force check

### Update fails with permission denied

**Problem:** Cannot write to binary location

**Solutions:**
- Run with sudo: `sudo ./sqlstudio update`
- Copy binary to user-writable location
- Use package manager (brew, apt) instead

### GitHub API rate limit

**Problem:** Too many requests to GitHub API

**Solutions:**
- Wait for rate limit to reset (1 hour)
- Use authenticated requests (set GITHUB_TOKEN)
- Increase check interval

### Binary not found after update

**Problem:** Update succeeds but binary doesn't work

**Solutions:**
- Check file permissions: `chmod +x ./sqlstudio`
- Verify checksum: `shasum -a 256 ./sqlstudio`
- Restore from backup: `./sqlstudio.backup`

## Future Enhancements

Potential improvements for future versions:

1. **Delta Updates**: Download only binary diff instead of full binary
2. **Rollback Command**: Easy rollback to previous version
3. **Update Channels**: Support for stable/beta/alpha channels
4. **Auto-Apply Updates**: Optional automatic installation (with flag)
5. **Update Notifications**: Desktop notifications when update available
6. **Version History**: Show changelog between versions
7. **Pre-Update Hooks**: Run custom scripts before/after update
8. **Update Server**: Self-hosted update server (not just GitHub)

## References

- [Semantic Versioning](https://semver.org/)
- [GitHub Releases API](https://docs.github.com/en/rest/releases)
- [Go Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [SHA256 Checksums](https://en.wikipedia.org/wiki/SHA-2)
