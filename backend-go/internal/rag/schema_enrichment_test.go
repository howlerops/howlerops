package rag

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaEnricher_EnrichColumn_Categorical(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	enricher := NewSchemaEnricher(db, logger)

	schema := "public"
	table := "users"
	column := "status"
	dataType := "varchar"

	// Mock distinct count query
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT status\\) FROM public.users").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	// Mock null count query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM public.users WHERE status IS NULL").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Mock categorical samples query (top 10 values)
	mock.ExpectQuery("SELECT status, COUNT\\(\\*\\) as cnt").
		WillReturnRows(
			sqlmock.NewRows([]string{"status", "cnt"}).
				AddRow("active", 100).
				AddRow("inactive", 50).
				AddRow("pending", 10),
		)

	ctx := context.Background()
	stats, err := enricher.EnrichColumn(ctx, schema, table, column, dataType)

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(3), stats.DistinctCount)
	assert.Equal(t, int64(0), stats.NullCount)
	assert.Len(t, stats.SampleValues, 3)
	assert.Equal(t, []string{"active", "inactive", "pending"}, stats.SampleValues)
	assert.Equal(t, 100, stats.TopValues["active"])
	assert.Equal(t, 50, stats.TopValues["inactive"])
	assert.Equal(t, 10, stats.TopValues["pending"])

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchemaEnricher_EnrichColumn_Numeric(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	enricher := NewSchemaEnricher(db, logger)

	schema := "public"
	table := "products"
	column := "price"
	dataType := "decimal"

	// Mock distinct count query
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT price\\) FROM public.products").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(500))

	// Mock null count query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM public.products WHERE price IS NULL").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Mock numeric stats query
	mock.ExpectQuery("SELECT MIN\\(price\\), MAX\\(price\\), AVG\\(price\\)").
		WillReturnRows(
			sqlmock.NewRows([]string{"min", "max", "avg"}).
				AddRow(9.99, 999.99, 149.50),
		)

	ctx := context.Background()
	stats, err := enricher.EnrichColumn(ctx, schema, table, column, dataType)

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(500), stats.DistinctCount)
	assert.Equal(t, int64(5), stats.NullCount)
	assert.Equal(t, 9.99, stats.MinValue)
	assert.Equal(t, 999.99, stats.MaxValue)
	assert.Equal(t, 149.50, stats.AvgValue)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchemaEnricher_EnrichColumn_Text(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	enricher := NewSchemaEnricher(db, logger)

	schema := "public"
	table := "posts"
	column := "title"
	dataType := "text"

	// Mock distinct count query (high cardinality)
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT title\\) FROM public.posts").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1000))

	// Mock null count query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM public.posts WHERE title IS NULL").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Mock sample values query
	mock.ExpectQuery("SELECT DISTINCT title").
		WillReturnRows(
			sqlmock.NewRows([]string{"title"}).
				AddRow("Introduction to Go").
				AddRow("Testing Best Practices").
				AddRow("Database Design Patterns").
				AddRow("API Security").
				AddRow("Performance Optimization"),
		)

	ctx := context.Background()
	stats, err := enricher.EnrichColumn(ctx, schema, table, column, dataType)

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(1000), stats.DistinctCount)
	assert.Equal(t, int64(2), stats.NullCount)
	assert.Len(t, stats.SampleValues, 5)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchemaEnricher_IsCategoricalType(t *testing.T) {
	logger := logrus.New()
	enricher := NewSchemaEnricher(nil, logger)

	tests := []struct {
		name          string
		dataType      string
		distinctCount int64
		expected      bool
	}{
		{"enum type", "enum('a','b','c')", 100, true},
		{"boolean type", "boolean", 2, true},
		{"low cardinality", "varchar", 10, true},
		{"high cardinality", "varchar", 1000, false},
		{"zero cardinality", "varchar", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enricher.isCategoricalType(tt.dataType, tt.distinctCount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaEnricher_IsNumericType(t *testing.T) {
	logger := logrus.New()
	enricher := NewSchemaEnricher(nil, logger)

	tests := []struct {
		dataType string
		expected bool
	}{
		{"int", true},
		{"integer", true},
		{"bigint", true},
		{"smallint", true},
		{"decimal(10,2)", true},
		{"numeric", true},
		{"float", true},
		{"double", true},
		{"real", true},
		{"varchar", false},
		{"text", false},
		{"date", false},
	}

	for _, tt := range tests {
		t.Run(tt.dataType, func(t *testing.T) {
			result := enricher.isNumericType(tt.dataType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaEnricher_EnrichColumn_PartialFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	enricher := NewSchemaEnricher(db, logger)

	schema := "public"
	table := "products"
	column := "price"
	dataType := "decimal"

	// Mock distinct count query - succeeds
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT price\\) FROM public.products").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(500))

	// Mock null count query - succeeds
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM public.products WHERE price IS NULL").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Mock numeric stats query - fails
	mock.ExpectQuery("SELECT MIN\\(price\\), MAX\\(price\\), AVG\\(price\\)").
		WillReturnError(sql.ErrConnDone)

	ctx := context.Background()
	stats, err := enricher.EnrichColumn(ctx, schema, table, column, dataType)

	// Should still succeed with partial data
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(500), stats.DistinctCount) // Succeeded
	assert.Equal(t, int64(5), stats.NullCount)       // Succeeded
	assert.Nil(t, stats.MinValue)                     // Failed
	assert.Nil(t, stats.MaxValue)                     // Failed
	assert.Nil(t, stats.AvgValue)                     // Failed

	require.NoError(t, mock.ExpectationsWereMet())
}
