# HowlerOps macOS Deployment - Quick Start Guide

## 🚀 Overview

This guide helps you set up automated deployment for your Wails macOS application (HowlerOps/SQL Studio) with:
- ✅ Code signing and notarization
- ✅ GitHub Actions CI/CD
- ✅ Homebrew distribution
- ✅ Automated releases on git tag push

**Total Setup Time:** 2-4 hours
**Cost:** $99/year (Apple Developer Program)

---

## 📁 Documentation Files

Your comprehensive deployment plan includes:

1. **[wails-macos-cicd-deployment.plan.md](./wails-macos-cicd-deployment.plan.md)**
   - Complete technical architecture
   - Code changes required
   - Testing strategy
   - Troubleshooting guide

2. **[wails-macos-user-actions.plan.md](./wails-macos-user-actions.plan.md)**
   - Step-by-step action items for YOU
   - Apple Developer setup
   - GitHub configuration
   - First release walkthrough

3. **[.github/workflows/release-macos.yml](../.github/workflows/release-macos.yml)**
   - Ready-to-use GitHub Actions workflow
   - Handles build, sign, notarize, release
   - Automatically updates Homebrew formula

---

## ⚡ Quick Start (30-Second Summary)

```bash
# 1. Prerequisites (one-time setup, ~2 hours)
- Enroll in Apple Developer Program ($99/year)
- Create Developer ID Application certificate
- Export certificate as P12 and convert to base64
- Create app-specific password
- Add 6 secrets to GitHub
- Create Homebrew tap repository

# 2. Code Changes (10 minutes)
- Update wails.json signing identity
- Customize .github/workflows/release-macos.yml (replace YOUR_USERNAME)

# 3. First Release (5 minutes)
git tag v1.0.0
git push origin v1.0.0

# 4. Done!
# GitHub Actions automatically:
# - Builds universal binary
# - Signs with your certificate
# - Notarizes with Apple
# - Creates GitHub Release
# - Updates Homebrew formula

# Users can install with:
brew tap YOUR_USERNAME/howlerops
brew install --cask howlerops
```

---

## 📋 Your Action Checklist

### Phase 1: Apple Setup (Required First)

**Read:** [wails-macos-user-actions.plan.md](./wails-macos-user-actions.plan.md) - Sections 1.1-1.4

