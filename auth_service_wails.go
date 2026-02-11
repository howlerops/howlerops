package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jbeck018/howlerops/pkg/auth"
)

// WailsAuthService handles authentication operations for Wails v3
type WailsAuthService struct {
	deps            *SharedDeps
	githubOAuth     *auth.OAuth2Manager
	googleOAuth     *auth.OAuth2Manager
	secureStorage   *auth.SecureStorage
	webauthnManager *auth.WebAuthnManager
	credentialStore *auth.CredentialStore
	sessionStore    *auth.SessionStore
}

// NewWailsAuthService creates a new WailsAuthService instance
func NewWailsAuthService(
	deps *SharedDeps,
	githubOAuth *auth.OAuth2Manager,
	googleOAuth *auth.OAuth2Manager,
	secureStorage *auth.SecureStorage,
	webauthnManager *auth.WebAuthnManager,
	credentialStore *auth.CredentialStore,
	sessionStore *auth.SessionStore,
) *WailsAuthService {
	return &WailsAuthService{
		deps:            deps,
		githubOAuth:     githubOAuth,
		googleOAuth:     googleOAuth,
		secureStorage:   secureStorage,
		webauthnManager: webauthnManager,
		credentialStore: credentialStore,
		sessionStore:    sessionStore,
	}
}

// =============================================================================
// OAuth Authentication Methods
// =============================================================================

// GetOAuthURL generates an OAuth authorization URL for the specified provider
func (a *WailsAuthService) GetOAuthURL(provider string) (map[string]string, error) {
	a.deps.Logger.WithField("provider", provider).Info("Generating OAuth URL")

	var manager *auth.OAuth2Manager
	switch provider {
	case "github":
		manager = a.githubOAuth
	case "google":
		manager = a.googleOAuth
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	if manager == nil {
		return nil, fmt.Errorf("OAuth not configured for provider: %s (check environment variables)", provider)
	}

	result, err := manager.GetAuthURL()
	if err != nil {
		a.deps.Logger.WithError(err).WithField("provider", provider).Error("Failed to generate OAuth URL")
		return nil, err
	}

	a.deps.Logger.WithField("provider", provider).Info("OAuth URL generated successfully")
	return result, nil
}

// Logout removes the stored token for the specified provider
func (a *WailsAuthService) Logout(provider string) error {
	a.deps.Logger.WithField("provider", provider).Info("Logging out")

	if provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}

	if err := a.secureStorage.DeleteToken(provider); err != nil {
		a.deps.Logger.WithError(err).WithField("provider", provider).Error("Failed to logout")
		return err
	}

	a.deps.Logger.WithField("provider", provider).Info("Logged out successfully")
	return nil
}

// CheckStoredToken checks if a valid token exists for the specified provider
func (a *WailsAuthService) CheckStoredToken(provider string) (bool, error) {
	a.deps.Logger.WithField("provider", provider).Debug("Checking for stored token")

	if provider == "" {
		return false, fmt.Errorf("provider cannot be empty")
	}

	exists, err := a.secureStorage.CheckTokenExists(provider)
	if err != nil {
		a.deps.Logger.WithError(err).WithField("provider", provider).Error("Failed to check stored token")
		return false, err
	}

	a.deps.Logger.WithFields(logrus.Fields{
		"provider": provider,
		"exists":   exists,
	}).Debug("Stored token check complete")

	return exists, nil
}

// GetStoredUserInfo retrieves the stored user info for the specified provider
func (a *WailsAuthService) GetStoredUserInfo(provider string) (map[string]interface{}, error) {
	a.deps.Logger.WithField("provider", provider).Debug("Retrieving stored user info")

	if provider == "" {
		return nil, fmt.Errorf("provider cannot be empty")
	}

	token, err := a.secureStorage.RetrieveToken(provider)
	if err != nil {
		a.deps.Logger.WithError(err).WithField("provider", provider).Error("Failed to retrieve stored token")
		return nil, err
	}

	if token == nil {
		return nil, nil // No token stored
	}

	// Return user info without access token
	userInfo := map[string]interface{}{
		"provider": token.Provider,
		"userId":   token.UserID,
		"email":    token.Email,
	}

	if !token.ExpiresAt.IsZero() {
		userInfo["expiresAt"] = token.ExpiresAt.Format(time.RFC3339)
	}

	a.deps.Logger.WithField("provider", provider).Debug("Retrieved stored user info")
	return userInfo, nil
}

