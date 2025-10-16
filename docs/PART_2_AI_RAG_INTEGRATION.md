# Part 2: AI/RAG Integration for Enhanced SQL Intelligence

## Executive Summary

This document outlines the implementation of AI and RAG (Retrieval-Augmented Generation) capabilities to transform SQL Studio into an intelligent database assistant. Building on Part 1's multi-database query support, this adds context-aware SQL generation, performance optimization, and smart visualizations.

## Core Vision

Transform SQL Studio from a query tool into an intelligent database assistant that:
- **Understands** your data model and business logic
- **Learns** from query patterns and user behavior
- **Optimizes** queries before they cause problems
- **Visualizes** data intelligently without manual configuration

## HowlerOps Architecture Integration

### Existing Infrastructure

HowlerOps already has RAG components in place:
- **AI Service**: `backend-go/internal/ai/service.go` - Provider abstraction (OpenAI, Anthropic, Ollama, etc.)
- **RAG Components**: `backend-go/internal/rag/` - Vector store, context builder, SQL generator
- **Database Manager**: `backend-go/pkg/database/manager.go` - Connection and query management

### Integration Strategy

The RAG features will integrate by:
1. **Extending AI Service** - Add RAG-enhanced methods to existing AI service
2. **Creating Service Wrapper** - New `services/ai.go` to wrap AI+RAG for Wails
3. **Wails Bindings** - Expose AI methods through `app.go`
4. **Frontend Components** - React components for NL query, suggestions, visualization
5. **Learning Hooks** - Integrate learning pipeline into query execution flow

### Data Flow

```
User Natural Language Query
  ↓
Frontend AI Editor Component
  ↓
Wails Binding (app.go)
  ↓
AI Service Wrapper (services/ai.go)
  ↓
RAG Context Builder (internal/rag/context_builder.go)
  ├─→ Vector Store (internal/rag/vector_store.go) → Qdrant
  ├─→ Schema Cache (database/manager.go)
  └─→ Query History
  ↓
AI Provider (internal/ai/service.go)
  ↓
Enhanced SQL Response
  ↓
Frontend Display
```

## 1. RAG Architecture for Databases

### A. What We Embed and Index

```yaml
Schema Embeddings:
  - Table names and descriptions
  - Column names, types, and comments
  - Foreign key relationships
  - Indexes and constraints
  - Table statistics (row counts, sizes)

Query Embeddings:
  - Successful query patterns
  - Query templates by use case
  - Error patterns and fixes
  - Performance characteristics

Business Context:
  - Domain terminology (e.g., "revenue" → specific tables/columns)
  - Common calculations and KPIs
  - Business rules and constraints
  - Data quality notes

Performance Data:
  - Execution plans
  - Query runtimes
  - Index usage patterns
  - Resource consumption
```

### B. Leverage Existing Vector Store Implementation

**Location**: `backend-go/internal/rag/vector_store.go` (Already exists)

The vector store implementation is already in place with Qdrant support. Key features:
- Document indexing with metadata
- Similarity search
- Collection management
- Statistics and monitoring

**No changes needed** - This component is ready to use.

### C. Integrate with Existing AI Service

**Location**: `backend-go/internal/ai/service.go`

Extend the existing AI service with RAG-enhanced methods:

```go
// backend-go/internal/ai/service.go
package ai

import (
    "github.com/sql-studio/backend-go/internal/rag"
)

type VectorStore struct {
    client     *qdrant.Client
    embedder   *EmbeddingService
    cache      *redis.Client
}

type Document struct {
    ID         string                 `json:"id"`
    Content    string                 `json:"content"`
    Embedding  []float32             `json:"embedding"`
    Metadata   map[string]interface{} `json:"metadata"`
    Type       DocumentType          `json:"type"`
    Score      float32               `json:"score"`
}

type DocumentType string
const (
    SchemaDoc   DocumentType = "schema"
    QueryDoc    DocumentType = "query"
    BusinessDoc DocumentType = "business"
    PerfDoc     DocumentType = "performance"
)

func (v *VectorStore) IndexSchema(conn *DatabaseConnection) error {
    // Extract schema information
    tables := conn.GetTables()

    for _, table := range tables {
        // Create document for table
        doc := Document{
            ID:      fmt.Sprintf("schema_%s_%s", conn.ID, table.Name),
            Content: v.buildTableDescription(table),
            Type:    SchemaDoc,
            Metadata: map[string]interface{}{
                "connection_id": conn.ID,
                "table_name":   table.Name,
                "columns":      table.Columns,
                "row_count":    table.RowCount,
            },
        }

        // Generate embedding
        doc.Embedding = v.embedder.Embed(doc.Content)

        // Store in vector database
        v.client.Upsert(doc)
    }

    return nil
}

func (v *VectorStore) SearchSimilar(query string, filters map[string]interface{}) ([]Document, error) {
    // Generate query embedding
    queryEmbedding := v.embedder.Embed(query)

    // Search with filters
    results := v.client.Search(SearchRequest{
        Vector:  queryEmbedding,
        TopK:    10,
        Filters: filters,
    })

    return results, nil
}
```

### D. Extend AI Service with RAG Methods

**Location**: `backend-go/internal/ai/service.go`

Add RAG-enhanced SQL generation to the existing AI service:

```go
// backend-go/internal/ai/service.go
package ai

// Add to serviceImpl struct:
type serviceImpl struct {
    config       *Config
    providers    map[Provider]AIProvider
    logger       *logrus.Logger
    usage        map[Provider]*Usage
    usageMu      sync.RWMutex
    started      bool
    mu           sync.RWMutex
    
    // NEW: RAG components
    ragGenerator *rag.SmartSQLGenerator
    vectorStore  rag.VectorStore
    contextBuilder *rag.ContextBuilder
}

// Add RAG-enhanced methods:

// GenerateSQLWithRAG generates SQL using RAG context
func (s *serviceImpl) GenerateSQLWithRAG(ctx context.Context, req *SQLRequest, connectionID string) (*SQLResponse, error) {
    // Build context from RAG
    ragContext, err := s.contextBuilder.BuildContext(ctx, req.Prompt, connectionID)
    if err != nil {
        s.logger.WithError(err).Warn("Failed to build RAG context, falling back to basic generation")
        return s.GenerateSQL(ctx, req)
    }

    // Use RAG-enhanced generation
    generated, err := s.ragGenerator.Generate(ctx, req.Prompt, connectionID)
    if err != nil {
        return nil, err
    }

    // Convert to SQLResponse format
    response := &SQLResponse{
        Query:       generated.Query,
        Explanation: generated.Explanation,
        Confidence:  float64(generated.Confidence),
        Suggestions: generated.AlternativeQueries,
        Warnings:    generated.Warnings,
        Provider:    req.Provider,
        Model:       req.Model,
        Metadata: map[string]string{
            "relevant_tables": strings.Join(generated.Tables, ","),
            "context_used":    "rag",
        },
    }

    // Update usage stats
    s.updateUsage(req.Provider, req.Model, true, time.Since(time.Now()), response)

    return response, nil
}

// GetQueryContext retrieves RAG context for a query
func (s *serviceImpl) GetQueryContext(ctx context.Context, prompt string, connectionID string) (*rag.QueryContext, error) {
    return s.contextBuilder.BuildContext(ctx, prompt, connectionID)
}

// LearnFromQuery indexes a successful query execution
func (s *serviceImpl) LearnFromQuery(ctx context.Context, execution *QueryExecution) error {
    if s.vectorStore == nil {
        return fmt.Errorf("vector store not initialized")
    }

    // Create document from execution
    doc := &rag.Document{
        ID:           uuid.New().String(),
        ConnectionID: execution.ConnectionID,
        Type:         rag.DocumentTypeQuery,
        Content:      execution.Query,
        Metadata: map[string]interface{}{
            "duration_ms":    execution.Duration.Milliseconds(),
            "rows_returned":  execution.RowCount,
            "success":        execution.Success,
            "tables":         execution.Tables,
            "timestamp":      time.Now(),
        },
    }

    // Index in vector store
    return s.vectorStore.IndexDocument(ctx, doc)
}

// IndexSchema indexes database schema for a connection
func (s *serviceImpl) IndexSchema(ctx context.Context, connectionID string, schema *database.TableStructure) error {
    if s.vectorStore == nil {
        return fmt.Errorf("vector store not initialized")
    }

    // Build schema description
    content := fmt.Sprintf("Table: %s.%s\\n", schema.Table.Schema, schema.Table.Name)
    if schema.Table.Comment != "" {
        content += fmt.Sprintf("Description: %s\\n", schema.Table.Comment)
    }
    
    content += "Columns:\\n"
    for _, col := range schema.Columns {
        content += fmt.Sprintf("  - %s (%s)", col.Name, col.DataType)
        if col.PrimaryKey {
            content += " [PRIMARY KEY]"
        }
        if col.Comment != "" {
            content += fmt.Sprintf(": %s", col.Comment)
        }
        content += "\\n"
    }

    doc := &rag.Document{
        ID:           fmt.Sprintf("schema_%s_%s_%s", connectionID, schema.Table.Schema, schema.Table.Name),
        ConnectionID: connectionID,
        Type:         rag.DocumentTypeSchema,
        Content:      content,
        Metadata: map[string]interface{}{
            "schema":    schema.Table.Schema,
            "table":     schema.Table.Name,
            "row_count": schema.Table.RowCount,
            "columns":   len(schema.Columns),
        },
    }

    return s.vectorStore.IndexDocument(ctx, doc)
}

// SuggestVisualization recommends visualization for query results
func (s *serviceImpl) SuggestVisualization(ctx context.Context, data [][]interface{}, columns []database.ColumnInfo) (*VisualizationSuggestion, error) {
    // Use visualization engine from RAG package
    vizEngine := rag.NewVisualizationEngine(s.logger)
    return vizEngine.SuggestVisualization(data, columns)
}

// QueryExecution represents a query execution for learning
type QueryExecution struct {
    ConnectionID string
    Query        string
    Duration     time.Duration
    RowCount     int64
    Success      bool
    Tables       []string
    Error        string
}

// VisualizationSuggestion represents a visualization recommendation
type VisualizationSuggestion struct {
    ChartType    string                 `json:"chart_type"`
    Config       map[string]interface{} `json:"config"`
    Reason       string                 `json:"reason"`
    Confidence   float32                `json:"confidence"`
    Alternatives []AlternativeViz       `json:"alternatives"`
}

type AlternativeViz struct {
    ChartType  string  `json:"chart_type"`
    Confidence float32 `json:"confidence"`
}
```

