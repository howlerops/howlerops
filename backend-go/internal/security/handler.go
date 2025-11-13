package security

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/sql-studio/backend-go/internal/apikeys"
	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/internal/sso"
)

// Handler handles security-related HTTP endpoints
type Handler struct {
	ssoService         *sso.Service
	twoFactorService   *auth.TwoFactorService
	apiKeyService      *apikeys.Service
	ipWhitelistService *middleware.IPWhitelistService
	eventLogger        SecurityEventLogger
	logger             *logrus.Logger
}

// SecurityEventLogger interface for logging security events
type SecurityEventLogger interface {
	LogSecurityEvent(ctx context.Context, eventType, userID, orgID, ipAddress, userAgent string, details map[string]interface{}) error
}

// NewHandler creates a new security handler
func NewHandler(
	ssoService *sso.Service,
	twoFactorService *auth.TwoFactorService,
	apiKeyService *apikeys.Service,
	ipWhitelistService *middleware.IPWhitelistService,
	eventLogger SecurityEventLogger,
	logger *logrus.Logger,
) *Handler {
	return &Handler{
		ssoService:         ssoService,
		twoFactorService:   twoFactorService,
		apiKeyService:      apiKeyService,
		ipWhitelistService: ipWhitelistService,
		eventLogger:        eventLogger,
		logger:             logger,
	}
}

// RegisterRoutes registers all security-related routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// SSO Configuration
	router.HandleFunc("/api/organizations/{id}/sso", h.ConfigureSSO).Methods("POST")
	router.HandleFunc("/api/organizations/{id}/sso", h.GetSSOConfig).Methods("GET")
	router.HandleFunc("/api/organizations/{id}/sso", h.DisableSSO).Methods("DELETE")
	router.HandleFunc("/api/auth/sso/{org_id}/login", h.InitiateSSOLogin).Methods("GET")
	router.HandleFunc("/api/auth/sso/callback", h.SSOCallback).Methods("POST")

	// IP Whitelist
	router.HandleFunc("/api/organizations/{id}/ip-whitelist", h.AddIPToWhitelist).Methods("POST")
	router.HandleFunc("/api/organizations/{id}/ip-whitelist", h.GetIPWhitelist).Methods("GET")
	router.HandleFunc("/api/organizations/{id}/ip-whitelist/{ip_id}", h.RemoveIPFromWhitelist).Methods("DELETE")

	// Two-Factor Authentication
	router.HandleFunc("/api/auth/2fa/enable", h.EnableTwoFactor).Methods("POST")
	router.HandleFunc("/api/auth/2fa/confirm", h.ConfirmTwoFactor).Methods("POST")
	router.HandleFunc("/api/auth/2fa/disable", h.DisableTwoFactor).Methods("POST")
	router.HandleFunc("/api/auth/2fa/validate", h.ValidateTwoFactor).Methods("POST")
	router.HandleFunc("/api/auth/2fa/backup-codes", h.RegenerateBackupCodes).Methods("GET")
	router.HandleFunc("/api/auth/2fa/status", h.GetTwoFactorStatus).Methods("GET")

	// API Keys
	router.HandleFunc("/api/api-keys", h.CreateAPIKey).Methods("POST")
	router.HandleFunc("/api/api-keys", h.ListAPIKeys).Methods("GET")
	router.HandleFunc("/api/api-keys/{id}", h.GetAPIKey).Methods("GET")
	router.HandleFunc("/api/api-keys/{id}", h.RevokeAPIKey).Methods("DELETE")

	// Security Events
	router.HandleFunc("/api/security/events", h.ListSecurityEvents).Methods("GET")
}

// SSO Endpoints

// ConfigureSSO configures SSO for an organization
func (h *Handler) ConfigureSSO(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	var config sso.SSOConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user from context
	userID := getUserIDFromContext(r.Context())
	config.CreatedBy = userID

	if err := h.ssoService.ConfigureSSO(r.Context(), orgID, &config); err != nil {
		h.logger.WithError(err).Error("Failed to configure SSO")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "SSO configured successfully",
	}); err != nil {
		h.logger.WithError(err).Error("Failed to encode SSO config response")
	}
}

