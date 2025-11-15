package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	servicesauth "github.com/sql-studio/sql-studio/services/auth"
)

// Handler handles authentication HTTP requests
type Handler struct {
	service         *Service
	logger          *logrus.Logger
	githubOAuth     *servicesauth.OAuth2Manager
	googleOAuth     *servicesauth.OAuth2Manager
	webauthnManager *servicesauth.WebAuthnManager
}

// NewHandler creates a new authentication handler
func NewHandler(service *Service, logger *logrus.Logger, githubOAuth *servicesauth.OAuth2Manager, googleOAuth *servicesauth.OAuth2Manager, webauthnManager *servicesauth.WebAuthnManager) *Handler {
	return &Handler{
		service:         service,
		logger:          logger,
		githubOAuth:     githubOAuth,
		googleOAuth:     googleOAuth,
		webauthnManager: webauthnManager,
	}
}

// RegisterRoutes registers authentication routes on the router
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Public authentication endpoints (no auth middleware required)
	r.HandleFunc("/api/auth/signup", h.HandleSignup).Methods("POST")
	r.HandleFunc("/api/auth/login", h.HandleLogin).Methods("POST")
	r.HandleFunc("/api/auth/refresh", h.HandleRefresh).Methods("POST")

	// OAuth endpoints
	r.HandleFunc("/api/auth/oauth/initiate", h.HandleOAuthInitiate).Methods("POST")
	r.HandleFunc("/api/auth/oauth/callback", h.HandleOAuthCallback).Methods("GET")
	r.HandleFunc("/api/auth/oauth/exchange", h.HandleOAuthExchange).Methods("POST")

	// WebAuthn endpoints
	r.HandleFunc("/api/auth/webauthn/register/begin", h.HandleWebAuthnRegisterBegin).Methods("POST")
	r.HandleFunc("/api/auth/webauthn/register/finish", h.HandleWebAuthnRegisterFinish).Methods("POST")
	r.HandleFunc("/api/auth/webauthn/login/begin", h.HandleWebAuthnLoginBegin).Methods("POST")
	r.HandleFunc("/api/auth/webauthn/login/finish", h.HandleWebAuthnLoginFinish).Methods("POST")
	r.HandleFunc("/api/auth/webauthn/available", h.HandleWebAuthnAvailable).Methods("GET")

	// Protected endpoints would be registered with auth middleware elsewhere
}

// SignupRequest represents a signup request
type SignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// HandleSignup handles user registration
func (h *Handler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Username == "" {
		h.respondError(w, http.StatusBadRequest, "username is required")
		return
	}
	if req.Email == "" {
		h.respondError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		h.respondError(w, http.StatusBadRequest, "password is required")
		return
	}
	if len(req.Password) < 8 {
		h.respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Create user
	user := &User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     "user", // Default role
	}

	if err := h.service.CreateUser(ctx, user); err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
			h.respondError(w, http.StatusConflict, "username or email already exists")
			return
		}
		h.logger.WithError(err).Error("Failed to create user")
		h.respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Send welcome email (non-blocking)
	go func() {
		if err := h.service.SendWelcomeEmail(context.Background(), user.ID); err != nil {
			h.logger.WithError(err).Warn("Failed to send welcome email")
		}
	}()

	// Auto-login after signup
	ipAddress := h.extractIPAddress(r)
	loginReq := &LoginRequest{
		Username:  req.Username,
		Password:  req.Password,
		IPAddress: ipAddress,
		UserAgent: r.UserAgent(),
	}

	loginResp, err := h.service.Login(ctx, loginReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to auto-login after signup")
		// Still return success for signup
		h.respondJSON(w, http.StatusCreated, map[string]interface{}{
			"success": true,
			"message": "user created successfully",
			"user": map[string]interface{}{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
			},
		})
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success":       true,
		"message":       "user created successfully",
		"user":          loginResp.User,
		"token":         loginResp.Token,
		"refresh_token": loginResp.RefreshToken,
		"expires_at":    loginResp.ExpiresAt,
	})
}

