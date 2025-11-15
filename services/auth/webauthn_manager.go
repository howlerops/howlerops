package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnUser implements the webauthn.User interface
type WebAuthnUser struct {
	ID          []byte
	Name        string
	DisplayName string
	credentials []webauthn.Credential
}

// WebAuthnID returns the user's ID
func (u *WebAuthnUser) WebAuthnID() []byte {
	return u.ID
}

// WebAuthnName returns the user's username
func (u *WebAuthnUser) WebAuthnName() string {
	return u.Name
}

// WebAuthnDisplayName returns the user's display name
func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.DisplayName
}

// WebAuthnIcon returns the user's icon URL (optional)
func (u *WebAuthnUser) WebAuthnIcon() string {
	return ""
}

// WebAuthnCredentials returns the user's credentials
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// WebAuthnManager handles WebAuthn registration and authentication
type WebAuthnManager struct {
	webauthn         *webauthn.WebAuthn
	credentialStore  *CredentialStore
	sessionStore     *SessionStore
	registrationOpts []webauthn.RegistrationOption
	loginOpts        []webauthn.LoginOption
}

// NewWebAuthnManager creates a new WebAuthn manager
func NewWebAuthnManager(credentialStore *CredentialStore, sessionStore *SessionStore) (*WebAuthnManager, error) {
	wconfig := &webauthn.Config{
		RPDisplayName: "HowlerOps",                   // Display name for the relying party
		RPID:          "localhost",                   // Relying Party ID (for development)
		RPOrigins:     []string{"wails://localhost"}, // Origins for Wails app
	}

	web, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn instance: %w", err)
	}

	// Registration options
	registrationOpts := []webauthn.RegistrationOption{
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform, // Prefer platform authenticators (Touch ID, Windows Hello)
			RequireResidentKey:      protocol.ResidentKeyNotRequired(),
			UserVerification:        protocol.VerificationPreferred, // Request user verification when possible
		}),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
	}

	// Login options
	loginOpts := []webauthn.LoginOption{
		webauthn.WithUserVerification(protocol.VerificationPreferred),
	}

	return &WebAuthnManager{
		webauthn:         web,
		credentialStore:  credentialStore,
		sessionStore:     sessionStore,
		registrationOpts: registrationOpts,
		loginOpts:        loginOpts,
	}, nil
}

// BeginRegistration starts the WebAuthn registration process
func (wm *WebAuthnManager) BeginRegistration(userID, userName string) ([]byte, error) {
	// Create user
	user := &WebAuthnUser{
		ID:          []byte(userID),
		Name:        userName,
		DisplayName: userName,
	}

	// Load existing credentials if any
	existingCreds, err := wm.credentialStore.GetCredentials(userID)
	if err == nil && len(existingCreds) > 0 {
		user.credentials = existingCreds
	}

	// Begin registration
	options, session, err := wm.webauthn.BeginRegistration(user, wm.registrationOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to begin registration: %w", err)
	}

	// Store session
	sessionID := userID + "_registration"
	if err := wm.sessionStore.StoreSession(sessionID, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	// Marshal options to JSON
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal options: %w", err)
	}

	return optionsJSON, nil
}

// FinishRegistration completes the WebAuthn registration process
func (wm *WebAuthnManager) FinishRegistration(userID, credentialJSON string) error {
	// Get session
	sessionID := userID + "_registration"
	session, err := wm.sessionStore.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	defer wm.sessionStore.DeleteSession(sessionID)

	// Create user (we need it for validation)
	user := &WebAuthnUser{
		ID:          []byte(userID),
		Name:        userID,
		DisplayName: userID,
	}

	// Parse the credential creation response from HTTP request body
	reader := bytes.NewReader([]byte(credentialJSON))
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(reader)
	if err != nil {
		return fmt.Errorf("failed to parse credential: %w", err)
	}

	// Finish registration
	credential, err := wm.webauthn.CreateCredential(user, *session, parsedResponse)
	if err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}

	// Store credential
	if err := wm.credentialStore.StoreCredential(userID, credential); err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	return nil
}

// BeginAuthentication starts the WebAuthn authentication process
func (wm *WebAuthnManager) BeginAuthentication(userID string) ([]byte, error) {
	// Create user with stored credentials
	user := &WebAuthnUser{
		ID:          []byte(userID),
		Name:        userID,
		DisplayName: userID,
	}

	// Load credentials
	credentials, err := wm.credentialStore.GetCredentials(userID)
	if err != nil {
		return nil, fmt.Errorf("no credentials found for user: %w", err)
	}
	user.credentials = credentials

	// Begin login
	options, session, err := wm.webauthn.BeginLogin(user, wm.loginOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to begin login: %w", err)
	}

	// Store session
	sessionID := userID + "_authentication"
	if err := wm.sessionStore.StoreSession(sessionID, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	// Marshal options to JSON
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal options: %w", err)
	}

	return optionsJSON, nil
}

// FinishAuthentication completes the WebAuthn authentication process
func (wm *WebAuthnManager) FinishAuthentication(userID, assertionJSON string) (string, error) {
	// Get session
	sessionID := userID + "_authentication"
	session, err := wm.sessionStore.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}
	defer wm.sessionStore.DeleteSession(sessionID)

	// Create user with stored credentials
	user := &WebAuthnUser{
		ID:          []byte(userID),
		Name:        userID,
		DisplayName: userID,
	}

	// Load credentials
	credentials, err := wm.credentialStore.GetCredentials(userID)
	if err != nil {
		return "", fmt.Errorf("no credentials found for user: %w", err)
	}
	user.credentials = credentials

	// Parse the credential assertion response from HTTP request body
	reader := bytes.NewReader([]byte(assertionJSON))
	parsedAssertion, err := protocol.ParseCredentialRequestResponseBody(reader)
	if err != nil {
		return "", fmt.Errorf("failed to parse assertion: %w", err)
	}

	// Verify the assertion
	credential, err := wm.webauthn.ValidateLogin(user, *session, parsedAssertion)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	// Update credential (counter, etc.)
	if err := wm.credentialStore.UpdateCredential(userID, credential); err != nil {
		return "", fmt.Errorf("failed to update credential: %w", err)
	}

	// Generate authentication token (simple JWT-like token for now)
	token := generateAuthToken(userID)

	return token, nil
}

// generateAuthToken generates a simple authentication token
// In production, use a proper JWT library with expiration
func generateAuthToken(userID string) string {
	// Simple token format: userID:timestamp
	// In production, use github.com/golang-jwt/jwt for proper JWT tokens
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s:%d", userID, timestamp)
}