### E. Create AIService Wrapper for Wails

**Location**: `services/ai.go` (NEW FILE)

Create a service wrapper that combines AI and RAG for Wails integration:

```go
// services/ai.go
package services

import (
    "context"
    "sync"
    "time"

    "github.com/sirupsen/logrus"
    "github.com/wailsapp/wails/v2/pkg/runtime"

    "github.com/sql-studio/backend-go/internal/ai"
    "github.com/sql-studio/backend-go/internal/rag"
    "github.com/sql-studio/backend-go/pkg/database"
)

// AIService wraps AI and RAG services for Wails
type AIService struct {
    aiService    ai.Service
    ragGenerator *rag.SmartSQLGenerator
    vectorStore  rag.VectorStore
    dbManager    *database.Manager
    logger       *logrus.Logger
    ctx          context.Context
    mu           sync.RWMutex
}

// NewAIService creates a new AI service wrapper
func NewAIService(
    aiConfig *ai.Config,
    ragConfig *rag.Config,
    dbManager *database.Manager,
    logger *logrus.Logger,
) (*AIService, error) {
    // Initialize AI service
    aiService, err := ai.NewService(aiConfig, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create AI service: %w", err)
    }

    // Initialize vector store
    vectorStore, err := rag.NewQdrantVectorStore(ragConfig.VectorStore, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create vector store: %w", err)
    }

    // Initialize context builder
    contextBuilder := rag.NewContextBuilder(vectorStore, dbManager, logger)

    // Initialize RAG SQL generator
    ragGenerator := rag.NewSmartSQLGenerator(contextBuilder, aiService, logger)

    return &AIService{
        aiService:    aiService,
        ragGenerator: ragGenerator,
        vectorStore:  vectorStore,
        dbManager:    dbManager,
        logger:       logger,
    }, nil
}

// SetContext sets the Wails context
func (s *AIService) SetContext(ctx context.Context) {
    s.ctx = ctx
}

// GenerateSQL generates SQL from natural language
func (s *AIService) GenerateSQL(req GenerateSQLRequest) (*GenerateSQLResponse, error) {
    s.logger.WithFields(logrus.Fields{
        "prompt":        req.Prompt,
        "connection_id": req.ConnectionID,
    }).Info("Generating SQL from natural language")

    // Convert to AI request
    aiReq := &ai.SQLRequest{
        Prompt:      req.Prompt,
        Provider:    ai.Provider(req.Provider),
        Model:       req.Model,
        MaxTokens:   req.MaxTokens,
        Temperature: req.Temperature,
    }

    // Generate SQL with RAG context
    generated, err := s.ragGenerator.Generate(s.ctx, req.Prompt, req.ConnectionID)
    if err != nil {
        s.logger.WithError(err).Error("SQL generation failed")
        return &GenerateSQLResponse{
            Error: err.Error(),
        }, nil
    }

    // Emit generation event
    runtime.EventsEmit(s.ctx, "ai:sql-generated", map[string]interface{}{
        "confidence": generated.Confidence,
        "tables":     generated.Tables,
    })

    return &GenerateSQLResponse{
        Query:        generated.Query,
        Explanation:  generated.Explanation,
        Confidence:   generated.Confidence,
        Tables:       generated.Tables,
        Warnings:     generated.Warnings,
        Alternatives: generated.AlternativeQueries,
    }, nil
}

// FixSQL fixes SQL based on error message
func (s *AIService) FixSQL(req FixSQLRequest) (*FixSQLResponse, error) {
    s.logger.WithFields(logrus.Fields{
        "query": req.Query,
        "error": req.Error,
    }).Info("Fixing SQL query")

    aiReq := &ai.SQLRequest{
        Query:       req.Query,
        Error:       req.Error,
        Provider:    ai.Provider(req.Provider),
        Model:       req.Model,
        MaxTokens:   2000,
        Temperature: 0.3,
    }

    response, err := s.aiService.FixSQL(s.ctx, aiReq)
    if err != nil {
        return &FixSQLResponse{
            Error: err.Error(),
        }, nil
    }

    return &FixSQLResponse{
        Query:       response.Query,
        Explanation: response.Explanation,
        Changes:     response.Suggestions,
    }, nil
}

// OptimizeQuery provides query optimization suggestions
func (s *AIService) OptimizeQuery(req OptimizeQueryRequest) (*OptimizationResponse, error) {
    // Get EXPLAIN plan
    explainPlan, err := s.dbManager.ExplainQuery(s.ctx, req.ConnectionID, req.Query)
    if err != nil {
        return nil, err
    }

    // Analyze with AI
    aiReq := &ai.SQLRequest{
        Prompt:      fmt.Sprintf("Optimize this SQL query. EXPLAIN plan:\\n%s\\n\\nQuery:\\n%s", explainPlan, req.Query),
        Query:       req.Query,
        Provider:    ai.Provider(req.Provider),
        Model:       req.Model,
        Temperature: 0.3,
    }

    response, err := s.aiService.GenerateSQL(s.ctx, aiReq)
    if err != nil {
        return nil, err
    }

    return &OptimizationResponse{
        OptimizedQuery:  response.Query,
        Improvements:    response.Suggestions,
        EstimatedGain:   "See explanation",
        Explanation:     response.Explanation,
    }, nil
}

// GetQuerySuggestions provides real-time query suggestions
func (s *AIService) GetQuerySuggestions(partialQuery string, cursorPos int) ([]Suggestion, error) {
    // TODO: Implement suggestion logic
    return []Suggestion{}, nil
}

// SuggestVisualization recommends visualization for results
func (s *AIService) SuggestVisualization(data QueryResultData) (*VizSuggestion, error) {
    viz, err := s.aiService.SuggestVisualization(s.ctx, data.Rows, data.Columns)
    if err != nil {
        return nil, err
    }

    return &VizSuggestion{
        ChartType:    viz.ChartType,
        Config:       viz.Config,
        Reason:       viz.Reason,
        Confidence:   viz.Confidence,
        Alternatives: convertAlternatives(viz.Alternatives),
    }, nil
}

// LearnFromExecution indexes a query execution
func (s *AIService) LearnFromExecution(execution QueryExecutionData) error {
    queryExec := &ai.QueryExecution{
        ConnectionID: execution.ConnectionID,
        Query:        execution.Query,
        Duration:     execution.Duration,
        RowCount:     execution.RowCount,
        Success:      execution.Success,
        Tables:       execution.Tables,
    }

    return s.aiService.LearnFromQuery(s.ctx, queryExec)
}

// IndexSchema indexes schema for a connection
func (s *AIService) IndexSchema(connectionID string, schema *database.TableStructure) error {
    return s.aiService.IndexSchema(s.ctx, connectionID, schema)
}

// GetProviderStatus returns status of AI providers
func (s *AIService) GetProviderStatus() (map[string]ProviderStatus, error) {
    health, err := s.aiService.GetAllProvidersHealth(s.ctx)
    if err != nil {
        return nil, err
    }

    status := make(map[string]ProviderStatus)
    for provider, h := range health {
        status[string(provider)] = ProviderStatus{
            Available:    h.Status == "healthy",
            Status:       h.Status,
            Message:      h.Message,
            ResponseTime: h.ResponseTime,
        }
    }

    return status, nil
}

// Request/Response types for service layer
type GenerateSQLRequest struct {
    Prompt       string  `json:"prompt"`
    ConnectionID string  `json:"connectionId"`
    Provider     string  `json:"provider"`
    Model        string  `json:"model"`
    MaxTokens    int     `json:"maxTokens"`
    Temperature  float64 `json:"temperature"`
}

type GenerateSQLResponse struct {
    Query        string   `json:"query"`
    Explanation  string   `json:"explanation"`
    Confidence   float32  `json:"confidence"`
    Tables       []string `json:"tables"`
    Warnings     []string `json:"warnings"`
    Alternatives []string `json:"alternatives"`
    Error        string   `json:"error,omitempty"`
}

type FixSQLRequest struct {
    Query    string `json:"query"`
    Error    string `json:"error"`
    Provider string `json:"provider"`
    Model    string `json:"model"`
}

type FixSQLResponse struct {
    Query       string   `json:"query"`
    Explanation string   `json:"explanation"`
    Changes     []string `json:"changes"`
    Error       string   `json:"error,omitempty"`
}

type OptimizeQueryRequest struct {
    Query        string `json:"query"`
    ConnectionID string `json:"connectionId"`
    Provider     string `json:"provider"`
    Model        string `json:"model"`
}

type OptimizationResponse struct {
    OptimizedQuery string   `json:"optimizedQuery"`
    Improvements   []string `json:"improvements"`
    EstimatedGain  string   `json:"estimatedGain"`
    Explanation    string   `json:"explanation"`
}

type Suggestion struct {
    Type        string  `json:"type"`
    Text        string  `json:"text"`
    Confidence  float32 `json:"confidence"`
    Explanation string  `json:"explanation"`
}

type VizSuggestion struct {
    ChartType    string                 `json:"chartType"`
    Config       map[string]interface{} `json:"config"`
    Reason       string                 `json:"reason"`
    Confidence   float32                `json:"confidence"`
    Alternatives []AlternativeChart     `json:"alternatives"`
}

type AlternativeChart struct {
    ChartType  string  `json:"chartType"`
    Confidence float32 `json:"confidence"`
}

type QueryResultData struct {
    Columns []database.ColumnInfo `json:"columns"`
    Rows    [][]interface{}       `json:"rows"`
}

type QueryExecutionData struct {
    ConnectionID string        `json:"connectionId"`
    Query        string        `json:"query"`
    Duration     time.Duration `json:"duration"`
    RowCount     int64         `json:"rowCount"`
    Success      bool          `json:"success"`
    Tables       []string      `json:"tables"`
}

type ProviderStatus struct {
    Available    bool          `json:"available"`
    Status       string        `json:"status"`
    Message      string        `json:"message"`
    ResponseTime time.Duration `json:"responseTime"`
}
```

