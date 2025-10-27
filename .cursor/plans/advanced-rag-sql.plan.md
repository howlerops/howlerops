<!-- 63bd9552-5437-409c-b2fb-1d5d8a1fc3dc 1eb8661d-c70b-463c-b55d-b3b6f78fb287 -->
# Advanced RAG for SQL Generation - Implementation Plan

## Overview

This plan implements advanced RAG improvements for SQL generation based on three key resources:

1. **gorag library** - Go interface patterns for embeddings and vector operations
2. **Ent Atlas pgvector article** - Schema-aware RAG with relationship graphs
3. **Advanced RAG without DB exposure** - Text-to-SQL patterns without data leakage

## Key Design Decisions

### ✅ ONNX Runtime (Not Ollama)

- **No external service** - Embedding model runs directly in your Go process
- **Truly offline** - Zero network calls for embeddings
- **Simpler deployment** - Just include model file, no daemon management
- **Faster inference** - Direct C bindings, no HTTP overhead
- **Proven approach** - Used by ChromaDB Go client

### 🔒 Security-First: Schema-Only Indexing

- **Never index data values** - Only metadata (table/column names, types, relationships)
- **Privacy guards** - Detect and block PII columns automatically
- **Pattern learning** - Extract SQL templates without actual query values

## Key Improvements

### 1. Local-First Embedding Model (Offline Capability)

**Problem**: Current system uses OpenAI embeddings exclusively, requiring internet and incurring costs.

**Solution**: Implement **ONNX Runtime-based local embedding model** (all-MiniLM-L6-v2) as primary, with OpenAI fallback. No external service required.

**Why ONNX over Ollama**:

- ✅ No daemon/service to run - embedded directly in Go binary
- ✅ Truly offline - no network calls at all
- ✅ Faster - direct C bindings, no HTTP overhead
- ✅ Smaller - ~90MB model file vs Ollama's larger footprint
- ✅ Simpler deployment - just include model file

**Files to create**:

- `backend-go/internal/rag/onnx_embedding_provider.go` - ONNX Runtime embedding provider
- `backend-go/internal/rag/models/` - Directory for ONNX model files
- `backend-go/configs/config.yaml` - Update embedding config

**Implementation**:

```go
// backend-go/internal/rag/onnx_embedding_provider.go (NEW)
// Uses onnxruntime-go to run sentence-transformers models locally

import (
    ort "github.com/yalue/onnxruntime_go"
)

type ONNXEmbeddingProvider struct {
    session    *ort.AdvancedSession
    modelPath  string
    dimension  int
    tokenizer  *Tokenizer // Simple wordpiece tokenizer
}

// NewONNXEmbeddingProvider loads a sentence-transformer ONNX model
func NewONNXEmbeddingProvider(modelPath string) (*ONNXEmbeddingProvider, error) {
    // Initialize ONNX Runtime
    ort.InitializeEnvironment()

    // Load model
    session, err := ort.NewAdvancedSession(modelPath,
        []string{"input_ids", "attention_mask"},
        []string{"sentence_embedding"},
        nil)
    if err != nil {
        return nil, fmt.Errorf("failed to load ONNX model: %w", err)
    }

    return &ONNXEmbeddingProvider{
        session:   session,
        modelPath: modelPath,
        dimension: 384, // all-MiniLM-L6-v2
    }, nil
}

func (o *ONNXEmbeddingProvider) EmbedText(ctx context.Context, text string) ([]float32, error) {
    // Tokenize text
    inputIDs, attentionMask := o.tokenizer.Tokenize(text)

    // Create input tensors
    inputTensor, _ := ort.NewTensor(ort.NewShape(1, len(inputIDs)), inputIDs)
    maskTensor, _ := ort.NewTensor(ort.NewShape(1, len(attentionMask)), attentionMask)

    // Run inference
    outputs, err := o.session.Run([]ort.Value{inputTensor, maskTensor})
    if err != nil {
        return nil, err
    }

    // Extract embedding
    embedding := outputs[0].GetData().([]float32)

    return embedding, nil
}
```

**Model Setup**:

