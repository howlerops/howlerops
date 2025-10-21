# HowlerOps Backend - Comprehensive Testing Strategy

## 🎯 Goal: 100% Coverage of All Major Components

**Target**: Achieve 100% test coverage across all critical business logic, APIs, and database layers.
**Timeline**: Phased approach over 2-3 weeks
**Current State**: 4 test files (multiquery, elasticsearch, storage)
**Target State**: ~61 test files with full coverage

---

## 📊 Architecture Analysis

### Backend Structure Overview

```
backend-go/
├── cmd/                       # Entry points
│   ├── server/               # Main server
│   └── migrate-vector-db/    # Migration tool
├── internal/                  # Private packages (32 files)
│   ├── ai/                   # AI service layer (11 files)
│   ├── auth/                 # Authentication (1 file)
│   ├── config/               # Configuration (1 file)
│   ├── middleware/           # HTTP/gRPC middleware (2 files)
│   ├── rag/                  # RAG system (8 files)
│   ├── server/               # HTTP/gRPC servers (2 files)
│   └── services/             # Service orchestration (2 files)
└── pkg/                       # Public packages (29 files + protobuf)
    ├── ai/                   # AI types (2 files)
    ├── database/             # Database drivers (15 files)
    ├── logger/               # Logging (1 file)
    ├── pb/                   # Protobuf (generated - exclude)
    ├── rag/                  # RAG interface (1 file)
    └── storage/              # Storage layer (5 files)
```

### Component Breakdown by Priority

| Component | Files | Complexity | Test Priority | Target Coverage |
|-----------|-------|------------|---------------|-----------------|
| **Database Layer** | 15 | High | CRITICAL | 100% |
| **AI Service** | 13 | High | CRITICAL | 100% |
| **RAG System** | 9 | High | CRITICAL | 100% |
| **HTTP/gRPC Servers** | 2 | Medium | HIGH | 100% |
| **Auth/Middleware** | 3 | Medium | HIGH | 100% |
| **Storage Layer** | 5 | Medium | HIGH | 95%+ |
| **Config** | 1 | Low | MEDIUM | 90% |
| **Logger** | 1 | Low | MEDIUM | 80% |
| **Services** | 2 | Low | MEDIUM | 90% |

---

## 🧪 Testing Philosophy & Strategy

### Testing Pyramid

```
         /\
        /e2\     E2E Tests (5%)
       /----\    - Full API workflows
      /Integ\   Integration Tests (15%)
     /-------\  - Database + external services
    /  Unit  \ Unit Tests (80%)
   /----------\ - Business logic, handlers, utils
```

### Coverage Goals by Layer

1. **Business Logic** → 100% (mandatory)
2. **API Endpoints** → 100% (mandatory)
3. **Database Operations** → 100% (mandatory)
4. **Error Handling** → 100% (mandatory)
5. **Middleware** → 100% (mandatory)
6. **Utils/Helpers** → 95%+
7. **Config/Init** → 85%+

### What to Exclude from Coverage

- Generated code (`.pb.go` files)
- `main.go` functions (move logic to testable functions)
- Third-party adapter wrappers (if thin)
- Platform-specific code with build tags

---

## 🛠️ Testing Tools & Setup

### Required Dependencies

```bash
# Install testing tools
go get -u github.com/stretchr/testify
go get -u github.com/DATA-DOG/go-sqlmock
go get -u github.com/vektra/mockery/v2
go get -u github.com/testcontainers/testcontainers-go

# Install mockery globally
go install github.com/vektra/mockery/v2@latest
```

### Project Configuration

**`.mockery.yaml`** (Mock generation config):
```yaml
with-expecter: true
dir: "{{.InterfaceDir}}/mocks"
outpkg: mocks
filename: "mock_{{.InterfaceName}}.go"
packages:
  github.com/sql-studio/backend-go/pkg/database:
    interfaces:
      Driver:
      VectorStore:
      SchemaCache:
  github.com/sql-studio/backend-go/internal/ai:
    interfaces:
      Provider:
      Service:
  github.com/sql-studio/backend-go/pkg/storage:
    interfaces:
      Store:
      Manager:
```