### F. Context Builder Service

**Location**: `backend-go/internal/rag/context_builder.go` (Already exists)

The context builder implementation is already in place. Key features:
- Retrieves relevant schemas from vector store
- Finds similar successful queries
- Gets performance hints
- Builds comprehensive context for AI

**No changes needed** - Use existing implementation.

```go
// backend-go/internal/rag/context_builder.go
package rag

type ContextBuilder struct {
    vectorStore *VectorStore
    queryHistory *QueryHistoryService
    statsService *StatisticsService
}

type QueryContext struct {
    RelevantSchemas   []SchemaContext      `json:"relevant_schemas"`
    SimilarQueries    []QueryPattern       `json:"similar_queries"`
    BusinessRules     []BusinessRule       `json:"business_rules"`
    PerformanceHints  []PerformanceHint    `json:"performance_hints"`
    DataStatistics    map[string]DataStats `json:"data_statistics"`
    Confidence        float32              `json:"confidence"`
}

func (c *ContextBuilder) BuildContext(naturalLanguage string, connectionIDs []string) (*QueryContext, error) {
    context := &QueryContext{}

    // 1. Find relevant schemas
    schemaDocs, _ := c.vectorStore.SearchSimilar(naturalLanguage, map[string]interface{}{
        "type": "schema",
        "connection_id": connectionIDs,
    })
    context.RelevantSchemas = c.extractSchemas(schemaDocs)

    // 2. Find similar successful queries
    queryDocs, _ := c.vectorStore.SearchSimilar(naturalLanguage, map[string]interface{}{
        "type": "query",
        "status": "success",
    })
    context.SimilarQueries = c.extractPatterns(queryDocs)

    // 3. Get business rules
    businessDocs, _ := c.vectorStore.SearchSimilar(naturalLanguage, map[string]interface{}{
        "type": "business",
    })
    context.BusinessRules = c.extractRules(businessDocs)

    // 4. Get performance hints
    context.PerformanceHints = c.getPerformanceHints(context.RelevantSchemas)

    // 5. Get data statistics
    context.DataStatistics = c.statsService.GetStats(context.RelevantSchemas)

    // 6. Calculate confidence
    context.Confidence = c.calculateConfidence(schemaDocs, queryDocs)

    return context, nil
}
```

### G. Wails App Layer - AI Methods

**Location**: `app.go`

Add AI-related fields and Wails-exported methods:

```go
// app.go
package main

// Add to App struct:
type App struct {
    ctx             context.Context
    logger          *logrus.Logger
    databaseService *services.DatabaseService
    fileService     *services.FileService
    keyboardService *services.KeyboardService
    aiService       *services.AIService  // NEW
}

// Update OnStartup to initialize AI service:
func (a *App) OnStartup(ctx context.Context) {
    a.ctx = ctx
    
    // Initialize existing services...
    a.databaseService.SetContext(ctx)
    
    // Initialize AI service if configured
    aiConfig := loadAIConfig()  // Load from config file
    if aiConfig.RAG.Enabled {
        aiService, err := services.NewAIService(aiConfig, ragConfig, a.databaseService.manager, a.logger)
        if err != nil {
            a.logger.WithError(err).Warn("Failed to initialize AI service")
        } else {
            a.aiService = aiService
            a.aiService.SetContext(ctx)
            a.logger.Info("AI/RAG service initialized")
        }
    }
}

// Wails-exported AI methods:

type NLQueryRequest struct {
    Prompt       string `json:"prompt"`
    ConnectionID string `json:"connectionId"`
    Provider     string `json:"provider,omitempty"`
    Model        string `json:"model,omitempty"`
}

type GeneratedSQLResponse struct {
    Query        string   `json:"query"`
    Explanation  string   `json:"explanation"`
    Confidence   float32  `json:"confidence"`
    Tables       []string `json:"tables"`
    Warnings     []string `json:"warnings"`
    Alternatives []string `json:"alternatives"`
    Error        string   `json:"error,omitempty"`
}

// GenerateSQLFromNaturalLanguage converts natural language to SQL using RAG
func (a *App) GenerateSQLFromNaturalLanguage(req NLQueryRequest) (*GeneratedSQLResponse, error) {
    if a.aiService == nil {
        return &GeneratedSQLResponse{
            Error: "AI service not initialized",
        }, nil
    }

    a.logger.WithFields(logrus.Fields{
        "prompt":        req.Prompt,
        "connection_id": req.ConnectionID,
    }).Info("Generating SQL from natural language")

    serviceReq := services.GenerateSQLRequest{
        Prompt:       req.Prompt,
        ConnectionID: req.ConnectionID,
        Provider:     req.Provider,
        Model:        req.Model,
        MaxTokens:    2000,
        Temperature:  0.7,
    }

    response, err := a.aiService.GenerateSQL(serviceReq)
    if err != nil {
        return &GeneratedSQLResponse{
            Error: err.Error(),
        }, nil
    }

    return (*GeneratedSQLResponse)(response), nil
}

// FixSQLError attempts to fix a SQL query based on error message
func (a *App) FixSQLError(query string, errorMsg string) (*FixedSQLResponse, error) {
    if a.aiService == nil {
        return nil, fmt.Errorf("AI service not initialized")
    }

    req := services.FixSQLRequest{
        Query:    query,
        Error:    errorMsg,
        Provider: "openai",  // Use default provider
        Model:    "gpt-4",
    }

    return a.aiService.FixSQL(req)
}

// OptimizeQuery provides optimization suggestions for a query
func (a *App) OptimizeQuery(query string, connectionID string) (*OptimizationResponse, error) {
    if a.aiService == nil {
        return nil, fmt.Errorf("AI service not initialized")
    }

    req := services.OptimizeQueryRequest{
        Query:        query,
        ConnectionID: connectionID,
        Provider:     "openai",
        Model:        "gpt-4",
    }

    return a.aiService.OptimizeQuery(req)
}

// GetQuerySuggestions provides real-time query completion suggestions
func (a *App) GetQuerySuggestions(partialQuery string, cursorPos int) ([]Suggestion, error) {
    if a.aiService == nil {
        return []Suggestion{}, nil
    }

    return a.aiService.GetQuerySuggestions(partialQuery, cursorPos)
}

// SuggestVisualization recommends visualization for query results
func (a *App) SuggestVisualization(resultData ResultData) (*VizSuggestion, error) {
    if a.aiService == nil {
        return nil, fmt.Errorf("AI service not initialized")
    }

    data := services.QueryResultData{
        Columns: resultData.Columns,
        Rows:    resultData.Rows,
    }

    return a.aiService.SuggestVisualization(data)
}

// GetAIProviderStatus returns health status of AI providers
func (a *App) GetAIProviderStatus() (map[string]ProviderStatus, error) {
    if a.aiService == nil {
        return nil, fmt.Errorf("AI service not initialized")
    }

    return a.aiService.GetProviderStatus()
}

// ConfigureAIProvider updates AI provider configuration
func (a *App) ConfigureAIProvider(provider string, config ProviderConfig) error {
    if a.aiService == nil {
        return fmt.Errorf("AI service not initialized")
    }

    // TODO: Implement provider configuration update
    return nil
}

// Hook into query execution to learn from queries
func (a *App) ExecuteQuery(req QueryRequest) (*QueryResponse, error) {
    // ... existing query execution code ...
    
    result, err := a.databaseService.ExecuteQuery(req.ConnectionID, req.Query, options)
    
    // Learn from successful execution
    if err == nil && a.aiService != nil {
        go func() {
            execution := services.QueryExecutionData{
                ConnectionID: req.ConnectionID,
                Query:        req.Query,
                Duration:     result.Duration,
                RowCount:     result.RowCount,
                Success:      true,
                Tables:       extractTables(req.Query),
            }
            a.aiService.LearnFromExecution(execution)
        }()
    }
    
    return result, err
}

// Hook into connection creation to index schema
func (a *App) CreateConnection(req ConnectionRequest) (*ConnectionInfo, error) {
    // ... existing connection creation code ...
    
    connection, err := a.databaseService.CreateConnection(config)
    if err != nil {
        return nil, err
    }
    
    // Index schema for RAG
    if a.aiService != nil {
        go func() {
            schemas, _ := a.databaseService.GetSchemas(connection.ID)
            for _, schema := range schemas {
                tables, _ := a.databaseService.GetTables(connection.ID, schema)
                for _, table := range tables {
                    structure, _ := a.databaseService.GetTableStructure(connection.ID, schema, table.Name)
                    a.aiService.IndexSchema(connection.ID, structure)
                }
            }
        }()
    }
    
    return connection, nil
}
```

## 2. Frontend Integration

### A. AI Query Editor Component

**Location**: `frontend/src/components/ai-query-editor.tsx`

```tsx
// frontend/src/components/ai-query-editor.tsx
import { useState } from 'react'
import { GenerateSQLFromNaturalLanguage, GetQuerySuggestions } from '../../wailsjs/go/main/App'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Card } from './ui/card'
import { Sparkles, CheckCircle, AlertCircle } from 'lucide-react'

interface AIQueryEditorProps {
  connectionId: string
  onQueryGenerated?: (query: string) => void
}

export function AIQueryEditor({ connectionId, onQueryGenerated }: AIQueryEditorProps) {
  const [prompt, setPrompt] = useState('')
  const [generatedSQL, setGeneratedSQL] = useState<GeneratedSQLResponse | null>(null)
  const [loading, setLoading] = useState(false)

  const handleGenerate = async () => {
    if (!prompt.trim()) return

    setLoading(true)
    try {
      const response = await GenerateSQLFromNaturalLanguage({
        prompt,
        connectionId,
        provider: 'openai',
        model: 'gpt-4',
      })

      setGeneratedSQL(response)
      if (response.query && onQueryGenerated) {
        onQueryGenerated(response.query)
      }
    } catch (error) {
      console.error('SQL generation failed:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      {/* Natural Language Input */}
      <div className="flex gap-2">
        <Input
          placeholder="Describe what you want in plain English..."
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          onKeyPress={(e) => e.key === 'Enter' && handleGenerate()}
          className="flex-1"
        />
        <Button onClick={handleGenerate} disabled={loading || !prompt.trim()}>
          <Sparkles className="h-4 w-4 mr-2" />
          {loading ? 'Generating...' : 'Generate SQL'}
        </Button>
      </div>

      {/* Generated SQL Display */}
      {generatedSQL && (
        <Card className="p-4 space-y-3">
          {/* Confidence Indicator */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              {generatedSQL.confidence > 0.7 ? (
                <CheckCircle className="h-5 w-5 text-green-500" />
              ) : (
                <AlertCircle className="h-5 w-5 text-yellow-500" />
              )}
              <span className="text-sm font-medium">
                Confidence: {(generatedSQL.confidence * 100).toFixed(0)}%
              </span>
            </div>
            {generatedSQL.tables && (
              <span className="text-xs text-muted-foreground">
                Tables: {generatedSQL.tables.join(', ')}
              </span>
            )}
          </div>

          {/* SQL Code */}
          <pre className="p-3 bg-muted rounded-md overflow-x-auto">
            <code>{generatedSQL.query}</code>
          </pre>

          {/* Explanation */}
          {generatedSQL.explanation && (
            <div className="text-sm text-muted-foreground">
              {generatedSQL.explanation}
            </div>
          )}

          {/* Warnings */}
          {generatedSQL.warnings && generatedSQL.warnings.length > 0 && (
            <div className="space-y-1">
              {generatedSQL.warnings.map((warning, i) => (
                <div key={i} className="flex items-start gap-2 text-sm text-yellow-600">
                  <AlertCircle className="h-4 w-4 mt-0.5" />
                  <span>{warning}</span>
                </div>
              ))}
            </div>
          )}

          {/* Alternatives */}
          {generatedSQL.alternatives && generatedSQL.alternatives.length > 0 && (
            <details className="text-sm">
              <summary className="cursor-pointer font-medium">
                Alternative Queries ({generatedSQL.alternatives.length})
              </summary>
              <div className="mt-2 space-y-2">
                {generatedSQL.alternatives.map((alt, i) => (
                  <pre key={i} className="p-2 bg-muted/50 rounded text-xs overflow-x-auto">
                    <code>{alt}</code>
                  </pre>
                ))}
              </div>
            </details>
          )}
        </Card>
      )}
    </div>
  )
}
```

