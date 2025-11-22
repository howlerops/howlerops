package rag

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTokenBudget(t *testing.T) {
	tests := []struct {
		name        string
		totalTokens int
		wantSystem  int
		wantOutput  int
		wantContext int
	}{
		{
			name:        "small model (4k)",
			totalTokens: 4096,
			wantSystem:  2000,
			wantOutput:  1024, // 25% capped at 2000
			wantContext: 800,  // Approximate after safety margin
		},
		{
			name:        "medium model (8k)",
			totalTokens: 8192,
			wantSystem:  2000,
			wantOutput:  2000,
			wantContext: 3500, // Approximate
		},
		{
			name:        "large model (128k)",
			totalTokens: 128000,
			wantSystem:  2000,
			wantOutput:  2000,   // Capped at 2000
			wantContext: 118000, // Most goes to context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := DefaultTokenBudget(tt.totalTokens)

			assert.Equal(t, tt.totalTokens, budget.TotalTokens)
			assert.Equal(t, tt.wantSystem, budget.SystemPrompt)
			assert.Equal(t, tt.wantOutput, budget.OutputBuffer)

			// Context available should be positive
			assert.Greater(t, budget.ContextAvailable, 0,
				"Should have positive context tokens available")

			// Context should be roughly in expected range (±10%)
			tolerance := float64(tt.wantContext) * 0.1
			assert.InDelta(t, tt.wantContext, budget.ContextAvailable, tolerance,
				"Context allocation should be approximately %d", tt.wantContext)

			// Total should not exceed available
			total := budget.SystemPrompt + budget.UserQuery + budget.OutputBuffer + budget.ContextAvailable
			assert.LessOrEqual(t, total, tt.totalTokens,
				"Total allocation should not exceed total tokens")
		})
	}
}

func TestAllocateContextBudget(t *testing.T) {
	t.Run("default priorities", func(t *testing.T) {
		budget := DefaultTokenBudget(8192)
		allocation := budget.AllocateContextBudget(nil)

		// Check that allocation was created
		require.NotNil(t, allocation)
		require.NotNil(t, allocation.Details)

		// Verify all components got allocations
		assert.Greater(t, allocation.Schema, 0, "Schema should get tokens")
		assert.Greater(t, allocation.Examples, 0, "Examples should get tokens")
		assert.Greater(t, allocation.Business, 0, "Business should get tokens")
		assert.Greater(t, allocation.Performance, 0, "Performance should get tokens")

		// Schema should have highest allocation (priority 10)
		assert.Greater(t, allocation.Schema, allocation.Examples,
			"Schema (priority 10) should get more than examples (priority 7)")
		assert.Greater(t, allocation.Examples, allocation.Business,
			"Examples (priority 7) should get more than business (priority 5)")
		assert.Greater(t, allocation.Business, allocation.Performance,
			"Business (priority 5) should get more than performance (priority 3)")

		// Total should be valid
		assert.NoError(t, allocation.Validate(8192))
	})

	t.Run("custom priorities", func(t *testing.T) {
		budget := DefaultTokenBudget(8192)
		priorities := map[string]int{
			"schema":      5,  // Lower schema priority
			"examples":    10, // Boost examples
			"business":    0,  // Disable business
			"performance": 0,  // Disable performance
		}

		allocation := budget.AllocateContextBudget(priorities)

		// Examples should get most tokens
		assert.Greater(t, allocation.Examples, allocation.Schema,
			"Examples should get more with higher priority")

		// Business and performance should get nothing
		assert.Equal(t, 0, allocation.Business, "Business should get 0 tokens")
		assert.Equal(t, 0, allocation.Performance, "Performance should get 0 tokens")
	})

	t.Run("single component", func(t *testing.T) {
		budget := DefaultTokenBudget(4096)
		priorities := map[string]int{
			"schema": 10,
		}

		allocation := budget.AllocateContextBudget(priorities)

		// All context should go to schema
		assert.Equal(t, budget.ContextAvailable, allocation.Schema,
			"All context tokens should go to schema")
		assert.Equal(t, 0, allocation.Examples)
		assert.Equal(t, 0, allocation.Business)
		assert.Equal(t, 0, allocation.Performance)
	})
}

