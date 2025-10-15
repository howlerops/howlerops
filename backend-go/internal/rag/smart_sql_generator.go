package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// SmartSQLGenerator generates SQL using RAG context
type SmartSQLGenerator struct {
	contextBuilder *ContextBuilder
	llmProvider    LLMProvider
	validator      *SQLValidator
	planner        *QueryPlanner
	joinDetector   *JoinDetector
	logger         *logrus.Logger
}

// LLMProvider interface for language model providers
type LLMProvider interface {
	GenerateSQL(ctx context.Context, prompt string, context *QueryContext) (*GeneratedSQL, error)
	ExplainSQL(ctx context.Context, sql string) (*SQLExplanation, error)
	OptimizeSQL(ctx context.Context, sql string, hints []OptimizationHint) (*OptimizedSQL, error)
}

// GeneratedSQL represents generated SQL with metadata
type GeneratedSQL struct {
	Query         string            `json:"query"`
	Explanation   string            `json:"explanation"`
	Confidence    float32           `json:"confidence"`
	Tables        []string          `json:"tables"`
	Columns       []string          `json:"columns"`
	Warnings      []string          `json:"warnings"`
	AlternativeQueries []string     `json:"alternative_queries,omitempty"`
}

// SQLExplanation represents SQL explanation
type SQLExplanation struct {
	Summary       string         `json:"summary"`
	Steps         []ExplanationStep `json:"steps"`
	Complexity    string         `json:"complexity"` // simple, moderate, complex
	EstimatedTime string         `json:"estimated_time"`
}

