package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// QueryContext represents enriched context for a query
type QueryContext struct {
	Query            string             `json:"query"`
	RelevantSchemas  []SchemaContext    `json:"relevant_schemas"`
	SimilarQueries   []QueryPattern     `json:"similar_queries"`
	BusinessRules    []BusinessRule     `json:"business_rules"`
	PerformanceHints []OptimizationHint `json:"performance_hints"`
	DataStatistics   *DataStats         `json:"data_statistics"`
	Suggestions      []Suggestion       `json:"suggestions"`
	Confidence       float32            `json:"confidence"`
}

// SchemaContext represents relevant schema information
type SchemaContext struct {
	TableName     string             `json:"table_name"`
	Columns       []ColumnInfo       `json:"columns"`
	Indexes       []IndexInfo        `json:"indexes"`
	Relationships []RelationshipInfo `json:"relationships"`
	RowCount      int64              `json:"row_count"`
	Description   string             `json:"description"`
	Relevance     float32            `json:"relevance"`
}

// ColumnInfo represents column metadata
type ColumnInfo struct {
	Name         string                 `json:"name"`
	DataType     string                 `json:"data_type"`
	IsNullable   bool                   `json:"is_nullable"`
	IsPrimaryKey bool                   `json:"is_primary_key"`
	IsForeignKey bool                   `json:"is_foreign_key"`
	Description  string                 `json:"description"`
	Statistics   map[string]interface{} `json:"statistics"`
}

