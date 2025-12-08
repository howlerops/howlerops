# PRD: AI-Powered Index Recommendations

**Version:** 1.0
**Date:** 2024-12-07
**Author:** HowlerOps Team
**Status:** Draft

---

## 1. Executive Summary

Index Recommendations is an AI-powered feature for HowlerOps that analyzes query patterns and database schema to provide actionable index optimization suggestions. The feature identifies missing indexes, detects unused indexes, and provides per-query impact analysis with one-click DDL generation.

This feature builds on existing infrastructure:
- Query analyzer (`backend-go/internal/analyzer/`) for SQL parsing and optimization hints
- AI service (`backend-go/internal/ai/`) for LLM-powered analysis
- RAG schema indexer (`backend-go/internal/rag/`) for schema context

---

## 2. Problem Statement

Database performance issues often stem from suboptimal indexing:

1. **Missing indexes**: Queries scan entire tables instead of using indexes
2. **Unused indexes**: Redundant indexes consume storage and slow writes
3. **No visibility**: Users lack insight into index effectiveness
4. **Manual analysis**: EXPLAIN plans require expertise to interpret
5. **Multi-database complexity**: Index strategies vary by database engine

---

## 3. Goals & Success Metrics

### Goals
- Reduce query execution time through better indexing
- Eliminate manual index analysis for common cases
- Provide actionable, database-specific recommendations

### Success Metrics
| Metric | Target |
|--------|--------|
| Index suggestion acceptance rate | >40% |
| Average query time improvement | >30% for suggested indexes |
| False positive rate | <15% |
| Time-to-recommendation | <3 seconds |

---

## 4. User Stories

### P0 - Must Have
1. **As a developer**, I want to see index suggestions when my query is slow, so I can improve performance without being a DBA.
2. **As a DBA**, I want to identify unused indexes, so I can reduce storage costs and write overhead.
3. **As a developer**, I want one-click index creation, so I don't need to manually write DDL.

### P1 - Should Have
4. **As a developer**, I want to see estimated performance impact before creating an index.
5. **As a DBA**, I want to analyze historical query patterns for comprehensive index recommendations.

### P2 - Nice to Have
6. **As a developer**, I want to compare query execution plans with and without proposed indexes.

---

## 5. Technical Requirements

### 5.1 Backend (Go)

#### New Package: `backend-go/internal/indexrecommend/`

**Core Types:**
```go
type IndexRecommendation struct {
    ID              string              `json:"id"`
    ConnectionID    string              `json:"connection_id"`
    TableName       string              `json:"table_name"`
    SchemaName      string              `json:"schema_name"`
    RecommendationType RecommendationType `json:"type"` // "create", "drop"
    Columns         []string            `json:"columns"`
    IndexType       string              `json:"index_type"` // btree, hash, gin, etc.
    Reason          string              `json:"reason"`
    DDL             string              `json:"ddl"`
    Impact          *ImpactAnalysis     `json:"impact,omitempty"`
    Confidence      float64             `json:"confidence"` // 0-1
    CreatedAt       time.Time           `json:"created_at"`
}

type ImpactAnalysis struct {
    EstimatedSpeedup    float64   `json:"estimated_speedup"` // e.g., 2.5x
    AffectedQueries     int       `json:"affected_queries"`
    StorageImpactBytes  int64     `json:"storage_impact_bytes"`
    WriteOverhead       float64   `json:"write_overhead"` // percentage
}

type RecommendationType string
const (
    RecommendCreate RecommendationType = "create"
    RecommendDrop   RecommendationType = "drop"
)
```

**Service Interface:**
```go
type IndexRecommendService interface {
    // Analyze single query for index opportunities
    AnalyzeQuery(ctx context.Context, req *AnalyzeQueryRequest) (*AnalysisResult, error)

    // Get recommendations for a table
    GetTableRecommendations(ctx context.Context, connID, schema, table string) ([]IndexRecommendation, error)

    // Detect unused indexes
    GetUnusedIndexes(ctx context.Context, connID string, minDays int) ([]UnusedIndex, error)

    // Generate DDL for recommendation
    GenerateDDL(ctx context.Context, rec *IndexRecommendation) (string, error)

    // Apply recommendation (create/drop index)
    ApplyRecommendation(ctx context.Context, rec *IndexRecommendation) error
}
```

**Integration Points:**
- Extend existing `QueryAnalyzer` to extract index candidates from WHERE/JOIN/ORDER BY clauses
- Use `ai.Service.Chat()` for LLM-powered analysis with schema context
- Leverage `rag.SchemaIndexer` for table/column metadata retrieval

### 5.2 Frontend (React/TypeScript)

#### New Components: `frontend/src/components/index-recommendations/`

**IndexRecommendationsPanel.tsx:**
```typescript
interface IndexRecommendation {
  id: string;
  connectionId: string;
  tableName: string;
  schemaName: string;
  type: 'create' | 'drop';
  columns: string[];
  indexType: string;
  reason: string;
  ddl: string;
  impact?: ImpactAnalysis;
  confidence: number;
}

interface IndexRecommendationsPanelProps {
  connectionId: string;
  currentQuery?: string;
  onApply?: (recommendation: IndexRecommendation) => void;
}
```

**UI Elements:**
1. Recommendation cards with severity indicators
2. DDL preview with copy button
3. Impact analysis visualization
4. One-click apply button with confirmation
5. Filter by table/recommendation type

---

## 6. API Design

### 6.1 REST Endpoints