```bash
# Download pre-converted ONNX model
# all-MiniLM-L6-v2: 384 dimensions, ~90MB
# Model available at: https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2

# Option 1: Use optimized ONNX version from Hugging Face
wget https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx

# Option 2: Convert yourself using Optimum
pip install optimum[onnxruntime]
optimum-cli export onnx --model sentence-transformers/all-MiniLM-L6-v2 all-MiniLM-L6-v2-onnx/
```

**Dependencies**:

```bash
# Add to go.mod
go get github.com/yalue/onnxruntime_go
```

**Tokenizer Implementation**:

For simplicity, you can either:

1. Use a simple wordpiece tokenizer in Go
2. Pre-tokenize text via Python bridge (one-time setup)
3. Use the tokenizers from https://github.com/sugarme/tokenizer (Go port of Hugging Face tokenizers)

### 2. Enhanced Schema Indexing with Relationships

**Problem**: Current schema indexing is basic - stores table/column names but lacks relationship context and statistics.

**Solution**: Create rich schema documents with:

- Foreign key relationships as graph edges
- Column statistics (data types, nullability, cardinality estimates)
- ⚠️ **NO sample values** - security risk for PII exposure
- Common JOIN patterns detected from query history

**Files to create/modify**:

- `backend-go/internal/rag/schema_indexer.go` (NEW) - Advanced schema document builder
- `backend-go/internal/rag/relationship_analyzer.go` (NEW) - FK relationship graph builder
- `backend-go/internal/rag/context_builder.go` - Enhance to use relationship context

**Schema Document Structure**:

```go
type EnhancedSchemaDocument struct {
    TableName    string
    Schema       string
    ConnectionID string
    
    // Column-level details
    Columns []ColumnDetail {
        Name         string
        Type         string
        IsNullable   bool
        IsPrimaryKey bool
        IsForeignKey bool
        SampleValues []string  // Top 5 sample values for semantic context
        Cardinality  int64     // Estimated unique values
        Description  string    // Generated semantic description
    }
    
    // Relationship graph
    Relationships []Relationship {
        Type           string  // one-to-one, one-to-many, many-to-many
        TargetTable    string
        LocalColumns   []string
        ForeignColumns []string
        JoinFrequency  int     // How often this join appears in history
    }
    
    // Usage statistics
    Stats {
        RowCount          int64
        QueryFrequency    int
        LastQueried       time.Time
        CommonFilters     []string  // WHERE clauses that appear often
        CommonAggregates  []string  // GROUP BY patterns
    }
}
```

### 3. Multi-Document Embedding Strategy

**Problem**: Single embeddings for entire tables don't capture column-level semantics well.

**Solution**: Create separate embeddings for:

- **Table-level**: Table name + description + overall purpose
- **Column-level**: Each column with type, samples, and semantic meaning
- **Relationship-level**: JOIN patterns between tables

**Files to modify**:

- `backend-go/internal/rag/sqlite_vector_store.go` - Add document type categorization
- `backend-go/internal/rag/context_builder.go` - Multi-level retrieval logic

**Indexing Strategy**:

```
Table Embedding: "users table stores customer account information with 50000 rows"
Column Embeddings: 
  - "email VARCHAR(255) unique identifier for user contact"
  - "created_at TIMESTAMP when user account was created"
Relationship Embedding: "users.id → orders.user_id one-to-many relationship for order history"
```

### 4. Query Pattern Learning Pipeline

**Problem**: Current system indexes queries but doesn't extract reusable patterns or learn from JOIN structures.

**Solution**: Implement learning pipeline that:

- Extracts query patterns (templates with parameter placeholders)
- Identifies common JOIN sequences
- Tracks query success rates and execution times
- Learns from query corrections (when AI regenerates SQL after error)

**Files to create**:

- `backend-go/internal/rag/learning_pipeline.go` (NEW)
- `backend-go/internal/rag/pattern_extractor.go` (NEW)
- `backend-go/internal/rag/query_template.go` (NEW)

**Pattern Extraction Example**:

