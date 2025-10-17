# Wails macOS CI/CD & Homebrew Deployment Plan

## Executive Summary

This plan outlines the complete CI/CD pipeline for building, signing, notarizing, and distributing HowlerOps (SQL Studio) as a macOS application via GitHub Actions and Homebrew Cask.

**Project Details:**
- App Name: HowlerOps
- Bundle ID: `com.howlerops.app`
- Current Version: 1.0.0
- Platform: macOS (Universal Binary - Intel & Apple Silicon)
- Distribution: GitHub Releases + Homebrew Cask

**Timeline:** 2-4 days of focused work
**Cost:** $99/year (Apple Developer Program) + GitHub Actions compute (free for public repos)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     GitHub Actions Workflow                      │
│  (Triggered on: git tag push matching v*.*.*)                   │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 1: Build Environment Setup                               │
│  • Checkout code                                                │
│  • Setup Go 1.21+                                              │
│  • Setup Node.js 20                                            │
│  • Install Wails CLI                                           │
│  • Install frontend dependencies                               │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 2: Code Signing Certificate Setup                        │
│  • Decode base64 certificate from GitHub Secrets               │
│  • Create temporary keychain                                    │
│  • Import Developer ID Application certificate                 │
│  • Configure keychain for codesign access                      │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 3: Build Universal Binary                                │
│  • Run: wails build -platform darwin/universal                  │
│  • Output: build/bin/howlerops.app                             │
│  • Includes: Intel (amd64) + Apple Silicon (arm64)            │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 4: Code Signing                                          │
│  • Sign with Developer ID Application certificate              │
│  • Enable hardened runtime                                      │
│  • Apply entitlements from wails.json                          │
│  • Deep sign all embedded frameworks and binaries              │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 5: Create Distribution Package                           │
│  • Option A: Create DMG with custom background/layout          │
│  • Option B: Create ZIP archive (simpler)                      │
│  • Recommended: ZIP for Homebrew simplicity                    │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 6: Notarization                                          │
│  • Submit to Apple notarization service                        │
│  • Use xcrun notarytool (macOS 12+)                           │
│  • Wait for approval (~2-10 minutes)                           │
│  • Staple notarization ticket to app bundle                    │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 7: GitHub Release                                        │
│  • Create GitHub Release with version tag                      │
│  • Upload signed & notarized DMG/ZIP                           │
│  • Generate release notes from commits                         │
│  • Mark as pre-release or production release                   │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 8: Update Homebrew Formula                               │
│  • Calculate SHA256 of release artifact                        │
│  • Clone homebrew-howlerops repository                         │
│  • Update Cask formula with new version & SHA256               │
│  • Commit and push to Homebrew tap                             │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│  Stage 9: Post-Release Validation                               │
│  • Test Homebrew installation                                   │
│  • Verify app launches without warnings                        │
│  • Send notification to Slack/Discord (optional)               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Code Changes Required

### 1. Update `wails.json` - Bundle Identifier and Signing

**File:** `/Users/jacob_1/projects/sql-studio/wails.json`

**Current State:** Lines 56-58 have empty signing identity
```json
"signing": {
  "identity": "",
  "embedProvisioningProfile": false
}
```

**Required Change:**
```json
"signing": {
  "identity": "Developer ID Application: YOUR NAME (TEAM_ID)",
  "embedProvisioningProfile": false
}
```

**Note:** The `identity` field should match your Apple Developer certificate name exactly. You can find this by running:
```bash
security find-identity -v -p codesigning
```

### 2. Add Hardened Runtime Entitlements

**Current entitlements** (lines 60-66) are good but should add hardened runtime flag:

```json
"entitlements": {
  "com.apple.security.app-sandbox": false,
  "com.apple.security.network.client": true,
  "com.apple.security.network.server": false,
  "com.apple.security.files.user-selected.read-write": true,
  "com.apple.security.files.downloads.read-write": true,
  "com.apple.security.cs.allow-unsigned-executable-memory": true
}
```