// HandleLogin handles user login
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RememberMe bool   `json:"remember_me"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Username == "" {
		h.respondError(w, http.StatusBadRequest, "username is required")
		return
	}
	if req.Password == "" {
		h.respondError(w, http.StatusBadRequest, "password is required")
		return
	}

	// Extract IP and user agent
	ipAddress := h.extractIPAddress(r)
	userAgent := r.UserAgent()

	// Attempt login
	loginReq := &LoginRequest{
		Username:   req.Username,
		Password:   req.Password,
		RememberMe: req.RememberMe,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}

	loginResp, err := h.service.Login(ctx, loginReq)
	if err != nil {
		if strings.Contains(err.Error(), "locked") {
			h.respondError(w, http.StatusTooManyRequests, err.Error())
			return
		}
		if strings.Contains(err.Error(), "disabled") {
			h.respondError(w, http.StatusForbidden, "user account is disabled")
			return
		}
		h.logger.WithError(err).Debug("Login failed")
		h.respondError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"user":          loginResp.User,
		"token":         loginResp.Token,
		"refresh_token": loginResp.RefreshToken,
		"expires_at":    loginResp.ExpiresAt,
	})
}

// HandleRefresh handles token refresh
func (h *Handler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.RefreshToken == "" {
		h.respondError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	// Refresh token
	loginResp, err := h.service.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		h.logger.WithError(err).Debug("Token refresh failed")
		h.respondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"user":          loginResp.User,
		"token":         loginResp.Token,
		"refresh_token": loginResp.RefreshToken,
		"expires_at":    loginResp.ExpiresAt,
	})
}

// OAuth HTTP Endpoints

// OAuthInitiateRequest represents an OAuth initiation request
type OAuthInitiateRequest struct {
	Provider string `json:"provider"` // "github" or "google"
	Platform string `json:"platform"` // "web" or "desktop"
}

// HandleOAuthInitiate initiates OAuth flow
// POST /api/auth/oauth/initiate
func (h *Handler) HandleOAuthInitiate(w http.ResponseWriter, r *http.Request) {
	var req OAuthInitiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate provider
	if req.Provider != "github" && req.Provider != "google" {
		h.respondError(w, http.StatusBadRequest, "provider must be 'github' or 'google'")
		return
	}

	// Get appropriate OAuth manager
	var manager *servicesauth.OAuth2Manager
	switch req.Provider {
	case "github":
		manager = h.githubOAuth
	case "google":
		manager = h.googleOAuth
	}

	if manager == nil {
		h.respondError(w, http.StatusServiceUnavailable, "OAuth provider not configured")
		return
	}

	// Generate auth URL with PKCE
	result, err := manager.GetAuthURL()
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate OAuth URL")
		h.respondError(w, http.StatusInternalServerError, "failed to generate auth URL")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"auth_url": result["authUrl"],
		"state":    result["state"],
		"provider": req.Provider,
	})
}

// OAuthExchangeRequest represents an OAuth code exchange request
type OAuthExchangeRequest struct {
	Provider string `json:"provider"` // "github" or "google"
	Code     string `json:"code"`
	State    string `json:"state"`
}

// HandleOAuthExchange exchanges OAuth code for tokens
// POST /api/auth/oauth/exchange
func (h *Handler) HandleOAuthExchange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req OAuthExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Provider == "" {
		h.respondError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if req.Code == "" {
		h.respondError(w, http.StatusBadRequest, "code is required")
		return
	}
	if req.State == "" {
		h.respondError(w, http.StatusBadRequest, "state is required")
		return
	}

	// Get appropriate OAuth manager
	var manager *servicesauth.OAuth2Manager
	switch req.Provider {
	case "github":
		manager = h.githubOAuth
	case "google":
		manager = h.googleOAuth
	default:
		h.respondError(w, http.StatusBadRequest, "invalid provider")
		return
	}

	if manager == nil {
		h.respondError(w, http.StatusServiceUnavailable, "OAuth provider not configured")
		return
	}

	// Exchange code for token and get user info
	oauthUser, err := manager.ExchangeCodeForToken(req.Code, req.State)
	if err != nil {
		h.logger.WithError(err).Error("Failed to exchange OAuth code")
		if strings.Contains(err.Error(), "state") {
			h.respondError(w, http.StatusBadRequest, "invalid or expired state")
		} else {
			h.respondError(w, http.StatusUnauthorized, "OAuth authentication failed")
		}
		return
	}

	// Create or use the same Login flow to handle session creation
	// For OAuth we just need to call Login but skip password verification
	ipAddress := h.extractIPAddress(r)

	// Try to use existing Login method by first creating/getting user, then calling Login
	// For simplicity, just mimic what Login does but skip password check
	loginReq := &LoginRequest{
		Username:  oauthUser.Login,
		IPAddress: ipAddress,
		UserAgent: r.UserAgent(),
	}

	// Use internal Login implementation to create proper session
	// We need to bypass password check for OAuth
	loginResp, err := h.service.LoginWithOAuthUser(ctx, oauthUser.Email, oauthUser.Login, loginReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to login with OAuth")
		h.respondError(w, http.StatusInternalServerError, "failed to complete OAuth login")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"user":          loginResp.User,
		"token":         loginResp.Token,
		"refresh_token": loginResp.RefreshToken,
		"expires_at":    loginResp.ExpiresAt,
	})
}

// HandleOAuthCallback handles OAuth callback (for web deployments)
// GET /api/auth/oauth/callback
func (h *Handler) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Extract code and state from query params
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		h.logger.WithField("error", errorParam).Warn("OAuth callback error")
		h.respondError(w, http.StatusBadRequest, "OAuth authentication failed: "+errorParam)
		return
	}

	if code == "" || state == "" {
		h.respondError(w, http.StatusBadRequest, "missing code or state parameter")
		return
	}

	// In web mode, we need to redirect to frontend with the code and state
	// The frontend will then call /api/auth/oauth/exchange
	// For now, return a simple HTML page that posts the data to the frontend
	html := `<!DOCTYPE html>
