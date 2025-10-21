# HowlerOps Backend - Testing Implementation Roadmap

## 🎯 Mission: Achieve 100% Test Coverage

**Status**: Ready to implement
**Timeline**: 3 weeks
**Team Size**: 1-2 developers

---

## 📦 Deliverables Created

### ✅ Configuration Files
- [x] `.mockery.yaml` - Mock generation configuration
- [x] `Makefile.test` - Testing commands and automation
- [x] `.github/workflows/test-coverage.yml` - CI/CD pipeline

### ✅ Test Infrastructure
- [x] `internal/testutil/fixtures.go` - Test data and database fixtures
- [x] `internal/testutil/assertions.go` - Custom test assertions
- [x] `internal/testutil/mock_http.go` - HTTP testing utilities
- [x] `internal/testutil/server.go` - gRPC testing utilities

### ✅ Documentation
- [x] `TESTING_STRATEGY.md` - Comprehensive testing strategy
- [x] `TESTING_ROADMAP.md` - This implementation roadmap

---

## 🚀 Quick Start Guide

### 1. Install Dependencies

```bash
cd backend-go

# Install testing tools
go get -u github.com/stretchr/testify
go get -u github.com/DATA-DOG/go-sqlmock
go get -u github.com/vektra/mockery/v2

# Install mockery globally
go install github.com/vektra/mockery/v2@latest

# Download all dependencies
go mod download
```

### 2. Generate Mocks

```bash
# Generate all mocks using mockery
mockery --all

# Or use Makefile
make -f Makefile.test mocks
```

### 3. Run Tests

```bash
# Run unit tests
make -f Makefile.test test-unit

# Run with coverage
make -f Makefile.test test-coverage

# Check coverage threshold
make -f Makefile.test test-coverage-check
```

---

## 📅 Implementation Schedule

### Week 1: Database & Infrastructure Layer (Days 1-5)

#### Day 1: Setup & Database Core
**Focus**: Get testing infrastructure working + core database tests

**Tasks**:
1. Install all dependencies
2. Generate initial mocks
3. Verify test infrastructure works
4. Create `pkg/database/manager_test.go`
5. Create `pkg/database/pool_test.go`

**Files to Create** (2 files):
- `pkg/database/manager_test.go`
- `pkg/database/pool_test.go`

**Success Criteria**:
- All tools installed and working
- First 2 test files running
- Coverage measurement working

---

#### Day 2: SQL Drivers (MySQL, Postgres, SQLite)
**Focus**: Test core database drivers

**Tasks**:
1. Create `pkg/database/mysql_test.go` - test MySQL driver
2. Create `pkg/database/postgres_test.go` - test Postgres driver
3. Create `pkg/database/sqlite_test.go` - test SQLite driver
4. Use sqlmock for unit tests
5. Test connection, query execution, error handling

**Files to Create** (3 files):
- `pkg/database/mysql_test.go`
- `pkg/database/postgres_test.go`
- `pkg/database/sqlite_test.go`

**Example Test Pattern**:
```go
func TestMySQL_Query(t *testing.T) {
\tdb, mock, err := sqlmock.New()
\trequire.NoError(t, err)
\tdefer db.Close()

\tmock.ExpectQuery("SELECT \\* FROM users WHERE id = \\?").
\t\tWithArgs(1).
\t\tWillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
\t\t\tAddRow(1, "Alice"))

\t// Test your driver
\tdriver := NewMySQLDriver(db)
\tresult, err := driver.Query("SELECT * FROM users WHERE id = ?", 1)

\tassert.NoError(t, err)
\tassert.NotNil(t, result)
\tassert.NoError(t, mock.ExpectationsWereMet())
}
```

---

#### Day 3: NoSQL & Specialized Drivers
**Focus**: MongoDB, Elasticsearch, ClickHouse, TiDB

**Tasks**:
1. Create `pkg/database/mongodb_test.go`
2. Create `pkg/database/clickhouse_test.go`
3. Create `pkg/database/tidb_test.go`
4. Enhance `pkg/database/elasticsearch_test.go` (exists but may need expansion)

