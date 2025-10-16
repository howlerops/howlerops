package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// SchemaCache provides intelligent caching of database schemas with change detection
type SchemaCache struct {
	mu          sync.RWMutex
	cache       map[string]*CachedSchema
	logger      *logrus.Logger
	defaultTTL  time.Duration
	maxCacheAge time.Duration
}

// CachedSchema represents a cached schema with metadata
type CachedSchema struct {
	ConnectionID    string
	Schemas         []string
	Tables          map[string][]TableInfo // schema -> tables
	Columns         map[string][]ColumnInfo // schema.table -> columns
	Hash            string    // Hash of schema structure
	MigrationHash   string    // Hash of migration state
	CachedAt        time.Time
	ExpiresAt       time.Time
	LastCheckedAt   time.Time
	ChangeDetectedAt *time.Time
}

// SchemaFingerprint represents the current state of a database schema
type SchemaFingerprint struct {
	MigrationState string   // From migrations table
	TableList      []string // Sorted list of tables
	Hash           string   // Combined hash
}

// NewSchemaCache creates a new schema cache
func NewSchemaCache(logger *logrus.Logger) *SchemaCache {
	return &SchemaCache{
		cache:       make(map[string]*CachedSchema),
		logger:      logger,
		defaultTTL:  1 * time.Hour,      // Cache for 1 hour by default
		maxCacheAge: 24 * time.Hour,     // Maximum cache age
	}
}

// GetCachedSchema retrieves cached schema if valid
func (sc *SchemaCache) GetCachedSchema(ctx context.Context, connectionID string, db Database) (*CachedSchema, error) {
	sc.mu.RLock()
	cached, exists := sc.cache[connectionID]
	sc.mu.RUnlock()

	if !exists {
		sc.logger.WithField("connection", connectionID).Debug("Schema cache miss")
		return nil, nil
	}

	// Check if cache is expired
	if time.Now().After(cached.ExpiresAt) {
		sc.logger.WithField("connection", connectionID).Debug("Schema cache expired")
		sc.InvalidateCache(connectionID)
		return nil, nil
	}

	// Quick check - if cache is fresh (< 5 minutes), return immediately
	if time.Since(cached.LastCheckedAt) < 5*time.Minute {
		sc.logger.WithField("connection", connectionID).Debug("Schema cache hit (fresh)")
		return cached, nil
	}

	// Otherwise, do a lightweight change detection
	hasChanged, err := sc.detectSchemaChange(ctx, connectionID, cached, db)
	if err != nil {
		sc.logger.WithError(err).Warn("Failed to detect schema changes, using cached")
		// Update last checked time even on error
		sc.mu.Lock()
		cached.LastCheckedAt = time.Now()
		sc.mu.Unlock()
		return cached, nil
	}

	if hasChanged {
		sc.logger.WithField("connection", connectionID).Info("Schema change detected, invalidating cache")
		sc.InvalidateCache(connectionID)
		return nil, nil
	}

	// Update last checked time
	sc.mu.Lock()
	cached.LastCheckedAt = time.Now()
	sc.mu.Unlock()

	sc.logger.WithField("connection", connectionID).Debug("Schema cache hit (verified)")
	return cached, nil
}

// CacheSchema stores schema information with metadata
func (sc *SchemaCache) CacheSchema(ctx context.Context, connectionID string, db Database, schemas []string, tables map[string][]TableInfo) error {
	// Generate fingerprint
	fingerprint, err := sc.generateFingerprint(ctx, db, schemas, tables)
	if err != nil {
		return fmt.Errorf("failed to generate fingerprint: %w", err)
	}

	now := time.Now()
	cached := &CachedSchema{
		ConnectionID:  connectionID,
		Schemas:       schemas,
		Tables:        tables,
		Hash:          fingerprint.Hash,
		MigrationHash: fingerprint.MigrationState,
		CachedAt:      now,
		ExpiresAt:     now.Add(sc.defaultTTL),
		LastCheckedAt: now,
	}

	sc.mu.Lock()
	sc.cache[connectionID] = cached
	sc.mu.Unlock()

	sc.logger.WithFields(logrus.Fields{
		"connection":   connectionID,
		"schemas":      len(schemas),
		"tables_total": sc.countTables(tables),
		"hash":         fingerprint.Hash[:8],
	}).Info("Schema cached")

	return nil
}

// InvalidateCache removes cached schema for a connection
func (sc *SchemaCache) InvalidateCache(connectionID string) {
	sc.mu.Lock()
	delete(sc.cache, connectionID)
	sc.mu.Unlock()
	
	sc.logger.WithField("connection", connectionID).Debug("Schema cache invalidated")
}

// InvalidateAll clears all cached schemas
func (sc *SchemaCache) InvalidateAll() {
	sc.mu.Lock()
	sc.cache = make(map[string]*CachedSchema)
	sc.mu.Unlock()
	
	sc.logger.Info("All schema caches invalidated")
}

// detectSchemaChange performs lightweight change detection
func (sc *SchemaCache) detectSchemaChange(ctx context.Context, connectionID string, cached *CachedSchema, db Database) (bool, error) {
	// Get current fingerprint (lightweight operation)
	fingerprint, err := sc.generateLightweightFingerprint(ctx, db)
	if err != nil {
		return false, err
	}

	// Compare hashes
	if fingerprint.Hash != cached.Hash {
		sc.logger.WithFields(logrus.Fields{
			"connection": connectionID,
			"old_hash":   cached.Hash[:8],
			"new_hash":   fingerprint.Hash[:8],
		}).Debug("Schema hash mismatch detected")
		return true, nil
	}

	// Check migration state separately for more granular detection
	if fingerprint.MigrationState != cached.MigrationHash {
		sc.logger.WithFields(logrus.Fields{
			"connection": connectionID,
			"old_migration": cached.MigrationHash[:8],
			"new_migration": fingerprint.MigrationState[:8],
		}).Debug("Migration state change detected")
		return true, nil
	}

	return false, nil
}