```
Original Query: "SELECT * FROM users WHERE email = 'test@example.com'"
Pattern: "SELECT * FROM users WHERE email = ?"
Template Tags: [filter:email, table:users]
```

### 5. Tier-Based Vector Storage Strategy

**Problem**: Need different vector storage approaches based on customer tier (local-only vs Turso sync).

**Solution**: Implement adaptive vector store that:

- **Local Tier**: SQLite vector store in `~/.howlerops/vectors.db`
- **Individual Tier**: SQLite + optional Turso sync for embeddings metadata
- **Team Tier**: Turso-backed with shared embeddings across team members

**Files to create/modify**:

- `backend-go/internal/rag/adaptive_vector_store.go` (NEW) - Tier-aware storage router
- `backend-go/internal/rag/turso_vector_sync.go` (NEW) - Sync logic for team tier
- `backend-go/internal/sync/turso_store.go` - Add vector embedding sync tables

**Architecture**:

```go
type AdaptiveVectorStore struct {
    tierLevel     string
    localStore    VectorStore          // Always available
    remoteStore   VectorStore          // Turso for Individual/Team
    syncEnabled   bool
}

func (a *AdaptiveVectorStore) IndexDocument(ctx context.Context, doc *Document) error {
    // Always index locally for performance
    if err := a.localStore.IndexDocument(ctx, doc); err != nil {
        return err
    }
    
    // Sync to remote for Individual/Team tiers
    if a.syncEnabled && (a.tierLevel == "individual" || a.tierLevel == "team") {
        go a.syncToRemote(ctx, doc)
    }
    
    return nil
}
```

### 6. Context Builder Enhancements

**Problem**: Context builder doesn't leverage relationship graphs or multi-level embeddings effectively.

**Solution**: Enhance context retrieval to:

- Use relationship graph to expand relevant tables (e.g., if query mentions "orders", also fetch "users" and "products")
- Retrieve similar query patterns with execution times
- Include column-level context for better type matching
- Add business rule validation (e.g., "always join users.id with orders.user_id")

**Files to modify**:

- `backend-go/internal/rag/context_builder.go` - Enhanced multi-level retrieval
- `app.go` - Update `buildDetailedSchemaContext` to use RAG

### 7. Hybrid Search (Vector + Full-Text)

**Problem**: Pure vector search may miss exact keyword matches (e.g., table names, column names).

**Solution**: Implement hybrid search combining:

- Vector similarity for semantic matching
- SQLite FTS5 for exact keyword matching
- Weighted score fusion (0.6 vector + 0.4 FTS)

**Files to modify**:

- `backend-go/internal/rag/sqlite_vector_store.go` - Implement HybridSearch method
- `backend-go/internal/rag/context_builder.go` - Use hybrid search for schema retrieval

### 8. Embedding Cache Optimization

**Problem**: Generating embeddings for every query is expensive.

**Solution**: Enhance caching with:

- Persistent cache in SQLite (survive app restarts)
- LRU eviction for memory management
- Cache warming on startup for frequently accessed schemas

**Files to modify**:

- `backend-go/internal/rag/embedding_service.go` - Persistent cache
- `backend-go/internal/rag/sqlite_vector_store.go` - Add cache tables

## Implementation Order

### Phase 1: Local Embeddings (Days 1-2)

1. Create `onnx_embedding_provider.go` with ONNX Runtime support
2. Download and integrate all-MiniLM-L6-v2 ONNX model
3. Implement simple tokenizer (or use pre-tokenized approach)
4. Add embedding provider fallback logic (ONNX → OpenAI)
5. Update config to use local ONNX model
6. Test embedding generation and caching

### Phase 2: Enhanced Schema Indexing (Days 3-4)

1. Create `schema_indexer.go` with rich schema documents
2. Implement `relationship_analyzer.go` for FK graph building
3. Add sample value collection (with privacy controls)
4. Create multi-level embeddings (table, column, relationship)

### Phase 3: Query Learning Pipeline (Days 5-6)

1. Create `learning_pipeline.go` for query pattern extraction
2. Implement `pattern_extractor.go` for template generation
3. Add query success tracking and performance metrics
4. Index successful queries with patterns

