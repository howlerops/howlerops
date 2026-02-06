package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// BuildContextWithBudget builds query context respecting token budget constraints
func (cb *ContextBuilder) BuildContextWithBudget(
	ctx context.Context,
	query string,
	connectionID string,
	maxTokens int,
	hasError bool,
) (*QueryContext, *BudgetAllocation, error) {
	// Create token budget
	budget := DefaultTokenBudget(maxTokens)

	// Prioritize components based on query characteristics
	priorities := PrioritizeComponents(query, hasError)

	// Allocate budget across components
	allocation := budget.AllocateContextBudget(priorities)

	// Validate allocation
	if err := allocation.Validate(maxTokens); err != nil {
		return nil, nil, fmt.Errorf("budget validation failed: %w", err)
	}

	cb.logger.WithFields(logrus.Fields{
		"total_tokens":       maxTokens,
		"context_available":  budget.ContextAvailable,
		"schema_budget":      allocation.Schema,
		"examples_budget":    allocation.Examples,
		"business_budget":    allocation.Business,
		"performance_budget": allocation.Performance,
	}).Debug("Token budget allocated")

	// Build context with budget constraints
	queryContext, err := cb.buildContextWithConstraints(ctx, query, connectionID, allocation)
	if err != nil {
		return nil, nil, err
	}

	// Track actual usage
	cb.updateAllocationUsage(queryContext, allocation)

	return queryContext, allocation, nil
}

// buildContextWithConstraints builds context while respecting token budgets
func (cb *ContextBuilder) buildContextWithConstraints(
	ctx context.Context,
	query string,
	connectionID string,
	allocation *BudgetAllocation,
) (*QueryContext, error) {
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

	// Fetch components in priority order, respecting budgets
	// This ensures high-priority components get their full budget
	// while lower-priority ones get what's left

	// 1. Schema (highest priority)
	if allocation.Schema > 0 {
		schemas, err := cb.fetchRelevantSchemasWithBudget(
			ctx,
			embedding,
			connectionID,
			allocation.Schema,
		)
		if err != nil {
			cb.logger.WithError(err).Warn("Failed to fetch schemas")
		} else {
			queryContext.RelevantSchemas = schemas
		}
	}

	// 2. Examples (high priority)
	if allocation.Examples > 0 {
		patterns, err := cb.findSimilarQueriesWithBudget(
			ctx,
			embedding,
			connectionID,
			allocation.Examples,
		)
		if err != nil {
			cb.logger.WithError(err).Warn("Failed to find similar queries")
		} else {
			queryContext.SimilarQueries = patterns
		}
	}

	// 3. Business rules (medium priority)
	if allocation.Business > 0 {
		rules, err := cb.extractBusinessRulesWithBudget(
			ctx,
			query,
			embedding,
			allocation.Business,
		)
		if err != nil {
			cb.logger.WithError(err).Warn("Failed to extract business rules")
		} else {
			queryContext.BusinessRules = rules
		}
	}

	// 4. Performance hints (lower priority)
	if allocation.Performance > 0 {
		hints, err := cb.generateOptimizationHintsWithBudget(
			ctx,
			query,
			connectionID,
			allocation.Performance,
		)
		if err != nil {
			cb.logger.WithError(err).Warn("Failed to generate optimization hints")
		} else {
			queryContext.PerformanceHints = hints
		}
	}

	// Generate suggestions
	queryContext.Suggestions = cb.generateSuggestions(queryContext)

	// Calculate confidence
	queryContext.Confidence = cb.calculateConfidence(queryContext)

	return queryContext, nil
}