// GetSSOConfig retrieves SSO configuration for an organization
func (h *Handler) GetSSOConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	config, err := h.ssoService.GetConfig(r.Context(), orgID)
	if err != nil {
		http.Error(w, "SSO not configured", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		h.logger.WithError(err).Error("Failed to encode SSO config")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// DisableSSO disables SSO for an organization
func (h *Handler) DisableSSO(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	if err := h.ssoService.DisableSSO(r.Context(), orgID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "SSO disabled successfully",
	}); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// InitiateSSOLogin initiates SSO login
func (h *Handler) InitiateSSOLogin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["org_id"]

	loginURL, err := h.ssoService.InitiateLogin(r.Context(), orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to SSO provider
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

// SSOCallback handles SSO callback
func (h *Handler) SSOCallback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrgID string `json:"org_id"`
		Code  string `json:"code"`
		State string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.ssoService.HandleCallback(r.Context(), req.OrgID, req.Code, req.State)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Here you would typically create a session and return auth tokens
	// For now, just return the user info
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// IP Whitelist Endpoints

// AddIPToWhitelist adds an IP to the organization's whitelist
func (h *Handler) AddIPToWhitelist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	var entry middleware.IPWhitelistEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	entry.OrganizationID = orgID
	entry.CreatedBy = getUserIDFromContext(r.Context())

	if err := h.ipWhitelistService.AddIP(r.Context(), &entry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetIPWhitelist retrieves the IP whitelist for an organization
func (h *Handler) GetIPWhitelist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	whitelist, err := h.ipWhitelistService.GetWhitelist(r.Context(), orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(whitelist); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// RemoveIPFromWhitelist removes an IP from the whitelist
func (h *Handler) RemoveIPFromWhitelist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]
	ipID := vars["ip_id"]

	userID := getUserIDFromContext(r.Context())
	if err := h.ipWhitelistService.RemoveIP(r.Context(), ipID, userID, orgID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Two-Factor Authentication Endpoints

// EnableTwoFactor initiates 2FA setup
func (h *Handler) EnableTwoFactor(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	userEmail := getUserEmailFromContext(r.Context())

	setup, err := h.twoFactorService.EnableTwoFactor(r.Context(), userID, userEmail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(setup) // Best-effort encode
}

// ConfirmTwoFactor confirms and enables 2FA
func (h *Handler) ConfirmTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r.Context())
	if err := h.twoFactorService.ConfirmTwoFactor(r.Context(), userID, req.Code); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Two-factor authentication enabled successfully",
	}) // Best-effort encode
}

// DisableTwoFactor disables 2FA
func (h *Handler) DisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r.Context())
	if err := h.twoFactorService.DisableTwoFactor(r.Context(), userID, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Two-factor authentication disabled",
	}) // Best-effort encode
}

// ValidateTwoFactor validates a 2FA code
func (h *Handler) ValidateTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r.Context())
	if err := h.twoFactorService.ValidateCode(r.Context(), userID, req.Code); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{
		"valid": true,
	}) // Best-effort encode
}

// RegenerateBackupCodes regenerates 2FA backup codes
func (h *Handler) RegenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())

	codes, err := h.twoFactorService.RegenerateBackupCodes(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"backup_codes": codes,
		"created_at":   time.Now(),
	}) // Best-effort encode
}

// GetTwoFactorStatus returns the 2FA status
func (h *Handler) GetTwoFactorStatus(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())

	status, err := h.twoFactorService.GetStatus(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status) // Best-effort encode
}

// API Key Endpoints

// CreateAPIKey creates a new API key
func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var input apikeys.CreateAPIKeyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r.Context())
	response, err := h.apiKeyService.CreateAPIKey(r.Context(), userID, &input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response) // Best-effort encode
}

// ListAPIKeys lists API keys for the user
func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())

	keys, err := h.apiKeyService.ListUserAPIKeys(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(keys) // Best-effort encode
}

// GetAPIKey retrieves API key details
func (h *Handler) GetAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["id"]

	userID := getUserIDFromContext(r.Context())
	apiKey, err := h.apiKeyService.GetAPIKey(r.Context(), keyID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(apiKey) // Best-effort encode
}

// RevokeAPIKey revokes an API key
func (h *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["id"]

	userID := getUserIDFromContext(r.Context())
	if err := h.apiKeyService.RevokeAPIKey(r.Context(), keyID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListSecurityEvents lists security events
func (h *Handler) ListSecurityEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")
	_ = r.URL.Query().Get("event_type") // TODO: Use eventType for filtering

	// This would typically query the security_events table
	// For now, return a mock response
	events := []map[string]interface{}{
		{
			"id":         "evt-1",
			"event_type": "login_success",
			"user_id":    userID,
			"org_id":     orgID,
			"ip_address": "192.168.1.1",
			"created_at": time.Now().Add(-1 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(events) // Best-effort encode
}

// Helper functions

func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}

func getUserEmailFromContext(ctx context.Context) string {
	if email, ok := ctx.Value("user_email").(string); ok {
		return email
	}
	return ""
}