- [ ] Enroll in Apple Developer Program at [developer.apple.com](https://developer.apple.com/programs/enroll/)
- [ ] Create Developer ID Application certificate
- [ ] Export certificate as P12 with password
- [ ] Convert P12 to base64: `base64 -i apple-developer-id.p12 -o apple-developer-id-base64.txt`
- [ ] Create app-specific password at [appleid.apple.com](https://appleid.apple.com/account/manage)
- [ ] Save your Team ID (10 characters, found in certificate)

**Time:** 1-2 hours (including Apple approval wait time)

---

### Phase 2: GitHub Configuration

**Read:** [wails-macos-user-actions.plan.md](./wails-macos-user-actions.plan.md) - Sections 2.1-2.3

- [ ] Add GitHub Secrets (Settings → Secrets and variables → Actions):
  - [ ] `APPLE_DEVELOPER_ID` - Your Apple ID email
  - [ ] `APPLE_APP_PASSWORD` - App-specific password
  - [ ] `APPLE_TEAM_ID` - 10-character team ID
  - [ ] `APPLE_CERTIFICATE_P12` - Base64 certificate string
  - [ ] `APPLE_CERTIFICATE_PASSWORD` - P12 export password
  - [ ] `HOMEBREW_TAP_TOKEN` - GitHub Personal Access Token

- [ ] Create Homebrew tap repository:
  ```bash
  # Create new repo on GitHub: homebrew-howlerops
  git clone https://github.com/YOUR_USERNAME/homebrew-howlerops.git
  cd homebrew-howlerops
  mkdir Casks
  # Add initial Cask formula (see user actions guide)
  ```

**Time:** 30 minutes

---

### Phase 3: Update Project Files

**Read:** [wails-macos-cicd-deployment.plan.md](./wails-macos-cicd-deployment.plan.md) - "Code Changes Required" section

- [ ] Update `wails.json` (line 57):
  ```json
  "signing": {
    "identity": "Developer ID Application: Your Name (TEAM_ID)",
    "embedProvisioningProfile": false
  }
  ```

- [ ] Customize `.github/workflows/release-macos.yml`:
  - Replace all instances of `YOUR_USERNAME` with your GitHub username
  - Verify `APP_NAME` and `BUNDLE_ID` match your app

- [ ] (Optional) Add DMG background image at `build/dmg-background.png`

**Time:** 10 minutes

---

### Phase 4: Test Locally (Recommended)

**Read:** [wails-macos-user-actions.plan.md](./wails-macos-user-actions.plan.md) - Section 4

```bash
# Test build
wails build -platform darwin/universal

# Test signing
codesign --deep --force --verify --verbose \
  --sign "Developer ID Application: Your Name (TEAM_ID)" \
  --options runtime \
  build/bin/howlerops.app

# Verify signature
codesign -dv --verbose=4 build/bin/howlerops.app
```

**Time:** 15 minutes

---

### Phase 5: First Release

**Read:** [wails-macos-user-actions.plan.md](./wails-macos-user-actions.plan.md) - Section 5

```bash
# Commit workflow file
git add .github/workflows/release-macos.yml wails.json
git commit -m "ci: add macOS release automation"
git push origin main

# Create and push first release tag
git tag v1.0.0
git push origin v1.0.0

# Monitor at: https://github.com/YOUR_USERNAME/sql-studio/actions
```

**What Happens:**
1. GitHub Actions workflow triggers on tag push
2. Builds universal binary (Intel + Apple Silicon)
3. Signs with your Developer ID
4. Submits to Apple for notarization (~2-10 min wait)
5. Creates GitHub Release with ZIP and DMG
6. Updates Homebrew formula automatically

**Time:** 5 minutes + 30-45 minutes automated build time

---

## 🎯 Expected Results

After successful release:

### GitHub Release
- Visit: `https://github.com/YOUR_USERNAME/sql-studio/releases`
- Should see: `v1.0.0` release with:
  - `howlerops-darwin-universal.zip`
  - `howlerops-darwin-universal.dmg`
  - `checksums.txt`
  - Auto-generated release notes

### Homebrew Formula
- Visit: `https://github.com/YOUR_USERNAME/homebrew-howlerops/blob/main/Casks/howlerops.rb`
- Should see: Updated version and SHA256

### User Installation
```bash
brew tap YOUR_USERNAME/howlerops
brew install --cask howlerops

# App launches without warnings
open /Applications/howlerops.app
```

---

## 🔄 Subsequent Releases

For future releases:

```bash
# 1. Make your changes, commit, push to main
git add .
git commit -m "feat: add new feature"
git push origin main

# 2. Update version in wails.json
vim wails.json
# Change: "productVersion": "1.0.1"

# 3. Commit version bump
git add wails.json
git commit -m "chore: bump version to 1.0.1"
git push origin main

# 4. Create and push tag
git tag v1.0.1
git push origin v1.0.1

# 5. Automation handles everything else!
```

**That's it!** GitHub Actions will:
- Build, sign, notarize
- Create release
- Update Homebrew
- Notify you of success/failure

---

## 🐛 Troubleshooting

### Common Issues

**Issue:** GitHub Actions fails with "certificate not found"
**Fix:** Verify all 5 Apple secrets are added correctly

**Issue:** Notarization fails with "Invalid"
**Fix:** Check that:
- Bundle ID in `wails.json` matches certificate
- Hardened runtime is enabled (in workflow)
- Entitlements are correct

**Issue:** Homebrew installation fails with "SHA256 mismatch"
**Fix:** Workflow should auto-update SHA256, but if not:
```bash
shasum -a 256 howlerops-darwin-universal.zip
# Update in Casks/howlerops.rb manually
```

**Issue:** App shows "damaged" warning
**Fix:** Ensure notarization succeeded and ticket was stapled

**Full troubleshooting guide:** See [wails-macos-cicd-deployment.plan.md](./wails-macos-cicd-deployment.plan.md) - "Troubleshooting Guide" section

---

## 📚 Deep Dive Resources

For detailed information on any step:

1. **Technical Architecture & Strategy**
   - Read: [wails-macos-cicd-deployment.plan.md](./wails-macos-cicd-deployment.plan.md)
   - Covers: Architecture, code changes, testing, monitoring

2. **Step-by-Step Actions**
   - Read: [wails-macos-user-actions.plan.md](./wails-macos-user-actions.plan.md)
   - Covers: Every action you need to take with commands

3. **Workflow Implementation**
   - Read: [.github/workflows/release-macos.yml](../.github/workflows/release-macos.yml)
   - Fully commented workflow with 13 stages

---

## 💰 Cost Breakdown

| Item | Cost | Frequency |
|------|------|-----------|
| Apple Developer Program | $99 | Annual |
| GitHub Actions (public repo) | Free | N/A |
| GitHub Actions (private repo) | ~$2.40 | Per release |
| GitHub Releases Storage | Free | N/A |
| Homebrew Distribution | Free | N/A |

**Total for Public Repo:** $99/year
**Total for Private Repo:** $99/year + ~$2.40/release

---

## ⏱️ Timeline Estimate

| Phase | Time | Can Parallelize? |
|-------|------|------------------|
| Apple Developer enrollment | 10 min + wait time | No |
| Certificate creation & export | 20 min | No |
| GitHub secrets setup | 15 min | Yes |
| Homebrew tap creation | 20 min | Yes |
| Code updates | 10 min | Yes |
| Local testing (optional) | 30 min | No |
| First release trigger | 5 min | No |
| Automated build & release | 30-45 min | Automatic |

**Total Active Time:** 2-3 hours
**Total Wall Time:** 3-4 hours (including waits)

---

## ✅ Success Criteria

You'll know you're successful when:

1. ✅ GitHub Actions workflow completes without errors
2. ✅ GitHub Release is created with ZIP and DMG files
3. ✅ Homebrew formula is automatically updated
4. ✅ You can install with: `brew install --cask YOUR_USERNAME/howlerops/howlerops`
5. ✅ App launches without Gatekeeper warnings
6. ✅ App is properly signed: `codesign -dv /Applications/howlerops.app`
7. ✅ App is notarized: `spctl -a -vv /Applications/howlerops.app` shows "accepted"

---

## 🚦 Next Steps

**Start here:**

1. **Read this document completely** (you are here ✓)
2. **Begin Phase 1:** [Apple Setup](./wails-macos-user-actions.plan.md#phase-1-apple-developer-setup-required)
3. **Continue to Phase 2:** [GitHub Configuration](./wails-macos-user-actions.plan.md#phase-2-github-configuration)
4. **Complete remaining phases** following the user actions guide

**Recommended order:**
1. Complete all Apple setup first (longest wait times)
2. Set up GitHub while waiting for Apple approvals
3. Update code files
4. Test locally
5. Push first release

---

## 📞 Getting Help

If you encounter issues:

1. Check the [troubleshooting section](#-troubleshooting) above
2. Review GitHub Actions logs in detail
3. Consult the comprehensive guides in [wails-macos-cicd-deployment.plan.md](./wails-macos-cicd-deployment.plan.md)
4. Check Apple Developer forums for notarization issues
5. Review Wails documentation: [https://wails.io](https://wails.io)

---

## 🎉 After Successful Launch

Once your automated deployment is working:

1. **Announce the release:**
   - Update README.md with Homebrew installation instructions
   - Post on social media / Product Hunt / Hacker News
   - Notify existing users

2. **Monitor:**
   - GitHub Actions success rate
   - User-reported installation issues
   - Download statistics

3. **Maintain:**
   - Monthly: Review dependencies and security advisories
   - Quarterly: Test on latest macOS version
   - Yearly: Renew Apple Developer membership

4. **Enhance (optional):**
   - Add auto-update mechanism
   - Implement crash reporting
   - Set up analytics (with user consent)
   - Create beta testing channel

---

## 📖 File Reference

All documentation is organized in `.cursor/plans/`:

```
.cursor/plans/
├── DEPLOYMENT-QUICK-START.md          ← You are here
├── wails-macos-cicd-deployment.plan.md   ← Technical details
└── wails-macos-user-actions.plan.md      ← Step-by-step actions

.github/workflows/
└── release-macos.yml                     ← CI/CD workflow
```

---

**Ready to start?** Head to [wails-macos-user-actions.plan.md](./wails-macos-user-actions.plan.md) and begin with Phase 1: Apple Developer Setup!

Good luck! 🚀
