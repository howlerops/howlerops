<!-- d3d028e9-954f-49d7-bbc5-0354be644945 99f07984-05ed-4e56-902c-5f37a605dfac -->
# AI Query Agent Tab Implementation Plan

## Overview

Create a comprehensive AI-powered query interface with multiple specialized agents (orchestrator, SQL generator, data analyst, report generator, chart generator) that can answer questions, generate queries, create visualizations, and produce reports - all in a streaming chat interface with read-only database access.

## Architecture

### Backend Components (Go)

#### 1. Agent System (`backend-go/internal/ai/agents/`)

- **Orchestrator Agent** - Routes user requests to appropriate specialized agents
- **SQL Generator Agent** - Converts questions to SQL (reuses existing GenerateSQL)
- **Data Analyst Agent** - Analyzes query results, identifies trends, provides insights
- **Report Generator Agent** - Creates formatted reports from data
- **Chart Generator Agent** - Suggests and generates chart specifications
- **Query Explainer Agent** - Explains what queries do (educational)

#### 2. Streaming Chat Service (`backend-go/internal/ai/chat_service.go`)

- WebSocket or SSE-based streaming endpoint
- Message queue for agent responses
- Token-by-token streaming from AI providers
- Support for multi-turn conversations with context

#### 3. Read-Only Query Executor (`backend-go/pkg/database/readonly_executor.go`)

- Validates queries are SELECT-only (no INSERT/UPDATE/DELETE/DROP)
- Wraps existing ExecuteQuery with read-only enforcement
- Adds query timeout and row limit protections
- Returns structured results with metadata

#### 4. App Methods (`app.go`)

- `StreamAIChat(sessionId, message)` - Main streaming chat endpoint
- `ExecuteReadOnlyQuery(connectionId, sql)` - Safe query execution
- `GenerateChartSpec(data, chartType)` - Chart configuration generator
- `GenerateReport(data, format)` - Report generation (markdown/PDF)
- `ExportAIOutput(outputId, format)` - Export charts/reports
- `GetAIChatSessions()` - List all AI chat sessions
- `DeleteAIChatSession(sessionId)` - Remove chat history

#### 5. Extended Memory System

- Extend existing `AIMemorySessionPayload` with agent-specific metadata
- Add `ChatTurn` tracking for multi-agent conversations
- Store generated charts/reports as artifacts
- Link to executed queries for context

### Frontend Components (React/TypeScript)

#### 1. Query Type Dropdown (`frontend/src/components/query-type-selector.tsx`)

- Replace "Add Query" button with dropdown
- Options: "SQL Editor" (existing), "AI Assistant" (new)
- Icon indicators for each type

#### 2. AI Query Tab (`frontend/src/components/ai-query-tab.tsx`)

- Main container for AI chat interface
- Manages session state and message history
- Handles streaming response updates
- Coordinates with other components

#### 3. Chat Interface (`frontend/src/components/ai-chat/`)

- **`chat-container.tsx`** - Main chat layout with message list
- **`message-bubble.tsx`** - Individual messages with agent badges
- **`chat-input.tsx`** - User input with send button, typing indicators
- **`streaming-indicator.tsx`** - Visual feedback during AI response
- **`agent-badge.tsx`** - Shows which agent is responding

#### 4. Output Renderers (`frontend/src/components/ai-chat/renderers/`)

- **`sql-output-renderer.tsx`** - SQL with syntax highlighting + "Run Query" button
- **`table-output-renderer.tsx`** - Data tables with sorting/filtering
- **`chart-renderer.tsx`** - Dynamic chart rendering (Recharts/Victory)
- **`report-renderer.tsx`** - Formatted reports with markdown support
- **`insight-renderer.tsx`** - Highlighted insights and analysis

#### 5. Export System (`frontend/src/components/ai-chat/export/`)

- **`export-button.tsx`** - Export dropdown for each output
- **`pdf-exporter.ts`** - PDF generation (jsPDF or react-pdf)
- **`image-exporter.ts`** - Chart to PNG/SVG export
- **`csv-exporter.ts`** - Data to CSV export

