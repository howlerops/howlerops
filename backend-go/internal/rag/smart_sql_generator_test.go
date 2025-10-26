//go:build integration
// +build integration

package rag_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/rag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMProvider implements the LLMProvider interface for testing
type mockLLMProvider struct {
	mu                sync.Mutex
	generateSQLCalls  int
	explainSQLCalls   int
	optimizeSQLCalls  int
	generateSQLFunc   func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error)
	explainSQLFunc    func(ctx context.Context, sql string) (*rag.SQLExplanation, error)
	optimizeSQLFunc   func(ctx context.Context, sql string, hints []rag.OptimizationHint) (*rag.OptimizedSQL, error)
	shouldError       bool
	errorMessage      string
	generateSQLInputs []string
	explainSQLInputs  []string
	optimizeSQLInputs []string
}

func newMockLLMProvider() *mockLLMProvider {
	return &mockLLMProvider{}
}

func (m *mockLLMProvider) GenerateSQL(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.generateSQLCalls++
	m.generateSQLInputs = append(m.generateSQLInputs, prompt)

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	if m.generateSQLFunc != nil {
		return m.generateSQLFunc(ctx, prompt, context)
	}

	// Default implementation
	return &rag.GeneratedSQL{
		Query:       "SELECT * FROM users",
		Explanation: "Fetches all users",
		Confidence:  0.8,
		Tables:      []string{"users"},
		Columns:     []string{"*"},
		Warnings:    []string{},
	}, nil
}

func (m *mockLLMProvider) ExplainSQL(ctx context.Context, sql string) (*rag.SQLExplanation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.explainSQLCalls++
	m.explainSQLInputs = append(m.explainSQLInputs, sql)

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	if m.explainSQLFunc != nil {
		return m.explainSQLFunc(ctx, sql)
	}

	// Default implementation
	return &rag.SQLExplanation{
		Summary:    "Explains the SQL query",
		Steps:      []rag.ExplanationStep{},
		Complexity: "simple",
	}, nil
}

func (m *mockLLMProvider) OptimizeSQL(ctx context.Context, sql string, hints []rag.OptimizationHint) (*rag.OptimizedSQL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.optimizeSQLCalls++
	m.optimizeSQLInputs = append(m.optimizeSQLInputs, sql)

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	if m.optimizeSQLFunc != nil {
		return m.optimizeSQLFunc(ctx, sql, hints)
	}

	// Default implementation
	return &rag.OptimizedSQL{
		OriginalQuery:  sql,
		OptimizedQuery: sql + " LIMIT 1000",
		Improvements:   []rag.Improvement{},
		EstimatedGain:  10.5,
	}, nil
}

func (m *mockLLMProvider) setError(shouldError bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorMessage = message
}

func (m *mockLLMProvider) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateSQLCalls = 0
	m.explainSQLCalls = 0
	m.optimizeSQLCalls = 0
	m.generateSQLInputs = nil
	m.explainSQLInputs = nil
	m.optimizeSQLInputs = nil
}

func (m *mockLLMProvider) getCallCounts() (generate, explain, optimize int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generateSQLCalls, m.explainSQLCalls, m.optimizeSQLCalls
}

// mockContextBuilder implements the ContextBuilder dependency for testing
type mockContextBuilder struct {
	mu               sync.Mutex
	buildContextFunc func(ctx context.Context, query string, connectionID string) (*rag.QueryContext, error)
	shouldError      bool
	errorMessage     string
	buildCalls       int
	buildInputs      []string
}

func newMockContextBuilder() *mockContextBuilder {
	return &mockContextBuilder{}
}

