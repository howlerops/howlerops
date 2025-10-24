# Version & Update - Quick Reference

## Commands

### Version Info
```bash
# CLI tool
./sqlstudio version

# Server
./sql-studio-backend --version
```

### Updates
```bash
# Check only (no install)
./sqlstudio update --check

# Install update
./sqlstudio update

# Force install (skip confirmation)
./sqlstudio update --force
```

## Building

### Development
```bash
make build          # Build server
make build-cli      # Build CLI
make build-all      # Build both
```

### Release
```bash
# Current platform
make release VERSION=2.1.0

# All platforms
make release-all VERSION=2.1.0

# Specific platforms
make release-darwin-amd64
make release-darwin-arm64
make release-linux-amd64
make release-linux-arm64
make release-windows-amd64
```

## Release Process

1. **Update version** (if needed):
   ```makefile
   VERSION ?= 2.1.0
   ```

2. **Build release**:
   ```bash
   make release-all VERSION=2.1.0
   ```

3. **Create Git tag**:
   ```bash
   git tag -a v2.1.0 -m "Release v2.1.0"
   git push origin v2.1.0
   ```

4. **GitHub Actions runs automatically**:
   - Builds all platforms
   - Creates GitHub Release
   - Uploads binaries and checksums

5. **Verify**:
   ```bash
   ./sqlstudio update --check
   ```

## Files Created

### Packages
- `pkg/version/version.go` - Version info
- `pkg/version/version_test.go` - Tests (100% coverage)
- `pkg/updater/updater.go` - Update logic
- `pkg/updater/updater_test.go` - Tests (50.8% coverage)

### CLI
- `cmd/sqlstudio/main.go` - CLI tool

### Server
- `cmd/server/main.go` - Updated with version flag & auto-check

### Documentation
- `VERSION_AND_UPDATE.md` - Comprehensive guide
- `VERSION_UPDATE_SUMMARY.md` - Implementation summary
- `VERSION_QUICK_REF.md` - This file

### Build
- `Makefile` - Updated with version injection
- `.github/workflows/release.yml` - Updated for auto-updater

## Key Features

- **Version injection** at build time (ldflags)
- **Auto-update check** on server startup (dev mode only)
- **SHA256 verification** for all downloads
- **Automatic backup** before update
- **Platform detection** (darwin/linux/windows, amd64/arm64)
- **Rate limiting** (24-hour interval)
- **Graceful errors** with helpful messages

## Testing

```bash
# Run tests
go test ./pkg/version ./pkg/updater -v

# With coverage
go test ./pkg/version ./pkg/updater -cover

# Build and test
make build-cli
./build/sqlstudio version
./build/sqlstudio update --check
```

## Configuration

### Auto-Update Check
- **Enabled**: Development mode only
- **Interval**: 24 hours
- **Timeout**: 10 seconds
- **Storage**: `~/.sqlstudio/.last_update_check`

### Customization
Edit constants in `pkg/updater/updater.go`:
```go
const UpdateCheckInterval = 24 * time.Hour
const DefaultTimeout = 30 * time.Second
```

## Common Issues

### Permission denied
```bash
sudo ./sqlstudio update
```

### Binary is running
```bash
# Stop server first
./sqlstudio update
```

### Network timeout
```bash
# Increase timeout or check internet connection
# Edit pkg/updater/updater.go
```

## GitHub Release Requirements

### Tag Format
```
v2.1.0
```

### Required Assets
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

### Checksum Format
```
abc123def456...  sql-studio-backend-darwin-amd64
```

## Version Variables

### In Code
```go
// Set via ldflags
var Version = "dev"
var Commit = "unknown"
var BuildDate = "unknown"

// Use in package
version.Version = Version
version.Commit = Commit
version.BuildDate = BuildDate
```

### In Makefile
```makefile
VERSION ?= 2.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags="-X main.Version=$(VERSION) ..."
```

### In GitHub Actions
```yaml
VERSION=${GITHUB_REF#refs/tags/v}
COMMIT=${{ github.sha }}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

## Quick Test

```bash
# 1. Build
cd backend-go
make build-cli

# 2. Check version
./build/sqlstudio version

# 3. Check updates (will fail if no newer version)
./build/sqlstudio update --check

# 4. Build all platforms
make release-all VERSION=2.0.1

# 5. Verify checksums
cd build
shasum -c checksums.txt
```

## Help

```bash
./sqlstudio help
make help
```

For detailed documentation, see:
- `VERSION_AND_UPDATE.md` - Full guide
- `VERSION_UPDATE_SUMMARY.md` - Implementation details
