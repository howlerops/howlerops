package catalog

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*sql.DB, Store) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	store := NewStore(db)
	err = store.Initialize(context.Background())
	require.NoError(t, err)

	return db, store
}

func TestCreateAndGetTableEntry(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	entry := &TableCatalogEntry{
		ConnectionID: "conn-123",
		SchemaName:   "public",
		TableName:    "customers",
		Description:  "Customer data table",
		Tags:         []string{TagPII, TagSensitive},
		CreatedBy:    "user-1",
	}

	// Create entry
	err := store.CreateTableEntry(ctx, entry)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.ID)

	// Get entry
	retrieved, err := store.GetTableEntry(ctx, "conn-123", "public", "customers")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, entry.ID, retrieved.ID)
	assert.Equal(t, "Customer data table", retrieved.Description)
	assert.Equal(t, 2, len(retrieved.Tags))
}

func TestUpdateTableEntry(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	entry := &TableCatalogEntry{
		ConnectionID: "conn-123",
		SchemaName:   "public",
		TableName:    "orders",
		Description:  "Order records",
		Tags:         []string{TagInternal},
		CreatedBy:    "user-1",
	}

	// Create entry
	err := store.CreateTableEntry(ctx, entry)
	require.NoError(t, err)

	// Wait to ensure different timestamp
	time.Sleep(100 * time.Millisecond)

	// Update entry
	entry.Description = "Updated order records"
	entry.Tags = []string{TagInternal, TagSensitive}

	err = store.UpdateTableEntry(ctx, entry)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.GetTableEntry(ctx, "conn-123", "public", "orders")
	require.NoError(t, err)
	assert.Equal(t, "Updated order records", retrieved.Description)
	assert.Equal(t, 2, len(retrieved.Tags))
	assert.True(t, retrieved.UpdatedAt.Unix() >= entry.CreatedAt.Unix())
}

func TestDeleteTableEntry(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	entry := &TableCatalogEntry{
		ConnectionID: "conn-123",
		SchemaName:   "public",
		TableName:    "temp_table",
		CreatedBy:    "user-1",
	}

	err := store.CreateTableEntry(ctx, entry)
	require.NoError(t, err)

	// Delete entry
	err = store.DeleteTableEntry(ctx, entry.ID)
	require.NoError(t, err)

	// Verify deletion
	retrieved, err := store.GetTableEntry(ctx, "conn-123", "public", "temp_table")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestListTableEntries(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create multiple entries
	entries := []*TableCatalogEntry{
		{
			ConnectionID: "conn-123",
			SchemaName:   "public",
			TableName:    "table1",
			CreatedBy:    "user-1",
		},
		{
			ConnectionID: "conn-123",
			SchemaName:   "public",
			TableName:    "table2",
			CreatedBy:    "user-1",
		},
		{
			ConnectionID: "conn-456",
			SchemaName:   "public",
			TableName:    "table3",
			CreatedBy:    "user-1",
		},
	}

	for _, entry := range entries {
		err := store.CreateTableEntry(ctx, entry)
		require.NoError(t, err)
	}

	// List entries for conn-123
	list, err := store.ListTableEntries(ctx, "conn-123")
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
}

func TestCreateAndGetColumnEntry(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table entry first
	tableEntry := &TableCatalogEntry{
		ConnectionID: "conn-123",
		SchemaName:   "public",
		TableName:    "customers",
		CreatedBy:    "user-1",
	}
	err := store.CreateTableEntry(ctx, tableEntry)
	require.NoError(t, err)

	// Create column entry
	piiType := "email"
	confidence := 0.95
	columnEntry := &ColumnCatalogEntry{
		TableCatalogID: tableEntry.ID,
		ColumnName:     "email_address",
		Description:    "Customer email",
		Tags:           []string{TagPII},
		PIIType:        &piiType,
		PIIConfidence:  &confidence,
	}

	err = store.CreateColumnEntry(ctx, columnEntry)
	require.NoError(t, err)
	assert.NotEmpty(t, columnEntry.ID)

	// Get column entry
	retrieved, err := store.GetColumnEntry(ctx, tableEntry.ID, "email_address")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Customer email", retrieved.Description)
	assert.Equal(t, "email", *retrieved.PIIType)
	assert.Equal(t, 0.95, *retrieved.PIIConfidence)
}

func TestListColumnEntries(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table entry
	tableEntry := &TableCatalogEntry{
		ConnectionID: "conn-123",
		SchemaName:   "public",
		TableName:    "users",
		CreatedBy:    "user-1",
	}
	err := store.CreateTableEntry(ctx, tableEntry)
	require.NoError(t, err)

	// Create multiple column entries
	columns := []*ColumnCatalogEntry{
		{
			TableCatalogID: tableEntry.ID,
			ColumnName:     "id",
			Description:    "User ID",
		},
		{
			TableCatalogID: tableEntry.ID,
			ColumnName:     "username",
			Description:    "Username",
		},
		{
			TableCatalogID: tableEntry.ID,
			ColumnName:     "email",
			Description:    "Email address",
			Tags:           []string{TagPII},
		},
	}

	for _, col := range columns {
		err := store.CreateColumnEntry(ctx, col)
		require.NoError(t, err)
	}

	// List columns
	list, err := store.ListColumnEntries(ctx, tableEntry.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(list))
}

func TestCreateAndListTags(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// System tags should already exist from Initialize
	tags, err := store.ListTags(ctx, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tags), 5) // At least 5 system tags

	// Create custom tag
	orgID := "org-123"
	customTag := &CatalogTag{
		Name:           "CustomTag",
		Color:          "#ff5733",
		Description:    "Custom organization tag",
		OrganizationID: &orgID,
	}

	err = store.CreateTag(ctx, customTag)
	require.NoError(t, err)

	// List tags for organization
	tags, err = store.ListTags(ctx, &orgID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tags), 6) // System tags + custom tag

	// Verify system tags are included
	hasSystemTag := false
	hasCustomTag := false
	for _, tag := range tags {
		if tag.Name == TagPII {
			hasSystemTag = true
		}
		if tag.Name == "CustomTag" {
			hasCustomTag = true
		}
	}
	assert.True(t, hasSystemTag)
	assert.True(t, hasCustomTag)
}

