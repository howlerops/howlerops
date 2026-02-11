package database

import (
	"strings"
	"sync"
	"time"
)

type tableStructureCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]*tableStructureCacheEntry
}

type tableStructureCacheEntry struct {
	structure *TableStructure
	expiresAt time.Time
}

func newTableStructureCache(ttl time.Duration) *tableStructureCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	return &tableStructureCache{
		ttl:     ttl,
		entries: make(map[string]*tableStructureCacheEntry),
	}
}

func (c *tableStructureCache) get(schema, table string) (*TableStructure, bool) {
	if c == nil {
		return nil, false
	}

	key := cacheKey(schema, table)

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		// Double-check expiration under write lock
		if current, exists := c.entries[key]; exists && time.Now().After(current.expiresAt) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, false
	}

	return cloneTableStructure(entry.structure), true
}

func (c *tableStructureCache) set(schema, table string, structure *TableStructure) {
	if c == nil || structure == nil {
		return
	}

	key := cacheKey(schema, table)
	entry := &tableStructureCacheEntry{
		structure: cloneTableStructure(structure),
		expiresAt: time.Now().Add(c.ttl),
	}

	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
}

func (c *tableStructureCache) invalidate(schema, table string) {
	if c == nil {
		return
	}

	key := cacheKey(schema, table)

	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

func (c *tableStructureCache) clear() {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.entries = make(map[string]*tableStructureCacheEntry)
	c.mu.Unlock()
}

func cacheKey(schema, table string) string {
	return strings.ToLower(strings.TrimSpace(schema) + "." + strings.TrimSpace(table))
}

func cloneTableStructure(structure *TableStructure) *TableStructure {
	if structure == nil {
		return nil
	}

	clone := &TableStructure{
		Table:      structure.Table,
		Statistics: make(map[string]string, len(structure.Statistics)),
	}

	if len(structure.Columns) > 0 {
		clone.Columns = make([]ColumnInfo, len(structure.Columns))
		copy(clone.Columns, structure.Columns)
	}

	if len(structure.Indexes) > 0 {
		clone.Indexes = make([]IndexInfo, len(structure.Indexes))
		copy(clone.Indexes, structure.Indexes)
	}

	if len(structure.ForeignKeys) > 0 {
		clone.ForeignKeys = make([]ForeignKeyInfo, len(structure.ForeignKeys))
		copy(clone.ForeignKeys, structure.ForeignKeys)
	}

	if len(structure.Triggers) > 0 {
		clone.Triggers = append([]string(nil), structure.Triggers...)
	}

	for key, value := range structure.Statistics {
		clone.Statistics[key] = value
	}

	return clone
}
