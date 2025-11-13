package quotas

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCheckQuota(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }() // Best-effort close in test

	store := NewStore(db)
	service := NewService(store, logrus.New())

	tests := []struct {
		name          string
		resourceType  ResourceType
		setupMock     func()
		expectError   bool
		expectQuotaEx bool
	}{
		{
			name:         "Query within quota",
			resourceType: ResourceQuery,
			setupMock: func() {
				// Mock GetQuota
				quotaRows := sqlmock.NewRows([]string{
					"organization_id", "max_connections", "max_queries_per_day",
					"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
					"max_team_members", "features_enabled", "created_at", "updated_at",
				}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_quotas").
					WithArgs("org-1").
					WillReturnRows(quotaRows)

				// Mock GetTodayUsage
				usageRows := sqlmock.NewRows([]string{
					"id", "organization_id", "usage_date", "connections_count",
					"queries_count", "storage_used_mb", "api_calls_count",
					"concurrent_queries_peak", "created_at", "updated_at",
				}).AddRow("usage-1", "org-1", time.Now().Unix(), 5, 500, 50.0, 100, 2, time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_usage").
					WithArgs("org-1", sqlmock.AnyArg()).
					WillReturnRows(usageRows)
			},
			expectError:   false,
			expectQuotaEx: false,
		},
		{
			name:         "Query quota exceeded",
			resourceType: ResourceQuery,
			setupMock: func() {
				// Mock GetQuota
				quotaRows := sqlmock.NewRows([]string{
					"organization_id", "max_connections", "max_queries_per_day",
					"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
					"max_team_members", "features_enabled", "created_at", "updated_at",
				}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_quotas").
					WithArgs("org-1").
					WillReturnRows(quotaRows)

				// Mock GetTodayUsage - at limit
				usageRows := sqlmock.NewRows([]string{
					"id", "organization_id", "usage_date", "connections_count",
					"queries_count", "storage_used_mb", "api_calls_count",
					"concurrent_queries_peak", "created_at", "updated_at",
				}).AddRow("usage-1", "org-1", time.Now().Unix(), 5, 1000, 50.0, 100, 2, time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_usage").
					WithArgs("org-1", sqlmock.AnyArg()).
					WillReturnRows(usageRows)
			},
			expectError:   true,
			expectQuotaEx: true,
		},
		{
			name:         "Connection within quota",
			resourceType: ResourceConnection,
			setupMock: func() {
				// Mock GetQuota
				quotaRows := sqlmock.NewRows([]string{
					"organization_id", "max_connections", "max_queries_per_day",
					"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
					"max_team_members", "features_enabled", "created_at", "updated_at",
				}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_quotas").
					WithArgs("org-1").
					WillReturnRows(quotaRows)

				// Mock GetTodayUsage
				usageRows := sqlmock.NewRows([]string{
					"id", "organization_id", "usage_date", "connections_count",
					"queries_count", "storage_used_mb", "api_calls_count",
					"concurrent_queries_peak", "created_at", "updated_at",
				}).AddRow("usage-1", "org-1", time.Now().Unix(), 5, 500, 50.0, 100, 2, time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_usage").
					WithArgs("org-1", sqlmock.AnyArg()).
					WillReturnRows(usageRows)
			},
			expectError:   false,
			expectQuotaEx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := service.CheckQuota(context.Background(), "org-1", tt.resourceType)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectQuotaEx {
					assert.True(t, IsQuotaExceeded(err))
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestIncrementUsage(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }() // Best-effort close in test

	store := NewStore(db)
	service := NewService(store, logrus.New())

	mock.ExpectExec("INSERT INTO organization_usage").
		WithArgs(
			sqlmock.AnyArg(), // id
			"org-1",          // organization_id
			sqlmock.AnyArg(), // usage_date
			1,                // amount
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			1,                // amount (for update)
			sqlmock.AnyArg(), // updated_at (for update)
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementUsage(context.Background(), "org-1", ResourceQuery)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateQuota(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }() // Best-effort close in test

	store := NewStore(db)
	service := NewService(store, logrus.New())

	// Mock GetQuota
	quotaRows := sqlmock.NewRows([]string{
		"organization_id", "max_connections", "max_queries_per_day",
		"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
		"max_team_members", "features_enabled", "created_at", "updated_at",
	}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

	mock.ExpectQuery("SELECT .* FROM organization_quotas").
		WithArgs("org-1").
		WillReturnRows(quotaRows)

	// Mock UpdateQuota
	mock.ExpectExec("UPDATE organization_quotas").
		WithArgs(
			20,               // max_connections
			sqlmock.AnyArg(), // max_queries_per_day
			sqlmock.AnyArg(), // max_storage_mb
			sqlmock.AnyArg(), // max_api_calls_per_hour
			sqlmock.AnyArg(), // max_concurrent_queries
			sqlmock.AnyArg(), // max_team_members
			sqlmock.AnyArg(), // features_enabled
			sqlmock.AnyArg(), // updated_at
			"org-1",          // WHERE organization_id
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	maxConn := 20
	req := &UpdateQuotaRequest{
		MaxConnections: &maxConn,
	}

	quota, err := service.UpdateQuota(context.Background(), "org-1", req)
	assert.NoError(t, err)
	assert.Equal(t, 20, quota.MaxConnections)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUsageStatistics(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }() // Best-effort close in test

	store := NewStore(db)
	service := NewService(store, logrus.New())

	// Mock GetQuota
	quotaRows := sqlmock.NewRows([]string{
		"organization_id", "max_connections", "max_queries_per_day",
		"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
		"max_team_members", "features_enabled", "created_at", "updated_at",
	}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

	mock.ExpectQuery("SELECT .* FROM organization_quotas").
		WithArgs("org-1").
		WillReturnRows(quotaRows)

	// Mock GetUsageHistory
	now := time.Now()
	usageRows := sqlmock.NewRows([]string{
		"id", "organization_id", "usage_date", "connections_count",
		"queries_count", "storage_used_mb", "api_calls_count",
		"concurrent_queries_peak", "created_at", "updated_at",
	}).
		AddRow("u1", "org-1", now.Unix(), 5, 100, 10.0, 50, 2, now.Unix(), now.Unix()).
		AddRow("u2", "org-1", now.AddDate(0, 0, -1).Unix(), 3, 80, 8.0, 40, 1, now.Unix(), now.Unix())

	mock.ExpectQuery("SELECT .* FROM organization_usage").
		WithArgs("org-1", sqlmock.AnyArg()).
		WillReturnRows(usageRows)

	stats, err := service.GetUsageStatistics(context.Background(), "org-1", 7)
	assert.NoError(t, err)
	assert.Equal(t, "org-1", stats.OrganizationID)
	assert.Equal(t, 180, stats.TotalQueries) // 100 + 80
	assert.Equal(t, 90, stats.TotalAPIcalls) // 50 + 40
	assert.Equal(t, 2, stats.PeakConcurrentQueries)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Benchmark quota checking
func BenchmarkCheckQuota(b *testing.B) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }() // Best-effort close in test

	store := NewStore(db)
	service := NewService(store, logrus.New())

	quotaRows := sqlmock.NewRows([]string{
		"organization_id", "max_connections", "max_queries_per_day",
		"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
		"max_team_members", "features_enabled", "created_at", "updated_at",
	}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

	usageRows := sqlmock.NewRows([]string{
		"id", "organization_id", "usage_date", "connections_count",
		"queries_count", "storage_used_mb", "api_calls_count",
		"concurrent_queries_peak", "created_at", "updated_at",
	}).AddRow("usage-1", "org-1", time.Now().Unix(), 5, 500, 50.0, 100, 2, time.Now().Unix(), time.Now().Unix())

	mock.ExpectQuery("SELECT .* FROM organization_quotas").WillReturnRows(quotaRows)
	mock.ExpectQuery("SELECT .* FROM organization_usage").WillReturnRows(usageRows)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.CheckQuota(context.Background(), "org-1", ResourceQuery) // Benchmark - error not relevant
	}
}