### B. Query Context Panel

**Location**: `frontend/src/components/query-context-panel.tsx`

```tsx
// frontend/src/components/query-context-panel.tsx
import { useEffect, useState } from 'react'
import { Card } from './ui/card'
import { Badge } from './ui/badge'
import { Database, History, Zap } from 'lucide-react'

interface QueryContextPanelProps {
  connectionId: string
  query: string
}

export function QueryContextPanel({ connectionId, query }: QueryContextPanelProps) {
  const [context, setContext] = useState<QueryContext | null>(null)

  useEffect(() => {
    if (query && connectionId) {
      // Fetch context from backend (implement API call)
      // This would call a backend method to get RAG context
    }
  }, [query, connectionId])

  return (
    <div className="space-y-4">
      {/* Relevant Tables */}
      <Card className="p-4">
        <h4 className="text-sm font-semibold flex items-center gap-2 mb-3">
          <Database className="h-4 w-4" />
          Relevant Tables
        </h4>
        {context?.relevantSchemas?.map((schema, i) => (
          <Badge key={i} variant="outline" className="mr-2 mb-2">
            {schema.table}
          </Badge>
        ))}
      </Card>

      {/* Similar Queries */}
      <Card className="p-4">
        <h4 className="text-sm font-semibold flex items-center gap-2 mb-3">
          <History className="h-4 w-4" />
          Similar Queries
        </h4>
        {context?.similarQueries?.map((similar, i) => (
          <div key={i} className="text-xs p-2 bg-muted rounded mb-2">
            <code>{similar.query}</code>
          </div>
        ))}
      </Card>

      {/* Performance Hints */}
      <Card className="p-4">
        <h4 className="text-sm font-semibold flex items-center gap-2 mb-3">
          <Zap className="h-4 w-4" />
          Performance Hints
        </h4>
        {context?.performanceHints?.map((hint, i) => (
          <div key={i} className="text-sm mb-2">
            {hint.message}
          </div>
        ))}
      </Card>
    </div>
  )
}
```

## 3. Configuration Management

**Location**: `backend-go/configs/config.yaml`

```yaml
# AI and RAG configuration
ai:
  enabled: true
  default_provider: openai
  
  openai:
    api_key: ${OPENAI_API_KEY}
    model: gpt-4
    max_tokens: 2000
    temperature: 0.7
    
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    model: claude-3-sonnet-20240229
    
  ollama:
    endpoint: http://localhost:11434
    model: llama2

# RAG configuration
rag:
  enabled: true
  
  vector_store:
    type: qdrant
    url: http://localhost:6333
    collection_prefix: howlerops
    api_key: ${QDRANT_API_KEY}
    
  embedding:
    provider: openai
    model: text-embedding-3-small
    dimension: 1536
    batch_size: 100
    
  learning:
    enabled: true
    min_confidence: 0.7
    auto_index_schemas: true
    index_on_connection: true
    learn_from_queries: true
    min_query_success_rate: 0.8
    
  context:
    max_relevant_tables: 10
    max_similar_queries: 5
    similarity_threshold: 0.75
```

## 4. Testing Strategy

### Backend Tests

**Location**: `backend-go/internal/rag/context_builder_test.go`

```go
package rag

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestContextBuilder_BuildContext(t *testing.T) {
    // Mock vector store
    mockVectorStore := &MockVectorStore{}
    builder := NewContextBuilder(mockVectorStore, nil, nil)
    
    ctx := context.Background()
    prompt := "Show me all users who joined last month"
    connectionID := "test-conn"
    
    queryContext, err := builder.BuildContext(ctx, prompt, connectionID)
    
    assert.NoError(t, err)
    assert.NotNil(t, queryContext)
    assert.Greater(t, len(queryContext.RelevantSchemas), 0)
}
```

### Frontend Tests

**Location**: `frontend/src/__tests__/ai-query-editor.test.tsx`

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { AIQueryEditor } from '../components/ai-query-editor'
import { GenerateSQLFromNaturalLanguage } from '../../wailsjs/go/main/App'

jest.mock('../../wailsjs/go/main/App')

describe('AIQueryEditor', () => {
  it('generates SQL from natural language', async () => {
    const mockGenerate = GenerateSQLFromNaturalLanguage as jest.Mock
    mockGenerate.mockResolvedValue({
      query: 'SELECT * FROM users',
      explanation: 'Retrieves all users',
      confidence: 0.95
    })
    
    const onQueryGenerated = jest.fn()
    render(<AIQueryEditor connectionId="test" onQueryGenerated={onQueryGenerated} />)
    
    const input = screen.getByPlaceholderText(/describe what you want/i)
    fireEvent.change(input, { target: { value: 'show me all users' } })
    
    const button = screen.getByText(/generate sql/i)
    fireEvent.click(button)
    
    await waitFor(() => {
      expect(mockGenerate).toHaveBeenCalled()
      expect(onQueryGenerated).toHaveBeenCalledWith('SELECT * FROM users')
    })
  })
})
```

## 5. Dependencies

### Backend Dependencies

**Add to `backend-go/go.mod`:**

```
require (
    github.com/qdrant/go-client v1.7.0
    github.com/sashabaranov/go-openai v1.17.9
    github.com/redis/go-redis/v9 v9.3.0
    github.com/google/uuid v1.5.0
)
```

### Frontend Dependencies

**Add to `frontend/package.json`:**

```json
{
  "dependencies": {
    "lucide-react": "^0.295.0",
    "@tanstack/react-query": "^5.8.4"
  }
}
```

## 6. Implementation Checklist

### Phase 1: AI Service Extensions (Week 1-2)
- [ ] Extend `backend-go/internal/ai/service.go` with RAG methods
- [ ] Add `GenerateSQLWithRAG`, `IndexSchema`, `LearnFromQuery`
- [ ] Test AI service integration

### Phase 2: Service Wrapper (Week 3)
- [ ] Create `services/ai.go` wrapper
- [ ] Implement all wrapper methods
- [ ] Add Wails event emissions

### Phase 3: Wails Bindings (Week 4)
- [ ] Add AI methods to `app.go`
- [ ] Define request/response types
- [ ] Hook into query execution for learning
- [ ] Hook into connection creation for schema indexing

### Phase 4: Frontend Components (Week 5-7)
- [ ] Create `ai-query-editor.tsx`
- [ ] Create `query-context-panel.tsx`
- [ ] Create `smart-visualization.tsx`
- [ ] Implement real-time suggestions

### Phase 5: Learning Pipeline (Week 8)
- [ ] Integrate learning into query execution
- [ ] Auto-index schemas on connection
- [ ] Test learning accuracy

### Phase 6: Testing & Polish (Week 9-10)
- [ ] Backend unit tests
- [ ] Frontend component tests
- [ ] E2E testing
- [ ] Performance optimization

## 2. Smart SQL Generation with RAG

### A. Enhanced SQL Generator

```go
// backend-go/internal/rag/smart_sql_generator.go
package rag