func TestBudgetAllocation_AdjustForActualUsage(t *testing.T) {
	budget := DefaultTokenBudget(8192)
	allocation := budget.AllocateContextBudget(nil)

	// Schema was allocated ~1200 tokens but only used 1000
	schemaAllocated := allocation.Schema
	allocation.AdjustForActualUsage("schema", 1000)

	// Check that usage was recorded
	details, exists := allocation.Details["schema"]
	require.True(t, exists)
	assert.Equal(t, 1000, details.Used)
	assert.Equal(t, schemaAllocated, details.Allocated)
}

func TestBudgetAllocation_GetComponentBudget(t *testing.T) {
	budget := DefaultTokenBudget(8192)
	allocation := budget.AllocateContextBudget(nil)

	schemaBudget := allocation.GetComponentBudget("schema")
	assert.Greater(t, schemaBudget, 0)
	assert.Equal(t, allocation.Schema, schemaBudget)

	// Non-existent component should return 0
	unknownBudget := allocation.GetComponentBudget("unknown")
	assert.Equal(t, 0, unknownBudget)
}

func TestBudgetAllocation_RemainingBudget(t *testing.T) {
	budget := DefaultTokenBudget(8192)
	allocation := budget.AllocateContextBudget(nil)

	// Initially no tokens used
	remaining := allocation.RemainingBudget()
	assert.Greater(t, remaining, 0)

	// Use some tokens
	allocation.AdjustForActualUsage("schema", 1000)
	allocation.AdjustForActualUsage("examples", 800)

	// Remaining should be positive and reasonable
	remaining = allocation.RemainingBudget()
	assert.Greater(t, remaining, 0)
}

func TestBudgetAllocation_Validate(t *testing.T) {
	t.Run("valid allocation", func(t *testing.T) {
		budget := DefaultTokenBudget(8192)
		allocation := budget.AllocateContextBudget(nil)

		err := allocation.Validate(8192)
		assert.NoError(t, err)
	})

	t.Run("exceeds maximum", func(t *testing.T) {
		budget := DefaultTokenBudget(8192)
		allocation := budget.AllocateContextBudget(nil)

		// Try to validate with lower max
		err := allocation.Validate(4096)
		assert.Error(t, err)
	})
}

func TestBudgetAllocation_Summary(t *testing.T) {
	budget := DefaultTokenBudget(8192)
	allocation := budget.AllocateContextBudget(nil)

	summary := allocation.Summary()

	// Summary should be human-readable
	assert.Contains(t, summary, "Token Budget Allocation")
	assert.Contains(t, summary, "System Prompt:")
	assert.Contains(t, summary, "schema:")
	assert.Contains(t, summary, "examples:")
	assert.Contains(t, summary, "priority:")

	// Should not be empty
	assert.Greater(t, len(summary), 100)
}

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "short text",
			text:     "Hello world",
			expected: 4, // ~11 chars / 4 + 10% overhead
		},
		{
			name:     "medium text",
			text:     strings.Repeat("word ", 50), // 250 characters
			expected: 70,                          // Approximate
		},
		{
			name:     "SQL query",
			text:     "SELECT id, name FROM users WHERE created_at > NOW() - INTERVAL '7 days'",
			expected: 21, // Approximate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := EstimateTokenCount(tt.text)

			if tt.expected == 0 {
				assert.Equal(t, 0, count)
			} else {
				// Allow 30% tolerance for estimation
				tolerance := float64(tt.expected) * 0.3
				assert.InDelta(t, tt.expected, count, tolerance,
					"Token estimate should be approximately correct")
			}
		})
	}
}

func TestTruncateToTokenBudget(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxTokens int
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:      "fits within budget",
			text:      "Short text",
			maxTokens: 100,
			checkFunc: func(t *testing.T, result string) {
				assert.Equal(t, "Short text", result, "Should not truncate")
			},
		},
		{
			name:      "needs truncation",
			text:      strings.Repeat("word ", 1000),
			maxTokens: 50,
			checkFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "[truncated]", "Should indicate truncation")
				assert.Less(t, len(result), len(strings.Repeat("word ", 1000)),
					"Should be shorter than original")

				// Estimate should be within budget
				estimate := EstimateTokenCount(result)
				assert.LessOrEqual(t, estimate, 55, // 10% tolerance
					"Truncated text should fit in budget")
			},
		},
		{
			name:      "truncate at sentence boundary",
			text:      "First sentence. Second sentence. Third sentence. Fourth sentence.",
			maxTokens: 10,
			checkFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "[truncated]")
				// Should try to break at period
				assert.True(t, strings.HasSuffix(strings.TrimSuffix(result, "...[truncated]"), "."),
					"Should truncate at sentence boundary when possible")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateToTokenBudget(tt.text, tt.maxTokens)
			tt.checkFunc(t, result)
		})
	}
}

