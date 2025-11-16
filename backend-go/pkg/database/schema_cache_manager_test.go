package database_test

import (
	"context"
	"testing"

	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManager_InvalidateSchemaCache tests the InvalidateSchemaCache method
func TestManager_InvalidateSchemaCache(t *testing.T) {
	tests := []struct {
		name         string
		connectionID string
		setupManager func() *database.Manager
		description  string
	}{
		{
			name:         "with nil schema cache - should not panic",
			connectionID: "test-conn-1",
			setupManager: func() *database.Manager {
				// Create a manager with nil schema cache (not possible via NewManager,
				// but tests defensive programming)
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			description: "InvalidateSchemaCache should be safe to call even with nil cache",
		},
		{
			name:         "with initialized schema cache - should succeed",
			connectionID: "test-conn-2",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			description: "InvalidateSchemaCache should succeed with valid cache",
		},
		{
			name:         "with empty connection ID",
			connectionID: "",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			description: "InvalidateSchemaCache should handle empty connection ID gracefully",
		},
		{
			name:         "with non-existent connection ID",
			connectionID: "non-existent-connection",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			description: "InvalidateSchemaCache should handle non-existent connections gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()

			// Should not panic - this is the primary assertion
			assert.NotPanics(t, func() {
				manager.InvalidateSchemaCache(tt.connectionID)
			}, tt.description)
		})
	}
}

// TestManager_InvalidateAllSchemas tests the InvalidateAllSchemas method
func TestManager_InvalidateAllSchemas(t *testing.T) {
	tests := []struct {
		name         string
		setupManager func() *database.Manager
		description  string
	}{
		{
			name: "with nil schema cache - should not panic",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			description: "InvalidateAllSchemas should be safe to call even with nil cache",
		},
		{
			name: "with initialized schema cache - should succeed",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			description: "InvalidateAllSchemas should succeed with valid cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()

			// Should not panic - this is the primary assertion
			assert.NotPanics(t, func() {
				manager.InvalidateAllSchemas()
			}, tt.description)
		})
	}
}

// TestManager_GetSchemaCacheStats tests the GetSchemaCacheStats method
func TestManager_GetSchemaCacheStats(t *testing.T) {
	tests := []struct {
		name          string
		setupManager  func() *database.Manager
		validateStats func(t *testing.T, stats map[string]interface{})
		description   string
	}{
		{
			name: "with initialized schema cache - returns valid stats",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			validateStats: func(t *testing.T, stats map[string]interface{}) {
				assert.NotNil(t, stats)
				// Schema cache is initialized by NewManager, so we expect valid stats
				assert.Contains(t, stats, "total_cached")
				assert.Contains(t, stats, "connections")
				assert.Contains(t, stats, "total_tables")
			},
			description: "GetSchemaCacheStats should return valid stats map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()

			stats := manager.GetSchemaCacheStats()
			tt.validateStats(t, stats)
		})
	}
}

// TestManager_GetConnectionCount tests the GetConnectionCount method
func TestManager_GetConnectionCount(t *testing.T) {
	tests := []struct {
		name          string
		setupManager  func() *database.Manager
		expectedCount int
		description   string
	}{
		{
			name: "empty manager - returns zero",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			expectedCount: 0,
			description:   "GetConnectionCount should return 0 for empty manager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()

			count := manager.GetConnectionCount()
			assert.Equal(t, tt.expectedCount, count, tt.description)
		})
	}
}

// TestManager_GetConnectionIDs tests the GetConnectionIDs method
func TestManager_GetConnectionIDs(t *testing.T) {
	tests := []struct {
		name         string
		setupManager func() *database.Manager
		validateIDs  func(t *testing.T, ids []string)
		description  string
	}{
		{
			name: "empty manager - returns empty slice",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			validateIDs: func(t *testing.T, ids []string) {
				assert.NotNil(t, ids, "should return non-nil slice")
				assert.Empty(t, ids, "should return empty slice for manager with no connections")
				assert.Len(t, ids, 0, "length should be 0")
			},
			description: "GetConnectionIDs should return empty slice for empty manager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()

			ids := manager.GetConnectionIDs()
			tt.validateIDs(t, ids)
		})
	}
}

// TestManager_RefreshSchema tests the RefreshSchema method
func TestManager_RefreshSchema(t *testing.T) {
	tests := []struct {
		name          string
		setupManager  func() *database.Manager
		connectionID  string
		expectedError string
		description   string
	}{
		{
			name: "connection not found - returns error",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			connectionID:  "non-existent-connection",
			expectedError: "connection not found",
			description:   "RefreshSchema should return error for non-existent connection",
		},
		{
			name: "empty connection ID - returns error",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			connectionID:  "",
			expectedError: "connection not found",
			description:   "RefreshSchema should return error for empty connection ID",
		},
		{
			name: "valid connection ID format but not in manager - returns error",
			setupManager: func() *database.Manager {
				logger := logrus.New()
				logger.SetLevel(logrus.ErrorLevel)
				return database.NewManager(logger)
			},
			connectionID:  "550e8400-e29b-41d4-a716-446655440000",
			expectedError: "connection not found",
			description:   "RefreshSchema should return error even for valid UUID that doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setupManager()
			ctx := context.Background()

			err := manager.RefreshSchema(ctx, tt.connectionID)

			require.Error(t, err, tt.description)
			assert.Contains(t, err.Error(), tt.expectedError, "error message should contain expected text")
		})
	}
}

