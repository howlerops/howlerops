package connections

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeck018/howlerops/internal/middleware"
	"github.com/jbeck018/howlerops/pkg/crypto"
	"github.com/jbeck018/howlerops/pkg/storage/turso"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests for connections
type Handler struct {
	service *Service
	logger  *logrus.Logger
}

// NewHandler creates a new connections handler
func NewHandler(service *Service, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers connection routes on the router
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Connection CRUD
	r.HandleFunc("/api/connections", h.CreateConnection).Methods("POST")
	r.HandleFunc("/api/connections", h.GetAccessibleConnections).Methods("GET")
	r.HandleFunc("/api/connections/{id}", h.UpdateConnection).Methods("PUT")
	r.HandleFunc("/api/connections/{id}", h.DeleteConnection).Methods("DELETE")

	// Connection sharing
	r.HandleFunc("/api/connections/{id}/share", h.ShareConnection).Methods("POST")
	r.HandleFunc("/api/connections/{id}/share-with-credential", h.ShareConnectionWithCredential).Methods("POST")
	r.HandleFunc("/api/connections/{id}/unshare", h.UnshareConnection).Methods("POST")
	r.HandleFunc("/api/connections/{id}/password", h.GetConnectionPassword).Methods("GET")

	// Organization connections
	r.HandleFunc("/api/organizations/{org_id}/connections", h.GetOrganizationConnections).Methods("GET")
}

// ShareConnectionRequest represents the request to share a connection
type ShareConnectionRequest struct {
	OrganizationID string `json:"organization_id" validate:"required"`
}

// ShareConnectionWithCredentialRequest includes encrypted password for OEK re-encryption
type ShareConnectionWithCredentialRequest struct {
	OrganizationID     string `json:"organization_id" validate:"required"`
	EncryptedMasterKey string `json:"encrypted_master_key"` // Base64-encoded, encrypted with session key
	Password           string `json:"password"`             // Already decrypted by frontend (temporary in memory)
}

// ShareConnectionWithCredential handles POST /api/connections/{id}/share-with-credential
// Share connection with password re-encryption for OEK
func (h *Handler) ShareConnectionWithCredential(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connectionID := vars["id"]
	if connectionID == "" {
		h.respondError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ShareConnectionWithCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrganizationID == "" {
		h.respondError(w, http.StatusBadRequest, "organization_id is required")
		return
	}

	// Decode master key from base64
	masterKey, err := base64.StdEncoding.DecodeString(req.EncryptedMasterKey)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid encrypted_master_key encoding")
		return
	}
	defer crypto.ClearBytes(masterKey)

	// Convert password to bytes
	password := []byte(req.Password)
	defer crypto.ClearBytes(password)

	if err := h.service.ShareConnectionWithCredential(
		r.Context(),
		connectionID,
		userID,
		req.OrganizationID,
		masterKey,
		password,
	); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"connection_id":   connectionID,
			"user_id":         userID,
			"organization_id": req.OrganizationID,
		}).Error("Failed to share connection with credential")

		status := http.StatusInternalServerError
		if err.Error() == "connection not found" {
			status = http.StatusNotFound
		} else if err.Error() == "only the creator can share this connection" ||
			err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to share connections" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection shared successfully with credential",
	})
}

// ShareConnection handles POST /api/connections/{id}/share
// Share connection in organization
func (h *Handler) ShareConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connectionID := vars["id"]
	if connectionID == "" {
		h.respondError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ShareConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrganizationID == "" {
		h.respondError(w, http.StatusBadRequest, "organization_id is required")
		return
	}

	if err := h.service.ShareConnection(r.Context(), connectionID, userID, req.OrganizationID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"connection_id":   connectionID,
			"user_id":         userID,
			"organization_id": req.OrganizationID,
		}).Error("Failed to share connection")

		status := http.StatusInternalServerError
		if err.Error() == "connection not found" {
			status = http.StatusNotFound
		} else if err.Error() == "only the creator can share this connection" ||
			err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to share connections" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection shared successfully",
	})
}

