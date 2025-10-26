package database

import (
	"context"
	"fmt"
)

// Schema cache management methods for Manager

// InvalidateSchemaCache invalidates the cached schema for a specific connection
func (m *Manager) InvalidateSchemaCache(connectionID string) {
	if m.schemaCache != nil {
		m.schemaCache.InvalidateCache(connectionID)
	}
}

// InvalidateAllSchemas invalidates all cached schemas
func (m *Manager) InvalidateAllSchemas() {
	if m.schemaCache != nil {
		m.schemaCache.InvalidateAll()
	}
}

// GetSchemaCacheStats returns statistics about the schema cache
func (m *Manager) GetSchemaCacheStats() map[string]interface{} {
	if m.schemaCache != nil {
		return m.schemaCache.GetCacheStats()
	}
	return map[string]interface{}{
		"error": "schema cache not initialized",
	}
}

// GetConnectionCount returns the number of active connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// GetConnectionIDs returns a list of all connection IDs
func (m *Manager) GetConnectionIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.connections))
	for id := range m.connections {
		ids = append(ids, id)
	}
	return ids
}

// RefreshSchema forces a refresh of the schema for a connection
func (m *Manager) RefreshSchema(ctx context.Context, connectionID string) error {
	// Invalidate cache
	m.InvalidateSchemaCache(connectionID)

	// Fetch fresh schema
	m.mu.RLock()
	db, exists := m.connections[connectionID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}

	// Get schemas
	schemas, err := db.GetSchemas(ctx)
	if err != nil {
		return err
	}

	// Get tables
	tablesMap := make(map[string][]TableInfo)
	for _, schema := range schemas {
		tables, err := db.GetTables(ctx, schema)
		if err != nil {
			continue
		}
		tablesMap[schema] = tables
	}

	// Cache the fresh schema
	if m.schemaCache != nil {
		return m.schemaCache.CacheSchema(ctx, connectionID, db, schemas, tablesMap)
	}

	return nil
}
