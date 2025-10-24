# Version and Auto-Update Implementation Summary

## What Was Added

### 1. Version Package (`pkg/version`)
- **Purpose**: Centralized version information management
- **Files**:
  - `version.go`: Core version info struct and formatting
  - `version_test.go`: Comprehensive test coverage (100%)

**Key Features**:
- Version, Commit, BuildDate injected at build time via ldflags
- Runtime info: Go version, OS, Architecture
- Multiple output formats: String(), ShortString()

**Example Usage**:
```go
version.Version = "2.0.0"
version.Commit = "abc123"
version.BuildDate = "2025-10-23T10:00:00Z"

info := version.GetInfo()
fmt.Println(info.String())
// Output:
// SQL Studio v2.0.0
// Commit: abc123
// Built: 2025-10-23T10:00:00Z
// Go: go1.24.5
// OS/Arch: darwin/arm64
```

### 2. Updater Package (`pkg/updater`)
- **Purpose**: Check for and install updates from GitHub releases
- **Files**:
  - `updater.go`: Core update functionality
  - `updater_test.go`: Unit tests with HTTP mocks (50.8% coverage)

**Key Features**:
- GitHub API integration for release checking
- Semantic version comparison
- Binary download with SHA256 verification
- Safe binary replacement with automatic backup
- Rate limiting (24-hour check interval)
- Platform-specific binary selection
- Graceful error handling

**Example Usage**:
```go
u := updater.NewUpdater("~/.sqlstudio")

// Check for updates
info, err := u.CheckForUpdate(ctx)
if err != nil {
    log.Fatal(err)
}

if info.Available {
    fmt.Printf("New version: %s\n", info.LatestVersion)

    // Download and install
    err := u.DownloadUpdate(ctx, info)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 3. CLI Tool (`cmd/sqlstudio`)
- **Purpose**: Standalone CLI for version and update commands
- **File**: `main.go`

**Commands**:
```bash
# Show version
./sqlstudio version

# Check for updates (no install)
./sqlstudio update --check

# Install update
./sqlstudio update

# Force update (skip confirmation)
./sqlstudio update --force
```

### 4. Server Integration (`cmd/server`)
- **Purpose**: Add version flag and auto-update check to server
- **Changes**:
  - Added `--version` flag
  - Background update check on startup (dev mode only)
  - Non-intrusive notification if update available

**Features**:
- Version flag: `./sql-studio-backend --version`
- Auto-check runs once per 24 hours
- Only in development mode (not production)
- Non-blocking, fails silently
- 10-second timeout

### 5. Build System Updates
- **File**: `Makefile`

**New/Updated Targets**:
```makefile
# Build with version info
make build          # Build server
make build-cli      # Build CLI tool
make build-all      # Build both

# Release builds
make release                # Current platform
make release-all            # All platforms
make release-darwin-amd64   # macOS Intel
make release-darwin-arm64   # macOS Apple Silicon
make release-linux-amd64    # Linux AMD64
make release-linux-arm64    # Linux ARM64
make release-windows-amd64  # Windows AMD64
make checksums              # Generate SHA256 checksums
```

**Version Variables**:
```makefile
VERSION ?= 2.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags="-X main.Version=$(VERSION) -X main.Commit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE) -w -s"
```

### 6. GitHub Actions Updates
- **File**: `.github/workflows/release.yml`

**Updated Release Workflow**:
- Changed binary names to match updater expectations
- Added raw binary uploads (in addition to archives)
- Generate SHA256 checksums for both archives and raw binaries
- Support multi-platform builds with version injection

**Release Assets**:
```
Archives (for manual download):
  sql-studio-2.0.0-darwin-amd64.tar.gz
  sql-studio-2.0.0-darwin-arm64.tar.gz
  sql-studio-2.0.0-linux-amd64.tar.gz
  sql-studio-2.0.0-linux-arm64.tar.gz
  sql-studio-2.0.0-windows-amd64.zip

Raw Binaries (for auto-updater):
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

Combined:
  checksums.txt