type SmartSQLGenerator struct {
    aiProvider    ai.Provider
    contextBuilder *ContextBuilder
    optimizer     *QueryOptimizer
}

type GenerateRequest struct {
    NaturalLanguage string   `json:"natural_language"`
    Connections     []string `json:"connections"`
    Options         Options  `json:"options"`
}

type GenerateResponse struct {
    SQL           string            `json:"sql"`
    Explanation   string            `json:"explanation"`
    Confidence    float32           `json:"confidence"`
    Alternatives  []Alternative     `json:"alternatives"`
    Optimizations []Optimization    `json:"optimizations"`
    Warnings      []string          `json:"warnings"`
}

func (g *SmartSQLGenerator) Generate(req GenerateRequest) (*GenerateResponse, error) {
    // 1. Build context from RAG
    context, err := g.contextBuilder.BuildContext(req.NaturalLanguage, req.Connections)
    if err != nil {
        return nil, err
    }

    // 2. Create enhanced prompt with context
    prompt := g.buildEnhancedPrompt(req.NaturalLanguage, context)

    // 3. Generate SQL using AI with context
    response, err := g.aiProvider.GenerateSQL(prompt)
    if err != nil {
        return nil, err
    }

    // 4. Optimize the generated SQL
    optimized := g.optimizer.Optimize(response.SQL, context)

    // 5. Generate alternatives
    alternatives := g.generateAlternatives(req, context)

    // 6. Validate against business rules
    warnings := g.validateBusinessRules(optimized.SQL, context.BusinessRules)

    return &GenerateResponse{
        SQL:           optimized.SQL,
        Explanation:   response.Explanation,
        Confidence:    context.Confidence,
        Alternatives:  alternatives,
        Optimizations: optimized.Suggestions,
        Warnings:      warnings,
    }, nil
}

func (g *SmartSQLGenerator) buildEnhancedPrompt(query string, context *QueryContext) string {
    prompt := fmt.Sprintf(`
Given the user query: "%s"

Available Schema:
%s

Similar Successful Queries:
%s

Business Rules:
%s

Data Statistics:
- Total rows in relevant tables: %v
- Common JOIN patterns: %v

Generate an optimized SQL query that:
1. Answers the user's question accurately
2. Follows the established query patterns
3. Uses appropriate indexes
4. Handles NULL values correctly
5. Includes helpful comments

SQL:
`, query,
   g.formatSchemas(context.RelevantSchemas),
   g.formatQueries(context.SimilarQueries),
   g.formatRules(context.BusinessRules),
   context.DataStatistics)

   return prompt
}
```

### B. Multi-Step Query Planning

```go
// backend-go/internal/rag/query_planner.go
package rag

type QueryPlanner struct {
    generator *SmartSQLGenerator
    executor  *QueryExecutor
}

type QueryPlan struct {
    Steps        []QueryStep    `json:"steps"`
    Dependencies []Dependency   `json:"dependencies"`
    TotalCost    float64       `json:"total_cost"`
}

type QueryStep struct {
    ID          string   `json:"id"`
    Description string   `json:"description"`
    SQL         string   `json:"sql"`
    DependsOn   []string `json:"depends_on"`
    OutputType  string   `json:"output_type"`
}

func (p *QueryPlanner) PlanComplexQuery(request string) (*QueryPlan, error) {
    // Break down complex request into steps
    // Example: "Show me user retention by cohort with revenue impact"

    plan := &QueryPlan{
        Steps: []QueryStep{
            {
                ID:          "cohort_definition",
                Description: "Define user cohorts by signup month",
                SQL:         "CREATE TEMP TABLE cohorts AS ...",
            },
            {
                ID:          "retention_calc",
                Description: "Calculate retention rates",
                SQL:         "WITH retention AS ...",
                DependsOn:   []string{"cohort_definition"},
            },
            {
                ID:          "revenue_impact",
                Description: "Join with revenue data",
                SQL:         "SELECT ... FROM retention JOIN revenue ...",
                DependsOn:   []string{"retention_calc"},
            },
        },
    }

    return plan, nil
}
```

## 3. Query Performance Optimization

### A. Query Analyzer

```go
// backend-go/internal/rag/query_analyzer.go
package rag

type QueryAnalyzer struct {
    explainParser *ExplainParser
    indexAdvisor  *IndexAdvisor
    costEstimator *CostEstimator
}

type AnalysisResult struct {
    ExecutionPlan    *ExecutionPlan      `json:"execution_plan"`
    BottleNecks     []BottleNeck        `json:"bottlenecks"`
    IndexSuggestions []IndexSuggestion   `json:"index_suggestions"`
    RewriteSuggestions []QueryRewrite    `json:"rewrite_suggestions"`
    EstimatedRuntime Duration            `json:"estimated_runtime"`
}

func (a *QueryAnalyzer) AnalyzeQuery(sql string, connectionID string) (*AnalysisResult, error) {
    // 1. Get execution plan
    plan, err := a.explainParser.GetPlan(sql, connectionID)
    if err != nil {
        return nil, err
    }

    // 2. Identify bottlenecks
    bottlenecks := a.identifyBottlenecks(plan)

    // 3. Suggest indexes
    indexSuggestions := a.indexAdvisor.SuggestIndexes(plan, bottlenecks)

    // 4. Suggest query rewrites
    rewrites := a.suggestRewrites(sql, bottlenecks)

    // 5. Estimate runtime
    runtime := a.costEstimator.EstimateRuntime(plan)

    return &AnalysisResult{
        ExecutionPlan:    plan,
        BottleNecks:     bottlenecks,
        IndexSuggestions: indexSuggestions,
        RewriteSuggestions: rewrites,
        EstimatedRuntime: runtime,
    }, nil
}

func (a *QueryAnalyzer) identifyBottlenecks(plan *ExecutionPlan) []BottleNeck {
    bottlenecks := []BottleNeck{}

    for _, node := range plan.Nodes {
        // Check for table scans
        if node.Type == "SeqScan" && node.Rows > 10000 {
            bottlenecks = append(bottlenecks, BottleNeck{
                Type:        "FullTableScan",
                Table:       node.Table,
                Impact:      "High",
                Description: fmt.Sprintf("Table scan on %s with %d rows", node.Table, node.Rows),
            })
        }

        // Check for missing indexes
        if node.Type == "Filter" && node.Cost > 1000 {
            bottlenecks = append(bottlenecks, BottleNeck{
                Type:        "MissingIndex",
                Table:       node.Table,
                Impact:      "Medium",
                Description: fmt.Sprintf("Expensive filter on %s", node.Table),
            })
        }
    }

    return bottlenecks
}
```

### B. Real-time Query Suggestions

```go
// backend-go/internal/rag/realtime_suggestions.go
package rag

type RealtimeSuggestionService struct {
    vectorStore    *VectorStore
    queryParser    *QueryParser
    contextBuilder *ContextBuilder
}

type Suggestion struct {
    Type        string  `json:"type"` // completion, correction, optimization
    Text        string  `json:"text"`
    Confidence  float32 `json:"confidence"`
    Explanation string  `json:"explanation"`
}

func (r *RealtimeSuggestionService) GetSuggestions(partialQuery string, cursorPosition int) ([]Suggestion, error) {
    suggestions := []Suggestion{}

    // 1. Parse partial query
    parsed, _ := r.queryParser.ParsePartial(partialQuery, cursorPosition)

    // 2. Determine context (what is user trying to write?)
    context := r.determineContext(parsed)

    switch context.Type {
    case "table_selection":
        // Suggest relevant tables based on previous tokens
        tables := r.suggestTables(parsed)
        for _, table := range tables {
            suggestions = append(suggestions, Suggestion{
                Type:       "completion",
                Text:       table.Name,
                Confidence: table.Relevance,
            })
        }

    case "join_condition":
        // Suggest JOIN conditions based on foreign keys
        joins := r.suggestJoins(parsed)
        for _, join := range joins {
            suggestions = append(suggestions, Suggestion{
                Type:        "completion",
                Text:        join.Condition,
                Confidence:  0.95,
                Explanation: "Based on foreign key relationship",
            })
        }

    case "where_clause":
        // Suggest common filters
        filters := r.suggestFilters(parsed)
        for _, filter := range filters {
            suggestions = append(suggestions, Suggestion{
                Type:       "completion",
                Text:       filter.Clause,
                Confidence: filter.Frequency,
            })
        }
    }

    // 3. Check for potential errors
    if error := r.checkForErrors(partialQuery); error != nil {
        suggestions = append(suggestions, Suggestion{
            Type:        "correction",
            Text:        error.Correction,
            Explanation: error.Reason,
            Confidence:  0.9,
        })
    }

    return suggestions, nil
}
```

## 4. Intelligent Visualization Engine

### A. Auto-Visualization Detection

```go
// backend-go/internal/rag/visualization_engine.go
package rag