**Why:** Some SQL drivers or native modules may require `allow-unsigned-executable-memory`.

### 3. Update Version Management

**Current:** Version is hardcoded in `wails.json` (lines 20 and 111)

**Recommendation:** Update version in `wails.json` before creating git tag, or use automated version bumping:

```bash
# Before release, update version
jq '.info.productVersion = "1.0.1"' wails.json > tmp.json && mv tmp.json wails.json
git add wails.json
git commit -m "chore: bump version to 1.0.1"
git tag v1.0.1
git push origin main --tags
```

**Alternative:** Use a version management tool like `npm version` or a custom script.

### 4. Add DMG Background Image (Optional)

**Location:** `build/dmg-background.png` (create this directory and file)

If using DMG distribution, add a custom background image for better UX:
- Size: 658x498 pixels (matching wails.json lines 69-71)
- Design: Include app icon, arrow pointing to Applications folder
- Format: PNG with transparency

### 5. Create `.github/workflows/release-macos.yml`

**Purpose:** New workflow file for automated macOS app releases

**Location:** `.github/workflows/release-macos.yml`

See separate workflow file in this plan.

---

## GitHub Secrets Configuration

The following secrets must be added to your GitHub repository:

### Required Secrets

| Secret Name | Description | How to Obtain |
|------------|-------------|---------------|
| `APPLE_DEVELOPER_ID` | Your Apple ID email | Your Apple Developer account email |
| `APPLE_APP_PASSWORD` | App-specific password | Generated at appleid.apple.com |
| `APPLE_TEAM_ID` | 10-character team ID | Found in Apple Developer account |
| `APPLE_CERTIFICATE_P12` | Base64-encoded certificate | Export from Keychain, convert to base64 |
| `APPLE_CERTIFICATE_PASSWORD` | Certificate export password | Password you set when exporting .p12 |
| `HOMEBREW_TAP_TOKEN` | GitHub PAT for Homebrew repo | GitHub Settings → Developer Settings → Personal Access Tokens |

### How to Add Secrets

1. Go to your GitHub repository
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add each secret listed above

---

## Homebrew Distribution

### Create Homebrew Tap Repository

**Repository Name:** `homebrew-howlerops` (or `homebrew-sqlstudio`)

**Location:** GitHub under your organization/username

**Structure:**
```
homebrew-howlerops/
├── README.md
└── Casks/
    └── howlerops.rb
```

**Initial Cask Formula:** `Casks/howlerops.rb`

```ruby
cask "howlerops" do
  version "1.0.0"
  sha256 "INITIAL_SHA256_PLACEHOLDER"

  url "https://github.com/YOUR_USERNAME/sql-studio/releases/download/v#{version}/HowlerOps-darwin-universal.zip"
  name "HowlerOps"
  desc "A powerful desktop SQL client built with Wails"
  homepage "https://github.com/YOUR_USERNAME/sql-studio"

  livecheck do
    url :url
    strategy :github_latest
  end

  app "howlerops.app"

  zap trash: [
    "~/Library/Application Support/howlerops",
    "~/Library/Caches/howlerops",
    "~/Library/Preferences/com.howlerops.app.plist",
    "~/Library/Saved Application State/com.howlerops.app.savedState",
  ]
end
```

### Installation Command for Users

Once published, users can install with:
```bash
brew tap YOUR_USERNAME/howlerops
brew install --cask howlerops
```

Or as a single command:
```bash
brew install --cask YOUR_USERNAME/howlerops/howlerops
```

---

## Testing Strategy

### Phase 1: Local Testing (Before CI/CD)

1. **Test Local Build:**
   ```bash
   wails build -platform darwin/universal
   ls -lah build/bin/
   ```

2. **Test Signing (requires certificate):**
   ```bash
   codesign --deep --force --verify --verbose \
     --sign "Developer ID Application: YOUR NAME (TEAM_ID)" \
     --options runtime \
     build/bin/howlerops.app

   # Verify signature
   codesign -dv --verbose=4 build/bin/howlerops.app
   spctl -a -vv build/bin/howlerops.app
   ```