**`Makefile`** (Testing commands):
```makefile
.PHONY: test test-coverage test-unit test-integration mocks

# Run all tests
test:
\tgo test -v -race ./...

# Run with coverage
test-coverage:
\tgo test -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./...
\tgo tool cover -html=coverage.out -o coverage.html
\t@echo "Coverage report: coverage.html"

# Unit tests only
test-unit:
\tgo test -short -v ./...

# Integration tests only
test-integration:
\tgo test -tags=integration -v ./...

# Generate mocks
mocks:
\tmockery --all

# Check coverage threshold
test-coverage-check: test-coverage
\t@go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//' | awk '{if ($$1 < 80) {print "Coverage is below 80%"; exit 1}}'
```

---

## 📋 Comprehensive Testing Plan by Component

### Phase 1: Critical Infrastructure (Week 1)

#### 1.1 Database Layer Testing

**Target**: `pkg/database/` - 15 files, 100% coverage

**Files to Test**:
- `manager.go` - Database connection management
- `pool.go` - Connection pooling
- `mysql.go`, `postgres.go`, `sqlite.go` - Driver implementations
- `mongodb.go`, `elasticsearch.go`, `clickhouse.go`, `tidb.go` - NoSQL/special drivers
- `ssh_tunnel.go` - SSH tunnel connections
- `types.go` - Type definitions
- `queryparser.go` - SQL parsing
- `schema_cache.go`, `schema_cache_manager.go` - Schema introspection
- `structure_cache.go` - Structure caching

**Testing Strategy**:
- **Unit Tests**: Use sqlmock for SQL driver operations
- **Integration Tests**: Use testcontainers for real database operations
- **Mocking**: Generate mocks for `Driver` interface

**Test Files to Create**:
```
pkg/database/
├── manager_test.go
├── pool_test.go
├── mysql_test.go
├── postgres_test.go
├── sqlite_test.go
├── mongodb_test.go
├── clickhouse_test.go
├── tidb_test.go
├── ssh_tunnel_test.go
├── queryparser_test.go
├── schema_cache_test.go
├── schema_cache_manager_test.go
├── structure_cache_test.go
└── integration_test.go  (+build integration)
```

**Example Test Pattern**:
```go
// mysql_test.go
package database_test

import (
\t"testing"
\t"github.com/DATA-DOG/go-sqlmock"
\t"github.com/stretchr/testify/assert"
)

func TestMySQL_Connect(t *testing.T) {
\ttests := []struct {
\t\tname    string
\t\tconfig  *Config
\t\twantErr bool
\t}{
\t\t{"valid config", validConfig(), false},
\t\t{"invalid host", invalidHostConfig(), true},
\t\t{"invalid credentials", invalidCredsConfig(), true},
\t}

\tfor _, tt := range tests {
\t\tt.Run(tt.name, func(t *testing.T) {
\t\t\tdriver := NewMySQLDriver()
\t\t\terr := driver.Connect(tt.config)
\t\t\tif tt.wantErr {
\t\t\t\tassert.Error(t, err)
\t\t\t} else {
\t\t\t\tassert.NoError(t, err)
\t\t\t\tdefer driver.Close()
\t\t\t}
\t\t})
\t}
}
```

#### 1.2 Storage Layer Testing

**Target**: `pkg/storage/` - 5 files (1 already has tests)

**Files to Test**:
- `interface.go` - Storage interface
- `manager.go` - Storage manager
- `sqlite_local.go` - SQLite storage (✅ has tests)
- `types.go` - Storage types

**Test Files to Create**:
```
pkg/storage/
├── manager_test.go
├── integration_test.go
```

#### 1.3 HTTP/gRPC Server Testing

**Target**: `internal/server/` - 2 files, 100% coverage

**Files to Test**:
- `http.go` - HTTP server and routing
- `grpc.go` - gRPC server and interceptors

**Testing Strategy**:
- Use `httptest.NewRecorder()` for HTTP tests
- Use `bufconn` for gRPC unit tests
- Test middleware chain execution
- Test TLS configuration
- Test health check endpoints

