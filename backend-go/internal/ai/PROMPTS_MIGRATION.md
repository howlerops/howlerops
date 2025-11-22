# Prompt System Migration Guide

This guide explains how to use the new universal SQL prompt system with token budget management.

## Overview

The new prompt system provides:

1. **Universal SQL Prompts**: Comprehensive, dialect-aware prompts for SQL generation and fixing
2. **Token Budget Management**: Smart allocation of context window across RAG components
3. **RAG Integration**: Automatic enrichment with schema, examples, business rules, and performance hints

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      AI Service                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         PromptBuilder (with ContextBuilder)          │  │
│  │  - Builds enhanced prompts with RAG context          │  │
│  │  - Manages token budgets                              │  │
│  │  - Detects SQL dialects and error categories         │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         ProviderWithPrompts (wrapper)                │  │
│  │  - Wraps existing providers                           │  │
│  │  - Coordinates prompt building and RAG               │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Provider (OpenAI, Anthropic, etc.)           │  │
│  │  - Uses ProviderHelpers for prompt extraction        │  │
│  │  - Sends enhanced prompts to LLM                      │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Option 1: Use Enhanced Service (Recommended)

The AI service can be configured to automatically use enhanced prompts:

```go
// When creating the service, pass a context builder
contextBuilder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

// Service automatically uses enhanced prompts when context builder is available
service, err := ai.NewService(config, logger)

// Generate SQL with automatic RAG enrichment
response, err := service.GenerateSQL(ctx, &ai.SQLRequest{
    Prompt:   "Show me all users who signed up last week",
    Provider: ai.ProviderOpenAI,
    Model:    "gpt-4o",
    Context: map[string]string{
        "connection_id":   "conn-123",
        "connection_type": "postgresql",
    },
})
```

### Option 2: Use Provider Wrapper

Wrap individual providers with enhanced prompt capabilities:

```go
// Create base provider
baseProvider, err := ai.NewOpenAIProvider(config, logger)

// Wrap with enhanced prompts
enhancedProvider := ai.WrapProviderWithPrompts(
    baseProvider,
    contextBuilder,
    ai.DialectPostgreSQL,
)

// Use enhanced generation
response, allocation, err := enhancedProvider.GenerateSQLWithEnhancedPrompts(ctx, req)

// allocation contains token budget information
fmt.Printf("Token budget: %s\n", allocation.Summary())
```

### Option 3: Update Provider Implementation

Update provider's GenerateSQL and FixSQL methods to use new prompts:

```go
func (p *openaiProvider) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
    helpers := ai.NewProviderHelpers()

    // Extract enhanced system prompt from context (if available)
    systemPrompt, userPrompt := helpers.BuildPromptForProvider(req, false)

    // Build messages
    messages := []openaiChatMessage{
        {Role: "system", Content: systemPrompt},
        {Role: "user", Content: userPrompt},
    }

    // Rest of implementation...
}
```

## SQL Dialect Support

The prompt system automatically detects and adapts to different SQL dialects:

- PostgreSQL
- MySQL/MariaDB
- SQLite
- Microsoft SQL Server
- Oracle
- Generic SQL (fallback)

Detection is automatic based on connection metadata:

```go
req := &ai.SQLRequest{
    Prompt: "List all active customers",
    Context: map[string]string{
        "connection_type": "postgresql", // Automatically uses PostgreSQL-specific prompt
    },
}
```

## Error Category Detection

For SQL fixing, the system automatically categorizes errors:

- **Syntax**: Parse errors, missing keywords, etc.
- **Reference**: Unknown tables/columns, ambiguous references
- **Type**: Type mismatches, invalid casts
- **Permission**: Access denied errors
- **Constraint**: Primary key, foreign key violations
- **Performance**: Timeouts, resource exhaustion

```go
category := ai.DetectErrorCategory("ERROR: column 'user_id' does not exist")
// Returns: ErrorCategoryReference
```

## Token Budget Management

The system intelligently allocates tokens across components:

```go
// Default allocation for 8192 token model:
// - System Prompt: 2000 tokens
// - User Query: 500 tokens
// - Output Buffer: 2000 tokens (25%)
// - RAG Context: 3500 tokens
//   - Schema: 35% (1225 tokens) - highest priority
//   - Examples: 28% (980 tokens) - high priority
//   - Business: 20% (700 tokens) - medium priority
//   - Performance: 12% (420 tokens) - lower priority

// Priorities adjust based on query:
// - Fixing errors: Examples boosted to 36%, Performance reduced
// - Business query: Business rules boosted to 32%
// - Performance query: Performance hints boosted to 28%
```

## RAG Context Format

The system formats RAG context in a structured, LLM-friendly format:

