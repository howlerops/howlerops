package rag

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type ColumnStatistics struct {
	DistinctCount int64
	NullCount     int64
	SampleValues  []string
	MinValue      interface{}
	MaxValue      interface{}
	AvgValue      interface{}
	TopValues     map[string]int // For categorical columns
}

type SchemaEnricher struct {
	db     *sql.DB
	logger *logrus.Logger
}

func NewSchemaEnricher(db *sql.DB, logger *logrus.Logger) *SchemaEnricher {
	return &SchemaEnricher{
		db:     db,
		logger: logger,
	}
}

// EnrichColumn adds statistics and samples to column metadata
func (se *SchemaEnricher) EnrichColumn(
	ctx context.Context,
	schema, table, column string,
	dataType string,
) (*ColumnStatistics, error) {
	stats := &ColumnStatistics{
		TopValues: make(map[string]int),
	}

	// Get distinct count
	distinctQuery := fmt.Sprintf(
		"SELECT COUNT(DISTINCT %s) FROM %s.%s",
		column, schema, table,
	)

	err := se.db.QueryRowContext(ctx, distinctQuery).Scan(&stats.DistinctCount)
	if err != nil {
		se.logger.WithError(err).Warnf("Failed to get distinct count for %s.%s.%s", schema, table, column)
		// Continue with other stats
	}

	// Get null count
	nullQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s.%s WHERE %s IS NULL",
		schema, table, column,
	)

	err = se.db.QueryRowContext(ctx, nullQuery).Scan(&stats.NullCount)
	if err != nil {
		se.logger.WithError(err).Warn("Failed to get null count")
	}

	// Type-specific enrichment
	if se.isCategoricalType(dataType, stats.DistinctCount) {
		stats.SampleValues, stats.TopValues = se.getCategoricalSamples(ctx, schema, table, column)
	} else if se.isNumericType(dataType) {
		stats.MinValue, stats.MaxValue, stats.AvgValue = se.getNumericStats(ctx, schema, table, column)
	} else {
		stats.SampleValues = se.getSampleValues(ctx, schema, table, column, 5)
	}

	return stats, nil
}

func (se *SchemaEnricher) isCategoricalType(dataType string, distinctCount int64) bool {
	dataType = strings.ToLower(dataType)

	// Type-based
	if strings.Contains(dataType, "enum") || strings.Contains(dataType, "bool") {
		return true
	}

	// Cardinality-based (low distinct count suggests categorical)
	return distinctCount > 0 && distinctCount < 50
}

func (se *SchemaEnricher) isNumericType(dataType string) bool {
	dataType = strings.ToLower(dataType)
	numericTypes := []string{"int", "integer", "bigint", "smallint", "decimal", "numeric", "float", "double", "real"}

	for _, numType := range numericTypes {
		if strings.Contains(dataType, numType) {
			return true
		}
	}

	return false
}

func (se *SchemaEnricher) getCategoricalSamples(
	ctx context.Context,
	schema, table, column string,
) ([]string, map[string]int) {
	// Get top 10 most common values
	query := fmt.Sprintf(`
        SELECT %s, COUNT(*) as cnt
        FROM %s.%s
        WHERE %s IS NOT NULL
        GROUP BY %s
        ORDER BY cnt DESC
        LIMIT 10
    `, column, schema, table, column, column)

	rows, err := se.db.QueryContext(ctx, query)
	if err != nil {
		se.logger.WithError(err).Warn("Failed to get categorical samples")
		return nil, nil
	}
	defer rows.Close()

	samples := []string{}
	topValues := make(map[string]int)

	for rows.Next() {
		var value string
		var count int

		if err := rows.Scan(&value, &count); err != nil {
			continue
		}

		samples = append(samples, value)
		topValues[value] = count
	}

	return samples, topValues
}

func (se *SchemaEnricher) getNumericStats(
	ctx context.Context,
	schema, table, column string,
) (interface{}, interface{}, interface{}) {
	query := fmt.Sprintf(`
        SELECT MIN(%s), MAX(%s), AVG(%s)
        FROM %s.%s
        WHERE %s IS NOT NULL
    `, column, column, column, schema, table, column)

	var min, max, avg interface{}
	err := se.db.QueryRowContext(ctx, query).Scan(&min, &max, &avg)
	if err != nil {
		se.logger.WithError(err).Warn("Failed to get numeric stats")
		return nil, nil, nil
	}

	return min, max, avg
}

func (se *SchemaEnricher) getSampleValues(
	ctx context.Context,
	schema, table, column string,
	limit int,
) []string {
	query := fmt.Sprintf(`
        SELECT DISTINCT %s
        FROM %s.%s
        WHERE %s IS NOT NULL
        LIMIT %d
    `, column, schema, table, column, limit)

	rows, err := se.db.QueryContext(ctx, query)
	if err != nil {
		se.logger.WithError(err).Warn("Failed to get sample values")
		return nil
	}
	defer rows.Close()

	samples := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			continue
		}
		samples = append(samples, value)
	}

	return samples
}