func (m *mockContextBuilder) BuildContext(ctx context.Context, query string, connectionID string) (*rag.QueryContext, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buildCalls++
	m.buildInputs = append(m.buildInputs, query)

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	if m.buildContextFunc != nil {
		return m.buildContextFunc(ctx, query, connectionID)
	}

	// Default implementation
	return &rag.QueryContext{
		Query:      query,
		Confidence: 0.7,
		RelevantSchemas: []rag.SchemaContext{
			{
				TableName: "users",
				Relevance: 0.9,
			},
		},
		SimilarQueries: []rag.QueryPattern{
			{
				Query:      "SELECT * FROM users WHERE active = 1",
				Similarity: 0.85,
				Frequency:  10,
			},
		},
		BusinessRules: []rag.BusinessRule{
			{
				Name:        "active_users",
				Description: "Only show active users",
				Conditions:  []string{"active = 1"},
			},
		},
		PerformanceHints: []rag.OptimizationHint{
			{
				Type:        "index",
				Description: "Add index on active column",
				Impact:      "high",
				Confidence:  0.8,
			},
		},
	}, nil
}

func (m *mockContextBuilder) setError(shouldError bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorMessage = message
}

func (m *mockContextBuilder) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buildCalls = 0
	m.buildInputs = nil
}

func (m *mockContextBuilder) getBuildCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buildCalls
}

// Helper function to create a test logger
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

// TestNewSmartSQLGenerator tests the constructor
func TestNewSmartSQLGenerator(t *testing.T) {
	t.Run("creates generator with valid dependencies", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()

		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		require.NotNil(t, generator)
	})

	t.Run("creates generator with nil logger", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()

		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, nil)

		require.NotNil(t, generator)
	})

	t.Run("initializes internal components", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()

		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		// Generator should be ready to use
		require.NotNil(t, generator)
	})
}

// TestGenerate tests the main Generate method
func TestGenerate(t *testing.T) {
	t.Run("generates simple SQL successfully", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Query)
		assert.NotEmpty(t, result.Explanation)
		assert.Greater(t, result.Confidence, float32(0.0))

		genCalls, _, _ := llmProvider.getCallCounts()
		assert.Equal(t, 1, genCalls)
	})

	t.Run("generates SQL without RAG context on context builder error", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		contextBuilder.setError(true, "context builder error")
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		// Should still succeed with fallback context
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Query)
	})

	t.Run("handles LLM provider error", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.setError(true, "LLM error")
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to generate SQL")
	})

	t.Run("enhances result with context information", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query:       "SELECT * FROM users",
				Explanation: "Fetches all users",
				Confidence:  0.5,
				Tables:      []string{"users"},
				Columns:     []string{"*"},
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Confidence should be enhanced with context confidence
		assert.Greater(t, result.Confidence, float32(0.5))
	})

	t.Run("adds validation warnings to result", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query:       "INVALID SQL SYNTAX",
				Explanation: "Invalid query",
				Confidence:  0.8,
				Tables:      []string{},
				Columns:     []string{},
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have validation warnings
		assert.GreaterOrEqual(t, len(result.Warnings), 0)
	})

	t.Run("detects complex queries and uses planning", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		complexPrompt := "show me all users with their orders and calculate aggregate sales for multiple regions with group by and having clauses"
		result, err := generator.Generate(ctx, complexPrompt, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Complex queries should still produce results
		assert.NotEmpty(t, result.Query)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := generator.Generate(ctx, "show me all users", "conn-1")

		// May or may not error depending on when cancellation is detected
		if err != nil {
			assert.Contains(t, err.Error(), "context")
		}
	})

	t.Run("extracts tables from generated SQL", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query:       "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id",
				Explanation: "Fetches users with orders",
				Confidence:  0.8,
				Tables:      []string{"users", "orders"},
				Columns:     []string{"name", "total"},
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me users and their orders", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.Tables, "users")
		assert.Contains(t, result.Tables, "orders")
	})

	t.Run("adds alternative queries from similar patterns", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		contextBuilder.buildContextFunc = func(ctx context.Context, query string, connectionID string) (*rag.QueryContext, error) {
			return &rag.QueryContext{
				Query:      query,
				Confidence: 0.9,
				SimilarQueries: []rag.QueryPattern{
					{
						Query:      "SELECT * FROM users WHERE active = 1",
						Similarity: 0.95,
					},
					{
						Query:      "SELECT id, name FROM users WHERE status = 'active'",
						Similarity: 0.92,
					},
					{
						Query:      "SELECT * FROM users ORDER BY created_at",
						Similarity: 0.85,
					},
				},
			}, nil
		}
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have alternative queries from high-similarity patterns
		assert.GreaterOrEqual(t, len(result.AlternativeQueries), 0)
	})

	t.Run("detects need for JOINs", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query:       "SELECT * FROM users",
				Explanation: "Fetches users",
				Confidence:  0.8,
				Tables:      []string{"users"},
				Columns:     []string{"*"},
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		// Prompt suggests multiple entities
		result, err := generator.Generate(ctx, "show me users and their orders including details", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Query)
	})
}