```markdown
## Database Schema

### users (relevance: 0.95)

User account information

Columns:
- id: INTEGER [PK] [NOT NULL]
- email: VARCHAR(255) [NOT NULL]
- created_at: TIMESTAMP [NOT NULL]

Relationships:
- users (user_id) -> orders (id)

## Similar Query Examples

### Example 1 (similarity: 0.92)

\`\`\`sql
SELECT u.email, COUNT(o.id) as order_count
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.created_at >= NOW() - INTERVAL '7 days'
GROUP BY u.id, u.email
ORDER BY order_count DESC;
\`\`\`

Used 15 times

## Applicable Business Rules

### Customer Active Status

A customer is considered active if they have placed an order in the last 90 days.

SQL Mapping: `CASE WHEN MAX(order_date) > NOW() - INTERVAL '90 days' THEN 'active' ELSE 'inactive' END`
```

## Response Format

All providers return responses in a consistent JSON format:

```json
{
  "query": "SELECT email FROM users WHERE created_at >= NOW() - INTERVAL '7 days'",
  "explanation": "This query retrieves email addresses of users who signed up in the last 7 days",
  "confidence": 0.95,
  "suggestions": [
    "Consider adding an index on created_at for better performance",
    "Use parameterized query for the date interval"
  ],
  "warnings": [
    "Large result sets may impact performance",
    "Ensure timezone handling is correct"
  ]
}
```

## Migration Path

### Phase 1: Add Helper Usage (No Breaking Changes)

Update providers to check for enhanced prompts:

```go
func (p *openaiProvider) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
    helpers := ai.NewProviderHelpers()
    systemPrompt, userPrompt := helpers.BuildPromptForProvider(req, false)

    // If no enhanced prompt in context, falls back to default
    // Existing behavior preserved
}
```

### Phase 2: Enable RAG Context (Opt-in)

Users can opt-in to RAG by providing connection context:

```go
req := &ai.SQLRequest{
    Prompt: "Show recent orders",
    Context: map[string]string{
        "connection_id": "conn-123", // Enables RAG context
    },
}
```

### Phase 3: Full Migration

Once validated, make enhanced prompts the default.

## Performance Considerations

### Token Budget Optimization

- **Small models (4k tokens)**: Reduce RAG context, focus on schema only
- **Medium models (8k-16k tokens)**: Balanced allocation across all components
- **Large models (100k+ tokens)**: Include extensive examples and documentation

### Caching

The system supports caching at multiple levels:

```go
// Context builder caches schema and pattern searches
// Vector store caches embeddings
// Token estimates cached for repeated content
```

### Monitoring

Track budget usage:

```go
response, allocation, err := enhancedProvider.GenerateSQLWithEnhancedPrompts(ctx, req)

log.Printf("Budget allocation:\n%s", allocation.Summary())
// Outputs:
// Token Budget Allocation (Total: 8192)
//   System Prompt: 2000 tokens
//   User Query: 500 tokens
//   Output Buffer: 2000 tokens
//   Context Components:
//     schema: 1225 tokens (priority: 10, ratio: 35.00%)
//       Used: 1100 tokens (89.8% of allocated)
//     examples: 980 tokens (priority: 7, ratio: 28.00%)
//       Used: 950 tokens (96.9% of allocated)
//   Remaining: 437 tokens
```

## Testing

Test with different scenarios:

```go
func TestEnhancedPrompts(t *testing.T) {
    // Test dialect detection
    dialect := ai.DetectDialect("postgresql")
    assert.Equal(t, ai.DialectPostgreSQL, dialect)

    // Test error categorization
    category := ai.DetectErrorCategory("syntax error at position 10")
    assert.Equal(t, ai.ErrorCategorySyntax, category)

    // Test token budget
    budget := rag.DefaultTokenBudget(8192)
    assert.Greater(t, budget.ContextAvailable, 3000)

    // Test allocation
    allocation := budget.AllocateContextBudget(nil)
    assert.NoError(t, allocation.Validate(8192))
}
```

## Examples

See test files for complete examples:

- `ai/prompts_test.go` - Prompt generation tests
- `rag/token_budget_test.go` - Budget allocation tests
- `ai/integration_test.go` - End-to-end integration tests

## Troubleshooting

### Issue: Context not being included

**Cause**: Missing connection_id in request context

**Solution**: Add connection metadata:
```go
req.Context = map[string]string{
    "connection_id": "conn-123",
}
```

### Issue: Token budget exceeded

**Cause**: Too much RAG context for small models

**Solution**: Reduce context priorities or use larger model:
```go
priorities := map[string]int{
    "schema": 10,  // Keep schema
    "examples": 0, // Disable examples
    "business": 0, // Disable business rules
    "performance": 0, // Disable hints
}
```

### Issue: Wrong SQL dialect

**Cause**: Incorrect connection type detection

**Solution**: Explicitly specify dialect:
```go
req.Context = map[string]string{
    "dialect": "postgresql",
}
```

## Future Enhancements

Planned improvements:

1. **Dynamic budget adjustment**: Learn optimal allocations from usage patterns
2. **Multi-turn conversations**: Maintain context across multiple queries
3. **Query validation**: Pre-execution validation using dialect rules
4. **Performance profiling**: Track which context components improve quality
5. **Custom prompts**: Allow users to override system prompts for specific use cases