// IndexInfo represents index metadata
type IndexInfo struct {
	Name      string   `json:"name"`
	Columns   []string `json:"columns"`
	Type      string   `json:"type"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
	Usage     int64    `json:"usage_count"`
}

// RelationshipInfo represents table relationships
type RelationshipInfo struct {
	Type          string `json:"type"` // one-to-one, one-to-many, many-to-many
	TargetTable   string `json:"target_table"`
	LocalColumn   string `json:"local_column"`
	ForeignColumn string `json:"foreign_column"`
	JoinFrequency int64  `json:"join_frequency"`
}

// QueryPattern represents a similar query pattern
type QueryPattern struct {
	Pattern          string        `json:"pattern"`
	Query            string        `json:"query"`
	Frequency        int           `json:"frequency"`
	AvgExecutionTime time.Duration `json:"avg_execution_time"`
	LastUsed         time.Time     `json:"last_used"`
	Similarity       float32       `json:"similarity"`
}

// BusinessRule represents a business logic rule
type BusinessRule struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	SQLMapping  string                 `json:"sql_mapping"`
	Conditions  []string               `json:"conditions"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// OptimizationHint represents a performance optimization suggestion
type OptimizationHint struct {
	Type        string  `json:"type"` // index, rewrite, partition, etc.
	Description string  `json:"description"`
	Impact      string  `json:"impact"` // high, medium, low
	SQLBefore   string  `json:"sql_before,omitempty"`
	SQLAfter    string  `json:"sql_after,omitempty"`
	Confidence  float32 `json:"confidence"`
}

// DataStats represents data statistics
type DataStats struct {
	TotalRows      int64              `json:"total_rows"`
	DataSize       int64              `json:"data_size_bytes"`
	LastAnalyzed   time.Time          `json:"last_analyzed"`
	GrowthRate     float64            `json:"growth_rate_percent"`
	AccessPatterns []AccessPattern    `json:"access_patterns"`
	Distribution   map[string]float64 `json:"distribution"`
}

// AccessPattern represents data access patterns
type AccessPattern struct {
	Pattern      string    `json:"pattern"`
	Frequency    int       `json:"frequency"`
	TimeOfDay    []int     `json:"time_of_day"` // hours when most accessed
	Users        []string  `json:"users"`
	LastAccessed time.Time `json:"last_accessed"`
}

// Suggestion represents a query suggestion
type Suggestion struct {
	Type        string  `json:"type"` // completion, correction, optimization
	Text        string  `json:"text"`
	Confidence  float32 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

// ContextBuilder builds enriched context for queries
type ContextBuilder struct {
	vectorStore      VectorStore
	embeddingService EmbeddingService
	schemaAnalyzer   *SchemaAnalyzer
	patternMatcher   *PatternMatcher
	statsCollector   *StatsCollector
	logger           *logrus.Logger
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(
	vectorStore VectorStore,
	embeddingService EmbeddingService,
	logger *logrus.Logger,
) *ContextBuilder {
	return &ContextBuilder{
		vectorStore:      vectorStore,
		embeddingService: embeddingService,
		schemaAnalyzer:   NewSchemaAnalyzer(logger),
		patternMatcher:   NewPatternMatcher(logger),
		statsCollector:   NewStatsCollector(logger),
		logger:           logger,
	}
}

// BuildContext builds comprehensive context for a query
func (cb *ContextBuilder) BuildContext(ctx context.Context, query string, connectionID string) (*QueryContext, error) {
	startTime := time.Now()

	// Generate embedding for the query
	embedding, err := cb.embeddingService.EmbedText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Create base context
	queryContext := &QueryContext{
		Query:      query,
		Confidence: 0.0,
	}

	// Parallel context enrichment
	errChan := make(chan error, 5)
	doneChan := make(chan bool, 5)

	// Fetch relevant schemas
	go func() {
		schemas, err := cb.fetchRelevantSchemas(ctx, embedding, connectionID)
		if err != nil {
			errChan <- err
			return
		}
		queryContext.RelevantSchemas = schemas
		doneChan <- true
	}()

	// Find similar queries
	go func() {
		patterns, err := cb.findSimilarQueries(ctx, embedding, connectionID)
		if err != nil {
			errChan <- err
			return
		}
		queryContext.SimilarQueries = patterns
		doneChan <- true
	}()

	// Extract business rules
	go func() {
		rules, err := cb.extractBusinessRules(ctx, query, embedding)
		if err != nil {
			errChan <- err
			return
		}
		queryContext.BusinessRules = rules
		doneChan <- true
	}()

	// Generate optimization hints
	go func() {
		hints, err := cb.generateOptimizationHints(ctx, query, connectionID)
		if err != nil {
			errChan <- err
			return
		}
		queryContext.PerformanceHints = hints
		doneChan <- true
	}()

	// Collect data statistics
	go func() {
		stats, err := cb.collectDataStatistics(ctx, query, connectionID)
		if err != nil {
			errChan <- err
			return
		}
		queryContext.DataStatistics = stats
		doneChan <- true
	}()

	// Wait for all goroutines to complete
	completed := 0
	for completed < 5 {
		select {
		case <-doneChan:
			completed++
		case err := <-errChan:
			cb.logger.WithError(err).Warn("Error during context enrichment")
			completed++
		case <-time.After(5 * time.Second):
			cb.logger.Warn("Context building timeout")
			break
		}
	}

	// Generate suggestions based on context
	queryContext.Suggestions = cb.generateSuggestions(queryContext)

	// Calculate overall confidence
	queryContext.Confidence = cb.calculateConfidence(queryContext)

	cb.logger.WithFields(logrus.Fields{
		"query":       query,
		"schemas":     len(queryContext.RelevantSchemas),
		"patterns":    len(queryContext.SimilarQueries),
		"rules":       len(queryContext.BusinessRules),
		"hints":       len(queryContext.PerformanceHints),
		"suggestions": len(queryContext.Suggestions),
		"confidence":  queryContext.Confidence,
		"duration":    time.Since(startTime),
	}).Debug("Query context built")

	return queryContext, nil
}

// fetchRelevantSchemas retrieves relevant schema information
func (cb *ContextBuilder) fetchRelevantSchemas(ctx context.Context, embedding []float32, connectionID string) ([]SchemaContext, error) {
    // Use hybrid search (vector + FTS) and filter in-memory for type/connection
    docs, err := cb.vectorStore.HybridSearch(ctx, "", embedding, 20)
	if err != nil {
		return nil, err
	}

    // Convert documents to schema contexts
	schemas := make([]SchemaContext, 0, len(docs))
	for _, doc := range docs {
        if doc.Type != DocumentTypeSchema {
            continue
        }
        if connectionID != "" && doc.ConnectionID != connectionID {
            continue
        }
		schema := cb.parseSchemaDocument(doc)
		schema.Relevance = doc.Score
		schemas = append(schemas, schema)
	}

	// Sort by relevance
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Relevance > schemas[j].Relevance
	})

	// Limit to top 5 most relevant
	if len(schemas) > 5 {
		schemas = schemas[:5]
	}

	return schemas, nil
}

// findSimilarQueries finds similar query patterns
func (cb *ContextBuilder) findSimilarQueries(ctx context.Context, embedding []float32, connectionID string) ([]QueryPattern, error) {
	// Search for similar query documents
	filter := map[string]interface{}{
		"connection_id": connectionID,
		"type":          string(DocumentTypeQuery),
	}

	docs, err := cb.vectorStore.SearchSimilar(ctx, embedding, 20, filter)
	if err != nil {
		return nil, err
	}

	// Group and analyze patterns
	patterns := cb.patternMatcher.ExtractPatterns(docs)

	// Sort by similarity and frequency
	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].Similarity != patterns[j].Similarity {
			return patterns[i].Similarity > patterns[j].Similarity
		}
		return patterns[i].Frequency > patterns[j].Frequency
	})

	// Limit to top 10 patterns
	if len(patterns) > 10 {
		patterns = patterns[:10]
	}

	return patterns, nil
}