type VisualizationEngine struct {
    dataAnalyzer *DataAnalyzer
    chartSelector *ChartSelector
    aggregator   *DataAggregator
}

type VisualizationSuggestion struct {
    ChartType    string                 `json:"chart_type"`
    Config       map[string]interface{} `json:"config"`
    Reason       string                 `json:"reason"`
    Confidence   float32                `json:"confidence"`
    Alternatives []Alternative          `json:"alternatives"`
}

func (v *VisualizationEngine) SuggestVisualization(data [][]interface{}, columns []Column) (*VisualizationSuggestion, error) {
    // 1. Analyze data characteristics
    analysis := v.dataAnalyzer.Analyze(data, columns)

    // 2. Determine best chart type
    suggestion := &VisualizationSuggestion{}

    switch {
    case analysis.HasTimeSeries && analysis.NumericColumns == 1:
        suggestion.ChartType = "line"
        suggestion.Config = map[string]interface{}{
            "x": analysis.TimeColumn,
            "y": analysis.NumericColumns[0],
        }
        suggestion.Reason = "Time series data with single metric"

    case analysis.HasCategories && analysis.NumericColumns > 0:
        if len(analysis.Categories) < 10 {
            suggestion.ChartType = "bar"
        } else {
            suggestion.ChartType = "horizontal_bar"
        }
        suggestion.Config = map[string]interface{}{
            "category": analysis.CategoryColumn,
            "value":    analysis.NumericColumns[0],
        }
        suggestion.Reason = "Categorical comparison"

    case analysis.NumericColumns >= 2:
        suggestion.ChartType = "scatter"
        suggestion.Config = map[string]interface{}{
            "x": analysis.NumericColumns[0],
            "y": analysis.NumericColumns[1],
        }
        suggestion.Reason = "Correlation between numeric values"

    case analysis.IsDistribution:
        suggestion.ChartType = "histogram"
        suggestion.Config = map[string]interface{}{
            "value": analysis.NumericColumns[0],
            "bins":  20,
        }
        suggestion.Reason = "Distribution of values"
    }

    // 3. Generate alternatives
    suggestion.Alternatives = v.generateAlternatives(analysis)

    // 4. Calculate confidence
    suggestion.Confidence = v.calculateConfidence(analysis, suggestion.ChartType)

    return suggestion, nil
}
```

### B. Natural Language to Visualization

```go
// backend-go/internal/rag/nl_to_viz.go
package rag

type NLToVizService struct {
    nlParser     *NaturalLanguageParser
    sqlGenerator *SmartSQLGenerator
    vizEngine    *VisualizationEngine
}

func (n *NLToVizService) GenerateVisualization(request string, connectionID string) (*Visualization, error) {
    // Example: "Show me monthly revenue as a line chart"

    // 1. Parse visualization intent
    intent := n.nlParser.ParseVisualizationIntent(request)
    // intent = { metric: "revenue", timeframe: "monthly", chartType: "line" }

    // 2. Generate SQL for the data
    sqlRequest := GenerateRequest{
        NaturalLanguage: intent.DataQuery,
        Connections:     []string{connectionID},
    }
    sqlResponse, _ := n.sqlGenerator.Generate(sqlRequest)

    // 3. Execute query
    data, columns, _ := n.executeQuery(sqlResponse.SQL, connectionID)

    // 4. Apply specified visualization or auto-detect
    var viz *VisualizationSuggestion
    if intent.ChartType != "" {
        viz = n.vizEngine.ForceChartType(data, columns, intent.ChartType)
    } else {
        viz, _ = n.vizEngine.SuggestVisualization(data, columns)
    }

    // 5. Apply any specific configurations
    if intent.Aggregation != "" {
        data = n.applyAggregation(data, columns, intent.Aggregation)
    }

    return &Visualization{
        SQL:       sqlResponse.SQL,
        Data:      data,
        ChartType: viz.ChartType,
        Config:    viz.Config,
    }, nil
}
```

## 5. Learning Pipeline

### A. Query History Learning

```go
// backend-go/internal/rag/learning_pipeline.go
package rag

type LearningPipeline struct {
    vectorStore   *VectorStore
    patternExtractor *PatternExtractor
    feedbackProcessor *FeedbackProcessor
}

func (l *LearningPipeline) LearnFromExecution(execution QueryExecution) error {
    // 1. Extract patterns from successful queries
    if execution.Success && execution.Runtime < 1000 { // Fast, successful query
        pattern := l.patternExtractor.Extract(execution)

        // 2. Create embedding
        doc := Document{
            ID:      execution.ID,
            Content: execution.SQL,
            Type:    QueryDoc,
            Metadata: map[string]interface{}{
                "pattern":     pattern,
                "runtime_ms":  execution.Runtime,
                "rows_returned": execution.RowCount,
                "user":        execution.UserID,
                "timestamp":   execution.Timestamp,
            },
        }

        // 3. Store in vector database
        l.vectorStore.Index(doc)
    }

    // 4. Learn from errors
    if !execution.Success {
        l.learnFromError(execution)
    }

    // 5. Update statistics
    l.updateStatistics(execution)

    return nil
}

func (l *LearningPipeline) ProcessUserFeedback(feedback UserFeedback) error {
    // Thumbs up/down on generated SQL
    if feedback.Type == "sql_quality" {
        // Update embedding weights
        doc, _ := l.vectorStore.Get(feedback.QueryID)
        if feedback.Positive {
            doc.Score *= 1.1 // Boost good examples
        } else {
            doc.Score *= 0.9 // Reduce bad examples
        }
        l.vectorStore.Update(doc)
    }

    return nil
}
```

## 6. Frontend Integration

### A. RAG-Enhanced Query Editor

```tsx
// frontend/src/components/RAGQueryEditor.tsx
import { useRAGContext } from '@/hooks/useRAGContext'
import { useRealtimeSuggestions } from '@/hooks/useRealtimeSuggestions'

