package rag

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type TrackedQueryPattern struct {
	Pattern      string
	Tables       []string
	Columns      []string
	WhereColumns []string
	JoinColumns  []string
	Frequency    int
	AvgDuration  float64
	LastUsed     time.Time
}

type QueryPatternTracker struct {
	vectorStore VectorStore
	logger      *logrus.Logger
}

func NewQueryPatternTracker(vectorStore VectorStore, logger *logrus.Logger) *QueryPatternTracker {
	return &QueryPatternTracker{
		vectorStore: vectorStore,
		logger:      logger,
	}
}

// TrackQuery records a successful query for pattern extraction
func (qpt *QueryPatternTracker) TrackQuery(
	ctx context.Context,
	sql string,
	duration float64,
	connectionID string,
) error {
	// Extract pattern
	pattern := qpt.normalizeToPattern(sql)

	// Extract metadata
	tables := qpt.extractTables(sql)
	whereColumns := qpt.extractWhereColumns(sql)
	joinColumns := qpt.extractJoinColumns(sql)

	// Create document
	content := fmt.Sprintf(
		"Query pattern: %s\nTables: %s\nFilters: %s",
		qpt.describePattern(sql),
		strings.Join(tables, ", "),
		strings.Join(whereColumns, ", "),
	)

	doc := &Document{
		ID:           fmt.Sprintf("pattern:%s:%s", connectionID, hashString(pattern)),
		ConnectionID: connectionID,
		Type:         DocumentTypeQuery,
		Content:      content,
		Metadata: map[string]interface{}{
			"pattern":       pattern,
			"tables":        tables,
			"where_columns": whereColumns,
			"join_columns":  joinColumns,
			"avg_duration":  duration,
			"frequency":     1,
			"last_used":     time.Now(),
		},
	}

	// Index pattern (will be embedded and searchable)
	return qpt.vectorStore.IndexDocument(ctx, doc)
}

func (qpt *QueryPatternTracker) normalizeToPattern(sql string) string {
	// Remove specific values, keep structure
	pattern := sql

	// Replace string literals
	pattern = regexp.MustCompile(`'[^']*'`).ReplaceAllString(pattern, "'?'")

	// Replace numbers
	pattern = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(pattern, "?")

	// Normalize whitespace
	pattern = regexp.MustCompile(`\s+`).ReplaceAllString(pattern, " ")

	return strings.TrimSpace(pattern)
}

func (qpt *QueryPatternTracker) extractTables(sql string) []string {
	// Simple regex-based extraction (could be improved with SQL parser)
	fromRegex := regexp.MustCompile(`(?i)FROM\s+([a-zA-Z0-9_]+)`)
	joinRegex := regexp.MustCompile(`(?i)JOIN\s+([a-zA-Z0-9_]+)`)

	tables := []string{}

	if matches := fromRegex.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				tables = append(tables, match[1])
			}
		}
	}

	if matches := joinRegex.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				tables = append(tables, match[1])
			}
		}
	}

	return qpt.unique(tables)
}

func (qpt *QueryPatternTracker) extractWhereColumns(sql string) []string {
	// Extract columns mentioned in WHERE clause
	whereRegex := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:GROUP|ORDER|LIMIT|$)`)

	columns := []string{}

	if matches := whereRegex.FindStringSubmatch(sql); len(matches) > 1 {
		whereClause := matches[1]

		// Extract column names (simple approach)
		colRegex := regexp.MustCompile(`([a-zA-Z0-9_]+)\s*[=<>]`)
		if colMatches := colRegex.FindAllStringSubmatch(whereClause, -1); len(colMatches) > 0 {
			for _, match := range colMatches {
				if len(match) > 1 {
					columns = append(columns, match[1])
				}
			}
		}
	}

	return qpt.unique(columns)
}

func (qpt *QueryPatternTracker) extractJoinColumns(sql string) []string {
	// Extract columns mentioned in JOIN ON clauses
	joinRegex := regexp.MustCompile(`(?i)ON\s+([a-zA-Z0-9_.]+)\s*=\s*([a-zA-Z0-9_.]+)`)

	columns := []string{}

	if matches := joinRegex.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 2 {
				columns = append(columns, match[1], match[2])
			}
		}
	}

	return qpt.unique(columns)
}

func (qpt *QueryPatternTracker) describePattern(sql string) string {
	// Generate human-readable description
	upper := strings.ToUpper(sql)

	if strings.Contains(upper, "INSERT") {
		return "Insert data"
	} else if strings.Contains(upper, "UPDATE") {
		return "Update records"
	} else if strings.Contains(upper, "DELETE") {
		return "Delete records"
	} else if strings.Contains(upper, "SELECT") {
		if strings.Contains(upper, "JOIN") {
			return "Query with joins"
		} else if strings.Contains(upper, "GROUP BY") {
			return "Aggregate query"
		} else if strings.Contains(upper, "WHERE") {
			return "Filtered query"
		}
		return "Simple select"
	}

	return "Unknown pattern"
}

func (qpt *QueryPatternTracker) unique(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// hashString creates a hash of a string for use in IDs
func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