```

## File Structure

```
backend-go/
├── cmd/
│   ├── server/
│   │   └── main.go                    # Updated with version flag & auto-check
│   └── sqlstudio/
│       └── main.go                    # NEW: CLI tool
├── pkg/
│   ├── version/
│   │   ├── version.go                 # NEW: Version info
│   │   └── version_test.go            # NEW: Tests (100% coverage)
│   └── updater/
│       ├── updater.go                 # NEW: Update functionality
│       └── updater_test.go            # NEW: Tests (50.8% coverage)
├── Makefile                           # Updated with version injection
├── VERSION_AND_UPDATE.md              # NEW: Comprehensive documentation
└── VERSION_UPDATE_SUMMARY.md          # NEW: This file

.github/
└── workflows/
    └── release.yml                    # Updated for auto-updater support
```

## Testing

### Test Results
```bash
# Version package: 100% coverage
go test ./pkg/version -cover
PASS
coverage: 100.0% of statements

# Updater package: 50.8% coverage
go test ./pkg/updater -cover
PASS
coverage: 50.8% of statements
```

### Build Tests
```bash
# Build CLI
make build-cli
# Output: ✓ Built: build/sqlstudio

# Test version command
./build/sqlstudio version
# Output:
# SQL Studio v2.0.0
# Commit: 86b8752
# Built: 2025-10-23T22:46:03Z
# Go: go1.24.5
# OS/Arch: darwin/arm64

# Build server
make build
# Output: Built: build/sql-studio-backend

# Test server version flag
./build/sql-studio-backend --version
# Output: Same as above
```

## How It Works

### Version Injection Flow

1. **Build Time**:
   ```bash
   make build VERSION=2.0.0
   ```

2. **Makefile extracts Git info**:
   ```makefile
   GIT_COMMIT := $(shell git rev-parse --short HEAD)
   BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
   ```

3. **Go build with ldflags**:
   ```bash
   go build -ldflags="-X main.Version=2.0.0 -X main.Commit=86b8752 -X main.BuildDate=2025-10-23T22:46:03Z"
   ```

4. **Runtime**:
   ```go
   // In main.go
   version.Version = Version  // From ldflags
   version.Commit = Commit
   version.BuildDate = BuildDate

   info := version.GetInfo()  // Access version info
   ```

### Update Check Flow

1. **User runs update command**:
   ```bash
   ./sqlstudio update --check
   ```

2. **Updater checks GitHub API**:
   ```
   GET https://api.github.com/repos/sql-studio/sql-studio/releases/latest
   ```

3. **Parse response**:
   ```json
   {
     "tag_name": "v2.1.0",
     "body": "Release notes...",
     "assets": [
       {
         "name": "sql-studio-backend-darwin-arm64",
         "browser_download_url": "https://..."
       },
       {
         "name": "sql-studio-backend-darwin-arm64.sha256",
         "browser_download_url": "https://..."
       }
     ]
   }
   ```

4. **Compare versions**:
   ```go
   current := "2.0.0"
   latest := "2.1.0"
   isNewer := latest > current  // true
   ```

5. **If newer, download binary**:
   - Find appropriate asset for OS/Arch
   - Download binary
   - Download checksum
   - Verify SHA256

6. **Install update**:
   - Create backup: `current.backup`
   - Write new binary: `current.tmp`
   - Replace: `mv current.tmp current`
   - Clean up: `rm current.backup`

### Auto-Update Check Flow

1. **Server starts** (development mode):
   ```go
   if !cfg.IsProduction() {
       go checkForUpdates(appLogger)
   }
   ```

2. **Check if should run**:
   ```go
   if !u.ShouldCheckForUpdate() {
       return  // Checked within last 24 hours
   }
   ```

3. **Check for updates** (10-second timeout):
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
   defer cancel()

   updateInfo, err := u.CheckForUpdate(ctx)
   ```

4. **Record check time**:
   ```go
   u.RecordUpdateCheck()  // Writes ~/.sqlstudio/.last_update_check
   ```

5. **Show notification if available**:
   ```go
   if updateInfo.Available {
       logger.Info("New version available! Run 'sqlstudio update' to upgrade")
   }
   ```

## Security

### Checksum Verification
All downloads are verified with SHA256:
```go
actualChecksum := sha256.Sum256(binaryData)
if actualChecksumStr != expectedChecksum {
    return fmt.Errorf("checksum mismatch")
}
```