**Test Files to Create**:
```
internal/server/
├── http_test.go
├── grpc_test.go
└── integration_test.go  (+build integration)
```

**Example HTTP Test**:
```go
func TestHTTPServer_HealthCheck(t *testing.T) {
\treq := httptest.NewRequest("GET", "/health", nil)
\tw := httptest.NewRecorder()

\thandler := createTestHandler()
\thandler.ServeHTTP(w, req)

\tassert.Equal(t, http.StatusOK, w.Code)
\tassert.Contains(t, w.Body.String(), "healthy")
}
```

**Example gRPC Test**:
```go
func TestGRPCServer_AIService(t *testing.T) {
\tlistener := bufconn.Listen(1024 * 1024)
\tserver := createTestGRPCServer(t, listener)
\tdefer server.Stop()

\tconn := createTestConnection(t, listener)
\tdefer conn.Close()

\tclient := pb.NewAIServiceClient(conn)
\tresp, err := client.Chat(context.Background(), &pb.ChatRequest{
\t\tMessage: "test",
\t})

\tassert.NoError(t, err)
\tassert.NotNil(t, resp)
}
```

---

### Phase 2: Business Logic (Week 2)

#### 2.1 AI Service Testing

**Target**: `internal/ai/` - 11 files, 100% coverage

**Files to Test**:
- `service.go` - Main AI service
- `provider.go` - Provider abstraction
- `anthropic.go`, `openai.go`, `ollama.go`, `huggingface.go` - Provider implementations
- `claudecode.go`, `codex.go` - Specialized providers
- `adapter_wrapper.go` - Adapter pattern
- `handlers.go` - HTTP handlers
- `grpc.go` - gRPC service implementation
- `ollama_detector.go` - Ollama detection
- `types.go`, `config.go` - Types and config

**Testing Strategy**:
- Mock external HTTP calls to AI APIs
- Test rate limiting and retries
- Test streaming responses
- Test error handling (API failures, timeouts)
- Test provider fallback logic

**Test Files to Create**:
```
internal/ai/
├── service_test.go
├── provider_test.go
├── anthropic_test.go
├── openai_test.go
├── ollama_test.go
├── huggingface_test.go
├── claudecode_test.go
├── codex_test.go
├── handlers_test.go
├── grpc_test.go
├── ollama_detector_test.go
└── integration_test.go  (+build integration)
```

**Example Provider Test**:
```go
func TestAnthropicProvider_Chat(t *testing.T) {
\t// Mock HTTP client
\tmockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
\t\tassert.Equal(t, "/v1/messages", r.URL.Path)
\t\tw.WriteHeader(http.StatusOK)
\t\tjson.NewEncoder(w).Encode(map[string]interface{}{
\t\t\t"content": []map[string]string{{"text": "Hello!"}},
\t\t})
\t}))
\tdefer mockServer.Close()

\tprovider := NewAnthropicProvider(mockServer.URL, "test-key")
\tresp, err := provider.Chat(context.Background(), &ChatRequest{
\t\tMessage: "Hi",
\t})

\tassert.NoError(t, err)
\tassert.Equal(t, "Hello!", resp.Content)
}
```

#### 2.2 RAG System Testing

**Target**: `internal/rag/` - 8 files, 100% coverage

**Files to Test**:
- `vector_store.go` - Vector store interface
- `embedding_service.go` - Embedding generation
- `embedding_utils.go` - Embedding utilities
- `smart_sql_generator.go` - SQL generation from embeddings
- `visualization_engine.go` - Visualization logic
- `context_builder.go` - Context building
- `sqlite_vector_store.go`, `mysql_vector_store.go` - Vector store implementations
- `helpers.go` - Helper functions

**Testing Strategy**:
- Use in-memory SQLite for vector store tests
- Mock embedding API calls
- Test similarity search algorithms
- Test SQL generation accuracy
- Validate visualization data structures

**Test Files to Create**:
```
internal/rag/
├── embedding_service_test.go
├── embedding_utils_test.go
├── smart_sql_generator_test.go
├── visualization_engine_test.go
├── context_builder_test.go
├── sqlite_vector_store_test.go
├── mysql_vector_store_test.go
├── helpers_test.go
└── integration_test.go  (+build integration)
```

