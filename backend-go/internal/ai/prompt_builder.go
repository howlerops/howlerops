package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jbeck018/howlerops/backend-go/internal/rag"
)

// PromptBuilder builds prompts for AI providers with RAG context
type PromptBuilder struct {
	contextBuilder *rag.ContextBuilder
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder(contextBuilder *rag.ContextBuilder) *PromptBuilder {
	return &PromptBuilder{
		contextBuilder: contextBuilder,
	}
}

// BuildSQLGenerationPrompt builds a complete prompt for SQL generation with RAG context
func (pb *PromptBuilder) BuildSQLGenerationPrompt(
	ctx context.Context,
	req *SQLRequest,
	dialect SQLDialect,
	maxTokens int,
) (systemPrompt string, userPrompt string, allocation *rag.BudgetAllocation, err error) {
	// Get universal SQL system prompt
	systemPrompt = GetUniversalSQLPrompt(dialect)

	// Build user prompt with RAG context if we have a context builder
	if pb.contextBuilder != nil && req.Context != nil {
		connectionID := ""
		if connID, ok := req.Context["connection_id"]; ok {
			connectionID = connID
		}

		// Build context with budget
		queryContext, budgetAlloc, err := pb.contextBuilder.BuildContextWithBudget(
			ctx,
			req.Prompt,
			connectionID,
			maxTokens,
			false, // Not fixing an error
		)

		if err != nil {
			// Fall back to simple prompt if context building fails
			userPrompt = pb.buildSimpleUserPrompt(req)
			return systemPrompt, userPrompt, nil, nil
		}

		allocation = budgetAlloc
		userPrompt = pb.formatUserPromptWithContext(req, queryContext, budgetAlloc)
	} else {
		// Simple prompt without RAG context
		userPrompt = pb.buildSimpleUserPrompt(req)
	}

	return systemPrompt, userPrompt, allocation, nil
}

// BuildSQLFixPrompt builds a complete prompt for fixing SQL with RAG context
func (pb *PromptBuilder) BuildSQLFixPrompt(
	ctx context.Context,
	req *SQLRequest,
	dialect SQLDialect,
	maxTokens int,
) (systemPrompt string, userPrompt string, allocation *rag.BudgetAllocation, err error) {
	// Detect error category from error message
	errorCategory := DetectErrorCategory(req.Error)

	// Get SQL fix system prompt
	systemPrompt = GetSQLFixPrompt(dialect, errorCategory)

	// Build user prompt with RAG context if we have a context builder
	if pb.contextBuilder != nil && req.Context != nil {
		connectionID := ""
		if connID, ok := req.Context["connection_id"]; ok {
			connectionID = connID
		}

		// Build context with budget (prioritize examples for fixing)
		queryContext, budgetAlloc, err := pb.contextBuilder.BuildContextWithBudget(
			ctx,
			req.Prompt,
			connectionID,
			maxTokens,
			true, // Fixing an error
		)

		if err != nil {
			// Fall back to simple prompt if context building fails
			userPrompt = pb.buildSimpleFixPrompt(req)
			return systemPrompt, userPrompt, nil, nil
		}

		allocation = budgetAlloc
		userPrompt = pb.formatFixPromptWithContext(req, queryContext, budgetAlloc)
	} else {
		// Simple prompt without RAG context
		userPrompt = pb.buildSimpleFixPrompt(req)
	}

	return systemPrompt, userPrompt, allocation, nil
}

// buildSimpleUserPrompt builds a basic user prompt without RAG context
func (pb *PromptBuilder) buildSimpleUserPrompt(req *SQLRequest) string {
	var parts []string

	if req.Schema != "" {
		parts = append(parts, "## Database Schema\n\n"+req.Schema)
	}

	parts = append(parts, "## Request\n\n"+req.Prompt)

	parts = append(parts, "\nProvide your response in JSON format with: query, explanation, confidence, suggestions, and warnings.")

	return strings.Join(parts, "\n\n")
}

// buildSimpleFixPrompt builds a basic fix prompt without RAG context
func (pb *PromptBuilder) buildSimpleFixPrompt(req *SQLRequest) string {
	var parts []string

	if req.Schema != "" {
		parts = append(parts, "## Database Schema\n\n"+req.Schema)
	}

	parts = append(parts, "## Error Message\n\n"+req.Error)
	parts = append(parts, "## Broken Query\n\n```sql\n"+req.Query+"\n```")

	if req.Prompt != "" {
		parts = append(parts, "## Additional Context\n\n"+req.Prompt)
	}

	parts = append(parts, "\nProvide the fixed SQL in JSON format with: query, explanation, confidence, suggestions, and warnings.")

	return strings.Join(parts, "\n\n")
}

// formatUserPromptWithContext formats user prompt with RAG context
func (pb *PromptBuilder) formatUserPromptWithContext(
	req *SQLRequest,
	queryContext *rag.QueryContext,
	allocation *rag.BudgetAllocation,
) string {
	var parts []string

	// Add RAG context
	ragContext := rag.FormatContextForPrompt(queryContext, allocation)
	if ragContext != "" {
		parts = append(parts, ragContext)
	}

	// Add any explicit schema from request
	if req.Schema != "" {
		parts = append(parts, "## Additional Schema Information\n\n"+req.Schema)
	}

	// Add user request
	parts = append(parts, "## User Request\n\n"+req.Prompt)

	// Add response format instruction
	parts = append(parts, "\nProvide your response in the following JSON format:\n"+
		"```json\n"+
		"{\n"+
		`  "query": "The generated SQL query",`+"\n"+
		`  "explanation": "Brief explanation of the query",`+"\n"+
		`  "confidence": 0.95,`+"\n"+
		`  "suggestions": ["Optional improvements"],`+"\n"+
		`  "warnings": ["Any caveats"]`+"\n"+
		"}\n"+
		"```")

	return strings.Join(parts, "\n\n")
}

// formatFixPromptWithContext formats fix prompt with RAG context
func (pb *PromptBuilder) formatFixPromptWithContext(
	req *SQLRequest,
	queryContext *rag.QueryContext,
	allocation *rag.BudgetAllocation,
) string {
	var parts []string

	// Add RAG context (especially examples of correct queries)
	ragContext := rag.FormatContextForPrompt(queryContext, allocation)
	if ragContext != "" {
		parts = append(parts, ragContext)
	}

	// Add any explicit schema from request
	if req.Schema != "" {
		parts = append(parts, "## Additional Schema Information\n\n"+req.Schema)
	}

	// Add error and broken query
	parts = append(parts, "## Error Message\n\n"+req.Error)
	parts = append(parts, "## Broken Query\n\n```sql\n"+req.Query+"\n```")

	// Add any additional context
	if req.Prompt != "" {
		parts = append(parts, "## Additional Context\n\n"+req.Prompt)
	}

	// Add response format instruction
	parts = append(parts, "\nProvide the fixed SQL in the following JSON format:\n"+
		"```json\n"+
		"{\n"+
		`  "query": "The corrected SQL query",`+"\n"+
		`  "explanation": "What was wrong and how it was fixed",`+"\n"+
		`  "confidence": 0.90,`+"\n"+
		`  "suggestions": ["Additional improvements"],`+"\n"+
		`  "warnings": ["Potential issues to watch for"]`+"\n"+
		"}\n"+
		"```")

	return strings.Join(parts, "\n\n")
}

// ParseSQLResponse parses the AI response into SQLResponse
func ParseSQLResponse(content string, provider Provider, model string) (*SQLResponse, error) {
	// Try to extract JSON from the response
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	// Parse JSON
	var parsed struct {
		Query       string   `json:"query"`
		Explanation string   `json:"explanation"`
		Confidence  float64  `json:"confidence"`
		Suggestions []string `json:"suggestions"`
		Warnings    []string `json:"warnings"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Validate required fields
	if parsed.Query == "" {
		return nil, fmt.Errorf("response missing required 'query' field")
	}

	// Set defaults
	if parsed.Confidence == 0 {
		parsed.Confidence = 0.5 // Default confidence if not provided
	}

	if parsed.Suggestions == nil {
		parsed.Suggestions = []string{}
	}

	if parsed.Warnings == nil {
		parsed.Warnings = []string{}
	}

	return &SQLResponse{
		Query:       strings.TrimSpace(parsed.Query),
		Explanation: parsed.Explanation,
		Confidence:  parsed.Confidence,
		Suggestions: parsed.Suggestions,
		Warnings:    parsed.Warnings,
		Provider:    provider,
		Model:       model,
	}, nil
}

// extractJSON extracts JSON from text that might contain markdown code blocks or other formatting
func extractJSON(text string) string {
	// Try to find JSON in markdown code blocks
	if strings.Contains(text, "```json") {
		start := strings.Index(text, "```json")
		if start != -1 {
			start += 7 // Skip "```json\n"
			end := strings.Index(text[start:], "```")
			if end != -1 {
				return strings.TrimSpace(text[start : start+end])
			}
		}
	}

	// Try to find JSON in plain code blocks
	if strings.Contains(text, "```") {
		start := strings.Index(text, "```")
		if start != -1 {
			start += 3 // Skip "```"
			// Skip to newline
			for start < len(text) && text[start] != '\n' {
				start++
			}
			if start < len(text) {
				start++ // Skip newline
				end := strings.Index(text[start:], "```")
				if end != -1 {
					candidate := strings.TrimSpace(text[start : start+end])
					if isLikelyJSON(candidate) {
						return candidate
					}
				}
			}
		}
	}

	// Try to find raw JSON by looking for { ... }
	start := strings.Index(text, "{")
	if start != -1 {
		// Find matching closing brace
		depth := 0
		for i := start; i < len(text); i++ {
			if text[i] == '{' {
				depth++
			} else if text[i] == '}' {
				depth--
				if depth == 0 {
					candidate := strings.TrimSpace(text[start : i+1])
					if isLikelyJSON(candidate) {
						return candidate
					}
					break
				}
			}
		}
	}

	// If nothing else works, try the whole text
	trimmed := strings.TrimSpace(text)
	if isLikelyJSON(trimmed) {
		return trimmed
	}

	return ""
}

// isLikelyJSON checks if a string is likely valid JSON
func isLikelyJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return false
	}
	// Quick validation - try to parse
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

