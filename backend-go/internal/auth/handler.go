package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Handler handles authentication HTTP requests
type Handler struct {
	service *Service
	logger  *logrus.Logger
}

// NewHandler creates a new authentication handler
func NewHandler(service *Service, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers authentication routes on the router
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Public authentication endpoints (no auth middleware required)
	r.HandleFunc("/api/auth/signup", h.HandleSignup).Methods("POST")
	r.HandleFunc("/api/auth/login", h.HandleLogin).Methods("POST")
	r.HandleFunc("/api/auth/refresh", h.HandleRefresh).Methods("POST")

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