<html>
<head><title>OAuth Callback</title></head>
<body>
<script>
	// Post message to opener window (if opened in popup)
	if (window.opener) {
		window.opener.postMessage({
			type: 'oauth-callback',
			code: '` + code + `',
			state: '` + state + `'
		}, '*');
		window.close();
	} else {
		// Redirect to frontend with query params
		window.location.href = '/?oauth_code=` + code + `&oauth_state=` + state + `';
	}
</script>
<p>Authenticating...</p>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(html)); err != nil {
		h.logger.WithError(err).Error("Failed to write callback response")
	}
}

// WebAuthn HTTP Endpoints

// WebAuthnRegisterBeginRequest represents a WebAuthn registration start request
type WebAuthnRegisterBeginRequest struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// HandleWebAuthnRegisterBegin starts WebAuthn registration
// POST /api/auth/webauthn/register/begin
func (h *Handler) HandleWebAuthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	var req WebAuthnRegisterBeginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.Username == "" {
		h.respondError(w, http.StatusBadRequest, "user_id and username are required")
		return
	}

	if h.webauthnManager == nil {
		h.respondError(w, http.StatusServiceUnavailable, "WebAuthn not configured")
		return
	}

	// Begin registration
	optionsJSON, err := h.webauthnManager.BeginRegistration(req.UserID, req.Username)
	if err != nil {
		h.logger.WithError(err).Error("Failed to begin WebAuthn registration")
		h.respondError(w, http.StatusInternalServerError, "failed to begin registration")
		return
	}

	// Parse the JSON to return as object
	var options interface{}
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		h.logger.WithError(err).Error("Failed to parse registration options")
		h.respondError(w, http.StatusInternalServerError, "failed to parse options")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"options": options,
	})
}

// WebAuthnRegisterFinishRequest represents a WebAuthn registration completion request
type WebAuthnRegisterFinishRequest struct {
	UserID         string `json:"user_id"`
	CredentialJSON string `json:"credential"`
}

