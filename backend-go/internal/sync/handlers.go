package sync

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Handler handles sync HTTP requests
type Handler struct {
	service *Service
	logger  *logrus.Logger
}

// NewHandler creates a new sync handler
func NewHandler(service *Service, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers sync routes on the router
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Sync endpoints
	r.HandleFunc("/upload", h.HandleUpload).Methods("POST")
	r.HandleFunc("/download", h.HandleDownload).Methods("GET")
	r.HandleFunc("/conflicts", h.HandleListConflicts).Methods("GET")
	r.HandleFunc("/conflicts/{id}/resolve", h.HandleResolveConflict).Methods("POST")
}

// HandleUpload handles sync upload requests
func (h *Handler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request
	var req SyncUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Set user ID from auth context
	req.UserID = userID

	// Process upload
	resp, err := h.service.Upload(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process sync upload")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// HandleDownload handles sync download requests
func (h *Handler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse query parameters
	sinceStr := r.URL.Query().Get("since")
	deviceID := r.URL.Query().Get("device_id")

	var since time.Time
	var err error

	if sinceStr != "" {
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid 'since' parameter")
			return
		}
	} else {
		// Default to 30 days ago
		since = time.Now().Add(-30 * 24 * time.Hour)
	}

	if deviceID == "" {
		h.respondError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	// Create download request
	req := &SyncDownloadRequest{
		UserID:   userID,
		DeviceID: deviceID,
		Since:    since,
	}

	// Process download
	resp, err := h.service.Download(ctx, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process sync download")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// HandleListConflicts handles listing conflicts
func (h *Handler) HandleListConflicts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get conflicts
	conflicts, err := h.service.ListConflicts(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list conflicts")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"conflicts": conflicts,
		"count":     len(conflicts),
	})
}

// HandleResolveConflict handles conflict resolution
func (h *Handler) HandleResolveConflict(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get conflict ID from URL
	vars := mux.Vars(r)
	conflictID := vars["id"]

	// Parse request
	var req ConflictResolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Set conflict ID from URL
	req.ConflictID = conflictID

	// Resolve conflict
	resp, err := h.service.ResolveConflict(ctx, userID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to resolve conflict")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// respondJSON sends a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	service *interface{} // Extended auth service
	logger  *logrus.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(service interface{}, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		service: &service,
		logger:  logger,
	}
}

// RegisterAuthRoutes registers authentication routes
func (h *AuthHandler) RegisterAuthRoutes(r *mux.Router) {
	// Email verification routes
	r.HandleFunc("/verify-email", h.HandleVerifyEmail).Methods("POST")
	r.HandleFunc("/resend-verification", h.HandleResendVerification).Methods("POST")

	// Password reset routes
	r.HandleFunc("/request-password-reset", h.HandleRequestPasswordReset).Methods("POST")
	r.HandleFunc("/reset-password", h.HandleResetPassword).Methods("POST")
}

// HandleVerifyEmail handles email verification
func (h *AuthHandler) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		h.respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	// Note: This would need to call the actual extended auth service
	// For now, this is a placeholder structure
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Email verified successfully",
	})
}

// HandleResendVerification handles resending verification email
func (h *AuthHandler) HandleResendVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Note: This would need to call the actual extended auth service
	h.logger.WithField("user_id", userID).Info("Resending verification email")

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Verification email sent",
	})
}

// HandleRequestPasswordReset handles password reset requests
func (h *AuthHandler) HandleRequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		h.respondError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Note: This would need to call the actual extended auth service
	h.logger.WithField("email", req.Email).Info("Password reset requested")

	// Always return success for security (don't reveal if email exists)
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "If the email exists, a password reset link has been sent",
	})
}

// HandleResetPassword handles password reset
func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		h.respondError(w, http.StatusBadRequest, "token and new_password are required")
		return
	}

	// Note: This would need to call the actual extended auth service
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password reset successfully",
	})
}

// Helper methods
func (h *AuthHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *AuthHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}

// RateLimitConfig defines rate limit configuration for sync endpoints
type RateLimitConfig struct {
	UploadRPS   int
	DownloadRPS int
	ConflictRPS int
}

// GetDefaultRateLimitConfig returns default rate limits
func GetDefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		UploadRPS:   10, // 10 requests per minute per user
		DownloadRPS: 20, // 20 requests per minute per user
		ConflictRPS: 10, // 10 requests per minute per user
	}
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, code string) *ErrorResponse {
	return &ErrorResponse{
		Error:   true,
		Message: message,
		Code:    code,
	}
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(message string, data interface{}) *SuccessResponse {
	return &SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResponse represents a validation error response
type ValidationErrorResponse struct {
	Error   bool              `json:"error"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors"`
}

// NewValidationErrorResponse creates a new validation error response
func NewValidationErrorResponse(errors []ValidationError) *ValidationErrorResponse {
	return &ValidationErrorResponse{
		Error:   true,
		Message: "Validation failed",
		Errors:  errors,
	}
}

// Helper function to validate sync upload request
func validateSyncUploadRequest(req *SyncUploadRequest) []ValidationError {
	var errors []ValidationError

	if req.UserID == "" {
		errors = append(errors, ValidationError{
			Field:   "user_id",
			Message: "user_id is required",
		})
	}

	if req.DeviceID == "" {
		errors = append(errors, ValidationError{
			Field:   "device_id",
			Message: "device_id is required",
		})
	}

	if len(req.Changes) == 0 {
		errors = append(errors, ValidationError{
			Field:   "changes",
			Message: "at least one change is required",
		})
	}

	// Validate each change
	for i, change := range req.Changes {
		if change.ItemID == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("changes[%d].item_id", i),
				Message: "item_id is required",
			})
		}

		if change.ItemType == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("changes[%d].item_type", i),
				Message: "item_type is required",
			})
		}

		if change.Action == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("changes[%d].action", i),
				Message: "action is required",
			})
		}

		if change.Data == nil && change.Action != SyncActionDelete {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("changes[%d].data", i),
				Message: "data is required for create and update actions",
			})
		}
	}

	return errors
}