3. **Test Notarization:**
   ```bash
   # Create ZIP
   cd build/bin
   zip -r howlerops.zip howlerops.app

   # Submit for notarization
   xcrun notarytool submit howlerops.zip \
     --apple-id YOUR_APPLE_ID \
     --password YOUR_APP_PASSWORD \
     --team-id YOUR_TEAM_ID \
     --wait

   # Staple ticket
   xcrun stapler staple howlerops.app
   ```

4. **Test Installation:**
   - Copy app to `/Applications`
   - Launch from Finder
   - Verify no Gatekeeper warnings
   - Test all database connections

### Phase 2: CI/CD Testing

1. **Create Test Tag:**
   ```bash
   git tag v1.0.0-beta.1
   git push origin v1.0.0-beta.1
   ```

2. **Monitor Workflow:**
   - Check GitHub Actions tab
   - Review logs for each step
   - Download artifacts if build succeeds

3. **Test Pre-Release:**
   - Download from GitHub Releases
   - Install and test on clean Mac
   - Report any issues

### Phase 3: Homebrew Testing

1. **Test Local Formula:**
   ```bash
   brew install --cask --formula ./Casks/howlerops.rb
   ```

2. **Test from Tap:**
   ```bash
   brew tap YOUR_USERNAME/howlerops
   brew install --cask howlerops
   ```

3. **Test Uninstall:**
   ```bash
   brew uninstall --cask howlerops
   ```

---

## Versioning and Release Process

### Semantic Versioning

Use semantic versioning: `v1.0.0`, `v1.0.1`, `v1.1.0`, `v2.0.0`

- **Major (v2.0.0):** Breaking changes
- **Minor (v1.1.0):** New features, backward compatible
- **Patch (v1.0.1):** Bug fixes

### Release Workflow

1. **Prepare Release:**
   ```bash
   # Update version in wails.json
   vim wails.json  # Update info.productVersion

   # Update CHANGELOG.md (recommended)
   vim CHANGELOG.md

   # Commit changes
   git add wails.json CHANGELOG.md
   git commit -m "chore: bump version to 1.0.1"
   ```

2. **Create and Push Tag:**
   ```bash
   git tag v1.0.1
   git push origin main
   git push origin v1.0.1
   ```

3. **Automated Process Starts:**
   - GitHub Actions detects tag push
   - Workflow runs automatically
   - Release created on GitHub
   - Homebrew formula updated

4. **Verify Release:**
   - Check GitHub Releases page
   - Test Homebrew installation
   - Monitor for user issues

### Hotfix Process

For critical bugs:

1. Create hotfix branch from main
2. Fix bug and test
3. Bump patch version
4. Create tag and push
5. Workflow handles rest

---

## Monitoring and Maintenance

### What to Monitor

1. **GitHub Actions:**
   - Build success rate
   - Notarization wait times
   - Failed workflows

2. **Homebrew:**
   - Installation success (via analytics if enabled)
   - User-reported issues
   - Outdated dependencies

3. **User Feedback:**
   - GitHub Issues
   - Crash reports
   - Feature requests

### Regular Maintenance

**Monthly:**
- Review and update dependencies
- Check for outdated npm packages
- Update Go modules
- Review security advisories

**Quarterly:**
- Update Wails framework
- Review and update entitlements
- Test on latest macOS version
- Update documentation

**Yearly:**
- Renew Apple Developer membership ($99)
- Review analytics and usage
- Plan major version updates

---

## Troubleshooting Guide

### Common Issues and Solutions

#### 1. Notarization Fails

**Symptom:** `xcrun notarytool` returns error

**Solutions:**
- Verify app-specific password is correct
- Check bundle ID matches certificate
- Ensure hardened runtime is enabled
- Review entitlements for restrictive settings

**Debug Command:**
```bash
xcrun notarytool log <submission-id> \
  --apple-id YOUR_APPLE_ID \
  --password YOUR_APP_PASSWORD \
  --team-id YOUR_TEAM_ID
```

