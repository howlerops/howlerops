package rag

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// Helper types and components for RAG functionality

// SchemaAnalyzer analyzes database schemas
type SchemaAnalyzer struct {
	logger *logrus.Logger
}

// NewSchemaAnalyzer creates a new schema analyzer
func NewSchemaAnalyzer(logger *logrus.Logger) *SchemaAnalyzer {
	return &SchemaAnalyzer{logger: logger}
}

// PatternMatcher matches query patterns
type PatternMatcher struct {
	logger *logrus.Logger
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(logger *logrus.Logger) *PatternMatcher {
	return &PatternMatcher{logger: logger}
}

// ExtractPatterns extracts patterns from documents
func (pm *PatternMatcher) ExtractPatterns(docs []*Document) []QueryPattern {
	// TODO: Implement pattern extraction
	return []QueryPattern{}
}

// StatsCollector collects statistics
type StatsCollector struct {
	logger *logrus.Logger
}

// NewStatsCollector creates a new stats collector
func NewStatsCollector(logger *logrus.Logger) *StatsCollector {
	return &StatsCollector{logger: logger}
}

// SQLValidator validates SQL queries
type SQLValidator struct {
	logger *logrus.Logger
}

// NewSQLValidator creates a new SQL validator
func NewSQLValidator(logger *logrus.Logger) *SQLValidator {
	return &SQLValidator{logger: logger}
}

// Validate validates a SQL query
func (v *SQLValidator) Validate(query string) error {
	// TODO: Implement SQL validation
	return nil
}

// QueryPlanner plans complex queries
type QueryPlanner struct {
	logger *logrus.Logger
}

// NewQueryPlanner creates a new query planner
func NewQueryPlanner(logger *logrus.Logger) *QueryPlanner {
	return &QueryPlanner{logger: logger}
}

// QueryStep represents a step in query execution
type QueryStep struct {
	Order       int
	Description string
	Complexity  string
}

// StepSQL represents SQL for a query step
type StepSQL struct {
	Step        QueryStep
	SQL         string
	Explanation string
}

// PlannedSQL represents a planned SQL query
type PlannedSQL struct {
	Query       string
	Explanation string
}

// DecomposeRequest decomposes a complex request into steps
func (qp *QueryPlanner) DecomposeRequest(prompt string, context *QueryContext) []QueryStep {
	// TODO: Implement request decomposition
	return []QueryStep{}
}

// CombineSteps combines steps into a final query
func (qp *QueryPlanner) CombineSteps(steps []StepSQL) PlannedSQL {
	// TODO: Implement step combination
	return PlannedSQL{}
}

// ValidateAndOptimize validates and optimizes a planned query
func (qp *QueryPlanner) ValidateAndOptimize(planned PlannedSQL) PlannedSQL {
	// TODO: Implement validation and optimization
	return planned
}

// JoinDetector detects and suggests table joins using FK relationships
type JoinDetector struct {
	logger *logrus.Logger
}

// NewJoinDetector creates a new join detector
func NewJoinDetector(logger *logrus.Logger) *JoinDetector {
	return &JoinDetector{logger: logger}
}

// JoinCondition represents a join condition
type JoinCondition struct {
	LeftTable   string
	RightTable  string
	LeftColumn  string
	RightColumn string
	JoinType    string // INNER, LEFT, RIGHT, FULL
}

// JoinPath represents a path of joins between tables
type JoinPath struct {
	Tables []string
	Joins  []JoinCondition
	Cost   int // Lower is better (fewer joins)
}

// tableEdge represents an edge in the table relationship graph
type tableEdge struct {
	targetTable   string
	localColumn   string
	foreignColumn string
	joinType      string
}

// DetectTables detects tables mentioned in a query using schema context
func (jd *JoinDetector) DetectTables(query string, context *QueryContext) []string {
	if context == nil {
		return []string{}
	}

	queryLower := strings.ToLower(query)
	detectedTables := make([]string, 0)
	seen := make(map[string]bool)

	// Check each schema's table name against the query
	for _, schema := range context.RelevantSchemas {
		tableName := strings.ToLower(schema.TableName)
		if tableName == "" {
			continue
		}

		// Check if table name appears in query (with word boundaries)
		if strings.Contains(queryLower, tableName) && !seen[schema.TableName] {
			detectedTables = append(detectedTables, schema.TableName)
			seen[schema.TableName] = true
		}

		// Also check for schema.table format
		for _, col := range schema.Columns {
			if col.IsForeignKey && strings.Contains(queryLower, strings.ToLower(col.Name)) {
				// The FK might reference another table
				if !seen[schema.TableName] {
					detectedTables = append(detectedTables, schema.TableName)
					seen[schema.TableName] = true
				}
			}
		}
	}

	jd.logger.WithFields(logrus.Fields{
		"query":           query[:min(50, len(query))],
		"detected_tables": detectedTables,
	}).Debug("Tables detected from query")

	return detectedTables
}

// FindJoinPath finds the optimal join path between tables using BFS
func (jd *JoinDetector) FindJoinPath(tables []string, schemas []SchemaContext) JoinPath {
	if len(tables) < 2 {
		return JoinPath{Tables: tables, Joins: []JoinCondition{}}
	}

	// Build adjacency graph from FK relationships
	graph := jd.buildRelationshipGraph(schemas)

	// Use BFS to find shortest path connecting all tables
	path := jd.bfsJoinPath(tables, graph, schemas)

	jd.logger.WithFields(logrus.Fields{
		"input_tables":  tables,
		"path_tables":   path.Tables,
		"joins_count":   len(path.Joins),
		"path_cost":     path.Cost,
	}).Debug("Join path computed")

	return path
}

// buildRelationshipGraph builds an adjacency list from FK relationships
func (jd *JoinDetector) buildRelationshipGraph(schemas []SchemaContext) map[string][]tableEdge {
	graph := make(map[string][]tableEdge)

	for _, schema := range schemas {
		tableName := schema.TableName
		if tableName == "" {
			continue
		}

		// Initialize node if not exists
		if _, exists := graph[tableName]; !exists {
			graph[tableName] = []tableEdge{}
		}

		// Add edges from relationships
		for _, rel := range schema.Relationships {
			if rel.TargetTable == "" {
				continue
			}

			// Add edge from this table to target
			graph[tableName] = append(graph[tableName], tableEdge{
				targetTable:   rel.TargetTable,
				localColumn:   rel.LocalColumn,
				foreignColumn: rel.ForeignColumn,
				joinType:      jd.inferJoinType(rel.Type),
			})

			// Add reverse edge (bidirectional graph)
			if _, exists := graph[rel.TargetTable]; !exists {
				graph[rel.TargetTable] = []tableEdge{}
			}
			graph[rel.TargetTable] = append(graph[rel.TargetTable], tableEdge{
				targetTable:   tableName,
				localColumn:   rel.ForeignColumn,
				foreignColumn: rel.LocalColumn,
				joinType:      jd.inferReverseJoinType(rel.Type),
			})
		}

		// Also consider FK columns as potential relationships
		for _, col := range schema.Columns {
			if col.IsForeignKey {
				// Try to infer target table from column name (e.g., user_id -> users)
				targetTable := jd.inferTargetTable(col.Name, schemas)
				if targetTable != "" && targetTable != tableName {
					graph[tableName] = append(graph[tableName], tableEdge{
						targetTable:   targetTable,
						localColumn:   col.Name,
						foreignColumn: "id", // Assume standard PK naming
						joinType:      "INNER",
					})
				}
			}
		}
	}

	return graph
}

// bfsJoinPath uses BFS to find optimal join path connecting all tables
func (jd *JoinDetector) bfsJoinPath(tables []string, graph map[string][]tableEdge, schemas []SchemaContext) JoinPath {
	if len(tables) == 0 {
		return JoinPath{}
	}
	if len(tables) == 1 {
		return JoinPath{Tables: tables, Joins: []JoinCondition{}, Cost: 0}
	}

	// Start from first table and find paths to all others
	startTable := tables[0]
	targetTables := make(map[string]bool)
	for i := 1; i < len(tables); i++ {
		targetTables[tables[i]] = true
	}

	// BFS state
	type bfsNode struct {
		table string
		path  []string
		joins []JoinCondition
	}

	visited := make(map[string]bool)
	queue := []bfsNode{{table: startTable, path: []string{startTable}, joins: []JoinCondition{}}}
	visited[startTable] = true

	result := JoinPath{Tables: []string{startTable}, Joins: []JoinCondition{}, Cost: 0}

	for len(queue) > 0 && len(targetTables) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Check if we reached a target table
		if targetTables[current.table] {
			delete(targetTables, current.table)
			result.Tables = current.path
			result.Joins = current.joins
			result.Cost = len(current.joins)
		}

		// Explore neighbors
		for _, edge := range graph[current.table] {
			if !visited[edge.targetTable] {
				visited[edge.targetTable] = true

				newPath := make([]string, len(current.path))
				copy(newPath, current.path)
				newPath = append(newPath, edge.targetTable)

				newJoins := make([]JoinCondition, len(current.joins))
				copy(newJoins, current.joins)
				newJoins = append(newJoins, JoinCondition{
					LeftTable:   current.table,
					RightTable:  edge.targetTable,
					LeftColumn:  edge.localColumn,
					RightColumn: edge.foreignColumn,
					JoinType:    edge.joinType,
				})

				queue = append(queue, bfsNode{
					table: edge.targetTable,
					path:  newPath,
					joins: newJoins,
				})
			}
		}
	}

	// If some tables weren't reachable, add them with CROSS JOIN warning
	for table := range targetTables {
		jd.logger.WithFields(logrus.Fields{
			"unreachable_table": table,
			"from":              startTable,
		}).Warn("No FK path found, would require CROSS JOIN")
		result.Tables = append(result.Tables, table)
	}

	return result
}