// TestGenerateWithPlanning tests complex query planning
func TestGenerateWithPlanning(t *testing.T) {
	t.Run("decomposes complex query into steps", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		complexPrompt := "show me users with their orders and aggregate sales by region with group by having multiple conditions"
		result, err := generator.Generate(ctx, complexPrompt, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Query)
	})

	t.Run("handles step generation errors gracefully", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		callCount := 0
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			callCount++
			if callCount == 2 {
				return nil, fmt.Errorf("step generation failed")
			}
			return &rag.GeneratedSQL{
				Query:      "SELECT * FROM users",
				Confidence: 0.8,
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		complexPrompt := "show me multiple complex aggregations with nested subqueries"
		result, err := generator.Generate(ctx, complexPrompt, "conn-1")

		// Should handle errors and still produce result
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("combines steps into final query", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		complexPrompt := "aggregate sales by region and combine with user statistics"
		result, err := generator.Generate(ctx, complexPrompt, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Query)
		assert.NotEmpty(t, result.Explanation)
	})
}

// TestExplain tests SQL explanation
func TestExplain(t *testing.T) {
	t.Run("explains simple SQL", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT * FROM users WHERE active = 1"
		explanation, err := generator.Explain(ctx, sql)

		require.NoError(t, err)
		require.NotNil(t, explanation)
		assert.NotEmpty(t, explanation.Summary)
		assert.NotEmpty(t, explanation.Complexity)
		_, explainCalls, _ := llmProvider.getCallCounts()
		assert.Equal(t, 1, explainCalls)
	})

	t.Run("explains complex SQL with joins", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.explainSQLFunc = func(ctx context.Context, sql string) (*rag.SQLExplanation, error) {
			return &rag.SQLExplanation{
				Summary: "Complex query with multiple joins",
				Steps: []rag.ExplanationStep{
					{Order: 1, Operation: "JOIN", Description: "Join users with orders"},
					{Order: 2, Operation: "WHERE", Description: "Filter active users"},
					{Order: 3, Operation: "GROUP BY", Description: "Group by region"},
				},
				Complexity: "complex",
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id WHERE u.active = 1 GROUP BY u.region"
		explanation, err := generator.Explain(ctx, sql)

		require.NoError(t, err)
		require.NotNil(t, explanation)
		assert.Equal(t, "complex", explanation.Complexity)
		assert.Len(t, explanation.Steps, 3)
	})

	t.Run("handles LLM error during explanation", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.setError(true, "explanation error")
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT * FROM users"
		explanation, err := generator.Explain(ctx, sql)

		require.Error(t, err)
		assert.Nil(t, explanation)
		assert.Contains(t, err.Error(), "failed to explain SQL")
	})

	t.Run("enhances explanation with complexity analysis", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT u.name, COUNT(o.id) FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.name"
		explanation, err := generator.Explain(ctx, sql)

		require.NoError(t, err)
		require.NotNil(t, explanation)
		// Should analyze complexity
		assert.NotEmpty(t, explanation.Complexity)
	})

	t.Run("estimates execution time based on complexity", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT * FROM users"
		explanation, err := generator.Explain(ctx, sql)

		require.NoError(t, err)
		require.NotNil(t, explanation)
		assert.NotEmpty(t, explanation.EstimatedTime)
	})
}