**Files to Create** (3 files + 1 enhancement):
- `pkg/database/mongodb_test.go`
- `pkg/database/clickhouse_test.go`
- `pkg/database/tidb_test.go`

---

#### Day 4: Schema & Caching
**Focus**: Schema introspection and caching layer

**Tasks**:
1. Create `pkg/database/queryparser_test.go`
2. Create `pkg/database/schema_cache_test.go`
3. Create `pkg/database/schema_cache_manager_test.go`
4. Create `pkg/database/structure_cache_test.go`
5. Create `pkg/database/ssh_tunnel_test.go`

**Files to Create** (5 files):
- `pkg/database/queryparser_test.go`
- `pkg/database/schema_cache_test.go`
- `pkg/database/schema_cache_manager_test.go`
- `pkg/database/structure_cache_test.go`
- `pkg/database/ssh_tunnel_test.go`

---

#### Day 5: Storage & Server Layer
**Focus**: Storage layer and HTTP/gRPC servers

**Tasks**:
1. Create `pkg/storage/manager_test.go`
2. Create `internal/server/http_test.go`
3. Create `internal/server/grpc_test.go`
4. Run coverage analysis
5. Fix any gaps in database layer

**Files to Create** (3 files):
- `pkg/storage/manager_test.go`
- `internal/server/http_test.go`
- `internal/server/grpc_test.go`

**Week 1 Goal**: 80%+ coverage of infrastructure layer

---

### Week 2: Business Logic & AI (Days 6-10)

#### Day 6: AI Service Core
**Focus**: AI service and provider abstraction

**Tasks**:
1. Create `internal/ai/service_test.go`
2. Create `internal/ai/provider_test.go`
3. Create `internal/ai/adapter_wrapper_test.go`
4. Create `internal/ai/types_test.go`

**Files to Create** (4 files):
- `internal/ai/service_test.go`
- `internal/ai/provider_test.go`
- `internal/ai/adapter_wrapper_test.go`
- `internal/ai/types_test.go`

---

#### Day 7: AI Providers (Anthropic, OpenAI, Ollama)
**Focus**: Individual AI provider implementations

**Tasks**:
1. Create `internal/ai/anthropic_test.go`
2. Create `internal/ai/openai_test.go`
3. Create `internal/ai/ollama_test.go`
4. Create `internal/ai/ollama_detector_test.go`
5. Mock HTTP calls to AI APIs

**Files to Create** (4 files):
- `internal/ai/anthropic_test.go`
- `internal/ai/openai_test.go`
- `internal/ai/ollama_test.go`
- `internal/ai/ollama_detector_test.go`

**Key Pattern**: Mock external HTTP calls
```go
func TestAnthropicProvider(t *testing.T) {
\tserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
\t\t// Mock Anthropic API response
\t\tw.WriteHeader(http.StatusOK)
\t\tjson.NewEncoder(w).Encode(mockResponse)
\t}))
\tdefer server.Close()

\tprovider := NewAnthropicProvider(server.URL, "test-key")
\t// test provider methods
}
```

---

#### Day 8: AI Specialized Providers & Handlers
**Focus**: ClaudeCode, Codex, HuggingFace, handlers

**Tasks**:
1. Create `internal/ai/claudecode_test.go`
2. Create `internal/ai/codex_test.go`
3. Create `internal/ai/huggingface_test.go`
4. Create `internal/ai/handlers_test.go`
5. Create `internal/ai/grpc_test.go`

**Files to Create** (5 files):
- `internal/ai/claudecode_test.go`
- `internal/ai/codex_test.go`
- `internal/ai/huggingface_test.go`
- `internal/ai/handlers_test.go`
- `internal/ai/grpc_test.go`

---

#### Day 9: RAG System (Part 1)
**Focus**: Embeddings and vector storage

**Tasks**:
1. Create `internal/rag/embedding_service_test.go`
2. Create `internal/rag/embedding_utils_test.go`
3. Create `internal/rag/sqlite_vector_store_test.go`
4. Create `internal/rag/mysql_vector_store_test.go`

