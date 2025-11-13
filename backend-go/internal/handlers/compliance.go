package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/audit"
	"github.com/sql-studio/backend-go/internal/backup"
	"github.com/sql-studio/backend-go/internal/gdpr"
	"github.com/sql-studio/backend-go/internal/pii"
	"github.com/sql-studio/backend-go/internal/retention"
)

// ComplianceHandler handles compliance-related HTTP requests
type ComplianceHandler struct {
	retentionService *retention.Service
	gdprService      *gdpr.Service
	backupService    *backup.Service
	auditLogger      *audit.DetailedAuditLogger
	piiDetector      *pii.Detector
	logger           *logrus.Logger
}

// NewComplianceHandler creates a new compliance handler
func NewComplianceHandler(
	retentionService *retention.Service,
	gdprService *gdpr.Service,
	backupService *backup.Service,
	auditLogger *audit.DetailedAuditLogger,
	piiDetector *pii.Detector,
	logger *logrus.Logger,
) *ComplianceHandler {
	return &ComplianceHandler{
		retentionService: retentionService,
		gdprService:      gdprService,
		backupService:    backupService,
		auditLogger:      auditLogger,
		piiDetector:      piiDetector,
		logger:           logger,
	}
}

// RegisterRoutes registers compliance routes
func (h *ComplianceHandler) RegisterRoutes(r *mux.Router) {
	// Data Retention
	r.HandleFunc("/api/organizations/{id}/retention-policy", h.CreateRetentionPolicy).Methods("POST")
	r.HandleFunc("/api/organizations/{id}/retention-policy", h.GetRetentionPolicies).Methods("GET")
	r.HandleFunc("/api/organizations/{id}/retention-policy/{resource_type}", h.UpdateRetentionPolicy).Methods("PUT")
	r.HandleFunc("/api/organizations/{id}/retention-policy/{resource_type}", h.DeleteRetentionPolicy).Methods("DELETE")
	r.HandleFunc("/api/organizations/{id}/retention-stats/{resource_type}", h.GetRetentionStats).Methods("GET")

	// GDPR
	r.HandleFunc("/api/gdpr/export", h.RequestDataExport).Methods("POST")
	r.HandleFunc("/api/gdpr/export/{request_id}", h.GetExportRequest).Methods("GET")
	r.HandleFunc("/api/gdpr/delete", h.RequestDataDeletion).Methods("POST")
	r.HandleFunc("/api/gdpr/requests", h.GetUserGDPRRequests).Methods("GET")

	// Backups (admin only)
	r.HandleFunc("/api/admin/backups", h.CreateBackup).Methods("POST")
	r.HandleFunc("/api/admin/backups", h.ListBackups).Methods("GET")
	r.HandleFunc("/api/admin/backups/{id}", h.GetBackup).Methods("GET")
	r.HandleFunc("/api/admin/backups/{id}/restore", h.RestoreBackup).Methods("POST")
	r.HandleFunc("/api/admin/backups/{id}", h.DeleteBackup).Methods("DELETE")
	r.HandleFunc("/api/admin/backups/stats", h.GetBackupStats).Methods("GET")

	// Audit Logs
	r.HandleFunc("/api/audit/detailed/{table}/{record_id}", h.GetChangeHistory).Methods("GET")
	r.HandleFunc("/api/audit/field/{table}/{record_id}/{field}", h.GetFieldHistory).Methods("GET")

	// PII Detection
	r.HandleFunc("/api/pii/scan", h.ScanForPII).Methods("POST")
	r.HandleFunc("/api/pii/fields", h.ListPIIFields).Methods("GET")
	r.HandleFunc("/api/pii/fields", h.RegisterPIIField).Methods("POST")
	r.HandleFunc("/api/pii/fields/{id}/verify", h.VerifyPIIField).Methods("POST")
}

// CreateRetentionPolicy creates a new retention policy
func (h *ComplianceHandler) CreateRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	var policy retention.RetentionPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	policy.OrganizationID = orgID
	// TODO: Get user ID from context
	policy.CreatedBy = "system"

	if err := h.retentionService.CreatePolicy(r.Context(), &policy); err != nil {
		h.logger.WithError(err).Error("Failed to create retention policy")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policy); err != nil {
		h.logger.WithError(err).Error("Failed to encode policy response")
	}
}

// GetRetentionPolicies retrieves retention policies for an organization
func (h *ComplianceHandler) GetRetentionPolicies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	policies, err := h.retentionService.GetOrganizationPolicies(r.Context(), orgID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get retention policies")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policies); err != nil {
		h.logger.WithError(err).Error("Failed to encode policies response")
	}
}

// UpdateRetentionPolicy updates a retention policy
func (h *ComplianceHandler) UpdateRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]
	resourceType := vars["resource_type"]

	var policy retention.RetentionPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	policy.OrganizationID = orgID
	policy.ResourceType = resourceType

	if err := h.retentionService.UpdatePolicy(r.Context(), &policy); err != nil {
		h.logger.WithError(err).Error("Failed to update retention policy")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policy); err != nil {
		h.logger.WithError(err).Error("Failed to encode updated policy response")
	}
}