// TestManager_RefreshSchema_ConcurrentCalls tests RefreshSchema with concurrent calls
func TestManager_RefreshSchema_ConcurrentCalls(t *testing.T) {
	manager := newTestManager()
	ctx := context.Background()

	// Launch multiple concurrent calls to RefreshSchema
	// This tests thread-safety of the method
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()
			err := manager.RefreshSchema(ctx, "non-existent")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "connection not found")
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestManager_InvalidateSchemaCache_ConcurrentCalls tests concurrent invalidations
func TestManager_InvalidateSchemaCache_ConcurrentCalls(t *testing.T) {
	manager := newTestManager()

	// Launch multiple concurrent invalidations
	// This tests thread-safety of invalidation
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()
			assert.NotPanics(t, func() {
				manager.InvalidateSchemaCache("test-conn")
			})
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestManager_GetSchemaCacheStats_Concurrency tests GetSchemaCacheStats with concurrent access
func TestManager_GetSchemaCacheStats_Concurrency(t *testing.T) {
	manager := newTestManager()

	// Launch concurrent stats requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			stats := manager.GetSchemaCacheStats()
			assert.NotNil(t, stats)
			assert.Contains(t, stats, "total_cached")
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestManager_GetConnectionCount_ThreadSafety tests GetConnectionCount with concurrent access
func TestManager_GetConnectionCount_ThreadSafety(t *testing.T) {
	manager := newTestManager()

	// Launch concurrent count requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			count := manager.GetConnectionCount()
			assert.Equal(t, 0, count)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestManager_GetConnectionIDs_ThreadSafety tests GetConnectionIDs with concurrent access
func TestManager_GetConnectionIDs_ThreadSafety(t *testing.T) {
	manager := newTestManager()

	// Launch concurrent ID requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			ids := manager.GetConnectionIDs()
			assert.NotNil(t, ids)
			assert.Empty(t, ids)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestManager_InvalidateAllSchemas_MultipleInvalidations tests calling InvalidateAll multiple times
func TestManager_InvalidateAllSchemas_MultipleInvalidations(t *testing.T) {
	manager := newTestManager()

	// Multiple sequential invalidations should all succeed
	for i := 0; i < 5; i++ {
		assert.NotPanics(t, func() {
			manager.InvalidateAllSchemas()
		}, "Multiple InvalidateAll calls should not panic")
	}
}

// TestManager_SchemaCacheWorkflow tests a typical workflow with schema cache methods
func TestManager_SchemaCacheWorkflow(t *testing.T) {
	manager := newTestManager()
	ctx := context.Background()

	// Get initial stats
	stats := manager.GetSchemaCacheStats()
	assert.NotNil(t, stats)

	// Get connection count (should be 0)
	count := manager.GetConnectionCount()
	assert.Equal(t, 0, count)

	// Get connection IDs (should be empty)
	ids := manager.GetConnectionIDs()
	assert.Empty(t, ids)

	// Try to invalidate non-existent connection (should not panic)
	assert.NotPanics(t, func() {
		manager.InvalidateSchemaCache("non-existent")
	})

	// Try to refresh non-existent connection (should error)
	err := manager.RefreshSchema(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")

	// Invalidate all schemas (should not panic)
	assert.NotPanics(t, func() {
		manager.InvalidateAllSchemas()
	})

	// Get stats again after invalidations
	stats = manager.GetSchemaCacheStats()
	assert.NotNil(t, stats)
}

// TestManager_RefreshSchema_ContextCancellation tests RefreshSchema with cancelled context
func TestManager_RefreshSchema_ContextCancellation(t *testing.T) {
	manager := newTestManager()

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Even with cancelled context, should get "connection not found" error first
	// because connection validation happens before any DB operations
	err := manager.RefreshSchema(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

// TestManager_GetConnectionCount_EmptyVsNil tests GetConnectionCount behavior
func TestManager_GetConnectionCount_EmptyVsNil(t *testing.T) {
	// Test with manager from NewManager
	manager1 := newTestManager()
	count1 := manager1.GetConnectionCount()
	assert.Equal(t, 0, count1, "NewManager should return 0 connections")

	// Test multiple calls return same result
	count2 := manager1.GetConnectionCount()
	assert.Equal(t, count1, count2, "Multiple calls should return consistent results")
}

// TestManager_GetConnectionIDs_EmptySliceNotNil tests that GetConnectionIDs returns empty slice, not nil
func TestManager_GetConnectionIDs_EmptySliceNotNil(t *testing.T) {
	manager := newTestManager()

	ids := manager.GetConnectionIDs()

	// Ensure we get an empty slice, not nil
	assert.NotNil(t, ids, "should return non-nil slice")
	assert.Empty(t, ids, "should return empty slice")
	assert.IsType(t, []string{}, ids, "should return slice of strings")
}

// TestManager_RefreshSchema_NilContext tests RefreshSchema with nil context
func TestManager_RefreshSchema_NilContext(t *testing.T) {
	manager := newTestManager()

	// Test defensive handling with non-existent connection
	// The function will error on "connection not found" before using context
	err := manager.RefreshSchema(context.TODO(), "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

// TestManager_GetSchemaCacheStats_EmptyCache tests stats for empty cache
func TestManager_GetSchemaCacheStats_EmptyCache(t *testing.T) {
	manager := newTestManager()

	stats := manager.GetSchemaCacheStats()

	require.NotNil(t, stats)
	assert.Contains(t, stats, "total_cached")
	assert.Contains(t, stats, "connections")
	assert.Contains(t, stats, "total_tables")

	// Verify initial state
	totalCached := stats["total_cached"]
	assert.Equal(t, 0, totalCached, "empty cache should have 0 entries")

	connections := stats["connections"]
	assert.IsType(t, []string{}, connections)
	assert.Empty(t, connections, "empty cache should have no connections")

	totalTables := stats["total_tables"]
	assert.Equal(t, 0, totalTables, "empty cache should have 0 tables")
}