// inferJoinType infers SQL JOIN type from relationship type
func (jd *JoinDetector) inferJoinType(relType string) string {
	switch strings.ToLower(relType) {
	case "one-to-many", "1:n":
		return "LEFT"
	case "many-to-one", "n:1":
		return "INNER"
	case "many-to-many", "n:n":
		return "INNER" // Assumes junction table
	case "one-to-one", "1:1":
		return "INNER"
	default:
		return "INNER"
	}
}

// inferReverseJoinType returns the reverse join type
func (jd *JoinDetector) inferReverseJoinType(relType string) string {
	switch strings.ToLower(relType) {
	case "one-to-many", "1:n":
		return "INNER" // Reverse of LEFT is INNER (child -> parent)
	case "many-to-one", "n:1":
		return "LEFT" // Reverse of INNER from many side
	default:
		return "INNER"
	}
}

// inferTargetTable attempts to infer target table from FK column name
func (jd *JoinDetector) inferTargetTable(columnName string, schemas []SchemaContext) string {
	colLower := strings.ToLower(columnName)

	// Common patterns: user_id -> users, category_id -> categories
	if strings.HasSuffix(colLower, "_id") {
		baseName := strings.TrimSuffix(colLower, "_id")

		// Try common pluralization patterns
		candidates := []string{
			baseName + "s",      // user -> users
			baseName + "es",     // status -> statuses
			baseName,            // exact match
		}

		// Handle special cases
		if strings.HasSuffix(baseName, "y") {
			candidates = append(candidates, strings.TrimSuffix(baseName, "y")+"ies") // category -> categories
		}

		for _, schema := range schemas {
			tableLower := strings.ToLower(schema.TableName)
			for _, candidate := range candidates {
				if tableLower == candidate {
					return schema.TableName
				}
			}
		}
	}

	return ""
}

// GenerateJoinConditions generates join conditions for a path
func (jd *JoinDetector) GenerateJoinConditions(path JoinPath) []JoinCondition {
	return path.Joins
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
