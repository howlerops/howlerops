package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

const (
	// PKCE constants
	codeVerifierLength = 128
	challengeMethod    = "S256"

	// State parameter constants
	stateLength = 32
	stateTTL    = 10 * time.Minute // States expire after 10 minutes
)

// OAuth2Manager manages OAuth2 authentication flows with PKCE
type OAuth2Manager struct {
	config        *oauth2.Config
	provider      string // "github" or "google"
	pendingTokens map[string]*PendingToken
	mu            sync.RWMutex
}

// NewOAuth2Manager creates a new OAuth2Manager for a specific provider
func NewOAuth2Manager(provider, clientID, clientSecret string) (*OAuth2Manager, error) {
	if provider != "github" && provider != "google" {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "howlerops://auth/callback",
		Scopes:       getScopes(provider),
		Endpoint:     getEndpoint(provider),
	}

	return &OAuth2Manager{
		config:        config,
		provider:      provider,
		pendingTokens: make(map[string]*PendingToken),
	}, nil
}

// getScopes returns appropriate scopes for the provider
func getScopes(provider string) []string {
	switch provider {
	case "github":
		return []string{"user:email"}
	case "google":
		return []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		}
	default:
		return []string{}
	}
}

// getEndpoint returns OAuth2 endpoint for the provider
func getEndpoint(provider string) oauth2.Endpoint {
	switch provider {
	case "github":
		return github.Endpoint
	case "google":
		return google.Endpoint
	default:
		return oauth2.Endpoint{}
	}
}

// generatePKCEPair generates a PKCE code verifier and challenge
func generatePKCEPair() (verifier, challenge string, err error) {
	// Generate random verifier (128 characters)
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	b := make([]byte, codeVerifierLength)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to valid verifier string
	verifierBytes := make([]byte, codeVerifierLength)
	for i := 0; i < codeVerifierLength; i++ {
		verifierBytes[i] = chars[int(b[i])%len(chars)]
	}
	verifier = string(verifierBytes)

	// Generate S256 challenge from verifier
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])

	return verifier, challenge, nil
}

// generateRandomState generates a random state parameter for CSRF protection
func generateRandomState() (string, error) {
	b := make([]byte, stateLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GetAuthURL generates an OAuth2 authorization URL with PKCE
func (om *OAuth2Manager) GetAuthURL() (map[string]string, error) {
	// Generate PKCE pair
	verifier, challenge, err := generatePKCEPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE pair: %w", err)
	}

	// Generate state parameter
	state, err := generateRandomState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Store pending token for later validation
	om.mu.Lock()
	now := time.Now()
	om.pendingTokens[state] = &PendingToken{
		State:         state,
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
		Provider:      om.provider,
		CreatedAt:     now,
		ExpiresAt:     now.Add(stateTTL),
	}
	om.mu.Unlock()

	// Generate auth URL with PKCE parameters
	authURL := om.config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", challengeMethod),
	)

	return map[string]string{
		"authUrl": authURL,
		"state":   state,
	}, nil
}

// ExchangeCodeForToken exchanges authorization code for access token
func (om *OAuth2Manager) ExchangeCodeForToken(code, state string) (*OAuthUser, error) {
	// Verify state parameter
	om.mu.Lock()
	pending, exists := om.pendingTokens[state]
	if !exists {
		om.mu.Unlock()
		return nil, errors.New("invalid state parameter: state not found")
	}

	// Check if state has expired
	if time.Now().After(pending.ExpiresAt) {
		delete(om.pendingTokens, state)
		om.mu.Unlock()
		return nil, errors.New("state parameter expired")
	}

	// Remove pending token (one-time use)
	delete(om.pendingTokens, state)
	codeVerifier := pending.CodeVerifier
	om.mu.Unlock()

	// Exchange code for token with PKCE verifier
	ctx := context.Background()
	token, err := om.config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user info from provider
	user, err := om.getUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Set provider and token
	user.Provider = om.provider
	user.AccessToken = token.AccessToken
	if !token.Expiry.IsZero() {
		user.ExpiresAt = token.Expiry
	}

	return user, nil
}

// getUserInfo fetches user information from the OAuth provider
func (om *OAuth2Manager) getUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUser, error) {
	client := om.config.Client(ctx, token)

	switch om.provider {
	case "github":
		return om.getGitHubUserInfo(client)
	case "google":
		return om.getGoogleUserInfo(client)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", om.provider)
	}
}

// getGitHubUserInfo fetches user info from GitHub API
func (om *OAuth2Manager) getGitHubUserInfo(client *http.Client) (*OAuthUser, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s - %s", resp.Status, string(body))
	}

	var ghUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub user: %w", err)
	}

	// If email is not public, fetch it from emails endpoint
	if ghUser.Email == "" {
		email, err := om.getGitHubPrimaryEmail(client)
		if err == nil {
			ghUser.Email = email
		}
	}

	return &OAuthUser{
		ID:        fmt.Sprintf("%d", ghUser.ID),
		Login:     ghUser.Login,
		Email:     ghUser.Email,
		Name:      ghUser.Name,
		AvatarURL: ghUser.AvatarURL,
	}, nil
}

// getGitHubPrimaryEmail fetches the primary email from GitHub
func (om *OAuth2Manager) getGitHubPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get emails: %s", resp.Status)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Find primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	// Fallback to first verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", errors.New("no verified email found")
}

// getGoogleUserInfo fetches user info from Google API
func (om *OAuth2Manager) getGoogleUserInfo(client *http.Client) (*OAuthUser, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get Google user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google API error: %s - %s", resp.Status, string(body))
	}

	var gUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, fmt.Errorf("failed to decode Google user: %w", err)
	}

	// Use email as login for Google
	login := gUser.Email
	if idx := strings.Index(login, "@"); idx > 0 {
		login = login[:idx] // Extract username part
	}

	return &OAuthUser{
		ID:        gUser.ID,
		Login:     login,
		Email:     gUser.Email,
		Name:      gUser.Name,
		AvatarURL: gUser.Picture,
	}, nil
}

// CleanupExpiredStates removes expired state parameters
func (om *OAuth2Manager) CleanupExpiredStates() {
	om.mu.Lock()
	defer om.mu.Unlock()

	now := time.Now()
	for state, pending := range om.pendingTokens {
		if now.After(pending.ExpiresAt) {
			delete(om.pendingTokens, state)
		}
	}
}