// OnUrlOpen handles custom protocol URL callbacks (e.g., OAuth redirects)
func (a *WailsAuthService) OnUrlOpen(url string) {
	a.deps.Logger.WithField("url", url).Info("Handling URL open callback")

	// Check if this is an OAuth callback
	if !strings.Contains(url, "howlerops://auth/callback") {
		a.deps.Logger.WithField("url", url).Debug("Not an OAuth callback URL, ignoring")
		return
	}

	// Parse URL to extract query parameters
	// Format: howlerops://auth/callback?code=xxx&state=yyy
	queryStart := strings.Index(url, "?")
	if queryStart == -1 {
		a.deps.Logger.Error("OAuth callback URL missing query parameters")
		a.deps.emitEvent("auth:error", "Invalid callback URL: missing parameters")
		return
	}

	// Parse query string
	query := url[queryStart+1:]
	params := make(map[string]string)
	for _, pair := range strings.Split(query, "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parts[1]
		}
	}

	// Extract required parameters
	code, hasCode := params["code"]
	state, hasState := params["state"]

	if !hasCode {
		a.deps.Logger.Error("OAuth callback missing authorization code")
		a.deps.emitEvent("auth:error", "Missing authorization code")
		return
	}

	if !hasState {
		a.deps.Logger.Error("OAuth callback missing state parameter")
		a.deps.emitEvent("auth:error", "Missing state parameter")
		return
	}

	// Try to exchange code with both providers (one will have the valid state)
	a.handleOAuthCallback(code, state)
}

// handleOAuthCallback processes the OAuth callback and exchanges the code for a token
func (a *WailsAuthService) handleOAuthCallback(code, state string) {
	a.deps.Logger.Info("Processing OAuth callback")

	var user *auth.OAuthUser
	var provider string
	var err error

	// Try GitHub first
	if a.githubOAuth != nil {
		user, err = a.githubOAuth.ExchangeCodeForToken(code, state)
		if err == nil {
			provider = "github"
		} else if !strings.Contains(err.Error(), "invalid state") {
			// Real error (not just wrong provider)
			a.deps.Logger.WithError(err).Error("GitHub OAuth exchange failed")
		}
	}

	// If GitHub didn't work, try Google
	if user == nil && a.googleOAuth != nil {
		user, err = a.googleOAuth.ExchangeCodeForToken(code, state)
		if err == nil {
			provider = "google"
		} else if !strings.Contains(err.Error(), "invalid state") {
			// Real error (not just wrong provider)
			a.deps.Logger.WithError(err).Error("Google OAuth exchange failed")
		}
	}

	// Check if we successfully got a user from any provider
	if user == nil {
		a.deps.Logger.WithError(err).Error("OAuth code exchange failed for all providers")
		a.deps.emitEvent("auth:error", "Authentication failed: "+err.Error())
		return
	}

	// Store token securely
	storedToken := &auth.StoredToken{
		AccessToken: user.AccessToken,
		Provider:    user.Provider,
		UserID:      user.ID,
		Email:       user.Email,
		ExpiresAt:   user.ExpiresAt,
	}

	if err := a.secureStorage.StoreToken(provider, storedToken); err != nil {
		a.deps.Logger.WithError(err).Error("Failed to store OAuth token")
		a.deps.emitEvent("auth:error", "Failed to store authentication token")
		return
	}

	a.deps.Logger.WithFields(logrus.Fields{
		"provider": provider,
		"userId":   user.ID,
		"email":    user.Email,
	}).Info("OAuth authentication successful")

	// Emit success event to frontend (without access token)
	userData := map[string]interface{}{
		"provider": user.Provider,
		"id":       user.ID,
		"login":    user.Login,
		"email":    user.Email,
		"name":     user.Name,
	}

	if user.AvatarURL != "" {
		userData["avatarUrl"] = user.AvatarURL
	}

	a.deps.emitEvent("auth:success", userData)
}

// =============================================================================
// WebAuthn Biometric Authentication Methods
// =============================================================================

// CheckBiometricAvailability checks if biometric authentication is available on the current platform
func (a *WailsAuthService) CheckBiometricAvailability() (map[string]interface{}, error) {
	a.deps.Logger.Debug("Checking biometric availability")

	capability, err := auth.CheckBiometricAvailability()
	if err != nil {
		a.deps.Logger.WithError(err).Error("Failed to check biometric availability")
		return map[string]interface{}{
			"available": false,
			"type":      "none",
			"error":     err.Error(),
		}, nil // Don't fail the call, just return unavailable
	}

	a.deps.Logger.WithFields(logrus.Fields{
		"available": capability["available"],
		"type":      capability["type"],
		"platform":  capability["platform"],
	}).Info("Biometric availability check complete")

	return capability, nil
}