**Files to Create** (4 files):
- `internal/rag/embedding_service_test.go`
- `internal/rag/embedding_utils_test.go`
- `internal/rag/sqlite_vector_store_test.go`
- `internal/rag/mysql_vector_store_test.go`

---

#### Day 10: RAG System (Part 2) & Auth/Middleware
**Focus**: SQL generation, visualization, context building

**Tasks**:
1. Create `internal/rag/smart_sql_generator_test.go`
2. Create `internal/rag/visualization_engine_test.go`
3. Create `internal/rag/context_builder_test.go`
4. Create `internal/rag/helpers_test.go`
5. Create `internal/auth/service_test.go`
6. Create `internal/middleware/auth_test.go`
7. Create `internal/middleware/ratelimit_test.go`

**Files to Create** (7 files):
- `internal/rag/smart_sql_generator_test.go`
- `internal/rag/visualization_engine_test.go`
- `internal/rag/context_builder_test.go`
- `internal/rag/helpers_test.go`
- `internal/auth/service_test.go`
- `internal/middleware/auth_test.go`
- `internal/middleware/ratelimit_test.go`

**Week 2 Goal**: 90%+ coverage of business logic

---

### Week 3: Integration, E2E & Polish (Days 11-15)

#### Day 11: Multiquery & Config
**Focus**: Finish multiquery package and config

**Tasks**:
1. Create `pkg/database/multiquery/merger_test.go`
2. Create `pkg/database/multiquery/types_test.go`
3. Create `internal/config/config_test.go`
4. Create `internal/services/services_test.go`
5. Create `internal/services/stores_test.go`

**Files to Create** (5 files):
- `pkg/database/multiquery/merger_test.go`
- `pkg/database/multiquery/types_test.go`
- `internal/config/config_test.go`
- `internal/services/services_test.go`
- `internal/services/stores_test.go`

---

#### Day 12: Integration Tests
**Focus**: Cross-component integration tests

**Tasks**:
1. Create `pkg/database/integration_test.go` (+build integration)
2. Create `pkg/storage/integration_test.go` (+build integration)
3. Create `internal/ai/integration_test.go` (+build integration)
4. Create `internal/rag/integration_test.go` (+build integration)
5. Create `internal/server/integration_test.go` (+build integration)

**Files to Create** (5 files):
- All with `// +build integration` tag
- Use testcontainers for real databases
- Test real API workflows

**Example Integration Test**:
```go
// +build integration

package database_test

import (
\t"testing"
\t"github.com/testcontainers/testcontainers-go"
)

func TestPostgresIntegration(t *testing.T) {
\tif testing.Short() {
\t\tt.Skip("skipping integration test")
\t}

\tctx := context.Background()
\tcontainer, err := testcontainers.RunContainer(ctx, ...)
\trequire.NoError(t, err)
\tdefer container.Terminate(ctx)

\t// Test with real Postgres
}
```

---

#### Day 13: E2E Tests
**Focus**: Full system end-to-end tests

**Tasks**:
1. Create `tests/e2e/api_test.go` (+build e2e)
2. Create `tests/e2e/ai_workflow_test.go` (+build e2e)
3. Create `tests/e2e/database_workflow_test.go` (+build e2e)
4. Test complete user workflows
5. Test error scenarios

**Files to Create** (3 files):
- Complete API workflows
- AI query workflows
- Database query workflows

---

#### Day 14: Coverage Analysis & Gap Filling
**Focus**: Achieve 100% on critical paths

**Tasks**:
1. Run full coverage analysis: `make -f Makefile.test test-coverage`
2. Identify gaps with `go tool cover -html=coverage.out`
3. Create tests for uncovered code paths
4. Focus on error handling paths
5. Review and improve test quality

**Commands**:
```bash
# Generate coverage report
make -f Makefile.test test-coverage

# View in browser
open coverage.html

# Check per-function coverage
make -f Makefile.test test-coverage-func
```

---

#### Day 15: CI/CD, Documentation & Polish
**Focus**: Finalize everything

**Tasks**:
1. Verify CI/CD pipeline works
2. Update documentation
3. Add code examples to README
4. Create pre-commit hooks
5. Final review and cleanup