#### 2.3 Auth & Middleware Testing

**Target**: `internal/auth/`, `internal/middleware/` - 3 files, 100% coverage

**Files to Test**:
- `auth/service.go` - Authentication service
- `middleware/auth.go` - Auth middleware
- `middleware/ratelimit.go` - Rate limiting middleware

**Testing Strategy**:
- Test JWT token generation and validation
- Test middleware chain execution
- Test rate limiting thresholds
- Test auth failure scenarios
- Test header propagation

**Test Files to Create**:
```
internal/auth/
└── service_test.go

internal/middleware/
├── auth_test.go
└── ratelimit_test.go
```

**Example Auth Test**:
```go
func TestAuthMiddleware_ValidToken(t *testing.T) {
\tauth := NewAuthMiddleware("secret", logger)
\ttoken := generateValidToken("user123")

\treq := httptest.NewRequest("GET", "/api/data", nil)
\treq.Header.Set("Authorization", "Bearer "+token)

\tctx, err := auth.Authenticate(req.Context(), req)
\tassert.NoError(t, err)
\tassert.Equal(t, "user123", getUserID(ctx))
}
```

---

### Phase 3: Supporting Components (Week 3)

#### 3.1 Configuration Testing

**Target**: `internal/config/` - 1 file

**Files to Test**:
- `config.go` - Configuration loading and validation

**Test Files to Create**:
```
internal/config/
└── config_test.go
```

#### 3.2 Services Orchestration Testing

**Target**: `internal/services/` - 2 files

**Files to Test**:
- `services.go` - Service initialization
- `stores.go` - Store initialization

**Test Files to Create**:
```
internal/services/
├── services_test.go
└── stores_test.go
```

#### 3.3 Multiquery System Testing

**Target**: `pkg/database/multiquery/` - 4 files (2 already have tests)

**Files to Test**:
- `parser.go` - SQL parsing (✅ has tests)
- `executor.go` - Query execution (✅ has tests)
- `merger.go` - Result merging
- `types.go` - Type definitions

**Test Files to Create**:
```
pkg/database/multiquery/
├── merger_test.go
└── types_test.go
```

---

## 🧩 Test Utilities & Helpers

### Shared Test Utilities

Create `internal/testutil/` package:

```
internal/testutil/
├── fixtures.go          # Test data fixtures
├── mock_db.go           # Common database mocks
├── mock_http.go         # HTTP client mocks
├── assertions.go        # Custom assertions
├── server.go            # Test server helpers
└── cleanup.go           # Test cleanup utilities
```

**Example fixtures.go**:
```go
package testutil

import "github.com/sql-studio/backend-go/pkg/database"

func NewTestConfig() *database.Config {
\treturn &database.Config{
\t\tHost:     "localhost",
\t\tPort:     3306,
\t\tDatabase: "test",
\t\tUsername: "test",
\t\tPassword: "test",
\t}
}

func NewTestConnection(t *testing.T) *sql.DB {
\tt.Helper()
\tdb, err := sql.Open("sqlite3", ":memory:")
\tif err != nil {
\t\tt.Fatalf("failed to create test db: %v", err)
\t}
\tt.Cleanup(func() { db.Close() })
\treturn db
}
```

---

## 🚀 Implementation Roadmap

### Week 1: Critical Infrastructure
- **Day 1-2**: Database layer testing (15 test files)
- **Day 3**: Storage layer testing (2 test files)
- **Day 4**: HTTP/gRPC server testing (3 test files)
- **Day 5**: Review, fix failures, measure coverage

**Deliverable**: 80%+ coverage of infrastructure layer

### Week 2: Business Logic
- **Day 1-2**: AI service testing (12 test files)
- **Day 3-4**: RAG system testing (9 test files)
- **Day 5**: Auth & middleware testing (3 test files)

**Deliverable**: 90%+ coverage of business logic