// fetchRelevantSchemasWithBudget retrieves schemas within token budget
func (cb *ContextBuilder) fetchRelevantSchemasWithBudget(
	ctx context.Context,
	embedding []float32,
	connectionID string,
	maxTokens int,
) ([]SchemaContext, error) {
	// Fetch more schemas than needed, then filter by budget
	docs, err := cb.vectorStore.HybridSearch(ctx, "", embedding, 20)
	if err != nil {
		return nil, err
	}

	schemas := make([]SchemaContext, 0)
	tokensUsed := 0

	for _, doc := range docs {
		if doc.Type != DocumentTypeSchema {
			continue
		}
		if connectionID != "" && doc.ConnectionID != connectionID {
			continue
		}

		schema := cb.parseSchemaDocument(doc)
		schema.Relevance = doc.Score

		// Estimate tokens for this schema
		schemaText := cb.formatSchemaForContext(schema)
		schemaTokens := EstimateTokenCount(schemaText)

		// Check if adding this schema would exceed budget
		if tokensUsed+schemaTokens > maxTokens {
			// Try to fit a truncated version
			remainingTokens := maxTokens - tokensUsed
			if remainingTokens > 100 { // Only include if at least 100 tokens available
				truncatedText := TruncateToTokenBudget(schemaText, remainingTokens)
				truncatedTokens := EstimateTokenCount(truncatedText)
				schemas = append(schemas, schema)
				tokensUsed += truncatedTokens
			}
			break // No more room in budget
		}

		schemas = append(schemas, schema)
		tokensUsed += schemaTokens
	}

	cb.logger.WithFields(logrus.Fields{
		"schemas_count": len(schemas),
		"tokens_used":   tokensUsed,
		"tokens_budget": maxTokens,
	}).Debug("Schemas fetched with budget")

	return schemas, nil
}

// findSimilarQueriesWithBudget finds similar queries within token budget
func (cb *ContextBuilder) findSimilarQueriesWithBudget(
	ctx context.Context,
	embedding []float32,
	connectionID string,
	maxTokens int,
) ([]QueryPattern, error) {
	filter := map[string]interface{}{
		"connection_id": connectionID,
		"type":          string(DocumentTypeQuery),
	}

	docs, err := cb.vectorStore.SearchSimilar(ctx, embedding, 30, filter)
	if err != nil {
		return nil, err
	}

	// Extract patterns
	allPatterns := cb.patternMatcher.ExtractPatterns(docs)

	// Filter by budget
	patterns := make([]QueryPattern, 0)
	tokensUsed := 0

	for _, pattern := range allPatterns {
		// Estimate tokens for this pattern
		patternText := cb.formatPatternForContext(pattern)
		patternTokens := EstimateTokenCount(patternText)

		if tokensUsed+patternTokens > maxTokens {
			break
		}

		patterns = append(patterns, pattern)
		tokensUsed += patternTokens
	}

	cb.logger.WithFields(logrus.Fields{
		"patterns_count": len(patterns),
		"tokens_used":    tokensUsed,
		"tokens_budget":  maxTokens,
	}).Debug("Patterns found with budget")

	return patterns, nil
}

// extractBusinessRulesWithBudget extracts business rules within token budget
func (cb *ContextBuilder) extractBusinessRulesWithBudget(
	ctx context.Context,
	query string,
	embedding []float32,
	maxTokens int,
) ([]BusinessRule, error) {
	filter := map[string]interface{}{
		"type": string(DocumentTypeBusiness),
	}

	docs, err := cb.vectorStore.SearchSimilar(ctx, embedding, 15, filter)
	if err != nil {
		return nil, err
	}

	rules := make([]BusinessRule, 0)
	tokensUsed := 0

	for _, doc := range docs {
		rule := cb.parseBusinessRule(doc)
		if !cb.isRuleApplicable(rule, query) {
			continue
		}

		// Estimate tokens for this rule
		ruleText := cb.formatRuleForContext(rule)
		ruleTokens := EstimateTokenCount(ruleText)

		if tokensUsed+ruleTokens > maxTokens {
			break
		}

		rules = append(rules, rule)
		tokensUsed += ruleTokens
	}

	cb.logger.WithFields(logrus.Fields{
		"rules_count":   len(rules),
		"tokens_used":   tokensUsed,
		"tokens_budget": maxTokens,
	}).Debug("Business rules extracted with budget")

	return rules, nil
}