// GetRecommendedTokenBudget returns recommended token budget based on model
func GetRecommendedTokenBudget(model string) int {
	modelLower := strings.ToLower(model)

	// Claude models
	if strings.Contains(modelLower, "claude-3-opus") {
		return 200000 // Claude 3 Opus has 200k context
	}
	if strings.Contains(modelLower, "claude-3-sonnet") {
		return 200000 // Claude 3 Sonnet has 200k context
	}
	if strings.Contains(modelLower, "claude-3-haiku") {
		return 200000 // Claude 3 Haiku has 200k context
	}
	if strings.Contains(modelLower, "claude-2") {
		return 100000 // Claude 2 has 100k context
	}

	// GPT-4 models
	if strings.Contains(modelLower, "gpt-4-turbo") {
		return 128000 // GPT-4 Turbo has 128k context
	}
	if strings.Contains(modelLower, "gpt-4-32k") {
		return 32000 // GPT-4-32k has 32k context
	}
	if strings.Contains(modelLower, "gpt-4") {
		return 8192 // Standard GPT-4 has 8k context
	}

	// GPT-3.5 models
	if strings.Contains(modelLower, "gpt-3.5-turbo-16k") {
		return 16384 // GPT-3.5-turbo-16k
	}
	if strings.Contains(modelLower, "gpt-3.5") {
		return 4096 // Standard GPT-3.5-turbo
	}

	// Default conservative budget
	return 4096
}

// DetectDialectFromConnection detects SQL dialect from connection metadata
func DetectDialectFromConnection(connectionType string, metadata map[string]string) SQLDialect {
	// First try direct detection from connection type
	dialect := DetectDialect(connectionType)
	if dialect != DialectGeneric {
		return dialect
	}

	// Try to detect from metadata
	if metadata != nil {
		if dbType, ok := metadata["database_type"]; ok {
			dialect = DetectDialect(dbType)
			if dialect != DialectGeneric {
				return dialect
			}
		}

		if driver, ok := metadata["driver"]; ok {
			dialect = DetectDialect(driver)
			if dialect != DialectGeneric {
				return dialect
			}
		}
	}

	return DialectGeneric
}