```
POST /api/index-recommend/analyze
  Request:  { sql: string, connection_id: string }
  Response: { recommendations: IndexRecommendation[], analysis: QueryAnalysis }

GET /api/index-recommend/table/{connection_id}/{schema}/{table}
  Response: { recommendations: IndexRecommendation[] }

GET /api/index-recommend/unused/{connection_id}
  Query:    ?min_days=7
  Response: { unused_indexes: UnusedIndex[] }

POST /api/index-recommend/apply
  Request:  { recommendation_id: string, connection_id: string }
  Response: { success: boolean, ddl_executed: string }
```

### 6.2 gRPC Service (proto/index_recommend.proto)

```protobuf
service IndexRecommendService {
  rpc AnalyzeQuery(AnalyzeQueryRequest) returns (AnalyzeQueryResponse);
  rpc GetTableRecommendations(TableRequest) returns (RecommendationsResponse);
  rpc GetUnusedIndexes(UnusedIndexesRequest) returns (UnusedIndexesResponse);
  rpc ApplyRecommendation(ApplyRequest) returns (ApplyResponse);
}

message IndexRecommendation {
  string id = 1;
  string connection_id = 2;
  string table_name = 3;
  string schema_name = 4;
  string type = 5;  // create, drop
  repeated string columns = 6;
  string index_type = 7;
  string reason = 8;
  string ddl = 9;
  ImpactAnalysis impact = 10;
  double confidence = 11;
}
```

---

## 7. AI/ML Approach

### 7.1 LLM Integration

Use existing AI service (`backend-go/internal/ai/`) with structured prompts:

**Index Analysis Prompt Template:**
```
Given the following SQL query and schema context, recommend index optimizations.

Query: {sql}

Schema Context:
{schema_context}

Current Indexes:
{existing_indexes}

Analyze and provide:
1. Missing indexes that would improve this query
2. Columns that should be part of composite indexes
3. Index types appropriate for the database engine
4. Estimated impact on query performance

Respond in JSON format: {json_schema}
```

### 7.2 Heuristic Analysis (Non-AI Fallback)

For quick analysis without LLM:
1. Extract WHERE clause columns not in indexes
2. Identify JOIN columns without indexes
3. Check ORDER BY columns for sort optimization
4. Flag indexes not referenced in query history

### 7.3 Pattern Learning

Extend `rag.QueryPatternTracker` to:
- Track index usage per query pattern
- Build corpus of effective indexes
- Learn from user acceptance/rejection of recommendations

---

## 8. Database-Specific Considerations

| Database | Index Types | Special Considerations |
|----------|-------------|----------------------|
| PostgreSQL | B-tree, Hash, GIN, GiST, BRIN | Partial indexes, INCLUDE columns |
| MySQL | B-tree, Hash, Full-text, Spatial | Prefix indexes, invisible indexes |
| SQLite | B-tree | Covering indexes, WITHOUT ROWID |
| MongoDB | Single, Compound, Text, Geo | Sparse indexes, TTL indexes |
| ClickHouse | Primary key (MergeTree), Skip | Granularity, projection |
| DuckDB | ART, Min-Max | Columnar considerations |
| TiDB | B-tree | TiKV region distribution |
| Elasticsearch | N/A | Mapping optimizations instead |

**DDL Generation:**
Generate database-specific syntax:
```go
func (s *service) GenerateDDL(rec *IndexRecommendation, dbType string) string {
    switch dbType {
    case "postgres":
        return fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON %s.%s (%s);",
            rec.IndexName, rec.SchemaName, rec.TableName,
            strings.Join(rec.Columns, ", "))
    case "mysql":
        return fmt.Sprintf("ALTER TABLE %s.%s ADD INDEX %s (%s);",
            rec.SchemaName, rec.TableName, rec.IndexName,
            strings.Join(rec.Columns, ", "))
    // ... other databases
    }
}
```

---

## 9. Implementation Plan

### Phase 1: Foundation (Week 1-2)
- [ ] Create `indexrecommend` package structure
- [ ] Implement basic heuristic analyzer (extend QueryAnalyzer)
- [ ] Add REST endpoints
- [ ] Basic frontend panel component

### Phase 2: AI Integration (Week 3)
- [ ] Build AI prompt templates
- [ ] Integrate with existing AI service
- [ ] Add confidence scoring
- [ ] Impact estimation logic

### Phase 3: Polish (Week 4)
- [ ] Database-specific DDL generation
- [ ] Unused index detection
- [ ] Frontend refinements
- [ ] Integration tests

### Phase 4: Enhancement (Post-MVP)
- [ ] Query history pattern analysis
- [ ] Batch recommendations
- [ ] Index simulation (EXPLAIN with hypothetical index)

---

## 10. Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| LLM hallucinations in DDL | High | Medium | Validate DDL syntax before display; heuristic fallback |
| Performance impact of analysis | Medium | Low | Async analysis; caching; rate limiting |
| Incorrect drop recommendations | High | Low | Require confirmation; show query count affected |
| Database-specific edge cases | Medium | Medium | Start with PostgreSQL/MySQL; expand coverage |
| User trust in AI recommendations | Medium | Medium | Show reasoning; provide confidence scores |

---

## Appendix: File Structure

```
backend-go/internal/indexrecommend/
├── service.go           # Main service implementation
├── analyzer.go          # Query analysis for index opportunities
├── ddl_generator.go     # Database-specific DDL generation
├── impact_analyzer.go   # Performance impact estimation
├── handler.go           # HTTP handlers
├── types.go             # Type definitions
└── service_test.go      # Tests

frontend/src/components/index-recommendations/
├── IndexRecommendationsPanel.tsx
├── RecommendationCard.tsx
├── ImpactVisualization.tsx
├── DDLPreview.tsx
└── index.ts
```