// HandleWebAuthnRegisterFinish completes WebAuthn registration
// POST /api/auth/webauthn/register/finish
func (h *Handler) HandleWebAuthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	var req WebAuthnRegisterFinishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.CredentialJSON == "" {
		h.respondError(w, http.StatusBadRequest, "user_id and credential are required")
		return
	}

	if h.webauthnManager == nil {
		h.respondError(w, http.StatusServiceUnavailable, "WebAuthn not configured")
		return
	}

	// Finish registration
	if err := h.webauthnManager.FinishRegistration(req.UserID, req.CredentialJSON); err != nil {
		h.logger.WithError(err).Error("Failed to finish WebAuthn registration")
		if strings.Contains(err.Error(), "session") {
			h.respondError(w, http.StatusBadRequest, "invalid or expired session")
		} else {
			h.respondError(w, http.StatusBadRequest, "registration failed")
		}
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "WebAuthn credential registered successfully",
	})
}

// WebAuthnLoginBeginRequest represents a WebAuthn login start request
type WebAuthnLoginBeginRequest struct {
	UserID string `json:"user_id"`
}

// HandleWebAuthnLoginBegin starts WebAuthn authentication
// POST /api/auth/webauthn/login/begin
func (h *Handler) HandleWebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	var req WebAuthnLoginBeginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if h.webauthnManager == nil {
		h.respondError(w, http.StatusServiceUnavailable, "WebAuthn not configured")
		return
	}

	// Begin authentication
	optionsJSON, err := h.webauthnManager.BeginAuthentication(req.UserID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to begin WebAuthn authentication")
		if strings.Contains(err.Error(), "credentials") {
			h.respondError(w, http.StatusNotFound, "no credentials found for user")
		} else {
			h.respondError(w, http.StatusInternalServerError, "failed to begin authentication")
		}
		return
	}

	// Parse the JSON to return as object
	var options interface{}
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		h.logger.WithError(err).Error("Failed to parse authentication options")
		h.respondError(w, http.StatusInternalServerError, "failed to parse options")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"options": options,
	})
}

// WebAuthnLoginFinishRequest represents a WebAuthn login completion request
type WebAuthnLoginFinishRequest struct {
	UserID        string `json:"user_id"`
	AssertionJSON string `json:"assertion"`
}

// HandleWebAuthnLoginFinish completes WebAuthn authentication
// POST /api/auth/webauthn/login/finish
func (h *Handler) HandleWebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req WebAuthnLoginFinishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.AssertionJSON == "" {
		h.respondError(w, http.StatusBadRequest, "user_id and assertion are required")
		return
	}

	if h.webauthnManager == nil {
		h.respondError(w, http.StatusServiceUnavailable, "WebAuthn not configured")
		return
	}

	// Finish authentication
	_, err := h.webauthnManager.FinishAuthentication(req.UserID, req.AssertionJSON)
	if err != nil {
		h.logger.WithError(err).Error("Failed to finish WebAuthn authentication")
		if strings.Contains(err.Error(), "session") {
			h.respondError(w, http.StatusBadRequest, "invalid or expired session")
		} else {
			h.respondError(w, http.StatusUnauthorized, "authentication failed")
		}
		return
	}

	// Create login request for session creation
	ipAddress := h.extractIPAddress(r)
	loginReq := &LoginRequest{
		IPAddress: ipAddress,
		UserAgent: r.UserAgent(),
	}

	// Use WebAuthn login helper to create session
	loginResp, err := h.service.LoginWithWebAuthn(ctx, req.UserID, loginReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to login with WebAuthn")
		h.respondError(w, http.StatusInternalServerError, "failed to complete WebAuthn login")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"user":          loginResp.User,
		"token":         loginResp.Token,
		"refresh_token": loginResp.RefreshToken,
		"expires_at":    loginResp.ExpiresAt,
	})
}

// HandleWebAuthnAvailable checks if WebAuthn is available
// GET /api/auth/webauthn/available
func (h *Handler) HandleWebAuthnAvailable(w http.ResponseWriter, r *http.Request) {
	available := h.webauthnManager != nil

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"available": available,
		"message":   "WebAuthn availability status",
	})
}

// respondJSON sends a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// respondError sends an error response
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}

// extractIPAddress extracts the client IP address from the request
func (h *Handler) extractIPAddress(r *http.Request) string {
	// Try X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Try X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}
