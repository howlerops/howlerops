# Release Process

This document describes the release process for SQL Studio Backend.

## Overview

SQL Studio uses semantic versioning (MAJOR.MINOR.PATCH) and follows a GitOps-based release workflow. Releases are triggered by creating and pushing Git tags, with GitHub Actions handling the build, packaging, and deployment process.

## Semantic Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: New functionality in a backwards compatible manner
- **PATCH** version: Backwards compatible bug fixes

Examples:
- `1.0.0` - Initial stable release
- `1.1.0` - Added new features (sync, auth improvements)
- `1.1.1` - Bug fixes
- `2.0.0` - Breaking changes (API redesign, new storage backend)

## Pre-Release Checklist

Before creating a release, ensure:

- [ ] All tests pass (`make test`)
- [ ] Code is linted and formatted (`make lint`, `make fmt`)
- [ ] Documentation is up to date
- [ ] CHANGELOG.md is updated with release notes
- [ ] Version number is decided (following semver)
- [ ] All PRs for this release are merged to `main`
- [ ] Local environment is clean (`git status`)

## Creating a New Release

### Step 1: Update Version Information

The version is managed in the Makefile and can be overridden during build:

```bash
# Default version is 2.0.0
make release-all

# Or specify a custom version
make release-all VERSION=2.1.0
```

### Step 2: Build and Test Locally

Build all platform binaries and test them:

```bash
# Build all platforms
cd backend-go
make release-all VERSION=2.1.0

# Test the binary (current platform)
./build/sql-studio-backend --version

# Verify output shows correct version
# Expected output:
# SQL Studio Backend
# Version:    2.1.0
# Commit:     abc1234
# Build Date: 2025-10-23T10:30:00Z
```

### Step 3: Create Git Tag

Create an annotated tag with release notes:

```bash
# Create annotated tag
git tag -a v2.1.0 -m "Release v2.1.0

What's New:
- Feature: Added multi-database sync support
- Feature: Improved authentication with email verification
- Enhancement: Better error handling in storage layer
- Fix: Resolved race condition in session cleanup

Breaking Changes:
- None

See CHANGELOG.md for full details."

# Verify tag
git tag -l v2.1.0
git show v2.1.0
```

### Step 4: Push Tag to GitHub

```bash
# Push the tag to origin
git push origin v2.1.0
```

### Step 5: Automated GitHub Actions Workflow

Once the tag is pushed, GitHub Actions automatically:

1. Detects the new tag (triggers on `v*` tags)
2. Builds binaries for all platforms (Darwin, Linux, Windows)
3. Runs full test suite
4. Creates release archives (.tar.gz, .zip)
5. Generates SHA256 checksums
6. Creates GitHub Release with artifacts
7. Deploys to Google Cloud Run (production)
8. Optionally deploys to Fly.io
9. Runs smoke tests to verify deployment

### Step 6: Verify Release

After GitHub Actions completes:

1. **Check GitHub Release Page**
   - Visit: https://github.com/yourusername/sql-studio/releases
   - Verify all artifacts are uploaded:
     - `sql-studio-2.1.0-darwin-amd64.tar.gz`
     - `sql-studio-2.1.0-darwin-arm64.tar.gz`
     - `sql-studio-2.1.0-linux-amd64.tar.gz`
     - `sql-studio-2.1.0-linux-arm64.tar.gz`
     - `sql-studio-2.1.0-windows-amd64.zip`
     - `checksums.txt`

2. **Test Installation**
   ```bash
   # Download and test (example for macOS)
   curl -L -o sql-studio.tar.gz \
     https://github.com/yourusername/sql-studio/releases/download/v2.1.0/sql-studio-2.1.0-darwin-arm64.tar.gz

   tar -xzf sql-studio.tar.gz
   ./sql-studio-darwin-arm64 --version
   ```

3. **Verify Production Deployment**
   ```bash
   # Check Cloud Run deployment
   curl https://your-service.run.app/health

   # Or use make command
   cd backend-go
   make prod-status
   ```

## Manual Steps (If Any)

In most cases, the release process is fully automated. Manual steps are only required for:

1. **Updating Documentation Sites**
   - Update docs.sqlstudio.io if major changes
   - Update marketing materials
   - Announce on social media