// extractBusinessRules extracts applicable business rules
func (cb *ContextBuilder) extractBusinessRules(ctx context.Context, query string, embedding []float32) ([]BusinessRule, error) {
	// Search for business rule documents
	filter := map[string]interface{}{
		"type": string(DocumentTypeBusiness),
	}

	docs, err := cb.vectorStore.SearchSimilar(ctx, embedding, 10, filter)
	if err != nil {
		return nil, err
	}

	// Parse and filter rules
	rules := make([]BusinessRule, 0)
	for _, doc := range docs {
		rule := cb.parseBusinessRule(doc)
		if cb.isRuleApplicable(rule, query) {
			rules = append(rules, rule)
		}
	}

	// Sort by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	return rules, nil
}

// generateOptimizationHints generates performance optimization hints
func (cb *ContextBuilder) generateOptimizationHints(ctx context.Context, query string, connectionID string) ([]OptimizationHint, error) {
	// Search for performance-related documents
	filter := map[string]interface{}{
		"connection_id": connectionID,
		"type":          string(DocumentTypePerformance),
	}

	embedding, _ := cb.embeddingService.EmbedText(ctx, query)
	docs, err := cb.vectorStore.SearchSimilar(ctx, embedding, 10, filter)
	if err != nil {
		return nil, err
	}

	// Generate hints based on patterns and statistics
	hints := make([]OptimizationHint, 0)

	// Check for missing indexes
	if hint := cb.checkMissingIndexes(query, docs); hint != nil {
		hints = append(hints, *hint)
	}

	// Check for query rewrites
	if hint := cb.suggestQueryRewrite(query, docs); hint != nil {
		hints = append(hints, *hint)
	}

	// Check for partitioning opportunities
	if hint := cb.checkPartitioning(query, docs); hint != nil {
		hints = append(hints, *hint)
	}

	return hints, nil
}

// collectDataStatistics collects relevant data statistics
func (cb *ContextBuilder) collectDataStatistics(ctx context.Context, query string, connectionID string) (*DataStats, error) {
	// This would typically query actual database statistics
	// For now, return mock statistics
	return &DataStats{
		TotalRows:    1000000,
		DataSize:     1024 * 1024 * 100, // 100MB
		LastAnalyzed: time.Now().Add(-24 * time.Hour),
		GrowthRate:   5.5,
		AccessPatterns: []AccessPattern{
			{
				Pattern:      "daily_aggregation",
				Frequency:    100,
				TimeOfDay:    []int{9, 10, 11, 14, 15, 16},
				LastAccessed: time.Now().Add(-1 * time.Hour),
			},
		},
		Distribution: map[string]float64{
			"uniform": 0.3,
			"skewed":  0.5,
			"sparse":  0.2,
		},
	}, nil
}

