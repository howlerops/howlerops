# Homebrew Tap Quick Reference

Quick reference guide for common Homebrew tap operations for SQL Studio maintainers.

## Table of Contents

- [User Commands](#user-commands)
- [Maintainer Commands](#maintainer-commands)
- [Testing Commands](#testing-commands)
- [Troubleshooting](#troubleshooting)
- [Release Checklist](#release-checklist)

## User Commands

### Installation

```bash
# Install from tap
brew install sql-studio/tap/sql-studio

# Or tap first, then install
brew tap sql-studio/tap
brew install sql-studio
```

### Updates

```bash
# Update Homebrew and upgrade SQL Studio
brew update
brew upgrade sql-studio

# Check for outdated formulas
brew outdated
```

### Information

```bash
# Show formula information
brew info sql-studio

# Show installation path
brew --prefix sql-studio

# List installed files
brew list sql-studio

# Show formula options
brew options sql-studio
```

### Uninstallation

```bash
# Uninstall SQL Studio
brew uninstall sql-studio

# Remove tap
brew untap sql-studio/tap
```

## Maintainer Commands

### Formula Management

```bash
# Audit formula
brew audit sql-studio
brew audit --strict sql-studio
brew audit --strict --online sql-studio

# Check formula style
brew style Formula/sql-studio.rb

# Test formula
brew test sql-studio

# Install from local formula
brew install --build-from-source ./Formula/sql-studio.rb

# Reinstall formula
brew reinstall sql-studio
```

### Tap Management

```bash
# Tap the repository locally
brew tap sql-studio/tap /path/to/homebrew-tap

# Untap
brew untap sql-studio/tap

# List all taps
brew tap

# Show tap info
brew tap-info sql-studio/tap
```

### Formula Updates

```bash
# Clone tap repository
git clone https://github.com/sql-studio/homebrew-tap.git
cd homebrew-tap

# Update formula version
# Edit Formula/sql-studio.rb

# Commit changes
git add Formula/sql-studio.rb
git commit -m "Update sql-studio to vX.Y.Z"
git push origin main
```

### Calculate Checksums

```bash
# Download release artifacts
VERSION="2.0.0"
curl -L -O "https://github.com/sql-studio/sql-studio/releases/download/v${VERSION}/sql-studio-darwin-amd64.tar.gz"
curl -L -O "https://github.com/sql-studio/sql-studio/releases/download/v${VERSION}/sql-studio-darwin-arm64.tar.gz"

# Calculate SHA256 checksums
shasum -a 256 sql-studio-darwin-amd64.tar.gz
shasum -a 256 sql-studio-darwin-arm64.tar.gz
```

### Automated Updates

```bash
# Run update script (from main repository)
cd /path/to/sql-studio
export GITHUB_TOKEN=your_token
./scripts/update-homebrew-formula.sh v2.0.0

# Dry run
DRY_RUN=true ./scripts/update-homebrew-formula.sh v2.0.0
```

## Testing Commands

### Local Testing

```bash
# Test formula syntax
ruby -c Formula/sql-studio.rb

# Test formula structure
brew info ./Formula/sql-studio.rb

# Run test script
./scripts/test-homebrew-formula.sh

# Skip installation tests (fast)
SKIP_INSTALL_TESTS=true ./scripts/test-homebrew-formula.sh
```

### Installation Testing

```bash
# Install from local formula
brew install --formula ./Formula/sql-studio.rb

# Verbose installation
brew install --verbose ./Formula/sql-studio.rb

# Install with debug output
brew install --debug ./Formula/sql-studio.rb

# Install and keep temporary files
brew install --keep-tmp ./Formula/sql-studio.rb
```

### Binary Testing

```bash
# Find binary location
which sql-studio
brew --prefix sql-studio

# Test binary
sql-studio --version
sql-studio --help

# Check binary architecture
file $(which sql-studio)
lipo -info $(which sql-studio)

# Verify code signature (if applicable)
codesign -dv $(which sql-studio)
```

### Cleanup

```bash
# Clean up downloads
brew cleanup sql-studio

# Clean all cached downloads
brew cleanup -s

# Remove all cached downloads
brew cleanup --prune=all
```

## Troubleshooting

### Common Issues

#### Installation Fails

```bash
# Update Homebrew
brew update

# Diagnose issues
brew doctor

# Check tap
brew tap-info sql-studio/tap

# Verbose installation
brew install --verbose sql-studio

# Debug installation
brew install --debug sql-studio
```

#### Wrong Architecture

```bash
# Check system architecture
uname -m
arch

# Check binary architecture
file $(which sql-studio)

# Force reinstall
brew reinstall sql-studio
```

#### Checksum Mismatch

```bash
# Clear cache
brew cleanup -s sql-studio

# Reinstall
brew reinstall sql-studio

# Manual verification
cd $(brew --cache)/downloads
shasum -a 256 *sql-studio*
```

#### Formula Not Found

```bash
# Update tap
brew update

# Check tap is installed
brew tap | grep sql-studio

# Re-tap
brew untap sql-studio/tap
brew tap sql-studio/tap
```

### Debug Commands

```bash
# Show Homebrew version and configuration
brew --version
brew config

# Show formula location
brew formula sql-studio

# Show cache location
brew --cache sql-studio

# Show cellar location
brew --cellar sql-studio

# Show prefix (installation) location
brew --prefix sql-studio

# Clear all caches
rm -rf $(brew --cache)
```

### Logs

```bash
# Show recent logs
brew log sql-studio

# Show installation logs
cat $(brew --cache)/Logs/sql-studio/*.log

# Show all logs
ls -la $(brew --cache)/Logs/
```

## Release Checklist

### Pre-Release

- [ ] Update version in formula
- [ ] Update download URLs
- [ ] Calculate SHA256 checksums
- [ ] Test formula locally
- [ ] Audit formula (`brew audit --strict`)
- [ ] Test installation on both architectures (if possible)

### Release

- [ ] Create release in main repository
- [ ] Tag release with version (e.g., `v2.0.0`)
- [ ] Push tag to trigger release workflow
- [ ] Verify GitHub Actions workflow succeeds
- [ ] Check release artifacts are uploaded

### Post-Release

- [ ] Verify Homebrew formula updated in tap repository
- [ ] Test installation from tap: `brew install sql-studio`
- [ ] Test upgrade: `brew upgrade sql-studio`
- [ ] Verify version: `sql-studio --version`
- [ ] Update release notes if needed
- [ ] Announce release

### Verification

```bash
# Quick verification script
brew update
brew reinstall sql-studio
sql-studio --version
brew test sql-studio
brew audit sql-studio
```

## Environment Variables

### For update-homebrew-formula.sh

```bash
# Required
export GITHUB_TOKEN="ghp_..."

# Optional
export HOMEBREW_TAP_REPO="sql-studio/homebrew-tap"
export DRY_RUN="true"
```

### For Homebrew

```bash
# Skip analytics
export HOMEBREW_NO_ANALYTICS=1

# Verbose output
export HOMEBREW_VERBOSE=1

# Debug mode
export HOMEBREW_DEBUG=1

# Skip auto-update
export HOMEBREW_NO_AUTO_UPDATE=1
```

## URLs and Paths

### GitHub URLs

```bash
# Main repository
https://github.com/sql-studio/sql-studio

# Tap repository
https://github.com/sql-studio/homebrew-tap

# Releases
https://github.com/sql-studio/sql-studio/releases

# Latest release
https://github.com/sql-studio/sql-studio/releases/latest
```

### Local Paths

```bash
# Formula location (after tapping)
$(brew --repository)/Library/Taps/sql-studio/homebrew-tap/Formula/sql-studio.rb

# Installation location
$(brew --prefix sql-studio)

# Binary location
$(brew --prefix sql-studio)/bin/sql-studio

# Cache location
$(brew --cache)/downloads/sql-studio-*

# Logs location
$(brew --cache)/Logs/sql-studio/
```

## Useful Scripts

### Update Formula Version

```bash
#!/bin/bash
VERSION="2.0.0"
sed -i '' "s/version \".*\"/version \"$VERSION\"/" Formula/sql-studio.rb
```

### Update Formula URLs

```bash
#!/bin/bash
VERSION="2.0.0"
sed -i '' "s|/v[0-9.]*/|/v$VERSION/|g" Formula/sql-studio.rb
```

### Quick Test

```bash
#!/bin/bash
brew uninstall sql-studio || true
brew install ./Formula/sql-studio.rb
sql-studio --version
brew test sql-studio
```

## CI/CD Integration

### GitHub Actions Secrets

Required secrets in main repository:

```yaml
secrets:
  HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}  # Auto-provided
```

### Workflow Triggers

```yaml
# Release workflow
on:
  push:
    tags:
      - 'v*'

# Tap workflow
on:
  push:
    branches:
      - main
    paths:
      - 'Formula/**'
```

## Support Resources

### Documentation

- [Homebrew Docs](https://docs.brew.sh/)
- [Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [How to Create a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
- [SQL Studio HOMEBREW.md](../HOMEBREW.md)

### Commands for Help

```bash
# General help
brew help

# Formula help
brew help formula

# Tap help
brew help tap

# Command-specific help
brew help install
brew help upgrade
```

### Contact

- Issues: https://github.com/sql-studio/sql-studio/issues
- Discussions: https://github.com/sql-studio/sql-studio/discussions

---

**Quick Reference Version**: 1.0
**Last Updated**: 2025-10-23
**Maintained By**: SQL Studio Team