#### 2. Code Signing Fails

**Symptom:** `codesign` command fails

**Solutions:**
- Verify certificate is installed in keychain
- Check certificate hasn't expired
- Ensure correct certificate type (Developer ID Application)
- Verify bundle structure is correct

**Debug Command:**
```bash
security find-identity -v -p codesigning
codesign -dv --verbose=4 build/bin/howlerops.app
```

#### 3. GitHub Actions Fails to Access Secrets

**Symptom:** Workflow fails with authentication error

**Solutions:**
- Verify secrets are added to repository (not organization)
- Check secret names match exactly (case-sensitive)
- Re-create secrets if corrupted
- Verify base64 encoding is correct

#### 4. Homebrew Formula Installation Fails

**Symptom:** `brew install` fails

**Solutions:**
- Verify SHA256 checksum matches
- Check URL is accessible
- Ensure app name matches in Cask
- Test formula syntax: `brew audit --cask howlerops`

#### 5. App Opens with Gatekeeper Warning

**Symptom:** "App is damaged" or "can't be opened" message

**Solutions:**
- Verify app was properly notarized
- Check notarization ticket is stapled
- Ensure user didn't modify app bundle
- Re-sign and re-notarize if needed

**User Workaround (temporary):**
```bash
xattr -cr /Applications/howlerops.app
```

---

## Cost Analysis

### One-Time Costs

| Item | Cost | Notes |
|------|------|-------|
| Apple Developer Program | $99/year | Required for code signing |
| Domain (optional) | $10-15/year | For custom website |

### Ongoing Costs

| Item | Cost | Notes |
|------|------|-------|
| GitHub Actions | Free* | Free for public repos; $0.08/min macOS runners for private |
| Storage (GitHub Releases) | Free | Included with GitHub |
| Bandwidth | Free | Included with GitHub |

**Total Annual Cost:** $99-114/year (minimum)

**For Private Repos:** Typical release build ~30 minutes = $2.40/release

---

## Additional Features to Consider

### 1. Auto-Update Mechanism

Implement in-app updates using:
- **Sparkle Framework:** Popular macOS updater
- **Wails Auto-Updater:** Built-in capability
- **Custom Solution:** Check GitHub Releases API

**Benefits:**
- Users get updates automatically
- Higher adoption of new features
- Faster bug fix deployment

### 2. Crash Reporting

Integrate crash reporting:
- **Sentry:** Popular, free tier available
- **Bugsnag:** Alternative option
- **Custom:** Upload to your own server

### 3. Analytics (Optional)

Track usage metrics (with user consent):
- Installation source tracking
- Feature usage analytics
- Database type popularity
- Crash rates by version

### 4. Beta Distribution

Set up beta testing channel:
- Create `beta` Homebrew Cask
- Use GitHub pre-releases
- Gather feedback before stable release

---

## Success Metrics

### Key Performance Indicators

1. **Build Success Rate:** >95% of workflow runs succeed
2. **Release Frequency:** 1-2 releases per month
3. **Time to Release:** <45 minutes from tag push to Homebrew availability
4. **User Adoption:** Track downloads and Homebrew installs
5. **Issue Resolution Time:** <7 days for bugs, <30 days for features

### Launch Checklist

- [ ] Apple Developer account active
- [ ] All GitHub Secrets configured
- [ ] Workflow file created and tested
- [ ] Homebrew tap repository created
- [ ] Initial Cask formula published
- [ ] Documentation updated (README.md)
- [ ] Test on multiple macOS versions
- [ ] Announce release on social media/blog

---

## Next Steps

See accompanying files:
1. `wails-macos-user-actions.plan.md` - Step-by-step user action guide
2. `wails-macos-workflow.yml` - Complete GitHub Actions workflow
3. `homebrew-deployment-guide.plan.md` - Detailed Homebrew setup

**Start Here:** Follow the user actions guide first to set up prerequisites.