### Phase 4: Tier-Based Storage (Days 7-8)

1. Create `adaptive_vector_store.go` with tier detection
2. Implement Turso sync for Individual/Team tiers
3. Add sync status tracking and conflict resolution
4. Test local vs synced storage paths

### Phase 5: Context Enhancement (Days 9-10)

1. Enhance `context_builder.go` with relationship expansion
2. Implement hybrid search (vector + FTS)
3. Add business rule extraction from query history
4. Optimize context ranking and relevance scoring

### Phase 6: Testing & Optimization (Days 11-12)

1. End-to-end testing with real schemas
2. Performance benchmarking (embedding generation, retrieval)
3. Cache optimization and warming strategies
4. Documentation and usage examples

## Key Files Summary

### New Files (9)

- `backend-go/internal/rag/onnx_embedding_provider.go`
- `backend-go/internal/rag/tokenizer.go`
- `backend-go/internal/rag/schema_indexer.go`
- `backend-go/internal/rag/relationship_analyzer.go`
- `backend-go/internal/rag/learning_pipeline.go`
- `backend-go/internal/rag/pattern_extractor.go`
- `backend-go/internal/rag/query_template.go`
- `backend-go/internal/rag/adaptive_vector_store.go`
- `backend-go/internal/rag/turso_vector_sync.go`

### Modified Files (5)

- `backend-go/internal/rag/embedding_service.go` - Add ONNX provider support & fallback
- `backend-go/internal/rag/context_builder.go` - Enhanced relationship expansion
- `backend-go/internal/rag/sqlite_vector_store.go` - Hybrid search implementation
- `backend-go/configs/config.yaml` - ONNX model configuration
- `app.go` - Update RAG initialization

## Success Metrics

1. **Embedding Cost Reduction**: 90%+ of embeddings generated locally
2. **Context Relevance**: Improved SQL generation confidence scores by 30%+
3. **Query Success Rate**: Reduced failed queries by 40%+
4. **JOIN Accuracy**: Better FK relationship detection in generated SQL
5. **Performance**: Sub-100ms context retrieval for most queries

## Configuration Changes

```yaml
# backend-go/configs/config.yaml
rag:
  embedding:
    # Prefer local ONNX model for offline-first approach
    provider: "onnx"  # Changed from "openai"
    model_path: "./internal/rag/models/all-MiniLM-L6-v2.onnx"
    fallback_provider: "openai"  # NEW: Fallback if ONNX fails
    fallback_model: "text-embedding-3-small"
    dimension: 384  # all-MiniLM-L6-v2 dimension
    
  schema_indexing:  # NEW section
    enabled: true
    index_relationships: true
    collect_sample_values: false  # SECURITY: Never enable in production
    pii_detection_enabled: true   # Block PII columns
    index_statistics: true        # Cardinality, row counts
    reindex_interval: "24h"
    
  learning:
    extract_patterns: true  # NEW: Extract query templates
    track_join_frequency: true  # NEW: Learn common JOINs
    min_pattern_frequency: 3  # NEW: Minimum occurrences to index pattern
    
  storage:  # NEW section
    strategy: "tier-adaptive"  # tier-adaptive, local-only, remote-only
    sync_embeddings: true  # Sync for Individual/Team tiers
    sync_interval: "1h"
```

### To-dos

- [ ] Implement ONNX Runtime-based local embedding provider with all-MiniLM-L6-v2 model and fallback to OpenAI
- [ ] Create enhanced schema indexing with relationships, cardinality stats, and privacy guards (NO sample values)
- [ ] Implement multi-document embedding strategy (table, column, relationship levels)
- [ ] Build query pattern learning pipeline with template extraction and JOIN frequency tracking
- [ ] Implement tier-adaptive vector storage with local SQLite and Turso sync for Individual/Team tiers
- [ ] Enhance context builder with relationship graph expansion and hybrid search (vector + FTS)
- [ ] Add persistent embedding cache with LRU eviction and cache warming on startup
- [ ] End-to-end testing, performance benchmarking, and validation per repository guidelines