func TestDeleteCustomTag(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	orgID := "org-123"
	customTag := &CatalogTag{
		Name:           "TempTag",
		Color:          "#000000",
		OrganizationID: &orgID,
	}

	err := store.CreateTag(ctx, customTag)
	require.NoError(t, err)

	// Delete custom tag
	err = store.DeleteTag(ctx, customTag.ID)
	require.NoError(t, err)

	// Cannot delete system tag
	err = store.DeleteTag(ctx, "tag-pii")
	assert.Error(t, err)
}

func TestSearchCatalog(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test data
	tableEntry := &TableCatalogEntry{
		ConnectionID: "conn-123",
		SchemaName:   "public",
		TableName:    "customers",
		Description:  "Customer information including contact details",
		Tags:         []string{TagPII, TagSensitive},
		CreatedBy:    "user-1",
	}
	err := store.CreateTableEntry(ctx, tableEntry)
	require.NoError(t, err)

	piiType := "email"
	columnEntry := &ColumnCatalogEntry{
		TableCatalogID: tableEntry.ID,
		ColumnName:     "email_address",
		Description:    "Primary email for customer communication",
		Tags:           []string{TagPII},
		PIIType:        &piiType,
	}
	err = store.CreateColumnEntry(ctx, columnEntry)
	require.NoError(t, err)

	// Search for "customer"
	results, err := store.SearchCatalog(ctx, "customer", &SearchFilters{Limit: 10})
	require.NoError(t, err)
	assert.Greater(t, len(results.Results), 0)

	// Search for "email"
	results, err = store.SearchCatalog(ctx, "email", &SearchFilters{Limit: 10})
	require.NoError(t, err)
	assert.Greater(t, len(results.Results), 0)

	// Search with connection filter
	connID := "conn-123"
	results, err = store.SearchCatalog(ctx, "customer", &SearchFilters{
		ConnectionID: &connID,
		Limit:        10,
	})
	require.NoError(t, err)
	assert.Greater(t, len(results.Results), 0)

	// Search with tag filter
	results, err = store.SearchCatalog(ctx, "customer", &SearchFilters{
		Tags:  []string{TagPII},
		Limit: 10,
	})
	require.NoError(t, err)
	assert.Greater(t, len(results.Results), 0)
}