#### 6. Store Integration (`frontend/src/store/ai-chat-store.ts`)

- Zustand store for AI chat state
- Message history management
- Active session tracking
- Output artifacts storage

#### 7. Hooks

- `useAIChatStream.ts` - WebSocket/SSE connection management
- `useReadOnlyQuery.ts` - Safe query execution
- `useChartGeneration.ts` - Chart specification and rendering
- `useReportExport.ts` - Export functionality

## Implementation Details

### Agent Capabilities

**Orchestrator**

- Analyzes user intent
- Routes to appropriate specialized agent(s)
- Coordinates multi-step workflows
- Examples: "Show sales trends" → SQL Agent + Data Analyst + Chart Agent

**SQL Generator** (existing, enhanced)

- Natural language to SQL
- Multi-database aware (uses @connection syntax)
- Schema-aware through RAG
- Confidence scoring

**Data Analyst**

- Statistical analysis (mean, median, trends, outliers)
- Pattern recognition (seasonality, correlations)
- Anomaly detection
- Natural language insights

**Report Generator**

- Executive summaries
- Formatted tables with highlighting
- Multi-section reports
- Export-ready layouts

**Chart Generator**

- Chart type suggestions based on data shape
- Recharts component specifications
- Interactive visualizations
- Responsive designs

**Query Explainer**

- Plain English query descriptions
- Performance implications
- Alternative approaches

### Read-Only Enforcement

```go
// Validation rules
- Reject: INSERT, UPDATE, DELETE, DROP, TRUNCATE, ALTER, CREATE
- Allow: SELECT, SHOW, DESCRIBE, EXPLAIN
- Add: LIMIT 10000 (if not specified)
- Add: Timeout 30s (configurable)
- Wrap in transaction (rollback-only)
```

### Streaming Protocol

```typescript
// Message types
type StreamMessage = 
  | { type: 'agent_start', agent: AgentType }
  | { type: 'token', content: string }
  | { type: 'sql_generated', sql: string, confidence: number }
  | { type: 'query_result', data: QueryResult }
  | { type: 'chart_spec', spec: ChartSpec }
  | { type: 'report', content: string }
  | { type: 'agent_complete', agent: AgentType }
  | { type: 'error', message: string }
```

### Chart Generation

- Support chart types: line, bar, area, pie, scatter, heatmap
- Use Recharts for rendering
- Generate responsive configurations
- Store specs for re-rendering/export

### Export Formats

- **PDF**: Charts + tables + insights combined
- **PNG/SVG**: Individual charts
- **CSV**: Raw data tables
- **Markdown**: Full conversation transcript
- **JSON**: Structured data + metadata

## File Structure

```
backend-go/
  internal/ai/
    agents/
      orchestrator.go       # Main routing agent
      sql_agent.go          # SQL generation
      analyst_agent.go      # Data analysis
      report_agent.go       # Report generation
      chart_agent.go        # Chart specifications
      explainer_agent.go    # Query explanation
      types.go              # Agent interfaces
    chat_service.go         # Streaming chat handler
    stream_handler.go       # SSE/WebSocket
  pkg/database/
    readonly_executor.go    # Safe query execution
    query_validator.go      # SQL validation

frontend/
  src/components/
    query-type-selector.tsx
    ai-query-tab.tsx
    ai-chat/
      chat-container.tsx
      message-bubble.tsx
      chat-input.tsx
      streaming-indicator.tsx
      agent-badge.tsx
      renderers/
        sql-output-renderer.tsx
        table-output-renderer.tsx
        chart-renderer.tsx
        report-renderer.tsx
        insight-renderer.tsx
      export/
        export-button.tsx
        pdf-exporter.ts
        image-exporter.ts
        csv-exporter.ts
  src/store/
    ai-chat-store.ts
  src/hooks/
    use-ai-chat-stream.ts
    use-readonly-query.ts
    use-chart-generation.ts
    use-report-export.ts
  src/types/
    ai-chat.ts
```