2. **Homebrew Formula Update** (coming soon)
   - Update homebrew-sqlstudio tap
   - Submit PR to homebrew/core if needed

3. **Docker Hub** (optional)
   - Tag and push Docker images
   - Update Docker Hub description

## Post-Release

After a successful release:

1. **Update CHANGELOG.md**
   - Move "Unreleased" section to the new version
   - Start new "Unreleased" section

2. **Announce Release**
   - Twitter/X announcement
   - Blog post (for major releases)
   - Update documentation site
   - Email newsletter (for major releases)

3. **Monitor Production**
   ```bash
   # Watch logs for errors
   make prod-logs

   # Check metrics
   make check-costs
   ```

4. **Create Next Milestone**
   - Create GitHub milestone for next version
   - Plan features for next release

## Rollback Procedure

If a release has critical issues:

### Option 1: Quick Rollback (Cloud Run)

```bash
cd backend-go

# List all revisions
make prod-revisions

# Rollback to previous revision
make prod-rollback REVISION=sql-studio-backend-00042-xyz

# Verify rollback
make prod-status
```

### Option 2: Hotfix Release

For bugs that need immediate fixing:

```bash
# Create hotfix branch from tag
git checkout -b hotfix/2.1.1 v2.1.0

# Make the fix
# ... edit files ...

# Commit and tag
git commit -am "fix: critical bug in session handler"
git tag -a v2.1.1 -m "Release v2.1.1 - Hotfix for session bug"

# Push
git push origin hotfix/2.1.1
git push origin v2.1.1

# Merge back to main
git checkout main
git merge hotfix/2.1.1
git push origin main
```

### Option 3: Delete Bad Release

If the release was never deployed:

```bash
# Delete GitHub release (via web UI)
# Delete local tag
git tag -d v2.1.0

# Delete remote tag
git push origin :refs/tags/v2.1.0
```

## Release Cadence

- **Patch releases**: As needed for bug fixes
- **Minor releases**: Every 2-4 weeks
- **Major releases**: Every 3-6 months

## Version Compatibility

- **API Compatibility**: Maintained within major versions
- **Database Schema**: Migrations included in releases
- **Client SDK**: Version pinning recommended

## Environment-Specific Releases

### Production (main branch)

```bash
git tag -a v2.1.0 -m "Production release v2.1.0"
git push origin v2.1.0
```

### Staging (optional)

```bash
git tag -a v2.1.0-rc.1 -m "Release candidate v2.1.0-rc.1"
git push origin v2.1.0-rc.1
```

### Development Builds

Development builds use the version "dev" and commit hash:

```bash
# Build development version
make build

# Check version
./build/sql-studio-backend --version
# Output:
# Version:    dev
# Commit:     abc1234
# Build Date: 2025-10-23T15:45:00Z
```

## Troubleshooting

### Build Fails During Release

```bash
# Check Go version
go version  # Should be 1.24+

# Verify dependencies
cd backend-go
go mod verify

# Try manual build
make release-all VERSION=2.1.0
```

### GitHub Actions Fails

1. Check workflow logs in GitHub Actions tab
2. Common issues:
   - Missing secrets (TURSO_URL, JWT_SECRET, etc.)
   - GCP permissions issue
   - Test failures

### Binary Won't Run

```bash
# Check if binary is executable
chmod +x sql-studio-backend

# Check dependencies (for Linux)
ldd sql-studio-backend

# Run with debug logging
LOG_LEVEL=debug ./sql-studio-backend
```

## Release Artifacts

Each release includes:

- **Source code** (zip and tar.gz)
- **Compiled binaries** for all platforms
- **Checksums file** (SHA256)
- **Release notes** (auto-generated from commits)
- **Docker images** (optional)

## Security Considerations

- All binaries are built in GitHub Actions (reproducible builds)
- Checksums provided for verification
- Binaries are signed (coming soon)
- SBOM generated for each release (coming soon)

## Additional Resources

- [Installation Guide](INSTALL.md)
- [Deployment Guide](backend-go/DEPLOYMENT.md)
- [Contributing Guide](docs/CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)

## Questions or Issues?

- Open an issue: https://github.com/yourusername/sql-studio/issues
- Join discussions: https://github.com/yourusername/sql-studio/discussions
- Email: support@sqlstudio.io
