# macOS Code Signing and Notarization Setup Guide

This guide walks you through setting up Apple code signing and notarization for HowlerOps releases. Once configured, users will be able to install and run HowlerOps without seeing macOS Gatekeeper warnings.

## Prerequisites

- Apple Developer Program membership ($99/year) - [Enroll here](https://developer.apple.com/programs/)
- macOS with Xcode Command Line Tools installed
- Access to the HowlerOps GitHub repository settings

## Overview

The release workflow requires these GitHub secrets:

| Secret Name | Description |
|-------------|-------------|
| `APPLE_CERTIFICATE_P12` | Base64-encoded Developer ID Application certificate |
| `APPLE_CERTIFICATE_PASSWORD` | Password for the .p12 certificate file |
| `APPLE_TEAM_ID` | Your 10-character Apple Team ID |
| `APPLE_DEVELOPER_ID` | Your Apple ID email address |
| `APPLE_APP_PASSWORD` | App-specific password for notarization |

## Step 1: Enroll in Apple Developer Program

1. Go to [developer.apple.com/programs](https://developer.apple.com/programs/)
2. Click "Enroll" and follow the enrollment process
3. Pay the $99/year fee
4. Wait for enrollment approval (usually 24-48 hours)

## Step 2: Create a Developer ID Application Certificate

### Option A: Using Xcode (Recommended)

1. Open Xcode
2. Go to **Xcode > Settings > Accounts**
3. Select your Apple ID and click **Manage Certificates**
4. Click the **+** button and select **Developer ID Application**
5. Xcode will create and install the certificate automatically

### Option B: Using Apple Developer Portal

1. Go to [developer.apple.com/account/resources/certificates](https://developer.apple.com/account/resources/certificates/list)
2. Click the **+** button to create a new certificate
3. Select **Developer ID Application**
4. Follow the instructions to create a Certificate Signing Request (CSR)
5. Upload the CSR and download the certificate
6. Double-click to install in Keychain

## Step 3: Export Certificate as .p12 File

1. Open **Keychain Access** (Applications > Utilities > Keychain Access)
2. In the left sidebar, select **login** keychain and **My Certificates** category
3. Find your **Developer ID Application** certificate
4. Right-click and select **Export "Developer ID Application: Your Name"...**
5. Choose a save location and select **.p12** format
6. Create a strong password (you'll need this for `APPLE_CERTIFICATE_PASSWORD`)
7. Save the file

## Step 4: Base64 Encode the Certificate

```bash
# Encode the certificate
base64 -i /path/to/your-certificate.p12 | pbcopy

# The encoded certificate is now in your clipboard
# This is your APPLE_CERTIFICATE_P12 value
```

Or without clipboard:
```bash
base64 -i /path/to/your-certificate.p12 > certificate-base64.txt
cat certificate-base64.txt
```

## Step 5: Find Your Team ID

1. Go to [developer.apple.com/account](https://developer.apple.com/account)
2. Scroll down to **Membership details**
3. Your **Team ID** is a 10-character alphanumeric string (e.g., `ABC1234567`)

Or using command line:
```bash
security find-identity -v -p codesigning | grep "Developer ID Application"
# Output: 1) ABC1234567 "Developer ID Application: Your Name (ABC1234567)"
#         The Team ID is the string in parentheses
```

## Step 6: Create an App-Specific Password

App-specific passwords are required for notarization (not your main Apple ID password).

1. Go to [appleid.apple.com](https://appleid.apple.com)
2. Sign in with your Apple ID
3. Go to **Sign-In and Security > App-Specific Passwords**
4. Click **Generate an app-specific password**
5. Enter a label (e.g., "HowlerOps Notarization")
6. Copy the generated password (format: `xxxx-xxxx-xxxx-xxxx`)
7. This is your `APPLE_APP_PASSWORD` value

## Step 7: Add Secrets to GitHub

1. Go to your GitHub repository: `github.com/jbeck018/howlerops`
2. Navigate to **Settings > Secrets and variables > Actions**
3. Click **New repository secret** for each secret:

### APPLE_CERTIFICATE_P12
- Name: `APPLE_CERTIFICATE_P12`
- Value: The base64-encoded certificate from Step 4

### APPLE_CERTIFICATE_PASSWORD
- Name: `APPLE_CERTIFICATE_PASSWORD`
- Value: The password you set when exporting the .p12 file

### APPLE_TEAM_ID
- Name: `APPLE_TEAM_ID`
- Value: Your 10-character Team ID from Step 5

### APPLE_DEVELOPER_ID
- Name: `APPLE_DEVELOPER_ID`
- Value: Your Apple ID email address

### APPLE_APP_PASSWORD
- Name: `APPLE_APP_PASSWORD`
- Value: The app-specific password from Step 6

## Step 8: Verify Configuration

After adding all secrets, create a test release to verify the configuration:

```bash
# Create a test tag
git tag v0.0.1-test
git push origin v0.0.1-test
```

Monitor the GitHub Actions workflow for any errors. Common issues:

### Certificate Import Fails
- Ensure the base64 encoding has no line breaks
- Verify the password is correct
- Check that the certificate type is "Developer ID Application"

### Notarization Fails
- Verify app-specific password is correct
- Ensure Team ID matches the certificate
- Check Apple ID email is correct

### Signing Fails
- Ensure entitlements.plist exists at `build/darwin/entitlements.plist`
- Verify the certificate is not expired
- Check the signing identity name matches

## Local Testing

You can test code signing locally before pushing:

```bash
# Build the app
wails build -platform darwin/universal -clean

# Sign manually (replace with your identity)
codesign --deep --force --verify --verbose \
  --sign "Developer ID Application: Your Name (TEAMID)" \
  --options runtime \
  --entitlements build/darwin/entitlements.plist \
  --timestamp \
  build/bin/howlerops.app

# Verify signature
codesign -dv --verbose=4 build/bin/howlerops.app

# Check Gatekeeper approval
spctl -a -vvv build/bin/howlerops.app
```

## Notarization Local Test

```bash
# Create a ZIP for notarization
cd build/bin
zip -r howlerops.app.zip howlerops.app

# Submit for notarization
xcrun notarytool submit howlerops.app.zip \
  --apple-id "your@email.com" \
  --team-id "TEAMID" \
  --password "app-specific-password" \
  --wait

# Staple the notarization ticket
xcrun stapler staple howlerops.app

# Verify stapling
xcrun stapler validate howlerops.app
```

## Troubleshooting

### "Developer ID Application certificate not found"
Your certificate may not be installed or may have expired:
```bash
security find-identity -v -p codesigning
```
If no "Developer ID Application" certificate appears, reinstall from Apple Developer portal.

### "The signature is invalid" or "code object is not signed at all"
The app wasn't signed properly. Check:
- All nested frameworks are signed
- The `--deep` flag was used
- Entitlements file exists and is valid

### "Unable to notarize: Invalid credentials"
- Verify app-specific password is correct (not your main password)
- Ensure Apple ID has accepted latest terms at developer.apple.com
- Check Team ID matches certificate

### "Notarization failed: The executable does not have the hardened runtime enabled"
Ensure `--options runtime` is included in codesign command.

### Certificate Expiration
Developer ID Application certificates expire after 5 years. To renew:
1. Create a new certificate in Apple Developer portal
2. Export as .p12 and update GitHub secrets
3. Revoke the old certificate

## Security Best Practices

1. **Never commit certificates or passwords** to the repository
2. **Use GitHub secrets** for all sensitive values
3. **Rotate app-specific passwords** periodically
4. **Monitor certificate expiration** dates
5. **Limit secret access** to necessary team members only

## Certificate Renewal Checklist

When your certificate expires:
- [ ] Generate new Developer ID Application certificate
- [ ] Export as .p12 with strong password
- [ ] Base64 encode the new certificate
- [ ] Update `APPLE_CERTIFICATE_P12` secret
- [ ] Update `APPLE_CERTIFICATE_PASSWORD` if changed
- [ ] Test with a new release
- [ ] Revoke the old certificate in Apple Developer portal

## Additional Resources

- [Apple Developer Documentation: Notarizing macOS Software](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)
- [Wails Code Signing Documentation](https://wails.io/docs/guides/signing)
- [GitHub Actions Encrypted Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)

---

**Last Updated**: 2024-12-01
**Maintained By**: HowlerOps Team