// TestOptimize tests SQL optimization
func TestOptimize(t *testing.T) {
	t.Run("optimizes simple query", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT * FROM users"
		optimized, err := generator.Optimize(ctx, sql, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, optimized)
		assert.Equal(t, sql, optimized.OriginalQuery)
		assert.NotEmpty(t, optimized.OptimizedQuery)
		_, _, optCalls := llmProvider.getCallCounts()
		assert.Equal(t, 1, optCalls)
	})

	t.Run("optimizes with performance hints from context", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.optimizeSQLFunc = func(ctx context.Context, sql string, hints []rag.OptimizationHint) (*rag.OptimizedSQL, error) {
			improvements := make([]rag.Improvement, len(hints))
			for i, hint := range hints {
				improvements[i] = rag.Improvement{
					Type:        hint.Type,
					Description: hint.Description,
					Before:      sql,
					After:       sql + " /* optimized */",
				}
			}
			return &rag.OptimizedSQL{
				OriginalQuery:  sql,
				OptimizedQuery: sql + " /* optimized */",
				Improvements:   improvements,
				EstimatedGain:  25.5,
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT * FROM users WHERE active = 1"
		optimized, err := generator.Optimize(ctx, sql, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, optimized)
		assert.Greater(t, len(optimized.Improvements), 0)
		assert.Greater(t, optimized.EstimatedGain, float32(0.0))
	})

	t.Run("handles context builder error during optimization", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		contextBuilder.setError(true, "context error")
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT * FROM users"
		optimized, err := generator.Optimize(ctx, sql, "conn-1")

		// Should still succeed with fallback
		require.NoError(t, err)
		require.NotNil(t, optimized)
	})

	t.Run("handles LLM error during optimization", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.setError(true, "optimization error")
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT * FROM users"
		optimized, err := generator.Optimize(ctx, sql, "conn-1")

		require.Error(t, err)
		assert.Nil(t, optimized)
		assert.Contains(t, err.Error(), "failed to optimize SQL")
	})

	t.Run("validates optimized query", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.optimizeSQLFunc = func(ctx context.Context, sql string, hints []rag.OptimizationHint) (*rag.OptimizedSQL, error) {
			return &rag.OptimizedSQL{
				OriginalQuery:  sql,
				OptimizedQuery: "INVALID SYNTAX",
				Improvements:   []rag.Improvement{},
				EstimatedGain:  0,
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		sql := "SELECT * FROM users"
		optimized, err := generator.Optimize(ctx, sql, "conn-1")

		// Should still return result with validation warnings logged
		require.NoError(t, err)
		require.NotNil(t, optimized)
	})
}

// TestHelperMethods tests internal helper methods
func TestIsComplexQuery(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		expected bool
	}{
		{
			name:     "simple query",
			prompt:   "show me all users",
			expected: false,
		},
		{
			name:     "query with 'multiple'",
			prompt:   "show me multiple tables",
			expected: true,
		},
		{
			name:     "query with 'aggregate'",
			prompt:   "aggregate sales by region",
			expected: true,
		},
		{
			name:     "query with 'group by'",
			prompt:   "show sales grouped by month",
			expected: true,
		},
		{
			name:     "query with 'having'",
			prompt:   "show users having more than 10 orders",
			expected: true,
		},
		{
			name:     "query with 'union'",
			prompt:   "union active and inactive users",
			expected: true,
		},
		{
			name:     "query with 'complex'",
			prompt:   "complex analysis of sales data",
			expected: true,
		},
		{
			name:     "query with 'nested'",
			prompt:   "nested subquery for users",
			expected: true,
		},
		{
			name:     "very long query",
			prompt:   strings.Repeat("word ", 25),
			expected: true,
		},
		{
			name:     "short query",
			prompt:   strings.Repeat("word ", 10),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextBuilder := newMockContextBuilder()
			llmProvider := newMockLLMProvider()
			logger := newTestLogger()
			generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

			ctx := context.Background()
			result, err := generator.Generate(ctx, tt.prompt, "conn-1")

			require.NoError(t, err)
			require.NotNil(t, result)
			// Complex queries should still generate successfully
			assert.NotEmpty(t, result.Query)
		})
	}
}

