# Multi-Database AI SQL Generation Guide

## Overview

The SQL Studio now supports AI-powered SQL generation that works seamlessly across multiple connected databases. This feature allows users to:

1. Generate SQL queries using natural language across multiple databases
2. Automatically use the correct `@connection.table` syntax for multi-DB queries
3. View and interact with schemas from all connected databases
4. Fix SQL errors with AI assistance that understands multi-DB context

## Architecture

### Frontend Components

#### AI Store (`/src/store/ai-store.ts`)
- Enhanced to use Wails bindings instead of REST API
- Supports passing multi-database schema context to AI
- Handles both single and multi-database modes

#### AI Schema Context Builder (`/src/lib/ai-schema-context.ts`)
- Builds comprehensive schema context for AI
- Generates appropriate SQL syntax examples
- Creates token-efficient prompts for AI models

#### AI Schema Display (`/src/components/ai-schema-display.tsx`)
- Interactive tree view of all connected databases
- Click-to-insert table references
- Visual indicators for connection status

#### Query Editor Integration
- AI Assistant dialog with schema browser
- Auto-detects query mode (single vs multi-DB)
- Passes complete schema context to AI

### Backend Integration

#### App.go Enhancements
- `GenerateSQLFromNaturalLanguage`: Enhanced to understand multi-DB context
- `FixSQLError`: Detects and fixes multi-DB syntax issues
- Smart context building based on query mode

## Usage

### Single Database Mode

In single database mode, the AI generates standard SQL:

```sql
-- User prompt: "Show me all active users"
SELECT * FROM users WHERE status = 'active';
```

### Multi-Database Mode

In multi-database mode, the AI generates queries with `@connection` syntax:

```sql
-- User prompt: "Join production users with analytics events"
SELECT u.*, e.event_type, e.created_at
FROM @Production.users u
JOIN @Analytics.events e ON u.id = e.user_id
WHERE u.status = 'active';
```

### Using the AI Assistant

1. **Open AI Assistant**: Click the "AI Assistant" button in the query editor toolbar
2. **View Available Schemas**: Browse connected databases and their tables
3. **Write Natural Language Query**: Describe what you want in plain English
4. **Click Table Names**: Click on table names to insert them into your prompt
5. **Generate SQL**: Click "Generate SQL" or press Ctrl+Enter

### Examples

#### Cross-Database Aggregation
```text
Prompt: "Show total revenue per user from production orders and staging invoices"

Generated SQL:
SELECT
    u.id,
    u.email,
    COALESCE(o.order_total, 0) as production_revenue,
    COALESCE(i.invoice_total, 0) as staging_revenue,
    COALESCE(o.order_total, 0) + COALESCE(i.invoice_total, 0) as total_revenue
FROM @Production.users u
LEFT JOIN (
    SELECT user_id, SUM(total) as order_total
    FROM @Production.orders
    GROUP BY user_id
) o ON u.id = o.user_id
LEFT JOIN (
    SELECT user_id, SUM(amount) as invoice_total
    FROM @Staging.invoices
    GROUP BY user_id
) i ON u.id = i.user_id
ORDER BY total_revenue DESC;
```

#### Data Migration Query
```text
Prompt: "Copy all users from staging to production that don't exist in production"

Generated SQL:
INSERT INTO @Production.users (id, email, name, created_at)
SELECT s.id, s.email, s.name, s.created_at
FROM @Staging.users s
LEFT JOIN @Production.users p ON s.email = p.email
WHERE p.id IS NULL;
```

## API Reference

### Frontend Hooks

#### `useAISchemaContext`
```typescript
const {
  buildAIContext,
  getCompactSchemaContext,
  generateAIPrompt,
  getSyntaxExamples,
  isContextReady,
  getTableCount,
  getDatabaseCount
} = useAISchemaContext(mode, multiDBSchemas)
```

### Backend Methods

#### `GenerateSQLFromNaturalLanguage`
```go
type NLQueryRequest struct {
    Prompt       string `json:"prompt"`
    ConnectionID string `json:"connectionId"`
    Context      string `json:"context,omitempty"`
}

func (a *App) GenerateSQLFromNaturalLanguage(req NLQueryRequest) (*GeneratedSQLResponse, error)
```

#### `FixSQLError`
```go
func (a *App) FixSQLError(query string, error string, connectionID string) (*FixedSQLResponse, error)
```

## Configuration

### Environment Variables

Set up AI provider credentials:

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Ollama (local)
export OLLAMA_ENDPOINT="http://localhost:11434"
```

### AI Settings

Configure AI behavior in the settings panel:
- **Provider**: Choose between OpenAI, Anthropic, Ollama, etc.
- **Model**: Select the specific model to use
- **Max Tokens**: Control response length
- **Temperature**: Adjust creativity (0.1 for SQL = more deterministic)

## Best Practices

### Schema Context Management

1. **Keep Schemas Updated**: Refresh schemas when database structure changes
2. **Filter Connections**: Use environment filters to reduce context size
3. **Limit Table Count**: Too many tables can exceed token limits

### Prompt Engineering

1. **Be Specific**: Mention exact table and column names when known
2. **Use Examples**: Provide sample data or expected output format
3. **Specify Joins**: Explicitly mention how tables should be joined

### Performance Optimization

1. **Cache Schemas**: Schemas are cached to reduce API calls
2. **Lazy Load Columns**: Column information is loaded on-demand
3. **Compact Context**: Use compact mode for large schemas

## Troubleshooting

### Common Issues

#### "AI service not configured"
- Ensure AI API keys are set in environment variables
- Restart the application after setting keys

#### "Connection not found"
- Verify all referenced connections are connected
- Check connection names are spelled correctly (case-sensitive)

#### "Table not found"
- Refresh schemas if recently added
- Ensure correct `@connection.table` syntax

#### Token Limit Exceeded
- Reduce number of connected databases
- Use environment filters to limit scope
- Switch to a model with higher token limits

## Testing

Run the test suite:

```bash
npm test src/lib/__tests__/ai-schema-context.test.ts
```

## Future Enhancements

- [ ] Query execution plan analysis
- [ ] Automatic index suggestions
- [ ] Query performance predictions
- [ ] Natural language to stored procedure generation
- [ ] Schema documentation integration
- [ ] Query result visualization recommendations