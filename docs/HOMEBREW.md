# Homebrew Tap for SQL Studio

This document describes the Homebrew tap setup for SQL Studio, allowing macOS users to install SQL Studio via Homebrew.

## Overview

SQL Studio is distributed via a custom Homebrew tap that supports both Intel (x86_64) and Apple Silicon (ARM64) macOS architectures.

**Tap Repository**: `https://github.com/sql-studio/homebrew-tap`

## User Installation Instructions

### Prerequisites
- macOS (Intel or Apple Silicon)
- Homebrew installed ([https://brew.sh](https://brew.sh))

### Installation Steps

1. **Install from the tap**:
   ```bash
   brew install sql-studio/tap/sql-studio
   ```

   Or tap first, then install:
   ```bash
   brew tap sql-studio/tap
   brew install sql-studio
   ```

2. **Verify installation**:
   ```bash
   sql-studio --version
   ```

3. **Run SQL Studio**:
   ```bash
   sql-studio
   ```

### Updating

To update to the latest version:
```bash
brew update
brew upgrade sql-studio
```

### Uninstalling

To remove SQL Studio:
```bash
brew uninstall sql-studio
brew untap sql-studio/tap
```

## Maintainer Setup

### Initial Tap Repository Setup

1. **Create the tap repository on GitHub**:
   - Repository name: `homebrew-tap`
   - Organization/User: `sql-studio`
   - Full URL: `https://github.com/sql-studio/homebrew-tap`
   - Description: "Homebrew tap for SQL Studio"

2. **Initialize the repository**:
   ```bash
   # Clone the tap repository
   git clone https://github.com/sql-studio/homebrew-tap.git
   cd homebrew-tap

   # Create the Formula directory
   mkdir -p Formula

   # Copy the formula from the main repository
   cp /path/to/sql-studio/Formula/sql-studio.rb Formula/

   # Add README
   cat > README.md << 'EOF'
   # SQL Studio Homebrew Tap

   Official Homebrew tap for [SQL Studio](https://github.com/sql-studio/sql-studio).

   ## Installation

   ```bash
   brew install sql-studio/tap/sql-studio
   ```

   ## Updating

   ```bash
   brew update
   brew upgrade sql-studio
   ```

   ## About

   SQL Studio is a modern SQL database client with cloud sync capabilities.
   EOF

   # Commit and push
   git add .
   git commit -m "Initial tap setup"
   git push origin main
   ```

3. **Configure repository settings**:
   - Enable "Automatically delete head branches"
   - Add repository description
   - Add topics: `homebrew`, `homebrew-tap`, `sql`, `database-client`

### Updating the Formula on New Releases

The formula is automatically updated via GitHub Actions when a new release is created. However, you can also update it manually.

#### Automatic Updates (Recommended)

The main SQL Studio repository includes a GitHub Actions workflow that:
1. Detects new releases
2. Downloads release artifacts
3. Calculates SHA256 checksums
4. Updates the formula in the tap repository
5. Commits and pushes the changes

**Requirements**:
- `HOMEBREW_TAP_TOKEN` secret configured in the main repository
  - Generate a Personal Access Token with `repo` scope
  - Add to repository secrets at: Settings > Secrets and variables > Actions
  - Name: `HOMEBREW_TAP_TOKEN`
  - Value: Your GitHub PAT

#### Manual Updates

If you need to update the formula manually:

1. **Run the update script**:
   ```bash
   cd /path/to/sql-studio
   ./scripts/update-homebrew-formula.sh v2.1.0
   ```

2. **Or update manually**:
   ```bash
   # Clone the tap repository
   git clone https://github.com/sql-studio/homebrew-tap.git
   cd homebrew-tap

   # Download the release artifacts
   VERSION="2.1.0"
   curl -L -O "https://github.com/sql-studio/sql-studio/releases/download/v${VERSION}/sql-studio-darwin-amd64.tar.gz"
   curl -L -O "https://github.com/sql-studio/sql-studio/releases/download/v${VERSION}/sql-studio-darwin-arm64.tar.gz"

   # Calculate SHA256 checksums
   AMD64_SHA=$(shasum -a 256 sql-studio-darwin-amd64.tar.gz | awk '{print $1}')
   ARM64_SHA=$(shasum -a 256 sql-studio-darwin-arm64.tar.gz | awk '{print $1}')

   echo "AMD64 SHA256: $AMD64_SHA"
   echo "ARM64 SHA256: $ARM64_SHA"

   # Update Formula/sql-studio.rb with:
   # - New version number
   # - New download URLs
   # - New SHA256 checksums

   # Clean up downloaded files
   rm sql-studio-darwin-*.tar.gz

   # Commit and push
   git add Formula/sql-studio.rb
   git commit -m "Update sql-studio to v${VERSION}"
   git push origin main
   ```

3. **Test the updated formula**:
   ```bash
   # Uninstall existing version
   brew uninstall sql-studio || true

   # Update tap
   brew update

   # Install new version
   brew install sql-studio/tap/sql-studio

   # Test
   sql-studio --version
   brew test sql-studio
   ```

### Formula Update Checklist

When updating the formula:

- [ ] Update version number in the formula
- [ ] Update both `on_intel` and `on_arm` download URLs
- [ ] Calculate and update both SHA256 checksums
- [ ] Verify URLs are accessible
- [ ] Test installation on both Intel and Apple Silicon if possible
- [ ] Update changelog/release notes if needed
- [ ] Run `brew audit --strict sql-studio` to check for issues

## Formula Structure

The formula consists of:

1. **Metadata**: Description, homepage, version, license
2. **Architecture blocks**: Separate URLs and checksums for Intel and ARM64
3. **Install method**: Steps to install the binary
4. **Caveats**: Post-installation message for users
5. **Test block**: Automated tests to verify installation

## Testing the Formula

### Local Testing

Before pushing formula updates:

```bash
# Audit the formula for issues
brew audit --strict Formula/sql-studio.rb

# Test installation locally
brew install --build-from-source Formula/sql-studio.rb

# Run formula tests
brew test sql-studio

# Verify the binary works
sql-studio --version
sql-studio --help
```

### CI Testing

The tap repository should include a GitHub Actions workflow to test formula updates:

```yaml
name: Test Formula
on: [pull_request, push]

jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - name: Test formula
        run: |
          brew test-bot --only-cleanup-before
          brew test-bot --only-setup
          brew test-bot --only-tap-syntax
          brew test-bot --only-formulae
```

## Troubleshooting

### Common Issues

1. **SHA256 mismatch**:
   - Download the release artifacts again
   - Recalculate checksums
   - Ensure you're using the correct version URLs

2. **Architecture detection issues**:
   - Verify `on_intel` and `on_arm` blocks are correct
   - Test on both architectures if possible

3. **Installation fails**:
   - Check that the binary name in the tarball matches `sql-studio`
   - Verify file permissions in the tarball
   - Check binary is executable

4. **Formula audit failures**:
   ```bash
   brew audit --strict --online Formula/sql-studio.rb
   ```
   - Fix any reported issues
   - Common issues: missing license, incorrect URL format, style violations

### Getting Help

- Homebrew documentation: https://docs.brew.sh/Formula-Cookbook
- Homebrew Ruby style guide: https://docs.brew.sh/Formula-Cookbook#ruby-style-guide
- SQL Studio issues: https://github.com/sql-studio/sql-studio/issues

## Formula Versioning Strategy

### Version Numbering

SQL Studio follows semantic versioning (MAJOR.MINOR.PATCH):
- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### Release Cadence

- Formula updates are automated via GitHub Actions
- New versions are published when releases are created
- Beta/RC versions are not published to the tap
- Only stable releases are included

### Deprecation Policy

When deprecating old versions:
1. Add deprecation notice to formula caveats
2. Update documentation
3. Remove deprecated versions after 6 months
4. Communicate deprecation in release notes

## Security Considerations

1. **Checksum verification**: Always include SHA256 checksums
2. **HTTPS URLs**: All download URLs must use HTTPS
3. **Token security**: Store GitHub tokens as repository secrets
4. **Code signing**: Future consideration for macOS notarization
5. **Dependency scanning**: Monitor for vulnerabilities in dependencies

## Advanced Configuration

### Adding Completion Scripts

If SQL Studio provides shell completions:

```ruby
def install
  bin.install "sql-studio"
  bash_completion.install "completions/sql-studio.bash"
  fish_completion.install "completions/sql-studio.fish"
  zsh_completion.install "completions/_sql-studio"
end
```

### Adding Man Pages

If documentation includes man pages:

```ruby
def install
  bin.install "sql-studio"
  man1.install "man/sql-studio.1"
end
```

### Service Management

If SQL Studio includes a background service:

```ruby
service do
  run [opt_bin/"sql-studio", "serve"]
  keep_alive true
  log_path var/"log/sql-studio.log"
  error_log_path var/"log/sql-studio.error.log"
end
```

## Monitoring and Analytics

### Tap Usage Statistics

Monitor tap usage via:
- Homebrew analytics (if opted in): `brew info sql-studio`
- GitHub repository insights: Stars, forks, clones
- Download statistics from GitHub releases

### Metrics to Track

- Install count (via Homebrew analytics)
- Update frequency
- Formula audit success rate
- Issue reports related to installation
- Architecture distribution (Intel vs ARM64)

## Contributing

Contributions to the Homebrew tap are welcome! Please:

1. Fork the tap repository
2. Create a feature branch
3. Test your changes locally
4. Submit a pull request
5. Ensure CI passes

For major changes, please open an issue first to discuss the proposed changes.

## License

The SQL Studio Homebrew formula is licensed under the MIT License, consistent with the main SQL Studio project.

## Contact

- Issues: https://github.com/sql-studio/sql-studio/issues
- Discussions: https://github.com/sql-studio/sql-studio/discussions
- Email: support@sql-studio.com (if applicable)

---

**Last Updated**: 2025-10-23
**Maintained By**: SQL Studio Team
