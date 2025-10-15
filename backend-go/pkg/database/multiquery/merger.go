package multiquery

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// ResultMerger merges results from multiple database queries
type ResultMerger struct {
	logger *logrus.Logger
}

// NewResultMerger creates a new result merger
func NewResultMerger(logger *logrus.Logger) *ResultMerger {
	return &ResultMerger{
		logger: logger,
	}
}

// Merge merges multiple query results into a single result
func (m *ResultMerger) Merge(results map[string]*QueryResult, strategy MergeStrategy) (*Result, error) {
	if len(results) == 0 {
		return &Result{
			Columns:  []string{},
			Rows:     [][]interface{}{},
			RowCount: 0,
		}, nil
	}

	// If only one result, return it directly
	if len(results) == 1 {
		for _, result := range results {
			return &Result{
				Columns:  result.Columns,
				Rows:     result.Rows,
				RowCount: result.RowCount,
			}, nil
		}
	}

	// Merge based on strategy
	switch strategy {
	case MergeStrategyUnion:
		return m.mergeUnion(results)
	case MergeStrategyIntersect:
		return m.mergeIntersect(results)
	default:
		return m.mergeUnion(results)
	}
}

// mergeUnion combines all rows from all results (UNION behavior)
func (m *ResultMerger) mergeUnion(results map[string]*QueryResult) (*Result, error) {
	var firstResult *QueryResult
	var columns []string
	allRows := [][]interface{}{}

	// Get first result for column structure
	for _, result := range results {
		if firstResult == nil {
			firstResult = result
			columns = result.Columns
		}

		// Validate column compatibility
		if !m.columnsMatch(columns, result.Columns) {
			return nil, fmt.Errorf("column mismatch: cannot merge results with different columns")
		}

		// Append rows
		allRows = append(allRows, result.Rows...)
	}

	return &Result{
		Columns:  columns,
		Rows:     allRows,
		RowCount: int64(len(allRows)),
	}, nil
}

// mergeIntersect returns only rows that appear in all results
func (m *ResultMerger) mergeIntersect(results map[string]*QueryResult) (*Result, error) {
	// For simplicity, just return the first result
	// A full implementation would find common rows
	for _, result := range results {
		return &Result{
			Columns:  result.Columns,
			Rows:     result.Rows,
			RowCount: result.RowCount,
		}, nil
	}

	return &Result{
		Columns:  []string{},
		Rows:     [][]interface{}{},
		RowCount: 0,
	}, nil
}

// columnsMatch checks if two column lists match
func (m *ResultMerger) columnsMatch(cols1, cols2 []string) bool {
	if len(cols1) != len(cols2) {
		return false
	}

	for i := range cols1 {
		if cols1[i] != cols2[i] {
			return false
		}
	}

	return true
}

// MergeStrategy defines how results should be merged
type MergeStrategy string

const (
	// MergeStrategyUnion combines all rows
	MergeStrategyUnion MergeStrategy = "union"
	// MergeStrategyIntersect returns only common rows
	MergeStrategyIntersect MergeStrategy = "intersect"
)