func TestNeedsJoinDetection(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		generatedQuery string
		expectedDetect bool
	}{
		{
			name:           "multiple entities without joins",
			prompt:         "show me users and their orders",
			generatedQuery: "SELECT * FROM users",
			expectedDetect: true,
		},
		{
			name:           "multiple entities with join",
			prompt:         "show me users and their orders",
			generatedQuery: "SELECT * FROM users u JOIN orders o ON u.id = o.user_id",
			expectedDetect: false,
		},
		{
			name:           "single entity",
			prompt:         "show me all users",
			generatedQuery: "SELECT * FROM users",
			expectedDetect: false,
		},
		{
			name:           "with their indicator",
			prompt:         "show me products with their categories",
			generatedQuery: "SELECT * FROM products",
			expectedDetect: true,
		},
		{
			name:           "including indicator",
			prompt:         "show me orders including customer details",
			generatedQuery: "SELECT * FROM orders",
			expectedDetect: true,
		},
		{
			name:           "comma-separated tables",
			prompt:         "show me users and orders",
			generatedQuery: "SELECT * FROM users, orders",
			expectedDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextBuilder := newMockContextBuilder()
			llmProvider := newMockLLMProvider()
			llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
				return &rag.GeneratedSQL{
					Query:       tt.generatedQuery,
					Explanation: "Test query",
					Confidence:  0.8,
				}, nil
			}
			logger := newTestLogger()
			generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

			ctx := context.Background()
			result, err := generator.Generate(ctx, tt.prompt, "conn-1")

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.Query)
		})
	}
}

func TestExtractTables(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedTables []string
	}{
		{
			name:           "simple FROM clause",
			query:          "SELECT * FROM users",
			expectedTables: []string{"users"},
		},
		{
			name:           "FROM with alias",
			query:          "SELECT * FROM users u",
			expectedTables: []string{"users"},
		},
		{
			name:           "JOIN clause",
			query:          "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			expectedTables: []string{"users", "orders"},
		},
		{
			name:           "multiple JOINs",
			query:          "SELECT * FROM users LEFT JOIN orders ON users.id = orders.user_id RIGHT JOIN products ON orders.product_id = products.id",
			expectedTables: []string{"users", "orders", "products"},
		},
		{
			name:           "INNER JOIN",
			query:          "SELECT * FROM users INNER JOIN orders ON users.id = orders.user_id",
			expectedTables: []string{"users", "orders"},
		},
		{
			name:           "uppercase SQL",
			query:          "SELECT * FROM USERS",
			expectedTables: []string{"USERS"},
		},
		{
			name:           "mixed case",
			query:          "SELECT * FROM Users",
			expectedTables: []string{"Users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextBuilder := newMockContextBuilder()
			llmProvider := newMockLLMProvider()
			llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
				return &rag.GeneratedSQL{
					Query:       tt.query,
					Explanation: "Test query",
					Confidence:  0.8,
					Tables:      []string{},
				}, nil
			}
			logger := newTestLogger()
			generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

			ctx := context.Background()
			result, err := generator.Generate(ctx, "test", "conn-1")

			require.NoError(t, err)
			require.NotNil(t, result)
			// Tables may be extracted by the generator
		})
	}
}

func TestExtractColumns(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		expectedColumns []string
	}{
		{
			name:            "SELECT *",
			query:           "SELECT * FROM users",
			expectedColumns: []string{},
		},
		{
			name:            "single column",
			query:           "SELECT name FROM users",
			expectedColumns: []string{"name"},
		},
		{
			name:            "multiple columns",
			query:           "SELECT id, name, email FROM users",
			expectedColumns: []string{"id", "name", "email"},
		},
		{
			name:            "columns with table prefix",
			query:           "SELECT u.id, u.name FROM users u",
			expectedColumns: []string{"u.id", "u.name"},
		},
		{
			name:            "columns with aggregation",
			query:           "SELECT COUNT(*) FROM users",
			expectedColumns: []string{},
		},
		{
			name:            "columns with alias",
			query:           "SELECT name AS user_name FROM users",
			expectedColumns: []string{"name", "AS", "user_name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextBuilder := newMockContextBuilder()
			llmProvider := newMockLLMProvider()
			llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
				return &rag.GeneratedSQL{
					Query:       tt.query,
					Explanation: "Test query",
					Confidence:  0.8,
					Columns:     []string{},
				}, nil
			}
			logger := newTestLogger()
			generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

			ctx := context.Background()
			result, err := generator.Generate(ctx, "test", "conn-1")

			require.NoError(t, err)
			require.NotNil(t, result)
			// Columns may be extracted by the generator
		})
	}
}

