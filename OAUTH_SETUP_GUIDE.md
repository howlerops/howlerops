# OAuth Provider Setup Guide

This guide walks you through setting up OAuth authentication with Google and GitHub for HowlerOps.

## Prerequisites

- HowlerOps OAuth and WebAuthn implementation (already completed)
- Access to Google Cloud Console (for Google OAuth)
- GitHub account with developer access (for GitHub OAuth)

## Part 1: Google OAuth Setup

### 1.1 Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Select a project" → "New Project"
3. Enter project name: "HowlerOps"
4. Click "Create"

### 1.2 Configure OAuth Consent Screen

1. Navigate to "APIs & Services" → "OAuth consent screen"
2. Select "External" user type
3. Fill in required fields:
   - **App name**: HowlerOps
   - **User support email**: Your email
   - **Developer contact email**: Your email
4. Click "Save and Continue"
5. Skip "Scopes" (default scopes are sufficient)
6. Add test users if needed (for development)
7. Click "Save and Continue"

### 1.3 Create OAuth Credentials

1. Navigate to "APIs & Services" → "Credentials"
2. Click "Create Credentials" → "OAuth client ID"
3. Select application type: "Desktop app"
4. Name: "HowlerOps Desktop"
5. Click "Create"
6. **Save the credentials**:
   - Client ID: `YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com`
   - Client Secret: `YOUR_GOOGLE_CLIENT_SECRET`

### 1.4 Configure Authorized Redirect URIs

1. Edit the OAuth client you just created
2. Add authorized redirect URI:
   ```
   howlerops://auth/callback
   ```
3. Also add for local development:
   ```
   http://localhost:34115
   ```
4. Click "Save"

## Part 2: GitHub OAuth Setup

### 2.1 Create GitHub OAuth App

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "OAuth Apps" → "New OAuth App"
3. Fill in the form:
   - **Application name**: HowlerOps
   - **Homepage URL**: `https://github.com/yourusername/howlerops` (or your project URL)
   - **Application description**: Desktop SQL client with OAuth support
   - **Authorization callback URL**: `howlerops://auth/callback`
4. Click "Register application"

### 2.2 Generate Client Secret

1. On the OAuth app page, click "Generate a new client secret"
2. **Save the credentials**:
   - Client ID: `YOUR_GITHUB_CLIENT_ID`
   - Client Secret: `YOUR_GITHUB_CLIENT_SECRET` (only shown once!)

## Part 3: Configure HowlerOps

### 3.1 Create Environment File

Create a `.env` file in the project root:

```bash
# Google OAuth
GOOGLE_CLIENT_ID=YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=YOUR_GOOGLE_CLIENT_SECRET

# GitHub OAuth
GITHUB_CLIENT_ID=YOUR_GITHUB_CLIENT_ID
GITHUB_CLIENT_SECRET=YOUR_GITHUB_CLIENT_SECRET

# OAuth Redirect URI
OAUTH_REDIRECT_URI=howlerops://auth/callback
```

Replace `YOUR_*` placeholders with actual values from previous steps.

### 3.2 Add .env to .gitignore

Ensure your `.gitignore` includes:

```gitignore
.env
.env.local
.env.*.local
```

**IMPORTANT**: Never commit OAuth credentials to version control!

## Part 4: Register Custom URL Scheme (OS-Level)

The custom protocol handler `howlerops://` is already configured in `wails.json` for macOS. For production builds on other platforms:

### macOS
- Already handled by Wails configuration in `wails.json`
- The Info.plist will be generated automatically during build

### Windows
- Handled automatically by Wails during installation
- Registry entries created for `howlerops://` protocol

### Linux
- Create a `.desktop` file during installation:
```desktop
[Desktop Entry]
Type=Application
Name=HowlerOps
Exec=/path/to/howlerops %u
MimeType=x-scheme-handler/howlerops;
```

## Part 5: Test the Implementation

### 5.1 Start Development Server

```bash
cd /Users/jacob/projects/amplifier/ai_working/howlerops
wails dev
```

### 5.2 Test OAuth Flow

1. Launch HowlerOps
2. Navigate to Login/Signup page
3. Click "Google" button
   - Browser opens with Google consent screen
   - After approval, redirects to `howlerops://auth/callback`
   - App receives token and authenticates user
4. Try "GitHub" button
   - Similar flow with GitHub OAuth

### 5.3 Test Biometric Authentication

**macOS (Touch ID)**:
1. Click "Sign in with Touch ID" button
2. Touch ID prompt appears
3. Authenticate with fingerprint
4. User logged in

**Windows (Windows Hello)**:
1. Click "Sign in with Biometric" button
2. Windows Hello prompt appears
3. Authenticate with fingerprint/face/PIN
4. User logged in

## Part 6: Production Deployment

