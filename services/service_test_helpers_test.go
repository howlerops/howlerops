package services

import (
	"context"
	"sync"
	"time"

	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	"github.com/jbeck018/howlerops/backend-go/pkg/database/multiquery"
)

type eventRecord struct {
	name    string
	payload interface{}
}

type recordingEmitter struct {
	mu     sync.Mutex
	events []eventRecord
}

func newRecordingEmitter() *recordingEmitter {
	return &recordingEmitter{
		events: make([]eventRecord, 0),
	}
}

func (e *recordingEmitter) Emit(_ context.Context, event string, data interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, eventRecord{name: event, payload: data})
	return nil
}

func (e *recordingEmitter) Events() []eventRecord {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]eventRecord, len(e.events))
	copy(out, e.events)
	return out
}

func (e *recordingEmitter) Count(event string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	count := 0
	for _, rec := range e.events {
		if rec.name == event {
			count++
		}
	}
	return count
}

func (e *recordingEmitter) LastEvent() *eventRecord {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.events) == 0 {
		return nil
	}
	rec := e.events[len(e.events)-1]
	return &rec
}

func (e *recordingEmitter) WaitFor(name string, timeout time.Duration) (*eventRecord, bool) {
	deadline := time.Now().Add(timeout)
	for {
		e.mu.Lock()
		for _, rec := range e.events {
			if rec.name == name {
				copyRec := rec
				e.mu.Unlock()
				return &copyRec, true
			}
		}
		e.mu.Unlock()

		if time.Now().After(deadline) {
			return nil, false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type stubDatabaseManager struct {
	createConnectionFn      func(ctx context.Context, config database.ConnectionConfig) (*database.Connection, error)
	getMultiSchemaFn        func(ctx context.Context, connectionIDs []string) (*multiquery.CombinedSchema, error)
	testConnectionFn        func(ctx context.Context, config database.ConnectionConfig) error
	listConnectionsFn       func() []string
	removeConnectionFn      func(connectionID string) error
	listDatabasesFn         func(ctx context.Context, connectionID string) ([]string, error)
	getConnectionFn         func(connectionID string) (database.Database, error)
	updateRowFn             func(ctx context.Context, connectionID string, params database.UpdateRowParams) error
	insertRowFn             func(ctx context.Context, connectionID string, params database.InsertRowParams) (map[string]interface{}, error)
	deleteRowFn             func(ctx context.Context, connectionID string, params database.DeleteRowParams) error
	switchDatabaseFn        func(ctx context.Context, connectionID string, databaseName string) (database.ConnectionConfig, bool, error)
	getConnectionHealthFn   func(ctx context.Context, connectionID string) (*database.HealthStatus, error)
	getConnectionStatsFn    func() map[string]database.PoolStats
	healthCheckAllFn        func(ctx context.Context) map[string]*database.HealthStatus
	closeFn                 func() error
	executeMultiQueryFn     func(ctx context.Context, query string, options *multiquery.Options) (*multiquery.Result, error)
	parseMultiQueryFn       func(query string) (*multiquery.ParsedQuery, error)
	validateMultiQueryFn    func(parsed *multiquery.ParsedQuery) error
	invalidateSchemaCacheFn func(connectionID string)
	invalidateAllSchemasFn  func()
	refreshSchemaFn         func(ctx context.Context, connectionID string) error
	getSchemaCacheStatsFn   func() map[string]interface{}
	getConnectionCountFn    func() int
	getConnectionIDsFn      func() []string
}

func (s *stubDatabaseManager) CreateConnection(ctx context.Context, config database.ConnectionConfig) (*database.Connection, error) {
	if s.createConnectionFn != nil {
		return s.createConnectionFn(ctx, config)
	}
	return &database.Connection{}, nil
}

func (s *stubDatabaseManager) GetMultiConnectionSchema(ctx context.Context, connectionIDs []string) (*multiquery.CombinedSchema, error) {
	if s.getMultiSchemaFn != nil {
		return s.getMultiSchemaFn(ctx, connectionIDs)
	}
	return &multiquery.CombinedSchema{}, nil
}

func (s *stubDatabaseManager) TestConnection(ctx context.Context, config database.ConnectionConfig) error {
	if s.testConnectionFn != nil {
		return s.testConnectionFn(ctx, config)
	}
	return nil
}

func (s *stubDatabaseManager) ListConnections() []string {
	if s.listConnectionsFn != nil {
		return s.listConnectionsFn()
	}
	return nil
}

func (s *stubDatabaseManager) RemoveConnection(connectionID string) error {
	if s.removeConnectionFn != nil {
		return s.removeConnectionFn(connectionID)
	}
	return nil
}

func (s *stubDatabaseManager) ListDatabases(ctx context.Context, connectionID string) ([]string, error) {
	if s.listDatabasesFn != nil {
		return s.listDatabasesFn(ctx, connectionID)
	}
	return []string{}, nil
}

func (s *stubDatabaseManager) GetConnection(connectionID string) (database.Database, error) {
	if s.getConnectionFn != nil {
		return s.getConnectionFn(connectionID)
	}
	return nil, nil
}

func (s *stubDatabaseManager) UpdateRow(ctx context.Context, connectionID string, params database.UpdateRowParams) error {
	if s.updateRowFn != nil {
		return s.updateRowFn(ctx, connectionID, params)
	}
	return nil
}

func (s *stubDatabaseManager) InsertRow(ctx context.Context, connectionID string, params database.InsertRowParams) (map[string]interface{}, error) {
	if s.insertRowFn != nil {
		return s.insertRowFn(ctx, connectionID, params)
	}
	return map[string]interface{}{}, nil
}

func (s *stubDatabaseManager) DeleteRow(ctx context.Context, connectionID string, params database.DeleteRowParams) error {
	if s.deleteRowFn != nil {
		return s.deleteRowFn(ctx, connectionID, params)
	}
	return nil
}

func (s *stubDatabaseManager) SwitchDatabase(ctx context.Context, connectionID string, databaseName string) (database.ConnectionConfig, bool, error) {
	if s.switchDatabaseFn != nil {
		return s.switchDatabaseFn(ctx, connectionID, databaseName)
	}
	return database.ConnectionConfig{Database: databaseName}, false, nil
}

func (s *stubDatabaseManager) GetConnectionHealth(ctx context.Context, connectionID string) (*database.HealthStatus, error) {
	if s.getConnectionHealthFn != nil {
		return s.getConnectionHealthFn(ctx, connectionID)
	}
	return nil, nil
}

func (s *stubDatabaseManager) GetConnectionStats() map[string]database.PoolStats {
	if s.getConnectionStatsFn != nil {
		return s.getConnectionStatsFn()
	}
	return map[string]database.PoolStats{}
}

func (s *stubDatabaseManager) HealthCheckAll(ctx context.Context) map[string]*database.HealthStatus {
	if s.healthCheckAllFn != nil {
		return s.healthCheckAllFn(ctx)
	}
	return nil
}

func (s *stubDatabaseManager) Close() error {
	if s.closeFn != nil {
		return s.closeFn()
	}
	return nil
}

func (s *stubDatabaseManager) ExecuteMultiQuery(ctx context.Context, query string, options *multiquery.Options) (*multiquery.Result, error) {
	if s.executeMultiQueryFn != nil {
		return s.executeMultiQueryFn(ctx, query, options)
	}
	return &multiquery.Result{}, nil
}

func (s *stubDatabaseManager) ParseMultiQuery(query string) (*multiquery.ParsedQuery, error) {
	if s.parseMultiQueryFn != nil {
		return s.parseMultiQueryFn(query)
	}
	return &multiquery.ParsedQuery{}, nil
}

func (s *stubDatabaseManager) ValidateMultiQuery(parsed *multiquery.ParsedQuery) error {
	if s.validateMultiQueryFn != nil {
		return s.validateMultiQueryFn(parsed)
	}
	return nil
}

func (s *stubDatabaseManager) InvalidateSchemaCache(connectionID string) {
	if s.invalidateSchemaCacheFn != nil {
		s.invalidateSchemaCacheFn(connectionID)
	}
}

func (s *stubDatabaseManager) InvalidateAllSchemas() {
	if s.invalidateAllSchemasFn != nil {
		s.invalidateAllSchemasFn()
	}
}

func (s *stubDatabaseManager) RefreshSchema(ctx context.Context, connectionID string) error {
	if s.refreshSchemaFn != nil {
		return s.refreshSchemaFn(ctx, connectionID)
	}
	return nil
}

func (s *stubDatabaseManager) GetSchemaCacheStats() map[string]interface{} {
	if s.getSchemaCacheStatsFn != nil {
		return s.getSchemaCacheStatsFn()
	}
	return nil
}

func (s *stubDatabaseManager) GetConnectionCount() int {
	if s.getConnectionCountFn != nil {
		return s.getConnectionCountFn()
	}
	return 0
}

func (s *stubDatabaseManager) GetConnectionIDs() []string {
	if s.getConnectionIDsFn != nil {
		return s.getConnectionIDsFn()
	}
	return nil
}

type fakeDatabase struct {
	executeResult        *database.QueryResult
	executeErr           error
	executeStreamErr     error
	metadata             *database.EditableQueryMetadata
	metadataErr          error
	connectionInfo       map[string]interface{}
	connectionInfoErr    error
	editableMetadataArgs struct {
		query   string
		columns []string
	}
	beginTransactionCalled bool
	beginTransactionErr    error
	beginTransaction       database.Transaction
	insertRowResult        map[string]interface{}
	insertRowErr           error
	deleteRowErr           error
}

func (f *fakeDatabase) Connect(ctx context.Context, config database.ConnectionConfig) error {
	return nil
}

func (f *fakeDatabase) Disconnect() error { return nil }

func (f *fakeDatabase) Ping(ctx context.Context) error { return nil }

func (f *fakeDatabase) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	return f.connectionInfo, f.connectionInfoErr
}

func (f *fakeDatabase) Execute(ctx context.Context, query string, args ...interface{}) (*database.QueryResult, error) {
	return f.executeResult, f.executeErr
}

func (f *fakeDatabase) ExecuteWithOptions(ctx context.Context, query string, opts *database.QueryOptions, args ...interface{}) (*database.QueryResult, error) {
	return f.executeResult, f.executeErr
}

func (f *fakeDatabase) ExecuteStream(ctx context.Context, query string, batchSize int, callback func([][]interface{}) error, args ...interface{}) error {
	return f.executeStreamErr
}

func (f *fakeDatabase) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	return "", nil
}

func (f *fakeDatabase) ComputeEditableMetadata(ctx context.Context, query string, columns []string) (*database.EditableQueryMetadata, error) {
	f.editableMetadataArgs.query = query
	f.editableMetadataArgs.columns = columns
	return f.metadata, f.metadataErr
}

func (f *fakeDatabase) GetSchemas(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (f *fakeDatabase) GetTables(ctx context.Context, schema string) ([]database.TableInfo, error) {
	return nil, nil
}

func (f *fakeDatabase) GetTableStructure(ctx context.Context, schema, table string) (*database.TableStructure, error) {
	return nil, nil
}

func (f *fakeDatabase) BeginTransaction(ctx context.Context) (database.Transaction, error) {
	f.beginTransactionCalled = true
	if f.beginTransaction != nil {
		return f.beginTransaction, f.beginTransactionErr
	}
	if f.beginTransactionErr != nil {
		return nil, f.beginTransactionErr
	}
	return noOpTransaction{}, nil
}

func (f *fakeDatabase) UpdateRow(ctx context.Context, params database.UpdateRowParams) error {
	return nil
}

func (f *fakeDatabase) InsertRow(ctx context.Context, params database.InsertRowParams) (map[string]interface{}, error) {
	return f.insertRowResult, f.insertRowErr
}

func (f *fakeDatabase) DeleteRow(ctx context.Context, params database.DeleteRowParams) error {
	return f.deleteRowErr
}

func (f *fakeDatabase) ListDatabases(ctx context.Context) ([]string, error) {
	return []string{"testdb"}, nil
}

func (f *fakeDatabase) SwitchDatabase(ctx context.Context, databaseName string) error {
	return nil
}

func (f *fakeDatabase) GetDatabaseType() database.DatabaseType {
	return database.DatabaseType("fake")
}

func (f *fakeDatabase) GetConnectionStats() database.PoolStats {
	return database.PoolStats{}
}

func (f *fakeDatabase) QuoteIdentifier(identifier string) string {
	return `"` + identifier + `"`
}

func (f *fakeDatabase) GetDataTypeMappings() map[string]string {
	return map[string]string{}
}

type noOpTransaction struct{}

func (noOpTransaction) Execute(ctx context.Context, query string, args ...interface{}) (*database.QueryResult, error) {
	return &database.QueryResult{
		Rows:     [][]interface{}{},
		Columns:  []string{},
		Duration: time.Millisecond,
	}, nil
}

func (noOpTransaction) Commit() error   { return nil }
func (noOpTransaction) Rollback() error { return nil }