// StartWebAuthnRegistration initiates the WebAuthn registration process
func (a *WailsAuthService) StartWebAuthnRegistration(userID, userName string) (string, error) {
	a.deps.Logger.WithFields(logrus.Fields{
		"userID":   userID,
		"userName": userName,
	}).Info("Starting WebAuthn registration")

	if a.webauthnManager == nil {
		return "", fmt.Errorf("WebAuthn not initialized")
	}

	if userID == "" || userName == "" {
		return "", fmt.Errorf("userID and userName are required")
	}

	optionsJSON, err := a.webauthnManager.BeginRegistration(userID, userName)
	if err != nil {
		a.deps.Logger.WithError(err).Error("Failed to begin WebAuthn registration")
		return "", err
	}

	a.deps.Logger.WithField("userID", userID).Info("WebAuthn registration started successfully")
	return string(optionsJSON), nil
}

// FinishWebAuthnRegistration completes the WebAuthn registration process
func (a *WailsAuthService) FinishWebAuthnRegistration(userID, credentialJSON string) (bool, error) {
	a.deps.Logger.WithField("userID", userID).Info("Finishing WebAuthn registration")

	if a.webauthnManager == nil {
		return false, fmt.Errorf("WebAuthn not initialized")
	}

	if userID == "" || credentialJSON == "" {
		return false, fmt.Errorf("userID and credentialJSON are required")
	}

	err := a.webauthnManager.FinishRegistration(userID, credentialJSON)
	if err != nil {
		a.deps.Logger.WithError(err).Error("Failed to finish WebAuthn registration")
		return false, err
	}

	a.deps.Logger.WithField("userID", userID).Info("WebAuthn registration completed successfully")
	return true, nil
}

// StartWebAuthnAuthentication initiates the WebAuthn authentication process
func (a *WailsAuthService) StartWebAuthnAuthentication(userID string) (string, error) {
	a.deps.Logger.WithField("userID", userID).Info("Starting WebAuthn authentication")

	if a.webauthnManager == nil {
		return "", fmt.Errorf("WebAuthn not initialized")
	}

	if userID == "" {
		return "", fmt.Errorf("userID is required")
	}

	// Check if user has credentials
	if !a.credentialStore.HasCredentials(userID) {
		a.deps.Logger.WithField("userID", userID).Warn("No credentials found for user")
		return "", fmt.Errorf("no credentials registered for this user")
	}

	optionsJSON, err := a.webauthnManager.BeginAuthentication(userID)
	if err != nil {
		a.deps.Logger.WithError(err).Error("Failed to begin WebAuthn authentication")
		return "", err
	}

	a.deps.Logger.WithField("userID", userID).Info("WebAuthn authentication started successfully")
	return string(optionsJSON), nil
}

// FinishWebAuthnAuthentication completes the WebAuthn authentication process
func (a *WailsAuthService) FinishWebAuthnAuthentication(userID, assertionJSON string) (string, error) {
	a.deps.Logger.WithField("userID", userID).Info("Finishing WebAuthn authentication")

	if a.webauthnManager == nil {
		return "", fmt.Errorf("WebAuthn not initialized")
	}

	if userID == "" || assertionJSON == "" {
		return "", fmt.Errorf("userID and assertionJSON are required")
	}

	token, err := a.webauthnManager.FinishAuthentication(userID, assertionJSON)
	if err != nil {
		a.deps.Logger.WithError(err).Error("Failed to finish WebAuthn authentication")
		return "", err
	}

	a.deps.Logger.WithField("userID", userID).Info("WebAuthn authentication completed successfully")

	// Emit success event to frontend
	a.deps.emitEvent("webauthn:success", map[string]interface{}{
		"userID": userID,
		"token":  token,
	})

	return token, nil
}

// DeleteWebAuthnCredential removes a WebAuthn credential for a user
func (a *WailsAuthService) DeleteWebAuthnCredential(userID string) error {
	a.deps.Logger.WithField("userID", userID).Info("Deleting WebAuthn credentials")

	if a.credentialStore == nil {
		return fmt.Errorf("credential store not initialized")
	}

	if userID == "" {
		return fmt.Errorf("userID is required")
	}

	err := a.credentialStore.DeleteAllCredentials(userID)
	if err != nil {
		a.deps.Logger.WithError(err).Error("Failed to delete WebAuthn credentials")
		return err
	}

	a.deps.Logger.WithField("userID", userID).Info("WebAuthn credentials deleted successfully")
	return nil
}

// HasWebAuthnCredential checks if a user has registered WebAuthn credentials
func (a *WailsAuthService) HasWebAuthnCredential(userID string) (bool, error) {
	a.deps.Logger.WithField("userID", userID).Debug("Checking for WebAuthn credentials")

	if a.credentialStore == nil {
		return false, fmt.Errorf("credential store not initialized")
	}

	if userID == "" {
		return false, fmt.Errorf("userID is required")
	}

	hasCredentials := a.credentialStore.HasCredentials(userID)

	a.deps.Logger.WithFields(logrus.Fields{
		"userID":         userID,
		"hasCredentials": hasCredentials,
	}).Debug("WebAuthn credential check complete")

	return hasCredentials, nil
}