## Integration Points

### Existing Systems

- **AI Service**: Reuse existing provider infrastructure (OpenAI, Anthropic, Ollama)
- **Memory System**: Extend `AIMemorySessionPayload` for chat context
- **Database Service**: Wrap with read-only enforcement
- **Storage Manager**: Store chat sessions and artifacts
- **Query Store**: Track executed queries from AI suggestions

### New Wails Bindings

```go
// app.go additions
StreamAIChat(sessionId, message string) error
ExecuteReadOnlyQuery(connectionId, sql string) (*QueryResponse, error)
GenerateChartSpec(data ResultData, preferences map[string]string) (*ChartSpec, error)
ExportAIReport(sessionId string, format string) (string, error)
```

## Security & Safety

### Query Validation

1. Parse SQL AST to detect write operations
2. Reject stored procedures/functions
3. Enforce row limits (max 10k)
4. Set query timeout (30s default)
5. Run in read-only transaction

### Rate Limiting

- Max 10 AI requests per minute per session
- Queue requests if limit exceeded
- Show user-friendly rate limit message

### Data Privacy

- Don't send sensitive data in prompts (PII detection)
- Redact connection strings from context
- Clear chat sessions on demand

## Testing Strategy

### Backend Tests

- Agent routing logic
- Read-only query validation (positive/negative cases)
- Streaming message serialization
- Chart specification generation

### Frontend Tests

- Message rendering
- Streaming updates
- Export functionality
- Error handling

### Integration Tests

- End-to-end chat flow
- Query execution and result display
- Chart generation and rendering
- PDF export

## Success Criteria

✅ User can create AI query tab from dropdown

✅ Chat interface streams responses in real-time

✅ SQL queries are validated as read-only

✅ Generated SQL has "Run Query" button (user approval)

✅ Results displayed inline with charts/insights

✅ Reports exportable as PDF/PNG/CSV

✅ Multiple concurrent AI chat tabs supported

✅ Chat history persists and is searchable

✅ No accidental database modifications possible

✅ Graceful error handling with helpful messages

## Dependencies

### New Packages (Go)

- `github.com/jung-kurt/gofpdf` - PDF generation (optional, for backend PDF)

### New Packages (Frontend)

- `recharts` (likely already present)
- `jspdf` or `@react-pdf/renderer` - PDF export
- `html2canvas` - Chart to image conversion
- `markdown-to-jsx` - Report rendering

## Migration & Rollout

1. **Phase 1**: Backend agent system + read-only executor
2. **Phase 2**: Frontend dropdown + basic chat UI
3. **Phase 3**: Streaming + agent integration
4. **Phase 4**: Chart/report rendering
5. **Phase 5**: Export system
6. **Phase 6**: Polish + testing + documentation

### To-dos

- [ ] Create agent system infrastructure (orchestrator, SQL agent, analyst, report generator, chart generator, explainer)
- [ ] Implement read-only query executor with SQL validation and safety checks
- [ ] Build streaming chat service with WebSocket/SSE support and token-by-token streaming
- [ ] Add new Wails app methods for streaming chat, read-only queries, chart generation, and exports
- [ ] Replace 'Add Query' button with dropdown selector (SQL Editor / AI Assistant)
- [ ] Create Zustand store for AI chat state management (sessions, messages, artifacts)
- [ ] Build chat container, message bubbles, input, streaming indicators, and agent badges
- [ ] Implement useAIChatStream hook for WebSocket/SSE connection management
- [ ] Create output renderers for SQL, tables, charts, reports, and insights
- [ ] Implement chart generator with Recharts integration and dynamic configuration
- [ ] Build export functionality (PDF, PNG, CSV) with export button and format handlers
- [ ] Extend existing AI memory system to support chat sessions with agent metadata
- [ ] Add comprehensive tests for agent system, streaming, read-only execution, and exports