package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Service manages data retention policies
type Service struct {
	store    Store
	archiver Archiver
	logger   *logrus.Logger
}

// NewService creates a new retention service
func NewService(store Store, archiver Archiver, logger *logrus.Logger) *Service {
	return &Service{
		store:    store,
		archiver: archiver,
		logger:   logger,
	}
}

// CreatePolicy creates a new retention policy
func (s *Service) CreatePolicy(ctx context.Context, policy *RetentionPolicy) error {
	// Validate policy
	if err := s.validatePolicy(policy); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	return s.store.CreatePolicy(ctx, policy)
}

// GetPolicy retrieves a retention policy
func (s *Service) GetPolicy(ctx context.Context, orgID, resourceType string) (*RetentionPolicy, error) {
	return s.store.GetPolicy(ctx, orgID, resourceType)
}

// GetOrganizationPolicies retrieves all policies for an organization
func (s *Service) GetOrganizationPolicies(ctx context.Context, orgID string) ([]*RetentionPolicy, error) {
	return s.store.GetOrganizationPolicies(ctx, orgID)
}

// UpdatePolicy updates a retention policy
func (s *Service) UpdatePolicy(ctx context.Context, policy *RetentionPolicy) error {
	if err := s.validatePolicy(policy); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	return s.store.UpdatePolicy(ctx, policy)
}

// DeletePolicy deletes a retention policy
func (s *Service) DeletePolicy(ctx context.Context, orgID, resourceType string) error {
	return s.store.DeletePolicy(ctx, orgID, resourceType)
}

// ApplyRetentionPolicies applies all active retention policies
func (s *Service) ApplyRetentionPolicies(ctx context.Context) error {
	policies, err := s.store.GetAllPolicies(ctx)
	if err != nil {
		return fmt.Errorf("get policies: %w", err)
	}

	s.logger.WithField("policies", len(policies)).Info("Applying retention policies")

	for _, policy := range policies {
		if err := s.applyPolicy(ctx, policy); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"policy_id":       policy.ID,
				"organization_id": policy.OrganizationID,
				"resource_type":   policy.ResourceType,
			}).Error("Failed to apply retention policy")
			continue
		}
	}

	return nil
}

// applyPolicy applies a single retention policy
func (s *Service) applyPolicy(ctx context.Context, policy *RetentionPolicy) error {
	logger := s.logger.WithFields(logrus.Fields{
		"policy_id":       policy.ID,
		"organization_id": policy.OrganizationID,
		"resource_type":   policy.ResourceType,
		"retention_days":  policy.RetentionDays,
	})

	logger.Info("Applying retention policy")

	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -policy.RetentionDays)

	// Get data to archive/delete
	var oldRecords []map[string]interface{}
	var err error

	switch policy.ResourceType {
	case "query_history":
		oldRecords, err = s.store.GetOldQueryHistory(ctx, policy.OrganizationID, cutoffDate)
	case "audit_logs":
		oldRecords, err = s.store.GetOldAuditLogs(ctx, policy.OrganizationID, cutoffDate)
	case "connections":
		oldRecords, err = s.store.GetOldConnections(ctx, policy.OrganizationID, cutoffDate)
	case "templates":
		oldRecords, err = s.store.GetOldTemplates(ctx, policy.OrganizationID, cutoffDate)
	default:
		return fmt.Errorf("unknown resource type: %s", policy.ResourceType)
	}

	if err != nil {
		return fmt.Errorf("get old records: %w", err)
	}

	if len(oldRecords) == 0 {
		logger.Info("No records to archive")
		return nil
	}

	logger.WithField("records", len(oldRecords)).Info("Found records to archive")

	// Archive if enabled
	var archiveLocation string
	if policy.AutoArchive {
		archiveLocation, err = s.archiver.Archive(ctx, policy.ResourceType, oldRecords)
		if err != nil {
			return fmt.Errorf("archive data: %w", err)
		}
		logger.WithField("location", archiveLocation).Info("Archived data")
	}

	// Delete old records
	var deletedCount int64
	switch policy.ResourceType {
	case "query_history":
		deletedCount, err = s.store.DeleteQueryHistory(ctx, policy.OrganizationID, cutoffDate)
	case "audit_logs":
		deletedCount, err = s.store.DeleteAuditLogs(ctx, policy.OrganizationID, cutoffDate)
	case "connections":
		deletedCount, err = s.store.DeleteConnections(ctx, policy.OrganizationID, cutoffDate)
	case "templates":
		deletedCount, err = s.store.DeleteTemplates(ctx, policy.OrganizationID, cutoffDate)
	}

	if err != nil {
		return fmt.Errorf("delete old records: %w", err)
	}

	logger.WithField("deleted", deletedCount).Info("Deleted old records")

	// Log archive operation
	if policy.AutoArchive {
		archiveLog := &ArchiveLog{
			OrganizationID:  policy.OrganizationID,
			ResourceType:    policy.ResourceType,
			RecordsArchived: int(deletedCount),
			ArchiveLocation: archiveLocation,
			ArchiveDate:     time.Now(),
			CutoffDate:      cutoffDate,
		}
		if err := s.store.CreateArchiveLog(ctx, archiveLog); err != nil {
			s.logger.WithError(err).Error("Failed to create archive log")
		}
	}

	return nil
}

