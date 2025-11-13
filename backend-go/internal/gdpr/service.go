package gdpr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// Service handles GDPR compliance operations
type Service struct {
	store      Store
	exportPath string
	logger     *logrus.Logger
}

// NewService creates a new GDPR service
func NewService(store Store, exportPath string, logger *logrus.Logger) *Service {
	return &Service{
		store:      store,
		exportPath: exportPath,
		logger:     logger,
	}
}

// RequestDataExport creates a request to export all user data (GDPR Article 15 - Right to access)
func (s *Service) RequestDataExport(ctx context.Context, userID string) (*DataExportRequest, error) {
	// Check for existing pending/processing requests
	existing, err := s.store.GetUserExportRequests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check existing requests: %w", err)
	}

	for _, req := range existing {
		if req.RequestType == "export" && (req.Status == "pending" || req.Status == "processing") {
			return nil, fmt.Errorf("export request already in progress")
		}
	}

	// Create export request
	request := &DataExportRequest{
		UserID:      userID,
		RequestType: "export",
		Status:      "pending",
		RequestedAt: time.Now(),
	}

	err = s.store.CreateExportRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create export request: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"request_id": request.ID,
		"user_id":    userID,
	}).Info("Created data export request")

	// Start async export
	go s.performExport(request)

	return request, nil
}

// RequestDataDeletion creates a request to delete all user data (GDPR Article 17 - Right to erasure)
func (s *Service) RequestDataDeletion(ctx context.Context, userID string) (*DataExportRequest, error) {
	// Check for existing pending/processing deletion requests
	existing, err := s.store.GetUserExportRequests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check existing requests: %w", err)
	}

	for _, req := range existing {
		if req.RequestType == "delete" && (req.Status == "pending" || req.Status == "processing") {
			return nil, fmt.Errorf("deletion request already in progress")
		}
	}

	// Create deletion request
	request := &DataExportRequest{
		UserID:      userID,
		RequestType: "delete",
		Status:      "pending",
		RequestedAt: time.Now(),
	}

	err = s.store.CreateExportRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create deletion request: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"request_id": request.ID,
		"user_id":    userID,
	}).Warn("Created data deletion request")

	// Start async deletion
	go s.performDeletion(request)

	return request, nil
}

// GetExportRequest retrieves an export request
func (s *Service) GetExportRequest(ctx context.Context, requestID string) (*DataExportRequest, error) {
	return s.store.GetExportRequest(ctx, requestID)
}

// GetUserExportRequests retrieves all export requests for a user
func (s *Service) GetUserExportRequests(ctx context.Context, userID string) ([]*DataExportRequest, error) {
	return s.store.GetUserExportRequests(ctx, userID)
}

// performExport executes the data export in the background
func (s *Service) performExport(request *DataExportRequest) {
	ctx := context.Background()

	logger := s.logger.WithFields(logrus.Fields{
		"request_id": request.ID,
		"user_id":    request.UserID,
	})

	logger.Info("Starting data export")

	// Update status
	if err := s.store.UpdateRequestStatus(ctx, request.ID, "processing"); err != nil {
		logger.WithError(err).Error("Failed to update request status")
		return
	}

	// Collect all user data
	userData, err := s.collectUserData(ctx, request.UserID)
	if err != nil {
		logger.WithError(err).Error("Failed to collect user data")
		s.store.UpdateRequestFailed(ctx, request.ID, err.Error())
		return
	}

	// Export to JSON
	jsonData, err := json.MarshalIndent(userData, "", "  ")
	if err != nil {
		logger.WithError(err).Error("Failed to marshal user data")
		s.store.UpdateRequestFailed(ctx, request.ID, "Failed to serialize data")
		return
	}

	// Ensure export directory exists
	if err := os.MkdirAll(s.exportPath, 0750); err != nil {
		logger.WithError(err).Error("Failed to create export directory")
		s.store.UpdateRequestFailed(ctx, request.ID, "Failed to create export directory")
		return
	}

	// Save to file
	filename := fmt.Sprintf("user_%s_%d.json", request.UserID, time.Now().Unix())
	filePath := filepath.Join(s.exportPath, filename)

	if err := os.WriteFile(filePath, jsonData, 0600); err != nil {
		logger.WithError(err).Error("Failed to write export file")
		s.store.UpdateRequestFailed(ctx, request.ID, "Failed to write export file")
		return
	}

	logger.WithField("file_path", filePath).Info("Data export completed")

	// Update request with file location
	if err := s.store.UpdateRequestComplete(ctx, request.ID, filePath); err != nil {
		logger.WithError(err).Error("Failed to update request")
	}
}

