# Homebrew Tap Repository Setup Guide

This guide walks through creating and configuring the Howlerops Homebrew tap repository.

## Prerequisites

- GitHub account with permissions to create repositories in the `sql-studio` organization
- GitHub Personal Access Token with `repo` scope
- Git installed locally

## Step 1: Create the Tap Repository

1. **Create a new repository on GitHub**:
   - Navigate to: https://github.com/organizations/sql-studio/repositories/new
   - Repository name: `homebrew-tap`
   - Description: "Homebrew tap for Howlerops"
   - Visibility: Public
   - Initialize with: README (optional, we'll overwrite it)
   - License: MIT (same as main repository)

2. **Repository URL**: `https://github.com/sql-studio/homebrew-tap`

## Step 2: Initialize the Repository Locally

```bash
# Clone the newly created repository
git clone https://github.com/sql-studio/homebrew-tap.git
cd homebrew-tap

# Create the Formula directory
mkdir -p Formula

# Copy the formula from the main Howlerops repository
cp /path/to/sql-studio/Formula/sql-studio.rb Formula/

# Copy the README
cp /path/to/sql-studio/Formula/README.md .

# Add all files
git add .

# Commit
git commit -m "Initial tap setup with sql-studio formula"

# Push to GitHub
git push origin main
```

## Step 3: Configure Repository Settings

### Repository Details

1. Navigate to: https://github.com/sql-studio/homebrew-tap/settings
2. Update description: "Homebrew tap for Howlerops"
3. Update website: "https://github.com/sql-studio/sql-studio"
4. Add topics:
   - `homebrew`
   - `homebrew-tap`
   - `sql`
   - `database-client`
   - `macos`

### Branch Protection (Optional but Recommended)

1. Navigate to: Settings > Branches
2. Add rule for `main` branch:
   - Require pull request reviews before merging
   - Require status checks to pass before merging
   - Include administrators: No (for automated updates)

### Repository Features

1. Navigate to: Settings > General > Features
2. Enable:
   - Issues (for user feedback)
   - Discussions (optional)
3. Disable:
   - Wikis (use main repo for documentation)
   - Projects (use main repo)

## Step 4: Configure GitHub Secrets in Main Repository

The main Howlerops repository needs a token to push formula updates to the tap repository.

### Generate Personal Access Token

1. Navigate to: https://github.com/settings/tokens
2. Click "Generate new token" > "Generate new token (classic)"
3. Token name: `Howlerops Homebrew Tap Updater`
4. Expiration: No expiration (or 1 year with calendar reminder)
5. Select scopes:
   - `repo` (Full control of private repositories)
6. Generate token and copy it

### Add Secret to Main Repository

1. Navigate to: https://github.com/sql-studio/sql-studio/settings/secrets/actions
2. Click "New repository secret"
3. Name: `HOMEBREW_TAP_TOKEN`
4. Value: Paste the Personal Access Token
5. Click "Add secret"

**Important**: Store this token securely! It won't be shown again.

## Step 5: Test the Setup

### Test Manual Formula Update

```bash
# Clone the main repository
cd /path/to/sql-studio

# Set environment variables
export GITHUB_TOKEN=your_personal_access_token
export HOMEBREW_TAP_REPO=sql-studio/homebrew-tap

# Test the update script in dry-run mode
DRY_RUN=true ./scripts/update-homebrew-formula.sh latest

# If dry-run looks good, run for real
./scripts/update-homebrew-formula.sh latest
```

### Test Installation from Tap

```bash
# Add the tap
brew tap sql-studio/tap

# Install Howlerops
brew install sql-studio

# Verify installation
sql-studio --version

# Test the formula
brew test sql-studio

# Clean up (optional)
brew uninstall sql-studio
brew untap sql-studio/tap
```

## Step 6: Verify Automated Updates

### Create a Test Release

1. In the main Howlerops repository:
   ```bash
   git tag v2.0.0-test
   git push origin v2.0.0-test
   ```

2. Monitor the GitHub Actions workflow:
   - Navigate to: https://github.com/sql-studio/sql-studio/actions
   - Look for the "Release" workflow
   - Verify the "Update Homebrew Formula" job completes successfully

3. Check the tap repository:
   - Navigate to: https://github.com/sql-studio/homebrew-tap/commits/main
   - Verify a new commit with formula update appears

4. Clean up test release:
   ```bash
   # Delete local tag
   git tag -d v2.0.0-test

   # Delete remote tag
   git push --delete origin v2.0.0-test

   # Delete GitHub release via web interface
   ```

## Step 7: Documentation

### Update Main Repository Documentation

1. Add link to tap in main README.md:
   ```markdown
   ### Homebrew (macOS)

   ```bash
   brew install sql-studio/tap/sql-studio
   ```
   ```

2. Ensure HOMEBREW.md is committed to main repository

3. Update installation docs to include Homebrew option

## Troubleshooting

### Formula Update Fails

**Problem**: GitHub Actions job fails with "Permission denied"

**Solution**:
- Verify `HOMEBREW_TAP_TOKEN` secret is set correctly
- Ensure token has `repo` scope
- Check token hasn't expired

**Problem**: SHA256 checksum mismatch

**Solution**:
- Verify release artifacts are uploaded correctly
- Check network connectivity during download
- Re-run the workflow

### Installation Fails

**Problem**: `brew install sql-studio` fails

**Solution**:
```bash
# Update Homebrew
brew update

# Audit the formula
brew audit --strict sql-studio

# Check formula syntax
brew test-bot --only-tap-syntax

# Try verbose installation
brew install --verbose sql-studio
```

**Problem**: Binary not found in archive

**Solution**:
- Verify archive contains `sql-studio` binary (not `sql-studio-backend`)
- Check archive structure in release workflow
- Ensure binary is at root of archive

### Testing Issues

**Problem**: `brew test sql-studio` fails

**Solution**:
- Verify binary supports `--version` flag
- Check binary is executable
- Test binary manually:
  ```bash
  cd $(brew --cellar sql-studio)/2.0.0/bin
  ./sql-studio --version
  ```

## Maintenance

### Regular Tasks

1. **Monitor tap repository**:
   - Check for issues from users
   - Review automated formula updates
   - Verify checksums after each release

2. **Update formula structure** (when needed):
   - Add new install steps
   - Update caveats
   - Add completions/man pages

3. **Audit formula periodically**:
   ```bash
   brew audit --strict --online sql-studio
   ```

### Security

1. **Rotate Personal Access Token**:
   - Set calendar reminder for token expiration
   - Generate new token before expiration
   - Update `HOMEBREW_TAP_TOKEN` secret

2. **Monitor for vulnerabilities**:
   - Subscribe to security advisories
   - Update dependencies promptly
   - Test formula after security updates

3. **Review automated commits**:
   - Verify formula updates look correct
   - Check SHA256 checksums are valid
   - Ensure version numbers match releases

## Additional Resources

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Homebrew Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae)
- [How to Create and Maintain a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
- [Homebrew Ruby Style Guide](https://docs.brew.sh/Formula-Cookbook#ruby-style-guide)

## Support

For issues with:
- **Formula installation**: Open issue in homebrew-tap repository
- **Howlerops bugs**: Open issue in main repository
- **Tap setup**: See HOMEBREW.md or contact maintainers

---

**Created**: 2025-10-23
**Last Updated**: 2025-10-23
**Maintained By**: Howlerops Team