### Backup and Rollback
Before replacing binary:
```go
backupPath := currentExe + ".backup"
copyFile(currentExe, backupPath)

// If update fails
os.Rename(backupPath, currentExe)  // Restore
```

### Safe Update Process
1. Download to temporary file
2. Verify checksum
3. Create backup of current binary
4. Replace binary atomically
5. Only delete backup on success
6. Restore backup on any failure

## Usage Examples

### Check Version
```bash
# CLI tool
./sqlstudio version

# Server
./sql-studio-backend --version

# Both output:
# SQL Studio v2.0.0
# Commit: 86b8752
# Built: 2025-10-23T22:46:03Z
# Go: go1.24.5
# OS/Arch: darwin/arm64
```

### Check for Updates
```bash
./sqlstudio update --check

# Output:
# Checking for updates...
# Current version: 2.0.0
#
# New version available: 2.1.0
# Current version: 2.0.0
#
# Release Notes:
# ------------------------------------------------------------
# ## What's New
# - New feature: XYZ
# - Bug fix: ABC
# ------------------------------------------------------------
#
# To install the update, run: sqlstudio update
```

### Install Update
```bash
./sqlstudio update

# Output:
# Checking for updates...
# Current version: 2.0.0
#
# New version available: 2.1.0
#
# Release Notes: [...]
#
# Do you want to install this update? (y/N): y
#
# Downloading update...
# ✓ Update installed successfully!
# Updated from 2.0.0 to 2.1.0
#
# Please restart the application for the changes to take effect.
```

### Build Release
```bash
# Single platform
make release VERSION=2.1.0

# All platforms
make release-all VERSION=2.1.0

# Output:
# Building for Darwin AMD64...
# ✓ Built: build/sql-studio-darwin-amd64
# Building for Darwin ARM64...
# ✓ Built: build/sql-studio-darwin-arm64
# [...]
# Generating checksums...
# ✓ Release build complete for version 2.1.0
```

## Configuration

### Environment Variables
None required for basic usage.

### Config Files
- `~/.sqlstudio/.last_update_check`: Timestamp of last check
- Format: RFC3339 timestamp (e.g., "2025-10-23T10:00:00Z")

### Customization
Edit `pkg/updater/updater.go` constants:
```go
const (
    GitHubAPIURL = "https://api.github.com/repos/sql-studio/sql-studio/releases/latest"
    UpdateCheckInterval = 24 * time.Hour
    DefaultTimeout = 30 * time.Second
)
```

## Error Handling

### Network Errors
```
Error: Failed to check for updates: context deadline exceeded

Possible causes:
  - No internet connection
  - GitHub API rate limit exceeded
  - Network firewall blocking requests
```

### Permission Errors
```
Error: Failed to install update: permission denied

Possible causes:
  - Insufficient permissions (try running with sudo)
  - Binary is currently running (stop it first)
  - Installation method doesn't support auto-update (e.g., brew)
```

### Checksum Errors
```
Error: checksum mismatch: expected abc123..., got def456...
```

## Limitations

1. **No delta updates**: Downloads entire binary (typically 10-50MB)
2. **Manual restart required**: Update doesn't auto-restart application
3. **No rollback command**: Must manually restore `.backup` file
4. **Dev builds unsupported**: Version "dev" never shows updates
5. **Package managers**: Brew/apt installed binaries can't self-update

## Future Enhancements

Potential improvements:
1. Delta/patch updates (download only changes)
2. Auto-restart after update
3. Rollback command: `sqlstudio rollback`
4. Update channels: stable/beta/alpha
5. Desktop notifications
6. Version history command
7. Pre/post-update hooks
8. Self-hosted update server support

## Related Documentation

- **VERSION_AND_UPDATE.md**: Comprehensive guide with examples
- **Makefile**: Build targets and variables
- **.github/workflows/release.yml**: CI/CD release automation

## Support

For issues or questions:
1. Check logs: `~/.sqlstudio/.last_update_check`
2. Try manual update: Download from GitHub Releases
3. Report issue: github.com/sql-studio/sql-studio/issues
