package configexport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jbeck018/howlerops/backend-go/internal/middleware"
	"github.com/jbeck018/howlerops/backend-go/pkg/export"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests for config export/import
type Handler struct {
	service *Service
	logger  *logrus.Logger
}

// NewHandler creates a new config export handler
func NewHandler(service *Service, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// ExportConfigRequest is the request body for config export
type ExportConfigRequest struct {
	// Include connections (without passwords)
	IncludeConnections bool `json:"include_connections"`
	// Include saved queries
	IncludeSavedQueries bool `json:"include_saved_queries"`
	// Include query history (sanitized)
	IncludeQueryHistory bool `json:"include_query_history"`
	// Only export specific connection IDs
	ConnectionIDs []string `json:"connection_ids,omitempty"`
	// Only export queries with these tags
	QueryTags []string `json:"query_tags,omitempty"`
	// Include shared resources
	IncludeShared bool `json:"include_shared"`
	// Export metadata only (no query SQL)
	MetadataOnly bool `json:"metadata_only"`
	// Anonymize hostnames
	AnonymizeHosts bool `json:"anonymize_hosts"`
	// Output format: "json" or "pretty" (default: pretty)
	Format string `json:"format"`
}

// ImportConfigRequest is the request body for config import
type ImportConfigRequest struct {
	// The config data (JSON string)
	Config json.RawMessage `json:"config"`
	// Conflict handling strategy
	ConflictStrategy string `json:"conflict_strategy"`
	// Import connections
	ImportConnections bool `json:"import_connections"`
	// Import saved queries
	ImportSavedQueries bool `json:"import_saved_queries"`
	// Only import specific connection export IDs
	ConnectionExportIDs []string `json:"connection_export_ids,omitempty"`
	// Only import queries with these tags
	QueryTags []string `json:"query_tags,omitempty"`
	// Host overrides (export_id -> actual_host)
	HostOverrides map[string]string `json:"host_overrides,omitempty"`
	// Share with this organization
	ShareWithOrganization string `json:"share_with_organization,omitempty"`
	// Dry run (validate only)
	DryRun bool `json:"dry_run"`
}

// ExportConfig handles GET/POST /api/config/export
func (h *Handler) ExportConfig(w http.ResponseWriter, r *http.Request) {
	// Get user info from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userEmail, _ := r.Context().Value(middleware.UserEmailKey).(string)

	// Parse options from query params or body
	options := export.DefaultConfigExportOptions()

	if r.Method == http.MethodPost {
		var req ExportConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		options.IncludeConnections = req.IncludeConnections
		options.IncludeSavedQueries = req.IncludeSavedQueries
		options.IncludeQueryHistory = req.IncludeQueryHistory
		options.ConnectionIDs = req.ConnectionIDs
		options.QueryTags = req.QueryTags
		options.IncludeShared = req.IncludeShared
		options.MetadataOnly = req.MetadataOnly
		options.AnonymizeHosts = req.AnonymizeHosts
	} else {
		// Parse from query string for GET requests
		if r.URL.Query().Get("include_connections") == "true" {
			options.IncludeConnections = true
		}
		if r.URL.Query().Get("include_queries") == "true" {
			options.IncludeSavedQueries = true
		}
		if r.URL.Query().Get("include_history") == "true" {
			options.IncludeQueryHistory = true
		}
		if r.URL.Query().Get("include_shared") == "true" {
			options.IncludeShared = true
		}
		if r.URL.Query().Get("metadata_only") == "true" {
			options.MetadataOnly = true
		}
		if r.URL.Query().Get("anonymize") == "true" {
			options.AnonymizeHosts = true
		}
	}

	// Export config
	config, err := h.service.Export(r.Context(), userID, userEmail, options)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to export config")
		h.respondError(w, http.StatusInternalServerError, "failed to export config")
		return
	}

	// Determine format
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pretty"
	}

	data, err := config.ToJSON(format == "pretty")
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to serialize config")
		return
	}

	// Set headers for download
	filename := fmt.Sprintf("howlerops-config-%s.json", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// ImportConfig handles POST /api/config/import
func (h *Handler) ImportConfig(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ImportConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Parse the config
	config, err := export.ParseConfig(req.Config)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", err))
		return
	}

	// Validate config
	if issues := export.ValidateConfig(config); len(issues) > 0 {
		h.respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success":          false,
			"message":          "config validation failed",
			"validation_issues": issues,
		})
		return
	}

	// Build import options
	options := export.DefaultConfigImportOptions()
	options.ImportConnections = req.ImportConnections
	options.ImportSavedQueries = req.ImportSavedQueries
	options.ConnectionExportIDs = req.ConnectionExportIDs
	options.QueryTags = req.QueryTags
	options.HostOverrides = req.HostOverrides
	options.ShareWithOrganization = req.ShareWithOrganization
	options.DryRun = req.DryRun

	// Parse conflict strategy
	switch req.ConflictStrategy {
	case "skip", "":
		options.ConflictStrategy = export.ConflictSkip
	case "overwrite":
		options.ConflictStrategy = export.ConflictOverwrite
	case "rename":
		options.ConflictStrategy = export.ConflictRename
	case "merge":
		options.ConflictStrategy = export.ConflictMerge
	default:
		h.respondError(w, http.StatusBadRequest, "invalid conflict_strategy: must be skip, overwrite, rename, or merge")
		return
	}

	// Import
	result, err := h.service.Import(r.Context(), config, userID, options)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to import config")
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("import failed: %v", err))
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Imported %d connections, %d queries", result.ConnectionsImported, result.QueriesImported),
		"result":  result,
	})
}

// ValidateConfig handles POST /api/config/validate
func (h *Handler) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	// Read config from body
	data, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	config, err := export.ParseConfig(data)
	if err != nil {
		h.respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"error":  err.Error(),
			"issues": []string{},
		})
		return
	}

	issues := export.ValidateConfig(config)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":       len(issues) == 0,
		"issues":      issues,
		"connections": len(config.Connections),
		"queries":     len(config.SavedQueries),
		"tags":        config.Tags,
		"folders":     config.Folders,
		"exported_at": config.ExportedAt,
		"exported_by": config.ExportedBy,
	})
}

// PreviewImport handles POST /api/config/preview
func (h *Handler) PreviewImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ImportConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	config, err := export.ParseConfig(req.Config)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", err))
		return
	}

	// Force dry run for preview
	options := export.DefaultConfigImportOptions()
	options.ImportConnections = req.ImportConnections
	options.ImportSavedQueries = req.ImportSavedQueries
	options.DryRun = true

	result, err := h.service.Import(r.Context(), config, userID, options)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("preview failed: %v", err))
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"preview": result,
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
		"success": false,
		"error":   message,
	})
}