func TestPrioritizeComponents(t *testing.T) {
	t.Run("normal query", func(t *testing.T) {
		priorities := PrioritizeComponents("Show me all users", false)

		// Default priorities
		assert.Equal(t, 10, priorities["schema"])
		assert.Equal(t, 7, priorities["examples"])
		assert.Equal(t, 5, priorities["business"])
		assert.Equal(t, 3, priorities["performance"])
	})

	t.Run("error fixing", func(t *testing.T) {
		priorities := PrioritizeComponents("Fix this query", true)

		// Examples should be boosted for error fixing
		assert.Equal(t, 10, priorities["schema"])  // Still high
		assert.Equal(t, 9, priorities["examples"]) // Boosted
		assert.Equal(t, 5, priorities["business"])
		assert.Equal(t, 2, priorities["performance"]) // Lowered
	})

	t.Run("business query", func(t *testing.T) {
		priorities := PrioritizeComponents("Show revenue by customer", false)

		// Business should be boosted
		assert.Equal(t, 10, priorities["schema"])
		assert.Equal(t, 8, priorities["business"]) // Boosted for "revenue" keyword
	})

	t.Run("performance query", func(t *testing.T) {
		priorities := PrioritizeComponents("This query is slow, how to optimize?", false)

		// Performance should be boosted
		assert.Equal(t, 10, priorities["schema"])
		assert.Equal(t, 7, priorities["performance"]) // Boosted for "slow" keyword
	})

	t.Run("order query", func(t *testing.T) {
		priorities := PrioritizeComponents("List all orders with customer names", false)

		// Business should be boosted for "order" keyword
		assert.Equal(t, 8, priorities["business"])
	})
}

func TestTokenBudgetIntegration(t *testing.T) {
	t.Run("realistic scenario", func(t *testing.T) {
		// Simulate a realistic scenario
		totalTokens := 8192

		// Create budget
		budget := DefaultTokenBudget(totalTokens)

		// Allocate with custom priorities for error fixing
		priorities := PrioritizeComponents("Fix syntax error", true)
		allocation := budget.AllocateContextBudget(priorities)

		// Validate allocation
		require.NoError(t, allocation.Validate(totalTokens))

		// Simulate using tokens
		schemaText := strings.Repeat("Column definition. ", 50) // ~1000 chars
		schemaTokens := EstimateTokenCount(schemaText)
		allocation.AdjustForActualUsage("schema", schemaTokens)

		exampleText := strings.Repeat("SELECT example query. ", 40) // ~880 chars
		exampleTokens := EstimateTokenCount(exampleText)
		allocation.AdjustForActualUsage("examples", exampleTokens)

		// Check summary
		summary := allocation.Summary()
		assert.Contains(t, summary, "schema:")
		assert.Contains(t, summary, "examples:")
		assert.Contains(t, summary, "Used:")

		// Remaining budget should be reasonable
		remaining := allocation.RemainingBudget()
		assert.Greater(t, remaining, 0, "Should have remaining budget")
		assert.Less(t, remaining, totalTokens/2, "Shouldn't have too much remaining")
	})
}

func BenchmarkDefaultTokenBudget(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultTokenBudget(8192)
	}
}

func BenchmarkAllocateContextBudget(b *testing.B) {
	budget := DefaultTokenBudget(8192)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = budget.AllocateContextBudget(nil)
	}
}

func BenchmarkEstimateTokenCount(b *testing.B) {
	text := strings.Repeat("SELECT * FROM users WHERE id = ? ", 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = EstimateTokenCount(text)
	}
}

func BenchmarkTruncateToTokenBudget(b *testing.B) {
	text := strings.Repeat("This is a sentence. ", 500)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = TruncateToTokenBudget(text, 100)
	}
}

func BenchmarkPrioritizeComponents(b *testing.B) {
	query := "Show me revenue by customer for orders last month"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = PrioritizeComponents(query, false)
	}
}