// generateOptimizationHintsWithBudget generates hints within token budget
func (cb *ContextBuilder) generateOptimizationHintsWithBudget(
	ctx context.Context,
	query string,
	connectionID string,
	maxTokens int,
) ([]OptimizationHint, error) {
	filter := map[string]interface{}{
		"connection_id": connectionID,
		"type":          string(DocumentTypePerformance),
	}

	embedding, _ := cb.embeddingService.EmbedText(ctx, query)
	docs, err := cb.vectorStore.SearchSimilar(ctx, embedding, 10, filter)
	if err != nil {
		return nil, err
	}

	hints := make([]OptimizationHint, 0)
	tokensUsed := 0

	// Generate hints
	if hint := cb.checkMissingIndexes(query, docs); hint != nil {
		hintText := cb.formatHintForContext(*hint)
		hintTokens := EstimateTokenCount(hintText)
		if tokensUsed+hintTokens <= maxTokens {
			hints = append(hints, *hint)
			tokensUsed += hintTokens
		}
	}

	if hint := cb.suggestQueryRewrite(query, docs); hint != nil {
		hintText := cb.formatHintForContext(*hint)
		hintTokens := EstimateTokenCount(hintText)
		if tokensUsed+hintTokens <= maxTokens {
			hints = append(hints, *hint)
			tokensUsed += hintTokens
		}
	}

	if hint := cb.checkPartitioning(query, docs); hint != nil {
		hintText := cb.formatHintForContext(*hint)
		hintTokens := EstimateTokenCount(hintText)
		if tokensUsed+hintTokens <= maxTokens {
			hints = append(hints, *hint)
			tokensUsed += hintTokens
		}
	}

	cb.logger.WithFields(logrus.Fields{
		"hints_count":   len(hints),
		"tokens_used":   tokensUsed,
		"tokens_budget": maxTokens,
	}).Debug("Optimization hints generated with budget")

	return hints, nil
}

// updateAllocationUsage updates budget allocation with actual usage
func (cb *ContextBuilder) updateAllocationUsage(context *QueryContext, allocation *BudgetAllocation) {
	// Calculate actual tokens used by each component
	if len(context.RelevantSchemas) > 0 {
		tokensUsed := 0
		for _, schema := range context.RelevantSchemas {
			schemaText := cb.formatSchemaForContext(schema)
			tokensUsed += EstimateTokenCount(schemaText)
		}
		allocation.AdjustForActualUsage("schema", tokensUsed)
	}

	if len(context.SimilarQueries) > 0 {
		tokensUsed := 0
		for _, pattern := range context.SimilarQueries {
			patternText := cb.formatPatternForContext(pattern)
			tokensUsed += EstimateTokenCount(patternText)
		}
		allocation.AdjustForActualUsage("examples", tokensUsed)
	}

	if len(context.BusinessRules) > 0 {
		tokensUsed := 0
		for _, rule := range context.BusinessRules {
			ruleText := cb.formatRuleForContext(rule)
			tokensUsed += EstimateTokenCount(ruleText)
		}
		allocation.AdjustForActualUsage("business", tokensUsed)
	}

	if len(context.PerformanceHints) > 0 {
		tokensUsed := 0
		for _, hint := range context.PerformanceHints {
			hintText := cb.formatHintForContext(hint)
			tokensUsed += EstimateTokenCount(hintText)
		}
		allocation.AdjustForActualUsage("performance", tokensUsed)
	}
}

// Format helpers for token estimation

func (cb *ContextBuilder) formatSchemaForContext(schema SchemaContext) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Table: %s", schema.TableName))

	if schema.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", schema.Description))
	}

	if len(schema.Columns) > 0 {
		parts = append(parts, "Columns:")
		for _, col := range schema.Columns {
			colDesc := fmt.Sprintf("  - %s (%s)", col.Name, col.DataType)
			if col.IsPrimaryKey {
				colDesc += " PRIMARY KEY"
			}
			if col.IsForeignKey {
				colDesc += " FOREIGN KEY"
			}
			parts = append(parts, colDesc)
		}
	}

	if len(schema.Relationships) > 0 {
		parts = append(parts, "Relationships:")
		for _, rel := range schema.Relationships {
			parts = append(parts, fmt.Sprintf("  - %s -> %s.%s",
				rel.LocalColumn, rel.TargetTable, rel.ForeignColumn))
		}
	}

	return strings.Join(parts, "\n")
}

func (cb *ContextBuilder) formatPatternForContext(pattern QueryPattern) string {
	return fmt.Sprintf("Pattern: %s\nQuery: %s\nFrequency: %d\nSimilarity: %.2f",
		pattern.Pattern, pattern.Query, pattern.Frequency, pattern.Similarity)
}

