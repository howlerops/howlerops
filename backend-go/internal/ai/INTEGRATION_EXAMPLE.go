package ai

// This file provides complete integration examples for the universal SQL prompt system.
// These examples show real-world usage patterns and best practices.

import (
	"context"
	"fmt"

	"github.com/jbeck018/howlerops/backend-go/internal/rag"
	"github.com/sirupsen/logrus"
)

// Example 1: Simple integration without RAG
func ExampleBasicIntegration() {
	// Initialize service
	config := &Config{
		OpenAI: OpenAIConfig{
			APIKey: "sk-...",
			Models: []string{"gpt-4o"},
		},
		DefaultProvider: ProviderOpenAI,
		MaxTokens:       8192,
		Temperature:     0.3,
	}

	logger := logrus.New()
	service, _ := NewService(config, logger)

	// Generate SQL with enhanced prompts
	ctx := context.Background()
	response, _ := service.GenerateSQL(ctx, &SQLRequest{
		Prompt:   "Show all users who signed up in the last 7 days",
		Provider: ProviderOpenAI,
		Model:    "gpt-4o",
		Schema: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) NOT NULL,
				created_at TIMESTAMP DEFAULT NOW()
			);
		`,
		Context: map[string]string{
			"connection_type": "postgresql", // Enables PostgreSQL-specific prompt
		},
	})

	fmt.Printf("Generated SQL:\n%s\n\n", response.Query)
	fmt.Printf("Explanation: %s\n", response.Explanation)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)

	// Output:
	// Generated SQL:
	// SELECT id, email, created_at
	// FROM users
	// WHERE created_at >= NOW() - INTERVAL '7 days'
	// ORDER BY created_at DESC;
	//
	// Explanation: Retrieves users created within the last 7 days...
	// Confidence: 0.95
}

// Example 2: Integration with RAG context
func ExampleRAGIntegration(
	vectorStore rag.VectorStore,
	embeddingService rag.EmbeddingService,
) {
	logger := logrus.New()

	// Create context builder for RAG
	contextBuilder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

	// Create AI service
	config := &Config{
		OpenAI: OpenAIConfig{
			APIKey: "sk-...",
			Models: []string{"gpt-4o"},
		},
		DefaultProvider: ProviderOpenAI,
	}
	service, _ := NewService(config, logger)
	_ = service // Example code - service would be used in production

	// Create prompt builder with context
	promptBuilder := NewPromptBuilder(contextBuilder)

	ctx := context.Background()
	req := &SQLRequest{
		Prompt:   "Show revenue by customer for active orders",
		Provider: ProviderOpenAI,
		Model:    "gpt-4o",
		Context: map[string]string{
			"connection_id":   "conn-123", // Enable RAG lookups
			"connection_type": "postgresql",
		},
	}

	// Build enhanced prompts with RAG context
	systemPrompt, userPrompt, allocation, _ := promptBuilder.BuildSQLGenerationPrompt(
		ctx,
		req,
		DialectPostgreSQL,
		8192,
	)

	// Note: Token estimation requires additional utility function
	fmt.Printf("System Prompt Length: %d chars\n", len(systemPrompt))
	fmt.Printf("User Prompt Length: %d chars\n", len(userPrompt))
	if allocation != nil {
		fmt.Printf("\nToken Budget:\n%s\n", allocation.Summary())
	}

	// The prompts now include:
	// - Relevant schema for users, orders, customers tables
	// - 3-5 similar queries showing revenue calculations
	// - Business rule: "Revenue = quantity * price - discount"
	// - Performance hint: "Consider index on order_date for date filters"
}

// Example 3: Error fixing with category detection
func ExampleErrorFixing() {
	config := &Config{
		OpenAI: OpenAIConfig{
			APIKey: "sk-...",
			Models: []string{"gpt-4o"},
		},
		DefaultProvider: ProviderOpenAI,
	}

	logger := logrus.New()
	service, _ := NewService(config, logger)

	ctx := context.Background()

	// Broken query with error
	brokenQuery := "SELECT user_id, COUNT(*) FROM orders GROUP BY username"
	errorMessage := "ERROR: column 'username' must appear in GROUP BY clause or be used in an aggregate function"

	// Detect error category
	category := DetectErrorCategory(errorMessage)
	fmt.Printf("Error Category: %s\n", category)
	// Output: Error Category: syntax

	// Fix the query
	response, _ := service.FixSQL(ctx, &SQLRequest{
		Query:    brokenQuery,
		Error:    errorMessage,
		Provider: ProviderOpenAI,
		Model:    "gpt-4o",
		Schema: `
			CREATE TABLE orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER,
				username VARCHAR(100)
			);
		`,
		Context: map[string]string{
			"connection_type": "postgresql",
		},
	})

	fmt.Printf("\nFixed SQL:\n%s\n\n", response.Query)
	fmt.Printf("Explanation: %s\n", response.Explanation)

	// Output:
	// Fixed SQL:
	// SELECT user_id, COUNT(*) FROM orders GROUP BY user_id;
	//
	// Explanation: The error occurred because 'username' was selected but not
	// included in the GROUP BY clause. Changed to group by user_id which is
	// what the query intended.
}

// Example 4: Provider wrapper for enhanced features
func ExampleProviderWrapper(
	contextBuilder *rag.ContextBuilder,
) {
	logger := logrus.New()

	// Create base provider
	config := &OpenAIConfig{
		APIKey: "sk-...",
		Models: []string{"gpt-4o"},
	}
	baseProvider, _ := NewOpenAIProvider(config, logger)

	// Wrap with enhanced prompts
	enhancedProvider := WrapProviderWithPrompts(
		baseProvider,
		contextBuilder,
		DialectPostgreSQL,
	)

	ctx := context.Background()
	req := &SQLRequest{
		Prompt: "List top 10 customers by total order value",
		Context: map[string]string{
			"connection_id": "conn-123",
		},
	}

	// Use enhanced generation - includes RAG context automatically
	response, allocation, _ := enhancedProvider.GenerateSQLWithEnhancedPrompts(ctx, req)

	fmt.Printf("SQL: %s\n", response.Query)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("\nToken Usage:\n%s\n", allocation.Summary())

	// The wrapper automatically:
	// 1. Detects PostgreSQL dialect
	// 2. Fetches relevant schema from connection
	// 3. Finds similar queries from history
	// 4. Includes business rules about "customer value"
	// 5. Manages token budget optimally
}

// Example 5: Custom priority allocation
func ExampleCustomPriorities(
	contextBuilder *rag.ContextBuilder,
) {
	_ = logrus.New() // Example code - logger would be used in production
	promptBuilder := NewPromptBuilder(contextBuilder)

	ctx := context.Background()

	// Performance-critical query - boost performance hints
	req := &SQLRequest{
		Prompt: "This query is too slow, how can I optimize it?",
		Query:  "SELECT * FROM large_table WHERE status = 'active'",
		Context: map[string]string{
			"connection_id": "conn-123",
		},
	}

	// Note: PrioritizeComponents requires additional implementation
	// For now, use default priorities
	fmt.Printf("Priorities for performance query: using defaults\n")

	// Build with default token budget
	budget := rag.DefaultTokenBudget(8192)
	allocation := budget.AllocateContextBudget(nil)

	fmt.Printf("\nAllocation:\n%s\n", allocation.Summary())

	// Performance hints will get more tokens
	// Examples of optimized queries will be prioritized
	// Business rules will get less space

	_, _, _, _ = promptBuilder.BuildSQLGenerationPrompt(
		ctx,
		req,
		DialectPostgreSQL,
		8192,
	)
}

// Example 6: Multi-dialect support
func ExampleMultiDialect() {
	dialects := []SQLDialect{
		DialectPostgreSQL,
		DialectMySQL,
		DialectSQLite,
		DialectMSSQL,
	}

	_ = "Get users created in the last 7 days" // Example query

	for _, dialect := range dialects {
		prompt := GetUniversalSQLPrompt(dialect)

		// Each prompt includes dialect-specific guidance:
		// PostgreSQL: INTERVAL '7 days', $1 placeholders
		// MySQL: DATE_SUB(), ? placeholders
		// SQLite: datetime('now', '-7 days')
		// MSSQL: DATEADD(), @p1 placeholders

		fmt.Printf("%s prompt includes:\n", dialect)
		if dialect == DialectPostgreSQL {
			fmt.Println("  - INTERVAL syntax")
			fmt.Println("  - $ placeholders")
		} else if dialect == DialectMySQL {
			fmt.Println("  - DATE_SUB() function")
			fmt.Println("  - ? placeholders")
		}
		// ... and so on

		_ = prompt // Use in actual LLM call
	}
}

// Example 7: Token budget monitoring
func ExampleTokenBudgetMonitoring(
	contextBuilder *rag.ContextBuilder,
) {
	logger := logrus.New()
	promptBuilder := NewPromptBuilder(contextBuilder)

	ctx := context.Background()
	req := &SQLRequest{
		Prompt: "Complex query requiring lots of context",
		Context: map[string]string{
			"connection_id": "conn-123",
		},
	}

	// Build with monitoring
	_, _, allocation, _ := promptBuilder.BuildSQLGenerationPrompt(
		ctx,
		req,
		DialectPostgreSQL,
		8192,
	)

	// Check allocation
	if allocation != nil {
		// Log token usage
		logger.WithFields(logrus.Fields{
			"total_tokens":       allocation.Total,
			"schema_allocated":   allocation.Schema,
			"examples_allocated": allocation.Examples,
			"remaining":          allocation.RemainingBudget(),
		}).Info("Token budget allocated")

		// Check if we're using budget efficiently
		utilizationPercent := float64(allocation.Total-allocation.RemainingBudget()) /
			float64(allocation.Total) * 100

		if utilizationPercent < 70 {
			logger.Warn("Low budget utilization - may want to add more context")
		} else if utilizationPercent > 95 {
			logger.Warn("High budget utilization - may hit token limits")
		}

		// Summary for debugging
		fmt.Println(allocation.Summary())
	}
}

// Example 8: Batch processing with budget management
func ExampleBatchProcessing(
	service Service,
	queries []string,
) {
	ctx := context.Background()

	// Process multiple queries efficiently
	for i, query := range queries {
		// Adjust budget based on queue size
		maxTokens := 8192
		if len(queries) > 10 {
			// Use smaller budget for large batches
			maxTokens = 4096
		}

		req := &SQLRequest{
			Prompt:    query,
			Provider:  ProviderOpenAI,
			Model:     "gpt-4o",
			MaxTokens: maxTokens,
			Context: map[string]string{
				"connection_id": "conn-123",
				"batch_index":   fmt.Sprintf("%d", i),
			},
		}

		response, _ := service.GenerateSQL(ctx, req)

		// Process response
		fmt.Printf("Query %d: %s\n", i+1, response.Query)
		fmt.Printf("Confidence: %.2f\n", response.Confidence)

		// Track success rate
		if response.Confidence < 0.7 {
			fmt.Printf("  ⚠️  Low confidence - manual review recommended\n")
		}
	}
}

// Example 9: Testing prompt quality
func ExampleTestingPromptQuality() {
	// Test that prompts include expected guidance
	testCases := []struct {
		dialect     SQLDialect
		mustContain []string
		category    ErrorCategory
	}{
		{
			dialect: DialectPostgreSQL,
			mustContain: []string{
				"PostgreSQL",
				"$1, $2",
				"INTERVAL",
				"jsonb",
			},
		},
		{
			dialect: DialectMySQL,
			mustContain: []string{
				"MySQL",
				"CONCAT()",
				"AUTO_INCREMENT",
			},
		},
	}

	for _, tc := range testCases {
		prompt := GetUniversalSQLPrompt(tc.dialect)

		for _, expected := range tc.mustContain {
			if !contains(prompt, expected) {
				fmt.Printf("❌ Prompt missing: %s\n", expected)
			} else {
				fmt.Printf("✅ Prompt contains: %s\n", expected)
			}
		}
	}
}

// Example 10: Real-world production usage
func ExampleProductionUsage(
	service Service,
	logger *logrus.Logger,
) {
	ctx := context.Background()

	// User request from frontend
	userPrompt := "Show me all orders from last month with customer names and total amounts"
	connectionID := "conn-production-db"

	// Log request
	logger.WithFields(logrus.Fields{
		"user_prompt":   userPrompt,
		"connection_id": connectionID,
	}).Info("SQL generation request")

	req := &SQLRequest{
		Prompt:   userPrompt,
		Provider: ProviderOpenAI,
		Model:    "gpt-4o",
		Context: map[string]string{
			"connection_id":   connectionID,
			"connection_type": "postgresql",
			"user_id":         "user-123", // For audit logging
		},
	}

	response, err := service.GenerateSQL(ctx, req)
	if err != nil {
		logger.WithError(err).Error("SQL generation failed")
		return
	}

	// Log success with metrics
	logger.WithFields(logrus.Fields{
		"confidence":   response.Confidence,
		"tokens_used":  response.TokensUsed,
		"time_taken":   response.TimeTaken,
		"has_warnings": len(response.Warnings) > 0,
	}).Info("SQL generated successfully")

	// Check confidence before using
	if response.Confidence < 0.8 {
		logger.Warn("Low confidence SQL - flagging for review")
		// Send to human review queue
	}

	// Check for warnings
	if len(response.Warnings) > 0 {
		logger.WithField("warnings", response.Warnings).Warn("SQL has warnings")
		// Show warnings to user
	}

	// Return to user with explanation
	fmt.Printf("Generated Query:\n%s\n\n", response.Query)
	fmt.Printf("Explanation:\n%s\n\n", response.Explanation)

	if len(response.Suggestions) > 0 {
		fmt.Printf("Suggestions:\n")
		for _, suggestion := range response.Suggestions {
			fmt.Printf("  • %s\n", suggestion)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