export function RAGQueryEditor() {
  const [query, setQuery] = useState('')
  const [naturalLanguage, setNaturalLanguage] = useState('')
  const { context, loading: contextLoading } = useRAGContext(query)
  const { suggestions } = useRealtimeSuggestions(query)

  const handleGenerateSQL = async () => {
    const response = await generateSmartSQL({
      naturalLanguage,
      connections: selectedConnections,
    })

    setQuery(response.sql)
    setShowExplanation(true)
    setExplanation(response.explanation)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Natural Language Input */}
      <div className="p-4 border-b">
        <div className="flex gap-2">
          <Input
            placeholder="Describe what you want in plain English..."
            value={naturalLanguage}
            onChange={(e) => setNaturalLanguage(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleGenerateSQL()}
          />
          <Button onClick={handleGenerateSQL}>
            Generate SQL
          </Button>
        </div>
      </div>

      {/* SQL Editor with Context Panel */}
      <div className="flex flex-1">
        <div className="flex-1">
          <MonacoEditor
            value={query}
            onChange={setQuery}
            language="sql"
            options={{
              quickSuggestions: true,
              suggestOnTriggerCharacters: true,
            }}
          />

          {/* Real-time suggestions */}
          {suggestions.length > 0 && (
            <div className="absolute bottom-4 right-4 w-80 bg-background border rounded-lg p-3">
              <h4 className="font-semibold mb-2">Suggestions</h4>
              {suggestions.map(suggestion => (
                <SuggestionCard
                  key={suggestion.id}
                  suggestion={suggestion}
                  onApply={() => applySuggestion(suggestion)}
                />
              ))}
            </div>
          )}
        </div>

        {/* Context Panel */}
        <div className="w-80 border-l p-4 overflow-y-auto">
          <h3 className="font-semibold mb-3">Query Context</h3>

          {/* Relevant Tables */}
          <div className="mb-4">
            <h4 className="text-sm font-medium mb-2">Relevant Tables</h4>
            {context?.relevantSchemas.map(schema => (
              <SchemaCard key={schema.id} schema={schema} />
            ))}
          </div>

          {/* Similar Queries */}
          <div className="mb-4">
            <h4 className="text-sm font-medium mb-2">Similar Queries</h4>
            {context?.similarQueries.map(query => (
              <QueryPatternCard key={query.id} pattern={query} />
            ))}
          </div>

          {/* Performance Hints */}
          {context?.performanceHints.length > 0 && (
            <div className="mb-4">
              <h4 className="text-sm font-medium mb-2">Performance Tips</h4>
              {context.performanceHints.map(hint => (
                <Alert key={hint.id}>
                  <AlertDescription>{hint.message}</AlertDescription>
                </Alert>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
```

### B. Smart Visualization Component

```tsx
// frontend/src/components/SmartVisualization.tsx
import { useVisualizationEngine } from '@/hooks/useVisualizationEngine'

export function SmartVisualization({ data, columns }) {
  const { suggestion, alternatives } = useVisualizationEngine(data, columns)
  const [selectedChart, setSelectedChart] = useState(suggestion?.chartType)

  return (
    <div className="p-4">
      {/* Chart Type Selector */}
      <div className="flex items-center gap-4 mb-4">
        <span className="text-sm font-medium">Visualization:</span>
        <Select value={selectedChart} onValueChange={setSelectedChart}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={suggestion.chartType}>
              {suggestion.chartType} (Recommended)
            </SelectItem>
            {alternatives.map(alt => (
              <SelectItem key={alt.type} value={alt.type}>
                {alt.type}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className="text-xs text-muted-foreground">
          {suggestion.reason}
        </span>
      </div>

      {/* Dynamic Chart Rendering */}
      <div className="h-96">
        {selectedChart === 'line' && (
          <LineChart data={data} config={suggestion.config} />
        )}
        {selectedChart === 'bar' && (
          <BarChart data={data} config={suggestion.config} />
        )}
        {selectedChart === 'scatter' && (
          <ScatterPlot data={data} config={suggestion.config} />
        )}
        {/* ... other chart types */}
      </div>

      {/* Natural Language Viz Input */}
      <div className="mt-4">
        <Input
          placeholder="e.g., 'Show this as a pie chart' or 'Group by category'"
          onKeyPress={async (e) => {
            if (e.key === 'Enter') {
              const newViz = await transformVisualization(e.target.value, data)
              setSelectedChart(newViz.chartType)
            }
          }}
        />
      </div>
    </div>
  )
}
```

## 7. Implementation Timeline

### Phase 1: Foundation (Weeks 1-3)
#### Week 1: Vector Database Setup
- [ ] Install and configure Qdrant
- [ ] Create embedding service with OpenAI
- [ ] Design document schema

#### Week 2: Basic Indexing
- [ ] Index existing schemas
- [ ] Create embedding pipeline
- [ ] Implement basic search

#### Week 3: Context Builder
- [ ] Build context retrieval system
- [ ] Create relevance scoring
- [ ] Test with sample queries

### Phase 2: Smart SQL Generation (Weeks 4-6)
#### Week 4: RAG Integration
- [ ] Integrate context with AI providers
- [ ] Build enhanced prompt templates
- [ ] Create feedback loop

#### Week 5: Query Patterns
- [ ] Extract patterns from history
- [ ] Build pattern matching system
- [ ] Create template library

#### Week 6: Multi-step Planning
- [ ] Implement query decomposition
- [ ] Build dependency resolution
- [ ] Create execution planner

### Phase 3: Performance Optimization (Weeks 7-9)
#### Week 7: Query Analysis
- [ ] Build EXPLAIN parser
- [ ] Create bottleneck detection
- [ ] Implement cost estimation

#### Week 8: Index Advisor
- [ ] Build index recommendation engine
- [ ] Create impact analysis
- [ ] Test with real queries

#### Week 9: Query Rewriting
- [ ] Implement rewrite rules
- [ ] Build optimization suggestions
- [ ] Create performance predictor

### Phase 4: Intelligent Visualizations (Weeks 10-11)
#### Week 10: Auto-Detection
- [ ] Build data analyzer
- [ ] Create chart selector
- [ ] Implement config generator

#### Week 11: Natural Language
- [ ] Parse visualization intents
- [ ] Build NL to chart converter
- [ ] Create interactive refinement

### Phase 5: Real-time Assistance (Weeks 12-13)
#### Week 12: Live Suggestions
- [ ] Build real-time parser
- [ ] Create suggestion engine
- [ ] Implement error detection

#### Week 13: Query Completion
- [ ] Build autocomplete provider
- [ ] Create context-aware suggestions
- [ ] Test and optimize latency

### Phase 6: Polish & Testing (Weeks 14-15)
#### Week 14: Integration Testing
- [ ] End-to-end testing
- [ ] Performance benchmarking
- [ ] User acceptance testing

#### Week 15: Documentation & Launch
- [ ] User documentation
- [ ] API documentation
- [ ] Launch preparation

## 8. Success Metrics

### Performance Metrics
- **Context Retrieval**: < 200ms latency
- **SQL Generation**: < 2s for complex queries
- **Suggestion Latency**: < 100ms for real-time
- **Visualization Detection**: < 500ms

### Quality Metrics
- **SQL Accuracy**: > 85% first-try success
- **Context Relevance**: > 0.8 similarity score
- **Optimization Impact**: > 30% query speedup
- **Visualization Match**: > 90% appropriate chart selection

### User Metrics
- **Time Saved**: 50% reduction in query writing time
- **Error Reduction**: 40% fewer SQL errors
- **Feature Adoption**: 70% of users using AI features
- **Satisfaction Score**: > 4.5/5 rating

## 9. Technical Stack

### Backend Dependencies
```go
// go.mod additions
require (
    github.com/qdrant/go-client v1.7.0       // Vector database
    github.com/sashabaranov/go-openai v1.17.9 // Embeddings
    github.com/redis/go-redis/v9 v9.3.0      // Caching
    github.com/montanaflynn/stats v0.7.1     // Statistics
)
```

### Frontend Dependencies
```json
{
  "dependencies": {
    "@tanstack/react-query": "^5.8.4",  // Data fetching
    "recharts": "^2.10.1",               // Charts
    "d3": "^7.8.5",                      // Advanced viz
    "react-markdown": "^9.0.1"           // Explanations
  }
}
```

## 10. Security & Privacy

### Data Protection
- **Local Embeddings**: Option to use local models
- **Encrypted Storage**: All vectors encrypted at rest
- **Query Sanitization**: Prevent injection via RAG
- **Access Control**: Connection-level permissions

### Privacy Controls
- **Opt-in Learning**: Users control what's indexed
- **Data Retention**: Configurable retention policies
- **Audit Logging**: Track all AI interactions
- **Local-First**: Option to run everything locally

## Conclusion

This AI/RAG integration transforms SQL Studio into an intelligent database assistant that understands context, learns from patterns, and helps developers write better queries faster. The phased implementation ensures each component delivers value independently while building toward a comprehensive intelligent system.

The combination of vector search, contextual understanding, and continuous learning creates a uniquely powerful tool that gets better with use and truly understands each organization's specific data landscape.

---

**Status**: Ready for Implementation
**Estimated Duration**: 15 weeks
**Priority**: High
**Dependencies**: Part 1 (Multi-Database Query Support) should be stable