### Week 3: Integration & Polish
- **Day 1**: Configuration and services testing (4 test files)
- **Day 2**: Multiquery completion (2 test files)
- **Day 3**: Integration tests across all layers
- **Day 4**: E2E tests for critical workflows
- **Day 5**: Coverage analysis, gap filling, documentation

**Deliverable**: 100% coverage target achieved

---

## 📈 Coverage Tracking & CI/CD

### GitHub Actions Workflow

**`.github/workflows/test.yml`**:
```yaml
name: Test Coverage

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      mysql:
        image: mysql:8
        env:
          MYSQL_ROOT_PASSWORD: test
          MYSQL_DATABASE: test
        options: >-
          --health-cmd "mysqladmin ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

      - name: Install dependencies
        run: |
          go mod download
          go install github.com/vektra/mockery/v2@latest

      - name: Generate mocks
        run: make mocks

      - name: Run unit tests
        run: make test-unit

      - name: Run integration tests
        run: make test-integration
        env:
          POSTGRES_HOST: localhost
          POSTGRES_PORT: 5432
          MYSQL_HOST: localhost
          MYSQL_PORT: 3306

      - name: Run tests with coverage
        run: make test-coverage

      - name: Check coverage threshold
        run: make test-coverage-check

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
          fail_ci_if_error: true

      - name: Upload coverage HTML
        uses: actions/upload-artifact@v3
        with:
          name: coverage-report
          path: coverage.html
```

### Pre-commit Hook

**`.git/hooks/pre-commit`**:
```bash
#!/bin/bash

echo "Running tests before commit..."
make test-unit

if [ $? -ne 0 ]; then
    echo "Tests failed! Commit aborted."
    exit 1
fi

echo "Tests passed!"
```

---

## 🎯 Success Metrics

### Coverage Targets
- **Overall**: 95%+
- **Database Layer**: 100%
- **AI Service**: 100%
- **RAG System**: 100%
- **HTTP/gRPC**: 100%
- **Auth/Middleware**: 100%
- **Supporting**: 90%+

### Quality Metrics
- Zero flaky tests
- All tests complete in < 2 minutes (unit tests)
- Integration tests complete in < 5 minutes
- No test-only code in production paths
- All edge cases covered

### Documentation
- [ ] All public APIs have example tests
- [ ] Complex logic has explanatory test comments
- [ ] Test naming follows convention
- [ ] README includes testing instructions

---

## 🔧 Troubleshooting & Best Practices

### Common Issues

**1. Flaky Tests**
- Use `t.Cleanup()` for proper teardown
- Avoid `time.Sleep()` - use polling with timeout
- Ensure test isolation (no shared state)

**2. Slow Tests**
- Use `t.Parallel()` for independent tests
- Mock external dependencies
- Use in-memory databases

**3. Coverage Gaps**
- Identify with `go tool cover -html`
- Check error paths specifically
- Review panic recovery blocks

### Testing Best Practices

1. **Use Table-Driven Tests**
   ```go
   tests := []struct{ name string; input int; want int }{}
   ```

2. **Always Test Error Paths**
   ```go
   if err != nil { t.Errorf("unexpected error") }
   ```

3. **Use Subtests**
   ```go
   t.Run("success case", func(t *testing.T) { ... })
   ```

4. **Cleanup Resources**
   ```go
   t.Cleanup(func() { conn.Close() })
   ```

5. **Isolate Tests**
   - No shared global state
   - Each test creates own fixtures

---

## 📚 Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Mockery Documentation](https://vektra.github.io/mockery/)
- [Go Test Coverage](https://go.dev/blog/cover)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [SQL Mock](https://github.com/DATA-DOG/go-sqlmock)
- [Testcontainers](https://golang.testcontainers.org/)

---

## ✅ Next Steps

1. **Set up testing infrastructure**
   - Install dependencies
   - Create mockery config
   - Set up Makefile

2. **Start Phase 1**
   - Begin with database layer
   - Establish patterns
   - Create test utilities

3. **Iterate and measure**
   - Run coverage reports daily
   - Track progress in GitHub issues
   - Adjust strategy as needed

---

*Testing strategy created based on industry best practices and web research. Ready for implementation.*
