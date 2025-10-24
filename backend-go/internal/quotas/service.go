package quotas

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Service handles quota enforcement and usage tracking
type Service struct {
	store  *Store
	logger *logrus.Logger
}

// NewService creates a new quota service
func NewService(store *Store, logger *logrus.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
	}
}

// CheckQuota checks if organization has quota available for resource
func (s *Service) CheckQuota(ctx context.Context, orgID string, resourceType ResourceType) error {
	// Get organization quota
	quota, err := s.store.GetQuota(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get quota: %w", err)
	}

	// Get current usage
	usage, err := s.store.GetTodayUsage(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get usage: %w", err)
	}

	// Check limits
	switch resourceType {
	case ResourceConnection:
		if usage.ConnectionsCount >= quota.MaxConnections {
			return &QuotaExceededError{
				ResourceType: resourceType,
				Current:      usage.ConnectionsCount,
				Limit:        quota.MaxConnections,
				Message:      fmt.Sprintf("Connection quota exceeded (%d/%d)", usage.ConnectionsCount, quota.MaxConnections),
			}
		}

	case ResourceQuery:
		if usage.QueriesCount >= quota.MaxQueriesPerDay {
			return &QuotaExceededError{
				ResourceType: resourceType,
				Current:      usage.QueriesCount,
				Limit:        quota.MaxQueriesPerDay,
				Message:      fmt.Sprintf("Daily query quota exceeded (%d/%d)", usage.QueriesCount, quota.MaxQueriesPerDay),
			}
		}

	case ResourceStorage:
		if usage.StorageUsedMB >= float64(quota.MaxStorageMB) {
			return &QuotaExceededError{
				ResourceType: resourceType,
				Current:      int(usage.StorageUsedMB),
				Limit:        quota.MaxStorageMB,
				Message:      fmt.Sprintf("Storage quota exceeded (%.2fMB/%dMB)", usage.StorageUsedMB, quota.MaxStorageMB),
			}
		}

	case ResourceAPI:
		// For API, check hourly usage
		if err := s.checkHourlyAPIQuota(ctx, orgID, quota.MaxAPICallsPerHour); err != nil {
			return err
		}

	case ResourceConcurrentQuery:
		if usage.ConcurrentQueriesPeak >= quota.MaxConcurrentQueries {
			return &QuotaExceededError{
				ResourceType: resourceType,
				Current:      usage.ConcurrentQueriesPeak,
				Limit:        quota.MaxConcurrentQueries,
				Message:      fmt.Sprintf("Concurrent query quota exceeded (%d/%d)", usage.ConcurrentQueriesPeak, quota.MaxConcurrentQueries),
			}
		}

	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	return nil
}

// IncrementUsage increments usage counter
func (s *Service) IncrementUsage(ctx context.Context, orgID string, resourceType ResourceType) error {
	return s.IncrementUsageBy(ctx, orgID, resourceType, 1)
}

// IncrementUsageBy increments usage counter by specific amount
func (s *Service) IncrementUsageBy(ctx context.Context, orgID string, resourceType ResourceType, amount int) error {
	if err := s.store.IncrementUsage(ctx, orgID, resourceType, amount); err != nil {
		return fmt.Errorf("increment usage: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"resource_type":   resourceType,
		"amount":          amount,
	}).Debug("Usage incremented")

	return nil
}

// GetUsageStatistics retrieves aggregated usage statistics
func (s *Service) GetUsageStatistics(ctx context.Context, orgID string, days int) (*UsageStatistics, error) {
	// Get quota limits
	quota, err := s.store.GetQuota(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get quota: %w", err)
	}

	// Get usage history
	usageHistory, err := s.store.GetUsageHistory(ctx, orgID, days)
	if err != nil {
		return nil, fmt.Errorf("get usage history: %w", err)
	}

	// Calculate statistics
	stats := &UsageStatistics{
		OrganizationID: orgID,
		Period:         fmt.Sprintf("last_%d_days", days),
		QuotaLimits:    *quota,
		DailyUsage:     make([]DailyUsageSummary, 0),
	}

	totalQueries := 0
	totalAPICalls := 0
	peakConcurrent := 0
	currentStorage := 0.0

	for _, usage := range usageHistory {
		totalQueries += usage.QueriesCount
		totalAPICalls += usage.APICallsCount

		if usage.ConcurrentQueriesPeak > peakConcurrent {
			peakConcurrent = usage.ConcurrentQueriesPeak
		}

		if usage.StorageUsedMB > currentStorage {
			currentStorage = usage.StorageUsedMB
		}

		stats.DailyUsage = append(stats.DailyUsage, DailyUsageSummary{
			Date:             usage.UsageDate.Format("2006-01-02"),
			QueriesCount:     usage.QueriesCount,
			APICallsCount:    usage.APICallsCount,
			ConnectionsCount: usage.ConnectionsCount,
			StorageUsedMB:    usage.StorageUsedMB,
		})
	}

	stats.TotalQueries = totalQueries
	stats.TotalAPIcalls = totalAPICalls
	stats.PeakConcurrentQueries = peakConcurrent
	stats.CurrentStorageUsedMB = currentStorage

	if len(usageHistory) > 0 {
		stats.AverageQueriesPerDay = float64(totalQueries) / float64(len(usageHistory))
		stats.AverageAPICallsPerDay = float64(totalAPICalls) / float64(len(usageHistory))
	}

	return stats, nil
}

// GetQuota retrieves quota settings for an organization
func (s *Service) GetQuota(ctx context.Context, orgID string) (*OrganizationQuota, error) {
	return s.store.GetQuota(ctx, orgID)
}

// UpdateQuota updates quota settings
func (s *Service) UpdateQuota(ctx context.Context, orgID string, req *UpdateQuotaRequest) (*OrganizationQuota, error) {
	// Get current quota
	quota, err := s.store.GetQuota(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get quota: %w", err)
	}

	// Apply updates
	if req.MaxConnections != nil {
		if *req.MaxConnections < 1 {
			return nil, fmt.Errorf("max_connections must be at least 1")
		}
		quota.MaxConnections = *req.MaxConnections
	}

	if req.MaxQueriesPerDay != nil {
		if *req.MaxQueriesPerDay < 1 {
			return nil, fmt.Errorf("max_queries_per_day must be at least 1")
		}
		quota.MaxQueriesPerDay = *req.MaxQueriesPerDay
	}

	if req.MaxStorageMB != nil {
		if *req.MaxStorageMB < 1 {
			return nil, fmt.Errorf("max_storage_mb must be at least 1")
		}
		quota.MaxStorageMB = *req.MaxStorageMB
	}

	if req.MaxAPICallsPerHour != nil {
		if *req.MaxAPICallsPerHour < 1 {
			return nil, fmt.Errorf("max_api_calls_per_hour must be at least 1")
		}
		quota.MaxAPICallsPerHour = *req.MaxAPICallsPerHour
	}

	if req.MaxConcurrentQueries != nil {
		if *req.MaxConcurrentQueries < 1 {
			return nil, fmt.Errorf("max_concurrent_queries must be at least 1")
		}
		quota.MaxConcurrentQueries = *req.MaxConcurrentQueries
	}

	if req.MaxTeamMembers != nil {
		if *req.MaxTeamMembers < 1 {
			return nil, fmt.Errorf("max_team_members must be at least 1")
		}
		quota.MaxTeamMembers = *req.MaxTeamMembers
	}

	if req.FeaturesEnabled != nil {
		quota.FeaturesEnabled = *req.FeaturesEnabled
	}

	// Save
	if err := s.store.UpdateQuota(ctx, quota); err != nil {
		// Try to create if update failed (quota doesn't exist yet)
		if err := s.store.CreateQuota(ctx, quota); err != nil {
			return nil, fmt.Errorf("save quota: %w", err)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"quota":           quota,
	}).Info("Quota updated")

	return quota, nil
}

// checkHourlyAPIQuota checks API quota for current hour
func (s *Service) checkHourlyAPIQuota(ctx context.Context, orgID string, maxPerHour int) error {
	now := time.Now()
	startOfHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	query := `
		SELECT COALESCE(SUM(api_calls_count), 0)
		FROM organization_usage_hourly
		WHERE organization_id = ? AND usage_hour = ?
	`

	var count int
	err := s.store.db.QueryRowContext(ctx, query, orgID, startOfHour.Unix()).Scan(&count)
	if err != nil {
		return fmt.Errorf("query hourly usage: %w", err)
	}

	if count >= maxPerHour {
		return &QuotaExceededError{
			ResourceType: ResourceAPI,
			Current:      count,
			Limit:        maxPerHour,
			Message:      fmt.Sprintf("Hourly API quota exceeded (%d/%d)", count, maxPerHour),
		}
	}

	return nil
}

// QuotaExceededError represents a quota exceeded error
type QuotaExceededError struct {
	ResourceType ResourceType
	Current      int
	Limit        int
	Message      string
}

func (e *QuotaExceededError) Error() string {
	return e.Message
}

// IsQuotaExceeded checks if error is a quota exceeded error
func IsQuotaExceeded(err error) bool {
	_, ok := err.(*QuotaExceededError)
	return ok
}