// generateSuggestions generates query suggestions
func (cb *ContextBuilder) generateSuggestions(context *QueryContext) []Suggestion {
	suggestions := make([]Suggestion, 0)

	// Suggest based on similar queries
	if len(context.SimilarQueries) > 0 {
		for i, pattern := range context.SimilarQueries {
			if i >= 3 {
				break
			}
			suggestions = append(suggestions, Suggestion{
				Type:        "completion",
				Text:        pattern.Query,
				Confidence:  pattern.Similarity,
				Explanation: fmt.Sprintf("Similar query used %d times", pattern.Frequency),
			})
		}
	}

	// Suggest based on optimization hints
	for _, hint := range context.PerformanceHints {
		if hint.SQLAfter != "" {
			suggestions = append(suggestions, Suggestion{
				Type:        "optimization",
				Text:        hint.SQLAfter,
				Confidence:  hint.Confidence,
				Explanation: hint.Description,
			})
		}
	}

	return suggestions
}

// calculateConfidence calculates overall confidence score
func (cb *ContextBuilder) calculateConfidence(context *QueryContext) float32 {
	var confidence float32 = 0.0
	weights := 0

	// Weight based on schema relevance
	if len(context.RelevantSchemas) > 0 {
		var schemaScore float32 = 0.0
		for _, schema := range context.RelevantSchemas {
			schemaScore += schema.Relevance
		}
		confidence += schemaScore / float32(len(context.RelevantSchemas)) * 0.3
		weights++
	}

	// Weight based on similar queries
	if len(context.SimilarQueries) > 0 {
		var queryScore float32 = 0.0
		for _, pattern := range context.SimilarQueries {
			queryScore += pattern.Similarity
		}
		confidence += queryScore / float32(len(context.SimilarQueries)) * 0.3
		weights++
	}

	// Weight based on business rules
	if len(context.BusinessRules) > 0 {
		confidence += 0.2
		weights++
	}

	// Weight based on optimization hints
	if len(context.PerformanceHints) > 0 {
		confidence += 0.2
		weights++
	}

	if weights > 0 {
		return confidence
	}

	return 0.5 // Default confidence
}

// Helper methods for parsing and analysis

func (cb *ContextBuilder) parseSchemaDocument(doc *Document) SchemaContext {
	// Parse schema information from document
	// This would be implemented based on actual document structure
	return SchemaContext{
		TableName: doc.Metadata["table_name"].(string),
		// ... other fields
	}
}

func (cb *ContextBuilder) parseBusinessRule(doc *Document) BusinessRule {
	// Parse business rule from document
	return BusinessRule{
		Name:        doc.Metadata["name"].(string),
		Description: doc.Content,
		// ... other fields
	}
}

func (cb *ContextBuilder) isRuleApplicable(rule BusinessRule, query string) bool {
	// Check if rule applies to the query
	query = strings.ToLower(query)
	for _, condition := range rule.Conditions {
		if strings.Contains(query, strings.ToLower(condition)) {
			return true
		}
	}
	return false
}

func (cb *ContextBuilder) checkMissingIndexes(query string, docs []*Document) *OptimizationHint {
	// Analyze query for potential missing indexes
	// This is a simplified implementation
	if strings.Contains(strings.ToLower(query), "where") &&
		!strings.Contains(strings.ToLower(query), "index") {
		return &OptimizationHint{
			Type:        "index",
			Description: "Consider adding an index on the WHERE clause columns",
			Impact:      "high",
			Confidence:  0.7,
		}
	}
	return nil
}

func (cb *ContextBuilder) suggestQueryRewrite(query string, docs []*Document) *OptimizationHint {
	// Suggest query rewrites for better performance
	if strings.Contains(strings.ToLower(query), "select *") {
		return &OptimizationHint{
			Type:        "rewrite",
			Description: "Avoid SELECT *, specify only needed columns",
			Impact:      "medium",
			SQLBefore:   query,
			SQLAfter:    strings.Replace(query, "*", "specific_columns", 1),
			Confidence:  0.8,
		}
	}
	return nil
}

func (cb *ContextBuilder) checkPartitioning(query string, docs []*Document) *OptimizationHint {
	// Check if partitioning could help
	if strings.Contains(strings.ToLower(query), "between") ||
		strings.Contains(strings.ToLower(query), "date") {
		return &OptimizationHint{
			Type:        "partition",
			Description: "Consider partitioning the table by date for better performance",
			Impact:      "high",
			Confidence:  0.6,
		}
	}
	return nil
}