// generateFingerprint creates a full fingerprint of the database schema
func (sc *SchemaCache) generateFingerprint(ctx context.Context, db Database, schemas []string, tables map[string][]TableInfo) (*SchemaFingerprint, error) {
	// Get migration state
	migrationHash, err := sc.getMigrationStateHash(ctx, db)
	if err != nil {
		// If migrations table doesn't exist or fails, use empty hash
		migrationHash = ""
	}

	// Generate table list hash
	tableList := sc.extractTableList(tables)
	tableListHash := sc.hashStringList(tableList)

	// Combine hashes
	combinedHash := sc.combineHashes(migrationHash, tableListHash)

	return &SchemaFingerprint{
		MigrationState: migrationHash,
		TableList:      tableList,
		Hash:           combinedHash,
	}, nil
}

// generateLightweightFingerprint creates a quick fingerprint without full schema scan
func (sc *SchemaCache) generateLightweightFingerprint(ctx context.Context, db Database) (*SchemaFingerprint, error) {
	// Get migration state (fast query)
	migrationHash, err := sc.getMigrationStateHash(ctx, db)
	if err != nil {
		migrationHash = ""
	}

	// Get table count and names from information_schema (faster than full scan)
	schemas, err := db.GetSchemas(ctx)
	if err != nil {
		return nil, err
	}

	var tableList []string
	for _, schema := range schemas {
		tables, err := db.GetTables(ctx, schema)
		if err != nil {
			continue
		}
		for _, table := range tables {
			tableList = append(tableList, fmt.Sprintf("%s.%s", table.Schema, table.Name))
		}
	}

	sort.Strings(tableList)
	tableListHash := sc.hashStringList(tableList)
	combinedHash := sc.combineHashes(migrationHash, tableListHash)

	return &SchemaFingerprint{
		MigrationState: migrationHash,
		TableList:      tableList,
		Hash:           combinedHash,
	}, nil
}

// getMigrationStateHash gets a hash of the current migration state
func (sc *SchemaCache) getMigrationStateHash(ctx context.Context, db Database) (string, error) {
	// Try common migration table patterns
	migrationTables := []string{
		"schema_migrations",
		"migrations", 
		"flyway_schema_history",
		"_prisma_migrations",
		"django_migrations",
		"alembic_version",
	}

	for _, tableName := range migrationTables {
		// Try to query migration table
		query := fmt.Sprintf("SELECT version FROM %s ORDER BY version", tableName)
		result, err := db.Execute(ctx, query)
		if err != nil {
			// Table doesn't exist or query failed, try next
			continue
		}

		// Hash the migration versions
		var versions []string
		for _, row := range result.Rows {
			if len(row) > 0 {
				versions = append(versions, fmt.Sprintf("%v", row[0]))
			}
		}

		if len(versions) > 0 {
			return sc.hashStringList(versions), nil
		}
	}

	// No migration table found, return empty hash
	return "", nil
}

// extractTableList extracts a sorted list of all tables
func (sc *SchemaCache) extractTableList(tables map[string][]TableInfo) []string {
	var tableList []string
	for schema, tableInfos := range tables {
		for _, table := range tableInfos {
			tableList = append(tableList, fmt.Sprintf("%s.%s", schema, table.Name))
		}
	}
	sort.Strings(tableList)
	return tableList
}

// hashStringList creates a hash of a list of strings
func (sc *SchemaCache) hashStringList(list []string) string {
	h := sha256.New()
	for _, item := range list {
		h.Write([]byte(item))
		h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// combineHashes combines multiple hashes into one
func (sc *SchemaCache) combineHashes(hashes ...string) string {
	h := sha256.New()
	for _, hash := range hashes {
		h.Write([]byte(hash))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// countTables counts total tables across all schemas
func (sc *SchemaCache) countTables(tables map[string][]TableInfo) int {
	count := 0
	for _, tableList := range tables {
		count += len(tableList)
	}
	return count
}

// GetCacheStats returns statistics about the cache
func (sc *SchemaCache) GetCacheStats() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats := map[string]interface{}{
		"total_cached":     len(sc.cache),
		"connections":      []string{},
		"oldest_cache":     "",
		"newest_cache":     "",
		"total_tables":     0,
	}

	var oldest, newest time.Time
	connectionsList := []string{}
	totalTables := 0

	for connID, cached := range sc.cache {
		connectionsList = append(connectionsList, connID)
		totalTables += sc.countTables(cached.Tables)

		if oldest.IsZero() || cached.CachedAt.Before(oldest) {
			oldest = cached.CachedAt
		}
		if newest.IsZero() || cached.CachedAt.After(newest) {
			newest = cached.CachedAt
		}
	}

	stats["connections"] = connectionsList
	stats["total_tables"] = totalTables
	if !oldest.IsZero() {
		stats["oldest_cache"] = oldest.Format(time.RFC3339)
	}
	if !newest.IsZero() {
		stats["newest_cache"] = newest.Format(time.RFC3339)
	}

	return stats
}