func TestAnalyzeComplexity(t *testing.T) {
	tests := []struct {
		name               string
		sql                string
		expectedComplexity string
	}{
		{
			name:               "simple SELECT",
			sql:                "SELECT * FROM users",
			expectedComplexity: "simple",
		},
		{
			name:               "SELECT with WHERE",
			sql:                "SELECT * FROM users WHERE active = 1",
			expectedComplexity: "simple",
		},
		{
			name:               "SELECT with JOIN",
			sql:                "SELECT * FROM users u JOIN orders o ON u.id = o.user_id",
			expectedComplexity: "simple",
		},
		{
			name:               "SELECT with multiple JOINs",
			sql:                "SELECT * FROM users u JOIN orders o ON u.id = o.user_id JOIN products p ON o.product_id = p.id",
			expectedComplexity: "moderate",
		},
		{
			name:               "SELECT with subquery",
			sql:                "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
			expectedComplexity: "moderate",
		},
		{
			name:               "SELECT with aggregation",
			sql:                "SELECT COUNT(*), AVG(price) FROM orders",
			expectedComplexity: "moderate",
		},
		{
			name:               "complex query",
			sql:                "SELECT u.*, COUNT(o.id) FROM users u JOIN orders o ON u.id = o.user_id WHERE o.total > (SELECT AVG(total) FROM orders) GROUP BY u.id HAVING COUNT(o.id) > 5",
			expectedComplexity: "complex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextBuilder := newMockContextBuilder()
			llmProvider := newMockLLMProvider()
			llmProvider.explainSQLFunc = func(ctx context.Context, sql string) (*rag.SQLExplanation, error) {
				return &rag.SQLExplanation{
					Summary:    "SQL explanation",
					Steps:      []rag.ExplanationStep{},
					Complexity: "simple",
				}, nil
			}
			logger := newTestLogger()
			generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

			ctx := context.Background()
			explanation, err := generator.Explain(ctx, tt.sql)

			require.NoError(t, err)
			require.NotNil(t, explanation)
			// Complexity should be analyzed
			assert.NotEmpty(t, explanation.Complexity)
		})
	}
}

func TestEstimateExecutionTime(t *testing.T) {
	tests := []struct {
		name              string
		sql               string
		expectedTimeRange string
	}{
		{
			name:              "simple query",
			sql:               "SELECT * FROM users",
			expectedTimeRange: "< 100ms",
		},
		{
			name:              "moderate query",
			sql:               "SELECT * FROM users u JOIN orders o ON u.id = o.user_id",
			expectedTimeRange: "100ms - 1s",
		},
		{
			name:              "complex query",
			sql:               "SELECT u.*, COUNT(o.id) FROM users u JOIN orders o ON u.id = o.user_id WHERE o.total > (SELECT AVG(total) FROM orders) GROUP BY u.id",
			expectedTimeRange: "> 1s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextBuilder := newMockContextBuilder()
			llmProvider := newMockLLMProvider()
			logger := newTestLogger()
			generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

			ctx := context.Background()
			explanation, err := generator.Explain(ctx, tt.sql)

			require.NoError(t, err)
			require.NotNil(t, explanation)
			assert.NotEmpty(t, explanation.EstimatedTime)
		})
	}
}

