# HowlerOps macOS Deployment - User Action Checklist

## Overview

This document outlines **ALL actions YOU need to take** to set up automated macOS app deployment. Follow these steps in order.

**Time Required:** 2-4 hours (mostly waiting for Apple processes)
**Difficulty:** Intermediate (requires some terminal comfort)

---

## Phase 1: Apple Developer Setup (Required)

### Step 1.1: Enroll in Apple Developer Program

**Cost:** $99/year
**Time:** 10-30 minutes + approval time (usually instant to 24 hours)

1. Go to [https://developer.apple.com/programs/enroll/](https://developer.apple.com/programs/enroll/)
2. Sign in with your Apple ID (or create one)
3. Click "Start Your Enrollment"
4. Choose entity type:
   - **Individual:** Fastest, uses your name
   - **Organization:** Requires D-U-N-S number and business verification
5. Accept agreements
6. Pay $99 via credit card
7. Wait for approval email

**Verification:**
- You should receive email: "Welcome to the Apple Developer Program"
- You can access [https://developer.apple.com/account](https://developer.apple.com/account)

---

### Step 1.2: Create Developer ID Application Certificate

**Purpose:** This certificate allows you to sign apps for distribution outside the Mac App Store.

**Steps:**

1. **Open Keychain Access on your Mac:**
   - Applications → Utilities → Keychain Access

2. **Request Certificate:**
   - Menu: Keychain Access → Certificate Assistant → Request a Certificate From a Certificate Authority
   - Enter your email address (same as Apple ID)
   - Common Name: "Your Name" or "Company Name"
   - Select: "Saved to disk"
   - Click "Continue"
   - Save as: `CertificateSigningRequest.certSigningRequest`

3. **Upload to Apple Developer:**
   - Go to [https://developer.apple.com/account/resources/certificates/list](https://developer.apple.com/account/resources/certificates/list)
   - Click the "+" button
   - Select "Developer ID Application"
   - Click "Continue"
   - Upload your `.certSigningRequest` file
   - Click "Continue"
   - Download the certificate (`.cer` file)

4. **Install Certificate:**
   - Double-click the downloaded `.cer` file
   - It will be added to your Keychain
   - Verify: Open Keychain Access → My Certificates
   - You should see: "Developer ID Application: Your Name (TEAM_ID)"

5. **Find Your Team ID:**
   ```bash
   # Run this command to see your certificate details:
   security find-identity -v -p codesigning
   ```

   Look for a line like:
   ```
   1) ABCDEF1234567890 "Developer ID Application: Your Name (ABC1234DEF)"
                        ^^^^^^^^^^ This is your TEAM_ID
   ```

**Save These Values:**
- Certificate Name: `Developer ID Application: Your Name (ABC1234DEF)`
- Team ID: `ABC1234DEF` (10 characters)

---

### Step 1.3: Export Certificate as P12 File

**Purpose:** This allows GitHub Actions to use your certificate.

**Steps:**

1. Open Keychain Access
2. Go to "My Certificates"
3. Find your "Developer ID Application" certificate
4. Right-click → Export "Developer ID Application: Your Name"
5. Save As: `apple-developer-id.p12`
6. **Set a strong password** (you'll need this later)
7. Save the file to a secure location

**Convert to Base64:**

```bash
# Navigate to where you saved the P12 file
cd ~/Downloads  # or wherever you saved it

# Convert to base64
base64 -i apple-developer-id.p12 -o apple-developer-id-base64.txt

# Display the base64 string (you'll copy this to GitHub)
cat apple-developer-id-base64.txt
```

**Save These Values:**
- P12 Password: `[the password you set]`
- Base64 String: `[contents of apple-developer-id-base64.txt]`

---

### Step 1.4: Create App-Specific Password

**Purpose:** Required for notarization (automated signing verification by Apple).

**Steps:**

1. Go to [https://appleid.apple.com/account/manage](https://appleid.apple.com/account/manage)
2. Sign in with your Apple ID
3. Navigate to "Security" section
4. Under "App-Specific Passwords", click "Generate Password"
5. Label it: "GitHub Actions Notarization"
6. Click "Create"
7. **Copy the password immediately** (it won't be shown again)
   - Format: `abcd-efgh-ijkl-mnop`

**Save This Value:**
- App-Specific Password: `abcd-efgh-ijkl-mnop`

---

## Phase 2: GitHub Configuration

### Step 2.1: Add GitHub Secrets

**Steps:**

1. Go to your GitHub repository: `https://github.com/YOUR_USERNAME/sql-studio`
2. Click "Settings" tab
3. In left sidebar: "Secrets and variables" → "Actions"
4. Click "New repository secret" for each of the following:

**Add These Secrets:**

| Secret Name | Value | Source |
|-------------|-------|--------|
| `APPLE_DEVELOPER_ID` | Your Apple ID email | Your Apple account |
| `APPLE_APP_PASSWORD` | App-specific password | From Step 1.4 |
| `APPLE_TEAM_ID` | 10-character team ID | From Step 1.2 |
| `APPLE_CERTIFICATE_P12` | Base64 certificate string | From Step 1.3 |
| `APPLE_CERTIFICATE_PASSWORD` | P12 export password | From Step 1.3 |

**Example:**
```
Name: APPLE_DEVELOPER_ID
Value: jacob@example.com

Name: APPLE_APP_PASSWORD
Value: abcd-efgh-ijkl-mnop

Name: APPLE_TEAM_ID
Value: ABC1234DEF

Name: APPLE_CERTIFICATE_P12
Value: MIIKPAIBAzCCCf... (very long base64 string)

Name: APPLE_CERTIFICATE_PASSWORD
Value: YourStrongPassword123
```

**Verification:**
- You should see 5 secrets listed under "Repository secrets"
- Green checkmarks indicate they're saved

---

### Step 2.2: Create Homebrew Tap Repository

**Purpose:** This is where users will install your app from using Homebrew.

**Steps:**

1. **Create New GitHub Repository:**
   - Go to [https://github.com/new](https://github.com/new)
   - Repository name: `homebrew-howlerops` (or `homebrew-sqlstudio`)
     - **IMPORTANT:** Must start with `homebrew-`
   - Description: "Homebrew tap for HowlerOps"
   - Public repository
   - Initialize with README
   - Click "Create repository"

2. **Create Casks Directory:**
   ```bash
   # Clone your new repository
   git clone https://github.com/YOUR_USERNAME/homebrew-howlerops.git
   cd homebrew-howlerops

   # Create Casks directory
   mkdir Casks

   # Create initial Cask formula
   cat > Casks/howlerops.rb << 'EOF'
cask "howlerops" do
  version "1.0.0"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"

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
EOF

   # Replace YOUR_USERNAME with your actual GitHub username
   sed -i '' 's/YOUR_USERNAME/your-actual-username/g' Casks/howlerops.rb

   # Commit and push
   git add Casks/howlerops.rb
   git commit -m "feat: add initial HowlerOps cask formula"
   git push origin main
   ```

3. **Update README:**
   ```bash
   cat > README.md << 'EOF'
# Homebrew Tap for HowlerOps

## Installation

```bash
brew tap YOUR_USERNAME/howlerops
brew install --cask howlerops
```

Or install in one command:

```bash
brew install --cask YOUR_USERNAME/howlerops/howlerops
```

## Updating

```bash
brew upgrade --cask howlerops
```

## Uninstalling

```bash
brew uninstall --cask howlerops
```
EOF

   # Replace YOUR_USERNAME
   sed -i '' 's/YOUR_USERNAME/your-actual-username/g' README.md

   git add README.md
   git commit -m "docs: add installation instructions"
   git push origin main
   ```

---

### Step 2.3: Create Personal Access Token for Homebrew Updates

**Purpose:** Allows GitHub Actions to automatically update your Homebrew formula.

**Steps:**

1. Go to [https://github.com/settings/tokens](https://github.com/settings/tokens)
2. Click "Generate new token" → "Generate new token (classic)"
3. Note: "GitHub Actions - Homebrew Updates"
4. Expiration: "No expiration" (or set to 1 year)
5. Select scopes:
   - ✅ `repo` (all repo permissions)
   - ✅ `workflow`
6. Click "Generate token"
7. **Copy the token immediately** (it won't be shown again)
   - Format: `ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

8. **Add to GitHub Secrets:**
   - Go to `sql-studio` repository settings
   - Secrets and variables → Actions
   - New repository secret:
     - Name: `HOMEBREW_TAP_TOKEN`
     - Value: `ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

---

## Phase 3: Update Project Files

### Step 3.1: Update `wails.json` Configuration

**File:** `/Users/jacob_1/projects/sql-studio/wails.json`

**Changes Needed:**

1. **Update signing identity (line 57):**
   ```json
   "signing": {
     "identity": "Developer ID Application: Your Name (ABC1234DEF)",
     "embedProvisioningProfile": false
   }
   ```
   Replace with your actual certificate name from Step 1.2.

2. **Add hardened runtime entitlement (line 65):**
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
   Add the last line if not present (may be needed for database drivers).

**Command to edit:**
```bash
cd /Users/jacob_1/projects/sql-studio
code wails.json  # or vim, nano, etc.
```

---

### Step 3.2: Create GitHub Actions Workflow

**File:** `.github/workflows/release-macos.yml`

This file will be created separately (see Step 3.3).

---

## Phase 4: Test Locally (Before Pushing)

### Step 4.1: Test Build

```bash
cd /Users/jacob_1/projects/sql-studio

# Build for macOS
wails build -platform darwin/universal

# Check output
ls -lah build/bin/
# Should see: howlerops.app
```

---

### Step 4.2: Test Code Signing

```bash
# Sign the app
codesign --deep --force --verify --verbose \
  --sign "Developer ID Application: Your Name (ABC1234DEF)" \
  --options runtime \
  build/bin/howlerops.app

# Verify signature
codesign -dv --verbose=4 build/bin/howlerops.app

# Should show:
# Authority=Developer ID Application: Your Name (ABC1234DEF)
# Sealed Resources...
```

---

### Step 4.3: Test Notarization (Optional but Recommended)

```bash
cd build/bin

# Create ZIP
zip -r howlerops.zip howlerops.app

# Submit for notarization
xcrun notarytool submit howlerops.zip \
  --apple-id YOUR_APPLE_ID \
  --password YOUR_APP_PASSWORD \
  --team-id YOUR_TEAM_ID \
  --wait

# Should see: "status: Accepted"

# Staple the notarization ticket
xcrun stapler staple howlerops.app

# Verify
spctl -a -vv howlerops.app
# Should show: "accepted"
```

---

## Phase 5: First Release

### Step 5.1: Commit and Push Workflow

```bash
cd /Users/jacob_1/projects/sql-studio

# Add the workflow file (created in next steps)
git add .github/workflows/release-macos.yml
git add wails.json  # if you updated it

git commit -m "ci: add macOS release workflow with code signing and notarization"
git push origin main
```

---

### Step 5.2: Create First Release Tag

```bash
# Make sure version in wails.json is 1.0.0

# Create and push tag
git tag v1.0.0
git push origin v1.0.0
```

**What Happens Next:**
1. GitHub Actions workflow starts automatically
2. Builds macOS app
3. Signs with your certificate
4. Notarizes with Apple
5. Creates GitHub Release
6. Updates Homebrew formula

**Monitor Progress:**
- Go to: `https://github.com/YOUR_USERNAME/sql-studio/actions`
- Click on the running workflow
- Watch each step complete

---

### Step 5.3: Verify Release

1. **Check GitHub Release:**
   - Go to: `https://github.com/YOUR_USERNAME/sql-studio/releases`
   - Should see: "v1.0.0" release with attached ZIP file

2. **Check Homebrew Tap:**
   - Go to: `https://github.com/YOUR_USERNAME/homebrew-howlerops`
   - Check `Casks/howlerops.rb`
   - Should see updated version and SHA256

3. **Test Homebrew Installation:**
   ```bash
   # Install from your tap
   brew tap YOUR_USERNAME/howlerops
   brew install --cask howlerops

   # Launch the app
   open /Applications/howlerops.app

   # Should launch without any warnings
   ```

---

## Phase 6: Subsequent Releases

For future releases:

```bash
# 1. Update version in wails.json
vim wails.json
# Change: "productVersion": "1.0.1"

# 2. Commit changes
git add wails.json
git commit -m "chore: bump version to 1.0.1"
git push origin main

# 3. Create and push tag
git tag v1.0.1
git push origin v1.0.1

# 4. Automation handles the rest!
```

---

## Troubleshooting

### Issue: Certificate Not Found

**Error:** `codesign: no identity found`

**Solution:**
```bash
# List available certificates
security find-identity -v -p codesigning

# If empty, re-import your certificate
open ~/Downloads/apple-developer-id.p12
```

---

### Issue: Notarization Fails

**Error:** `status: Invalid`

**Solution:**
```bash
# Get detailed log
xcrun notarytool log SUBMISSION_ID \
  --apple-id YOUR_APPLE_ID \
  --password YOUR_APP_PASSWORD \
  --team-id YOUR_TEAM_ID

# Common fixes:
# 1. Ensure hardened runtime is enabled
# 2. Check entitlements are correct
# 3. Verify bundle ID matches certificate
```

---

### Issue: GitHub Actions Fails

**Error:** Various authentication or build errors

**Solutions:**
1. Check all secrets are added correctly
2. Verify base64 certificate is complete (no truncation)
3. Review workflow logs for specific error
4. Re-run workflow: GitHub Actions UI → "Re-run jobs"

---

### Issue: Homebrew Installation Fails

**Error:** `SHA256 mismatch`

**Solution:**
```bash
# Get correct SHA256
shasum -a 256 HowlerOps-darwin-universal.zip

# Update in Casks/howlerops.rb
# Commit and push
```

---

## Security Best Practices

### Certificate Security

- ✅ Store P12 file securely (encrypted drive)
- ✅ Use strong password for P12
- ✅ Don't commit P12 or certificates to git
- ✅ Rotate app-specific password yearly
- ✅ Use different Apple ID for development vs distribution (optional)

### GitHub Secrets

- ✅ Never commit secrets to git
- ✅ Use repository secrets (not organization for sensitive data)
- ✅ Rotate tokens annually
- ✅ Review secret access logs periodically

### Access Control

- ✅ Limit who can push tags (creates releases)
- ✅ Require code review before merging to main
- ✅ Enable branch protection on main
- ✅ Use 2FA on Apple ID and GitHub

---

## Quick Reference

### Commands You'll Use Often

```bash
# Build locally
wails build -platform darwin/universal

# Create release
git tag v1.0.X
git push origin v1.0.X

# Update Homebrew formula manually
cd ~/path/to/homebrew-howlerops
vim Casks/howlerops.rb
git add Casks/howlerops.rb
git commit -m "chore: update to v1.0.X"
git push origin main

# Test Homebrew installation
brew uninstall --cask howlerops
brew reinstall --cask howlerops
```

---

## URLs to Bookmark

- Apple Developer: [https://developer.apple.com/account](https://developer.apple.com/account)
- GitHub Actions: [https://github.com/YOUR_USERNAME/sql-studio/actions](https://github.com/YOUR_USERNAME/sql-studio/actions)
- GitHub Releases: [https://github.com/YOUR_USERNAME/sql-studio/releases](https://github.com/YOUR_USERNAME/sql-studio/releases)
- Homebrew Tap: [https://github.com/YOUR_USERNAME/homebrew-howlerops](https://github.com/YOUR_USERNAME/homebrew-howlerops)
- Apple ID: [https://appleid.apple.com](https://appleid.apple.com)

---

## Checklist Summary

Use this checklist to track your progress:

### Apple Setup
- [ ] Enrolled in Apple Developer Program ($99 paid)
- [ ] Created Developer ID Application certificate
- [ ] Exported certificate as P12 file
- [ ] Converted P12 to base64
- [ ] Created app-specific password
- [ ] Saved Team ID

### GitHub Setup
- [ ] Added `APPLE_DEVELOPER_ID` secret
- [ ] Added `APPLE_APP_PASSWORD` secret
- [ ] Added `APPLE_TEAM_ID` secret
- [ ] Added `APPLE_CERTIFICATE_P12` secret
- [ ] Added `APPLE_CERTIFICATE_PASSWORD` secret
- [ ] Created Homebrew tap repository
- [ ] Created Personal Access Token
- [ ] Added `HOMEBREW_TAP_TOKEN` secret

### Code Changes
- [ ] Updated `wails.json` signing identity
- [ ] Added hardened runtime entitlements
- [ ] Created `.github/workflows/release-macos.yml`

### Testing
- [ ] Tested local build
- [ ] Tested code signing locally
- [ ] Tested notarization locally (optional)
- [ ] Created first release tag
- [ ] Verified GitHub Actions workflow succeeded
- [ ] Verified GitHub Release created
- [ ] Tested Homebrew installation

### Launch
- [ ] Pushed v1.0.0 tag
- [ ] Verified release published
- [ ] Tested user installation flow
- [ ] Updated documentation
- [ ] Announced release

---

## Getting Help

If you encounter issues:

1. Check troubleshooting section above
2. Review GitHub Actions logs
3. Check Apple Developer forums
4. Review Wails documentation: [https://wails.io](https://wails.io)
5. Open GitHub issue in sql-studio repo

---

## Next Steps

After completing this checklist:
1. Review the GitHub Actions workflow file
2. Customize DMG appearance (optional)
3. Set up auto-update mechanism (future enhancement)
4. Add crash reporting (future enhancement)
5. Plan your release cadence (monthly, bi-weekly, etc.)