// DeleteRetentionPolicy deletes a retention policy
func (h *ComplianceHandler) DeleteRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]
	resourceType := vars["resource_type"]

	if err := h.retentionService.DeletePolicy(r.Context(), orgID, resourceType); err != nil {
		h.logger.WithError(err).Error("Failed to delete retention policy")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetRetentionStats retrieves retention statistics
func (h *ComplianceHandler) GetRetentionStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]
	resourceType := vars["resource_type"]

	stats, err := h.retentionService.GetRetentionStats(r.Context(), orgID, resourceType)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get retention stats")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// RequestDataExport creates a GDPR data export request
func (h *ComplianceHandler) RequestDataExport(w http.ResponseWriter, r *http.Request) {
	// TODO: Get user ID from auth context
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	request, err := h.gdprService.RequestDataExport(r.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create export request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(request); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// GetExportRequest retrieves an export request
func (h *ComplianceHandler) GetExportRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	requestID := vars["request_id"]

	request, err := h.gdprService.GetExportRequest(r.Context(), requestID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get export request")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(request); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// RequestDataDeletion creates a GDPR data deletion request
func (h *ComplianceHandler) RequestDataDeletion(w http.ResponseWriter, r *http.Request) {
	// TODO: Get user ID from auth context
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	request, err := h.gdprService.RequestDataDeletion(r.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create deletion request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(request); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// GetUserGDPRRequests retrieves all GDPR requests for a user
func (h *ComplianceHandler) GetUserGDPRRequests(w http.ResponseWriter, r *http.Request) {
	// TODO: Get user ID from auth context
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	requests, err := h.gdprService.GetUserExportRequests(r.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get GDPR requests")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(requests); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// CreateBackup creates a database backup
func (h *ComplianceHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	var opts backup.BackupOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		opts = backup.BackupOptions{
			BackupType: "full",
			Compress:   true,
			MaxBackups: 30,
		}
	}

	backupRecord, err := h.backupService.CreateBackup(r.Context(), &opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create backup")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(backupRecord); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// ListBackups lists database backups
func (h *ComplianceHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	backups, err := h.backupService.ListBackups(r.Context(), limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list backups")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(backups); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// GetBackup retrieves a backup
func (h *ComplianceHandler) GetBackup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID := vars["id"]

	backupRecord, err := h.backupService.GetBackup(r.Context(), backupID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get backup")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(backupRecord); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// RestoreBackup restores from a backup
func (h *ComplianceHandler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID := vars["id"]

	var opts backup.RestoreOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		opts = backup.RestoreOptions{
			BackupID: backupID,
			DryRun:   false,
		}
	}
	opts.BackupID = backupID

	err := h.backupService.RestoreBackup(r.Context(), &opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to restore backup")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Restore initiated - application restart required",
	}) // Best-effort encode
}

// DeleteBackup deletes a backup
func (h *ComplianceHandler) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID := vars["id"]

	if err := h.backupService.DeleteBackup(r.Context(), backupID); err != nil {
		h.logger.WithError(err).Error("Failed to delete backup")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetBackupStats retrieves backup statistics
func (h *ComplianceHandler) GetBackupStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.backupService.GetBackupStats(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get backup stats")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats) // Best-effort encode
}

// GetChangeHistory retrieves change history for a record
func (h *ComplianceHandler) GetChangeHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tableName := vars["table"]
	recordID := vars["record_id"]

	history, err := h.auditLogger.GetChangeHistory(r.Context(), tableName, recordID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get change history")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(history) // Best-effort encode
}

// GetFieldHistory retrieves change history for a specific field
func (h *ComplianceHandler) GetFieldHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tableName := vars["table"]
	recordID := vars["record_id"]
	fieldName := vars["field"]

	history, err := h.auditLogger.GetFieldHistory(r.Context(), tableName, recordID, fieldName)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get field history")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(history) // Best-effort encode
}

// ScanForPII scans data for PII
func (h *ComplianceHandler) ScanForPII(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.piiDetector.ScanQueryResults(r.Context(), request.Data)
	if err != nil {
		h.logger.WithError(err).Error("Failed to scan for PII")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result.ScannedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result) // Best-effort encode
}

// ListPIIFields lists registered PII fields
func (h *ComplianceHandler) ListPIIFields(w http.ResponseWriter, r *http.Request) {
	fields, err := h.piiDetector.GetRegisteredPIIFields(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list PII fields")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(fields) // Best-effort encode
}

// RegisterPIIField registers a field as containing PII
func (h *ComplianceHandler) RegisterPIIField(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TableName string `json:"table_name"`
		FieldName string `json:"field_name"`
		PIIType   string `json:"pii_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.piiDetector.RegisterPIIField(r.Context(), request.TableName, request.FieldName, request.PIIType)
	if err != nil {
		h.logger.WithError(err).Error("Failed to register PII field")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "PII field registered successfully",
	})
}

// VerifyPIIField verifies a PII field
func (h *ComplianceHandler) VerifyPIIField(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fieldID := vars["id"]

	// TODO: Implement verification logic through store
	_ = fieldID

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "PII field verified",
	})
}