// ExplanationStep represents a step in SQL explanation
type ExplanationStep struct {
	Order       int    `json:"order"`
	Operation   string `json:"operation"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// OptimizedSQL represents optimized SQL
type OptimizedSQL struct {
	OriginalQuery   string            `json:"original_query"`
	OptimizedQuery  string            `json:"optimized_query"`
	Improvements    []Improvement     `json:"improvements"`
	EstimatedGain   float32           `json:"estimated_gain_percent"`
}

// Improvement represents an optimization improvement
type Improvement struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Before      string `json:"before"`
	After       string `json:"after"`
}

// NewSmartSQLGenerator creates a new smart SQL generator
func NewSmartSQLGenerator(
	contextBuilder *ContextBuilder,
	llmProvider LLMProvider,
	logger *logrus.Logger,
) *SmartSQLGenerator {
	return &SmartSQLGenerator{
		contextBuilder: contextBuilder,
		llmProvider:    llmProvider,
		validator:      NewSQLValidator(logger),
		planner:        NewQueryPlanner(logger),
		joinDetector:   NewJoinDetector(logger),
		logger:         logger,
	}
}

// Generate generates SQL from natural language prompt
func (g *SmartSQLGenerator) Generate(ctx context.Context, prompt string, connectionID string) (*GeneratedSQL, error) {
	startTime := time.Now()

	// Build RAG context
	ragContext, err := g.contextBuilder.BuildContext(ctx, prompt, connectionID)
	if err != nil {
		g.logger.WithError(err).Warn("Failed to build RAG context")
		// Continue without context
		ragContext = &QueryContext{Query: prompt}
	}

	// Check if query needs multi-step planning
	if g.isComplexQuery(prompt) {
		return g.generateWithPlanning(ctx, prompt, ragContext)
	}

	// Generate SQL using LLM with context
	generated, err := g.llmProvider.GenerateSQL(ctx, prompt, ragContext)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Enhance with context information
	generated = g.enhanceWithContext(generated, ragContext)

	// Validate the generated SQL
	if err := g.validator.Validate(generated.Query); err != nil {
		generated.Warnings = append(generated.Warnings, fmt.Sprintf("Validation warning: %v", err))
	}

	// Auto-detect JOINs if needed
	if g.needsJoinDetection(prompt, generated) {
		g.detectAndAddJoins(generated, ragContext)
	}

	g.logger.WithFields(logrus.Fields{
		"prompt":      prompt,
		"confidence":  generated.Confidence,
		"tables":      len(generated.Tables),
		"duration":    time.Since(startTime),
	}).Info("SQL generated successfully")

	return generated, nil
}

// generateWithPlanning generates complex queries using multi-step planning
func (g *SmartSQLGenerator) generateWithPlanning(ctx context.Context, prompt string, context *QueryContext) (*GeneratedSQL, error) {
	// Decompose the request into steps
	steps := g.planner.DecomposeRequest(prompt, context)

	// Generate SQL for each step
	stepSQLs := make([]StepSQL, 0, len(steps))
	for _, step := range steps {
		stepSQL, err := g.generateStep(ctx, step, context)
		if err != nil {
			g.logger.WithError(err).Warnf("Failed to generate SQL for step: %v", step)
			continue
		}
		stepSQLs = append(stepSQLs, *stepSQL)
	}

	// Combine steps into final query
	finalSQL := g.planner.CombineSteps(stepSQLs)

	// Validate and optimize
	optimized := g.planner.ValidateAndOptimize(finalSQL)

	return &GeneratedSQL{
		Query:       optimized.Query,
		Explanation: optimized.Explanation,
		Confidence:  g.calculateConfidence(stepSQLs),
		Tables:      g.extractTables(optimized.Query),
		Columns:     g.extractColumns(optimized.Query),
	}, nil
}

// generateStep generates SQL for a single step
func (g *SmartSQLGenerator) generateStep(ctx context.Context, queryStep QueryStep, context *QueryContext) (*StepSQL, error) {
	// Generate SQL for the step
	generated, err := g.llmProvider.GenerateSQL(ctx, queryStep.Description, context)
	if err != nil {
		return nil, err
	}

	return &StepSQL{
		Step:        queryStep,
		SQL:         generated.Query,
		Explanation: generated.Explanation,
	}, nil
}

// Explain explains SQL in plain English
func (g *SmartSQLGenerator) Explain(ctx context.Context, sql string) (*SQLExplanation, error) {
	// Use LLM to explain the SQL
	explanation, err := g.llmProvider.ExplainSQL(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to explain SQL: %w", err)
	}

	// Enhance with query plan analysis
	g.enhanceExplanation(explanation, sql)

	return explanation, nil
}

// Optimize optimizes SQL query
func (g *SmartSQLGenerator) Optimize(ctx context.Context, sql string, connectionID string) (*OptimizedSQL, error) {
	// Build context for the SQL
	ragContext, err := g.contextBuilder.BuildContext(ctx, sql, connectionID)
	if err != nil {
		g.logger.WithError(err).Warn("Failed to build context for optimization")
		ragContext = &QueryContext{Query: sql}
	}

	// Use LLM to optimize with hints from context
	optimized, err := g.llmProvider.OptimizeSQL(ctx, sql, ragContext.PerformanceHints)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize SQL: %w", err)
	}

	// Validate the optimized query
	if err := g.validator.Validate(optimized.OptimizedQuery); err != nil {
		g.logger.WithError(err).Warn("Optimized query validation failed")
	}

	return optimized, nil
}

// enhanceWithContext enhances generated SQL with RAG context
func (g *SmartSQLGenerator) enhanceWithContext(generated *GeneratedSQL, context *QueryContext) *GeneratedSQL {
	// Add confidence from context
	if context.Confidence > generated.Confidence {
		generated.Confidence = (generated.Confidence + context.Confidence) / 2
	}

	// Add warnings from business rules
	for _, rule := range context.BusinessRules {
		if g.violatesRule(generated.Query, rule) {
			generated.Warnings = append(generated.Warnings,
				fmt.Sprintf("Business rule violation: %s", rule.Description))
		}
	}

	// Add alternative queries from similar patterns
	for i, pattern := range context.SimilarQueries {
		if i >= 3 {
			break
		}
		if pattern.Similarity > 0.8 {
			generated.AlternativeQueries = append(generated.AlternativeQueries, pattern.Query)
		}
	}

	return generated
}

// detectAndAddJoins detects and adds necessary JOINs
func (g *SmartSQLGenerator) detectAndAddJoins(generated *GeneratedSQL, context *QueryContext) {
	// Detect required tables from the query
	tables := g.joinDetector.DetectTables(generated.Query, context)

	// Find optimal join path
	joinPath := g.joinDetector.FindJoinPath(tables, context.RelevantSchemas)

	// Generate join conditions
	joinConditions := g.joinDetector.GenerateJoinConditions(joinPath)

	// Update the query with joins
	if len(joinConditions) > 0 {
		generated.Query = g.addJoinsToQuery(generated.Query, joinConditions)
		generated.Tables = append(generated.Tables, g.extractTablesFromJoins(joinConditions)...)
	}
}

// Helper methods

func (g *SmartSQLGenerator) isComplexQuery(prompt string) bool {
	// Check for indicators of complex queries
	complexIndicators := []string{
		"multiple", "combine", "aggregate", "group by",
		"having", "union", "intersect", "complex",
		"nested", "subquery", "with recursive",
	}

	prompt = strings.ToLower(prompt)
	for _, indicator := range complexIndicators {
		if strings.Contains(prompt, indicator) {
			return true
		}
	}

	// Check length as a simple heuristic
	return len(strings.Fields(prompt)) > 20
}

func (g *SmartSQLGenerator) needsJoinDetection(prompt string, generated *GeneratedSQL) bool {
	// Check if the prompt mentions multiple entities
	// but the generated SQL doesn't have JOINs
	promptLower := strings.ToLower(prompt)
	queryLower := strings.ToLower(generated.Query)

	hasMultipleEntities := strings.Count(promptLower, "and") > 1 ||
		strings.Contains(promptLower, "with their") ||
		strings.Contains(promptLower, "including")

	hasJoins := strings.Contains(queryLower, "join") ||
		strings.Contains(queryLower, "from") && strings.Contains(queryLower, ",")

	return hasMultipleEntities && !hasJoins
}

func (g *SmartSQLGenerator) violatesRule(query string, rule BusinessRule) bool {
	queryLower := strings.ToLower(query)

	// Check if query violates any conditions
	for _, condition := range rule.Conditions {
		if !strings.Contains(queryLower, strings.ToLower(condition)) {
			return true
		}
	}

	return false
}

func (g *SmartSQLGenerator) calculateConfidence(steps []StepSQL) float32 {
	if len(steps) == 0 {
		return 0.5
	}

	var totalConfidence float32 = 0
	for range steps {
		// This would be calculated based on step complexity and success
		totalConfidence += 0.8
	}

	return totalConfidence / float32(len(steps))
}

func (g *SmartSQLGenerator) extractTables(query string) []string {
	// Simple extraction - in production, use proper SQL parser
	tables := make([]string, 0)
	queryLower := strings.ToLower(query)

	// Look for FROM clause
	if fromIndex := strings.Index(queryLower, "from"); fromIndex >= 0 {
		afterFrom := query[fromIndex+5:]
		// Extract table name (simplified)
		parts := strings.Fields(afterFrom)
		if len(parts) > 0 {
			tableName := strings.TrimSuffix(parts[0], ",")
			tables = append(tables, tableName)
		}
	}

	// Look for JOIN clauses
	joinKeywords := []string{"join", "inner join", "left join", "right join", "full join"}
	for _, keyword := range joinKeywords {
		if joinIndex := strings.Index(queryLower, keyword); joinIndex >= 0 {
			afterJoin := query[joinIndex+len(keyword):]
			parts := strings.Fields(afterJoin)
			if len(parts) > 0 {
				tableName := strings.TrimSuffix(parts[0], ",")
				tables = append(tables, tableName)
			}
		}
	}

	return tables
}

func (g *SmartSQLGenerator) extractColumns(query string) []string {
	// Simple extraction - in production, use proper SQL parser
	columns := make([]string, 0)
	queryLower := strings.ToLower(query)

	// Look for SELECT clause
	if selectIndex := strings.Index(queryLower, "select"); selectIndex >= 0 {
		fromIndex := strings.Index(queryLower, "from")
		if fromIndex > selectIndex {
			selectClause := query[selectIndex+7 : fromIndex]
			// Extract column names (simplified)
			parts := strings.Split(selectClause, ",")
			for _, part := range parts {
				col := strings.TrimSpace(part)
				if col != "*" && !strings.Contains(col, "(") {
					columns = append(columns, col)
				}
			}
		}
	}

	return columns
}

func (g *SmartSQLGenerator) addJoinsToQuery(query string, conditions []JoinCondition) string {
	// This would properly parse and modify the SQL
	// For now, return the original query
	return query
}

func (g *SmartSQLGenerator) extractTablesFromJoins(conditions []JoinCondition) []string {
	tables := make([]string, 0)
	for _, condition := range conditions {
		tables = append(tables, condition.LeftTable, condition.RightTable)
	}
	return tables
}

func (g *SmartSQLGenerator) enhanceExplanation(explanation *SQLExplanation, sql string) {
	// Add complexity analysis
	complexity := g.analyzeComplexity(sql)
	explanation.Complexity = complexity

	// Estimate execution time
	explanation.EstimatedTime = g.estimateExecutionTime(sql, complexity)
}

func (g *SmartSQLGenerator) analyzeComplexity(sql string) string {
	sqlLower := strings.ToLower(sql)

	// Count complex operations
	joinCount := strings.Count(sqlLower, "join")
	subqueryCount := strings.Count(sqlLower, "select") - 1
	aggregateCount := 0
	for _, agg := range []string{"sum", "avg", "count", "max", "min"} {
		aggregateCount += strings.Count(sqlLower, agg)
	}

	complexity := joinCount + subqueryCount*2 + aggregateCount

	if complexity <= 1 {
		return "simple"
	} else if complexity <= 3 {
		return "moderate"
	}
	return "complex"
}

func (g *SmartSQLGenerator) estimateExecutionTime(sql string, complexity string) string {
	switch complexity {
	case "simple":
		return "< 100ms"
	case "moderate":
		return "100ms - 1s"
	case "complex":
		return "> 1s"
	default:
		return "unknown"
	}
}