### 6.1 Production OAuth Apps

Create separate OAuth applications for production:

**Google**:
- New OAuth client with production redirect URI
- Consider using "Web application" type for hosted version

**GitHub**:
- New OAuth app with production callback URL
- Configure organization access if needed

### 6.2 Environment Variables

Use platform-specific secure storage for production:

**macOS**:
```bash
security add-generic-password -a howlerops -s oauth_google_client_id -w "YOUR_CLIENT_ID"
```

**Windows**:
- Use Windows Credential Manager
- Store as generic credentials

**Linux**:
- Use GNOME Keyring or similar
- Store with `secret-tool` utility

### 6.3 Build Configuration

Update `wails.json` for production:
```json
{
  "info": {
    "productName": "HowlerOps",
    "productVersion": "1.0.0"
  },
  "darwin": {
    "identifier": "com.howlerops.app",
    "urlSchemes": ["howlerops"]
  }
}
```

## Part 7: Security Considerations

### 7.1 PKCE (Proof Key for Code Exchange)
- ✅ Already implemented in `oauth_manager.go`
- Prevents authorization code interception
- Required for desktop applications

### 7.2 State Parameter Validation
- ✅ Already implemented with 10-minute expiration
- Prevents CSRF attacks
- State tokens are cryptographically random

### 7.3 Token Storage
- ✅ Tokens stored in OS keyring (never localStorage)
- macOS: Keychain
- Windows: Credential Manager
- Linux: GNOME Keyring

### 7.4 Credential Storage
- ✅ WebAuthn credentials stored securely
- Challenge-response prevents replay attacks
- Credentials never leave the device

## Part 8: Troubleshooting

### OAuth Redirect Not Working

**Problem**: Browser opens but doesn't redirect back to app

**Solutions**:
1. Verify custom protocol registered:
   ```bash
   # macOS
   defaults read com.apple.LaunchServices/com.apple.launchservices.secure

   # Windows
   reg query HKEY_CLASSES_ROOT\howlerops
   ```

2. Check OAuth app configuration:
   - Redirect URI exactly matches: `howlerops://auth/callback`
   - No trailing slashes or extra parameters

3. Clear browser OAuth cache:
   ```bash
   # Chrome
   chrome://settings/content/all

   # Firefox
   about:preferences#privacy
   ```

### Biometric Auth Not Available

**Problem**: Biometric button doesn't appear

**Solutions**:
1. Check platform support:
   - macOS: Touch ID enabled in System Preferences
   - Windows: Windows Hello configured
   - Linux: WebAuthn support in browser

2. Verify HTTPS/Localhost:
   - WebAuthn requires secure context
   - Development: Use localhost
   - Production: Use HTTPS

### Token Storage Errors

**Problem**: Keyring access denied

**Solutions**:
1. Grant keychain access (macOS):
   ```bash
   security unlock-keychain ~/Library/Keychains/login.keychain-db
   ```

2. Reset credentials:
   ```bash
   # macOS
   security delete-generic-password -a howlerops -s google_token

   # Windows
   cmdkey /delete:howlerops_google_token
   ```

## Part 9: Testing Checklist

Before releasing to users:

- [ ] Google OAuth flow completes successfully
- [ ] GitHub OAuth flow completes successfully
- [ ] Tokens stored securely in OS keyring
- [ ] Tokens persist across app restarts
- [ ] Token refresh works automatically
- [ ] Biometric registration works (macOS Touch ID)
- [ ] Biometric authentication works (macOS Touch ID)
- [ ] Biometric authentication works (Windows Hello)
- [ ] Logout clears tokens from keyring
- [ ] State parameter validation prevents CSRF
- [ ] PKCE prevents code interception
- [ ] Error messages are user-friendly
- [ ] OAuth errors handled gracefully
- [ ] Custom protocol handler works after installation

## Part 10: Resources

- **OAuth 2.0 for Native Apps**: [RFC 8252](https://datatracker.ietf.org/doc/html/rfc8252)
- **PKCE**: [RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636)
- **WebAuthn Spec**: [W3C WebAuthn](https://www.w3.org/TR/webauthn/)
- **Wails Documentation**: [Custom Protocol](https://wails.io/docs/guides/application-development#custom-protocols)
- **Google OAuth**: [Desktop Apps Guide](https://developers.google.com/identity/protocols/oauth2/native-app)
- **GitHub OAuth**: [Authorizing Apps](https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps)

## Questions?

Refer to the technical implementation docs:
- `OAUTH_IMPLEMENTATION.md` - Complete OAuth implementation details
- `OAUTH_FRONTEND_GUIDE.md` - Frontend integration examples
- `README_AUTHENTICATION.md` - Wails authentication overview

---

**Implementation Status**: ✅ Complete
**Last Updated**: 2025-01-15