// GetRetentionStats returns statistics about data retention
func (s *Service) GetRetentionStats(ctx context.Context, orgID, resourceType string) (*RetentionStats, error) {
	stats, err := s.store.GetRetentionStats(ctx, orgID, resourceType)
	if err != nil {
		return nil, err
	}

	// Get policy to calculate records to archive
	policy, err := s.store.GetPolicy(ctx, orgID, resourceType)
	if err == nil {
		cutoffDate := time.Now().AddDate(0, 0, -policy.RetentionDays)

		var oldRecords []map[string]interface{}
		switch resourceType {
		case "query_history":
			oldRecords, _ = s.store.GetOldQueryHistory(ctx, orgID, cutoffDate)
		case "audit_logs":
			oldRecords, _ = s.store.GetOldAuditLogs(ctx, orgID, cutoffDate)
		case "connections":
			oldRecords, _ = s.store.GetOldConnections(ctx, orgID, cutoffDate)
		case "templates":
			oldRecords, _ = s.store.GetOldTemplates(ctx, orgID, cutoffDate)
		}

		stats.RecordsToArchive = len(oldRecords)
	}

	return stats, nil
}

// GetArchiveLogs returns archive logs for an organization
func (s *Service) GetArchiveLogs(ctx context.Context, orgID string, since time.Time) ([]*ArchiveLog, error) {
	return s.store.GetArchiveLogs(ctx, orgID, since)
}

// validatePolicy validates a retention policy
func (s *Service) validatePolicy(policy *RetentionPolicy) error {
	if policy.OrganizationID == "" {
		return fmt.Errorf("organization_id is required")
	}

	if policy.ResourceType == "" {
		return fmt.Errorf("resource_type is required")
	}

	validResourceTypes := map[string]bool{
		"query_history": true,
		"audit_logs":    true,
		"connections":   true,
		"templates":     true,
	}

	if !validResourceTypes[policy.ResourceType] {
		return fmt.Errorf("invalid resource_type: %s", policy.ResourceType)
	}

	if policy.RetentionDays < 1 {
		return fmt.Errorf("retention_days must be at least 1")
	}

	if policy.RetentionDays > 3650 { // 10 years max
		return fmt.Errorf("retention_days cannot exceed 3650 (10 years)")
	}

	if policy.CreatedBy == "" {
		return fmt.Errorf("created_by is required")
	}

	return nil
}

// StartScheduler starts the retention policy scheduler
// Runs daily at 2 AM local time
func (s *Service) StartScheduler(ctx context.Context) {
	s.logger.Info("Starting retention policy scheduler")

	// Calculate next 2 AM
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}

	// Wait until 2 AM
	duration := next.Sub(now)
	s.logger.WithField("next_run", next).Info("Scheduler will run at next scheduled time")

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Retention scheduler stopped")
			return
		case <-timer.C:
			s.logger.Info("Running scheduled retention policy enforcement")

			if err := s.ApplyRetentionPolicies(ctx); err != nil {
				s.logger.WithError(err).Error("Failed to apply retention policies")
			}

			// Schedule next run in 24 hours
			timer.Reset(24 * time.Hour)
		}
	}
}