**Final Checklist**:
- [ ] All test files created (~61 files)
- [ ] Coverage >= 95% overall
- [ ] Coverage == 100% on critical paths (database, AI, RAG)
- [ ] All tests passing in CI/CD
- [ ] No flaky tests
- [ ] Documentation complete
- [ ] Code reviewed

---

## 📊 Progress Tracking

### Coverage Milestones

| Milestone | Target | Components |
|-----------|--------|------------|
| Week 1 End | 80% | Database, Storage, Servers |
| Week 2 End | 90% | AI, RAG, Auth, Middleware |
| Week 3 End | 95%+ | Integration, E2E, Full coverage |

### Test File Count

| Component | Target Files | Created | Status |
|-----------|--------------|---------|--------|
| Database Layer | 15 | 4 ✅ | 🟡 In Progress |
| AI Service | 12 | 0 | ⚪ Pending |
| RAG System | 9 | 0 | ⚪ Pending |
| Server Layer | 3 | 0 | ⚪ Pending |
| Storage | 2 | 1 ✅ | 🟡 In Progress |
| Auth/Middleware | 3 | 0 | ⚪ Pending |
| Multiquery | 4 | 2 ✅ | 🟡 In Progress |
| Config/Services | 3 | 0 | ⚪ Pending |
| Integration Tests | 5 | 0 | ⚪ Pending |
| E2E Tests | 3 | 0 | ⚪ Pending |
| **TOTAL** | **~61** | **4** | **7% Complete** |

---

## 🛠️ Daily Workflow

### Morning Routine (30 min)
1. Pull latest changes
2. Review previous day's coverage
3. Plan day's test files
4. Generate/update mocks if needed

### Development (6-7 hours)
1. Write tests for assigned component
2. Run tests frequently: `go test ./...`
3. Check coverage: `make -f Makefile.test test-coverage`
4. Fix failing tests immediately
5. Commit completed test files

### End of Day (30 min)
1. Run full test suite
2. Check coverage metrics
3. Document any blockers
4. Update progress tracking
5. Push changes

---

## 🚨 Risk Mitigation

### Potential Blockers

**1. External Dependencies**
- **Risk**: AI APIs, databases unavailable
- **Mitigation**: Use mocks for unit tests, testcontainers for integration

**2. Flaky Tests**
- **Risk**: Tests fail intermittently
- **Mitigation**: Avoid time-dependent tests, use proper cleanup, test in isolation

**3. Coverage Gaps**
- **Risk**: Hard-to-test code paths
- **Mitigation**: Refactor for testability, use table-driven tests, focus on critical paths

**4. Time Constraints**
- **Risk**: 3 weeks too aggressive
- **Mitigation**: Prioritize critical paths (database, AI, RAG), defer nice-to-have tests

---

## 📈 Measuring Success

### Quantitative Metrics
- **Overall Coverage**: >= 95%
- **Critical Path Coverage**: 100% (database, AI, RAG, auth)
- **Test Execution Time**: < 2 minutes (unit tests)
- **CI/CD Build Time**: < 5 minutes (with integration tests)
- **Flaky Test Rate**: 0%

### Qualitative Metrics
- All edge cases documented and tested
- Error paths thoroughly tested
- Integration tests validate real workflows
- Tests serve as documentation
- Refactoring confidence

---

## 🎓 Learning Resources

### Go Testing
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Advanced Testing](https://golang.org/doc/tutorial/fuzz)

### Tools
- [Testify](https://github.com/stretchr/testify)
- [Mockery](https://vektra.github.io/mockery/)
- [SQL Mock](https://github.com/DATA-DOG/go-sqlmock)
- [Testcontainers](https://golang.testcontainers.org/)

### Best Practices
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [100 Go Mistakes](https://100go.co/)

---

## ✅ Ready to Start!

**All infrastructure is in place. Let's achieve 100% coverage! 🚀**

**First Command**:
```bash
cd backend-go
make -f Makefile.test test-unit
```

**First Task**: Create `pkg/database/manager_test.go` and `pkg/database/pool_test.go`

---

*Roadmap created with comprehensive research and industry best practices. Updated: 2025*