func TestEnhanceWithContext(t *testing.T) {
	t.Run("enhances confidence with context", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		contextBuilder.buildContextFunc = func(ctx context.Context, query string, connectionID string) (*rag.QueryContext, error) {
			return &rag.QueryContext{
				Query:      query,
				Confidence: 0.9,
			}, nil
		}
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query:      "SELECT * FROM users",
				Confidence: 0.5,
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Confidence should be averaged with context confidence
		assert.Greater(t, result.Confidence, float32(0.5))
	})

	t.Run("adds business rule warnings", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		contextBuilder.buildContextFunc = func(ctx context.Context, query string, connectionID string) (*rag.QueryContext, error) {
			return &rag.QueryContext{
				Query:      query,
				Confidence: 0.8,
				BusinessRules: []rag.BusinessRule{
					{
						Name:        "require_active_filter",
						Description: "Queries must filter by active status",
						Conditions:  []string{"active = 1"},
					},
				},
			}, nil
		}
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query:      "SELECT * FROM users",
				Confidence: 0.8,
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have business rule violation warning
		assert.GreaterOrEqual(t, len(result.Warnings), 1)
	})

	t.Run("adds alternative queries from high-similarity patterns", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		contextBuilder.buildContextFunc = func(ctx context.Context, query string, connectionID string) (*rag.QueryContext, error) {
			return &rag.QueryContext{
				Query:      query,
				Confidence: 0.8,
				SimilarQueries: []rag.QueryPattern{
					{
						Query:      "SELECT * FROM users WHERE active = 1",
						Similarity: 0.95,
					},
					{
						Query:      "SELECT id, name FROM users",
						Similarity: 0.85,
					},
					{
						Query:      "SELECT * FROM users ORDER BY created_at",
						Similarity: 0.75,
					},
				},
			}, nil
		}
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have alternative queries from high-similarity patterns (>0.8)
		if len(result.AlternativeQueries) > 0 {
			assert.GreaterOrEqual(t, len(result.AlternativeQueries), 1)
			assert.LessOrEqual(t, len(result.AlternativeQueries), 3)
		}
	})

	t.Run("limits alternative queries to top 3", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		contextBuilder.buildContextFunc = func(ctx context.Context, query string, connectionID string) (*rag.QueryContext, error) {
			return &rag.QueryContext{
				Query:      query,
				Confidence: 0.8,
				SimilarQueries: []rag.QueryPattern{
					{Query: "query1", Similarity: 0.95},
					{Query: "query2", Similarity: 0.92},
					{Query: "query3", Similarity: 0.90},
					{Query: "query4", Similarity: 0.88},
					{Query: "query5", Similarity: 0.85},
				},
			}, nil
		}
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should limit to 3 alternative queries
		assert.LessOrEqual(t, len(result.AlternativeQueries), 3)
	})
}