// UnshareConnection handles POST /api/connections/{id}/unshare
// Make connection personal
func (h *Handler) UnshareConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connectionID := vars["id"]
	if connectionID == "" {
		h.respondError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if err := h.service.UnshareConnection(r.Context(), connectionID, userID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"connection_id": connectionID,
			"user_id":       userID,
		}).Error("Failed to unshare connection")

		status := http.StatusInternalServerError
		if err.Error() == "connection not found" {
			status = http.StatusNotFound
		} else if err.Error() == "only the creator can unshare this connection" ||
			err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to unshare connections" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection unshared successfully",
	})
}

// GetOrganizationConnections handles GET /api/organizations/{org_id}/connections
// List shared connections in org
func (h *Handler) GetOrganizationConnections(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["org_id"]
	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization ID is required")
		return
	}

	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	connections, err := h.service.GetOrganizationConnections(r.Context(), orgID, userID)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"organization_id": orgID,
			"user_id":         userID,
		}).Error("Failed to get organization connections")

		status := http.StatusInternalServerError
		if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to view connections" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"connections": connections,
		"count":       len(connections),
	})
}

// GetAccessibleConnections handles GET /api/connections/accessible
// Get all connections accessible to the user (personal + shared)
func (h *Handler) GetAccessibleConnections(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	connections, err := h.service.GetAccessibleConnections(r.Context(), userID)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get accessible connections")
		h.respondError(w, http.StatusInternalServerError, "failed to get connections")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"connections": connections,
		"count":       len(connections),
	})
}

// CreateConnection handles POST /api/connections
func (h *Handler) CreateConnection(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var conn turso.Connection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.CreateConnection(r.Context(), &conn, userID); err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to create connection")

		status := http.StatusInternalServerError
		if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to create connections in organization" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, conn)
}

// UpdateConnection handles PUT /api/connections/{id}
func (h *Handler) UpdateConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connectionID := vars["id"]
	if connectionID == "" {
		h.respondError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var conn turso.Connection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	conn.ID = connectionID

	if err := h.service.UpdateConnection(r.Context(), &conn, userID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"connection_id": connectionID,
			"user_id":       userID,
		}).Error("Failed to update connection")

		status := http.StatusInternalServerError
		if err.Error() == "connection not found" {
			status = http.StatusNotFound
		} else if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to update this connection" ||
			err.Error() == "cannot update another user's personal connection" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, conn)
}

// DeleteConnection handles DELETE /api/connections/{id}
func (h *Handler) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connectionID := vars["id"]
	if connectionID == "" {
		h.respondError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if err := h.service.DeleteConnection(r.Context(), connectionID, userID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"connection_id": connectionID,
			"user_id":       userID,
		}).Error("Failed to delete connection")

		status := http.StatusInternalServerError
		if err.Error() == "connection not found" {
			status = http.StatusNotFound
		} else if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to delete this connection" ||
			err.Error() == "cannot delete another user's personal connection" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection deleted successfully",
	})
}

// GetConnectionPassword handles GET /api/connections/{id}/password
// Retrieve the decrypted password for a connection
func (h *Handler) GetConnectionPassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connectionID := vars["id"]
	if connectionID == "" {
		h.respondError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get orgID from query param (optional, for shared connections)
	orgID := r.URL.Query().Get("org_id")

	// Get master key from header (Base64-encoded)
	masterKeyHeader := r.Header.Get("X-Master-Key")
	if masterKeyHeader == "" {
		h.respondError(w, http.StatusBadRequest, "X-Master-Key header is required")
		return
	}

	// Decode master key from base64
	masterKey, err := base64.StdEncoding.DecodeString(masterKeyHeader)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid X-Master-Key encoding")
		return
	}
	defer crypto.ClearBytes(masterKey)

	// Get the password
	password, err := h.service.GetConnectionPassword(
		r.Context(),
		connectionID,
		userID,
		orgID,
		masterKey,
	)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"connection_id": connectionID,
			"user_id":       userID,
			"org_id":        orgID,
		}).Error("Failed to get connection password")

		status := http.StatusInternalServerError
		if err.Error() == "connection not found" {
			status = http.StatusNotFound
		} else if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to access this connection" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}
	defer crypto.ClearBytes(password)

	// Encode password to base64 for response
	encodedPassword := base64.StdEncoding.EncodeToString(password)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"password": encodedPassword,
	})
}

// Helper methods

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error": message,
	})
}