// collectUserData collects all user data for export
func (s *Service) collectUserData(ctx context.Context, userID string) (*UserDataExport, error) {
	export := &UserDataExport{
		ExportedAt:    time.Now(),
		ExportVersion: "1.0",
		Metadata: map[string]interface{}{
			"user_id": userID,
			"format":  "json",
		},
	}

	var err error

	// Collect user data
	export.User, err = s.store.GetUserData(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user data: %w", err)
	}

	// Collect connections
	export.Connections, err = s.store.GetConnections(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get connections")
		export.Connections = []interface{}{}
	}

	// Collect queries
	export.Queries, err = s.store.GetQueries(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get queries")
		export.Queries = []interface{}{}
	}

	// Collect query history
	export.QueryHistory, err = s.store.GetQueryHistory(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get query history")
		export.QueryHistory = []interface{}{}
	}

	// Collect templates
	export.Templates, err = s.store.GetTemplates(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get templates")
		export.Templates = []interface{}{}
	}

	// Collect schedules
	export.Schedules, err = s.store.GetSchedules(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get schedules")
		export.Schedules = []interface{}{}
	}

	// Collect organizations
	export.Organizations, err = s.store.GetOrganizations(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get organizations")
		export.Organizations = []interface{}{}
	}

	// Collect audit logs
	export.AuditLogs, err = s.store.GetAuditLogs(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get audit logs")
		export.AuditLogs = []interface{}{}
	}

	return export, nil
}

// performDeletion executes the data deletion in the background
func (s *Service) performDeletion(request *DataExportRequest) {
	ctx := context.Background()

	logger := s.logger.WithFields(logrus.Fields{
		"request_id": request.ID,
		"user_id":    request.UserID,
	})

	logger.Warn("Starting data deletion")

	// Update status
	if err := s.store.UpdateRequestStatus(ctx, request.ID, "processing"); err != nil {
		logger.WithError(err).Error("Failed to update request status")
		return
	}

	report := &DeletionReport{
		UserID:    request.UserID,
		DeletedAt: time.Now(),
		Details:   make(map[string]int),
	}

	// Delete connections
	if count, err := s.store.DeleteConnections(ctx, request.UserID); err != nil {
		logger.WithError(err).Error("Failed to delete connections")
	} else {
		report.ConnectionsDeleted = int(count)
		report.Details["connections"] = int(count)
	}

	// Delete queries
	if count, err := s.store.DeleteQueries(ctx, request.UserID); err != nil {
		logger.WithError(err).Error("Failed to delete queries")
	} else {
		report.QueriesDeleted = int(count)
		report.Details["queries"] = int(count)
	}

	// Delete query history
	if count, err := s.store.DeleteQueryHistory(ctx, request.UserID); err != nil {
		logger.WithError(err).Error("Failed to delete query history")
	} else {
		report.HistoryDeleted = int(count)
		report.Details["query_history"] = int(count)
	}

	// Delete templates
	if count, err := s.store.DeleteTemplates(ctx, request.UserID); err != nil {
		logger.WithError(err).Error("Failed to delete templates")
	} else {
		report.TemplatesDeleted = int(count)
		report.Details["templates"] = int(count)
	}

	// Delete schedules
	if count, err := s.store.DeleteSchedules(ctx, request.UserID); err != nil {
		logger.WithError(err).Error("Failed to delete schedules")
	} else {
		report.SchedulesDeleted = int(count)
		report.Details["schedules"] = int(count)
	}

	// Anonymize audit logs (keep records but remove PII)
	if count, err := s.store.AnonymizeAuditLogs(ctx, request.UserID); err != nil {
		logger.WithError(err).Error("Failed to anonymize audit logs")
	} else {
		report.AuditLogsAnonymized = int(count)
		report.Details["audit_logs_anonymized"] = int(count)
	}

	// Finally, delete user account
	if err := s.store.DeleteUser(ctx, request.UserID); err != nil {
		logger.WithError(err).Error("Failed to delete user")
		s.store.UpdateRequestFailed(ctx, request.ID, "Failed to delete user account")
		return
	}

	logger.WithFields(logrus.Fields{
		"connections":     report.ConnectionsDeleted,
		"queries":         report.QueriesDeleted,
		"history":         report.HistoryDeleted,
		"templates":       report.TemplatesDeleted,
		"schedules":       report.SchedulesDeleted,
		"logs_anonymized": report.AuditLogsAnonymized,
	}).Warn("Data deletion completed")

	// Serialize report
	reportJSON, _ := json.Marshal(report)

	// Mark complete
	if err := s.store.UpdateRequestComplete(ctx, request.ID, string(reportJSON)); err != nil {
		logger.WithError(err).Error("Failed to update request")
	}
}