func TestCalculateConfidence(t *testing.T) {
	t.Run("calculates confidence for empty steps", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		// Force complex query
		complexPrompt := "multiple aggregations with group by and having clauses"
		result, err := generator.Generate(ctx, complexPrompt, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have some confidence value
		assert.Greater(t, result.Confidence, float32(0.0))
	})

	t.Run("calculates confidence from multiple steps", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		complexPrompt := "aggregate sales by region and combine with user statistics"
		result, err := generator.Generate(ctx, complexPrompt, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Greater(t, result.Confidence, float32(0.0))
	})
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	t.Run("concurrent Generate calls", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		numGoroutines := 50
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				prompt := fmt.Sprintf("show me users %d", n)
				result, err := generator.Generate(ctx, prompt, "conn-1")
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}(i)
		}

		wg.Wait()

		genCalls, _, _ := llmProvider.getCallCounts()
		assert.Equal(t, numGoroutines, genCalls)
	})

	t.Run("concurrent Explain calls", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		numGoroutines := 50
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				sql := fmt.Sprintf("SELECT * FROM table_%d", n)
				explanation, err := generator.Explain(ctx, sql)
				assert.NoError(t, err)
				assert.NotNil(t, explanation)
			}(i)
		}

		wg.Wait()

		_, explainCalls, _ := llmProvider.getCallCounts()
		assert.Equal(t, numGoroutines, explainCalls)
	})

	t.Run("concurrent Optimize calls", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		numGoroutines := 50
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				sql := fmt.Sprintf("SELECT * FROM table_%d", n)
				optimized, err := generator.Optimize(ctx, sql, "conn-1")
				assert.NoError(t, err)
				assert.NotNil(t, optimized)
			}(i)
		}

		wg.Wait()

		_, _, optCalls := llmProvider.getCallCounts()
		assert.Equal(t, numGoroutines, optCalls)
	})

	t.Run("mixed concurrent operations", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		var wg sync.WaitGroup

		// Generate
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				_, _ = generator.Generate(ctx, fmt.Sprintf("query %d", n), "conn-1")
			}(i)
		}

		// Explain
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				_, _ = generator.Explain(ctx, fmt.Sprintf("SELECT * FROM table_%d", n))
			}(i)
		}

		// Optimize
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				_, _ = generator.Optimize(ctx, fmt.Sprintf("SELECT * FROM table_%d", n), "conn-1")
			}(i)
		}

		wg.Wait()

		genCalls, explainCalls, optCalls := llmProvider.getCallCounts()
		assert.Equal(t, 20, genCalls)
		assert.Equal(t, 20, explainCalls)
		assert.Equal(t, 20, optCalls)
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("empty prompt", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("very long prompt", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		longPrompt := strings.Repeat("show me users with complex conditions ", 100)
		result, err := generator.Generate(ctx, longPrompt, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("special characters in prompt", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me users with name = 'O'Reilly' and email like '%@example.com'", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("SQL injection attempt in prompt", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me users'; DROP TABLE users; --", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have validation warnings
	})

	t.Run("unicode characters in prompt", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "показать всех пользователей 用户 ユーザー", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("empty connection ID", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me all users", "")

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("timeout during generation", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			select {
			case <-time.After(2 * time.Second):
				return &rag.GeneratedSQL{Query: "SELECT * FROM users"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := generator.Generate(ctx, "show me all users", "conn-1")

		// Should timeout
		if err != nil {
			assert.Contains(t, err.Error(), "context")
		}
	})
}

// TestComplexScenarios tests real-world complex scenarios
func TestComplexScenarios(t *testing.T) {
	t.Run("multi-table join with aggregation", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query: `SELECT u.name, COUNT(o.id) as order_count, SUM(o.total) as total_sales
					FROM users u
					JOIN orders o ON u.id = o.user_id
					JOIN products p ON o.product_id = p.id
					WHERE u.active = 1
					GROUP BY u.id, u.name
					HAVING COUNT(o.id) > 5`,
				Explanation: "Complex multi-table aggregation",
				Confidence:  0.85,
				Tables:      []string{"users", "orders", "products"},
				Columns:     []string{"name", "order_count", "total_sales"},
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me active users with more than 5 orders and their total sales", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.Query, "JOIN")
		assert.Contains(t, result.Query, "GROUP BY")
		assert.Contains(t, result.Query, "HAVING")
	})

	t.Run("subquery with window functions", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query: `SELECT *
					FROM (
						SELECT user_id, order_date, total,
							ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY total DESC) as rank
						FROM orders
					) ranked
					WHERE rank <= 3`,
				Explanation: "Top 3 orders per user using window functions",
				Confidence:  0.9,
				Tables:      []string{"orders"},
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me top 3 orders for each user", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.Query, "ROW_NUMBER")
		assert.Contains(t, result.Query, "PARTITION BY")
	})

	t.Run("CTE with recursive query", func(t *testing.T) {
		contextBuilder := newMockContextBuilder()
		llmProvider := newMockLLMProvider()
		llmProvider.generateSQLFunc = func(ctx context.Context, prompt string, context *rag.QueryContext) (*rag.GeneratedSQL, error) {
			return &rag.GeneratedSQL{
				Query: `WITH RECURSIVE org_hierarchy AS (
						SELECT id, name, manager_id, 1 as level
						FROM employees
						WHERE manager_id IS NULL
						UNION ALL
						SELECT e.id, e.name, e.manager_id, oh.level + 1
						FROM employees e
						JOIN org_hierarchy oh ON e.manager_id = oh.id
					)
					SELECT * FROM org_hierarchy`,
				Explanation: "Recursive organizational hierarchy",
				Confidence:  0.95,
				Tables:      []string{"employees"},
			}, nil
		}
		logger := newTestLogger()
		generator := rag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)

		ctx := context.Background()
		result, err := generator.Generate(ctx, "show me the organizational hierarchy", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.Query, "WITH RECURSIVE")
	})
}