func (cb *ContextBuilder) formatRuleForContext(rule BusinessRule) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Rule: %s", rule.Name))
	parts = append(parts, fmt.Sprintf("Description: %s", rule.Description))

	if rule.SQLMapping != "" {
		parts = append(parts, fmt.Sprintf("SQL Mapping: %s", rule.SQLMapping))
	}

	if len(rule.Conditions) > 0 {
		parts = append(parts, fmt.Sprintf("Conditions: %s", strings.Join(rule.Conditions, ", ")))
	}

	return strings.Join(parts, "\n")
}

func (cb *ContextBuilder) formatHintForContext(hint OptimizationHint) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Type: %s", hint.Type))
	parts = append(parts, fmt.Sprintf("Impact: %s", hint.Impact))
	parts = append(parts, fmt.Sprintf("Description: %s", hint.Description))

	if hint.SQLAfter != "" {
		parts = append(parts, fmt.Sprintf("Suggested SQL: %s", hint.SQLAfter))
	}

	return strings.Join(parts, "\n")
}

// FormatContextForPrompt formats the query context for inclusion in AI prompts
func FormatContextForPrompt(context *QueryContext, allocation *BudgetAllocation) string {
	var sections []string

	// Schema section
	if len(context.RelevantSchemas) > 0 && allocation.GetComponentBudget("schema") > 0 {
		schemaSection := "## Database Schema\n\n"
		for _, schema := range context.RelevantSchemas {
			schemaSection += fmt.Sprintf("### %s (relevance: %.2f)\n\n", schema.TableName, schema.Relevance)

			if schema.Description != "" {
				schemaSection += fmt.Sprintf("%s\n\n", schema.Description)
			}

			if len(schema.Columns) > 0 {
				schemaSection += "Columns:\n"
				for _, col := range schema.Columns {
					flags := ""
					if col.IsPrimaryKey {
						flags += " [PK]"
					}
					if col.IsForeignKey {
						flags += " [FK]"
					}
					if !col.IsNullable {
						flags += " [NOT NULL]"
					}
					schemaSection += fmt.Sprintf("- %s: %s%s\n", col.Name, col.DataType, flags)
				}
				schemaSection += "\n"
			}

			if len(schema.Relationships) > 0 {
				schemaSection += "Relationships:\n"
				for _, rel := range schema.Relationships {
					schemaSection += fmt.Sprintf("- %s (%s) -> %s (%s)\n",
						schema.TableName, rel.LocalColumn, rel.TargetTable, rel.ForeignColumn)
				}
				schemaSection += "\n"
			}
		}
		sections = append(sections, schemaSection)
	}

	// Examples section
	if len(context.SimilarQueries) > 0 && allocation.GetComponentBudget("examples") > 0 {
		exampleSection := "## Similar Query Examples\n\n"
		for i, pattern := range context.SimilarQueries {
			if i >= 5 { // Limit to top 5 examples
				break
			}
			exampleSection += fmt.Sprintf("### Example %d (similarity: %.2f)\n\n", i+1, pattern.Similarity)
			exampleSection += fmt.Sprintf("```sql\n%s\n```\n\n", pattern.Query)
			if pattern.Frequency > 1 {
				exampleSection += fmt.Sprintf("Used %d times\n\n", pattern.Frequency)
			}
		}
		sections = append(sections, exampleSection)
	}

	// Business rules section
	if len(context.BusinessRules) > 0 && allocation.GetComponentBudget("business") > 0 {
		rulesSection := "## Applicable Business Rules\n\n"
		for _, rule := range context.BusinessRules {
			rulesSection += fmt.Sprintf("### %s\n\n", rule.Name)
			rulesSection += fmt.Sprintf("%s\n\n", rule.Description)
			if rule.SQLMapping != "" {
				rulesSection += fmt.Sprintf("SQL Mapping: `%s`\n\n", rule.SQLMapping)
			}
		}
		sections = append(sections, rulesSection)
	}

	// Performance hints section
	if len(context.PerformanceHints) > 0 && allocation.GetComponentBudget("performance") > 0 {
		hintsSection := "## Performance Considerations\n\n"
		for _, hint := range context.PerformanceHints {
			hintsSection += fmt.Sprintf("- **%s** (%s impact): %s\n", hint.Type, hint.Impact, hint.Description)
		}
		hintsSection += "\n"
		sections = append(sections, hintsSection)
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n")
}
