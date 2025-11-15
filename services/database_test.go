package services

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	"github.com/jbeck018/howlerops/backend-go/pkg/database/multiquery"
)

func newSilentLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestDatabaseServiceCreateConnectionEmitsEvent(t *testing.T) {
	emitter := newRecordingEmitter()
	manager := &stubDatabaseManager{
		createConnectionFn: func(ctx context.Context, cfg database.ConnectionConfig) (*database.Connection, error) {
			if cfg.Database != "analytics" {
				t.Fatalf("unexpected database config: %+v", cfg)
			}
			return &database.Connection{
				ID:   "conn-123",
				Name: "analytics",
			}, nil
		},
	}

	service := NewDatabaseServiceWithDependencies(newSilentLogger(), manager, emitter)
	service.SetContext(context.Background())

	conn, err := service.CreateConnection(database.ConnectionConfig{Type: "postgres", Database: "analytics"})
	if err != nil {
		t.Fatalf("CreateConnection returned error: %v", err)
	}
	if conn.ID != "conn-123" {
		t.Fatalf("expected connection ID conn-123, got %s", conn.ID)
	}

	if _, ok := emitter.WaitFor("connection:created", 100*time.Millisecond); !ok {
		t.Fatalf("expected connection:created event to be emitted")
	}
}

func TestDatabaseServiceRemoveConnectionEmitsEvent(t *testing.T) {
	emitter := newRecordingEmitter()
	var removed string
	manager := &stubDatabaseManager{
		removeConnectionFn: func(connectionID string) error {
			removed = connectionID
			return nil
		},
	}

	service := NewDatabaseServiceWithDependencies(newSilentLogger(), manager, emitter)
	service.SetContext(context.Background())

	if err := service.RemoveConnection("conn-456"); err != nil {
		t.Fatalf("RemoveConnection returned error: %v", err)
	}
	if removed != "conn-456" {
		t.Fatalf("expected removeConnection to receive conn-456, got %s", removed)
	}

	if _, ok := emitter.WaitFor("connection:removed", 100*time.Millisecond); !ok {
		t.Fatalf("expected connection:removed event to be emitted")
	}
}

func TestDatabaseServiceExecuteQueryEmitsEventsAndMetadataJob(t *testing.T) {
	emitter := newRecordingEmitter()
	fakeDB := &fakeDatabase{
		executeResult: &database.QueryResult{
			Columns:  []string{"id"},
			Rows:     [][]interface{}{{1}},
			RowCount: 1,
			Affected: 0,
			Duration: 5 * time.Millisecond,
			Editable: &database.EditableQueryMetadata{
				Enabled: true,
				Pending: true,
			},
		},
		metadata: &database.EditableQueryMetadata{
			Enabled: true,
		},
	}
	manager := &stubDatabaseManager{
		getConnectionFn: func(connectionID string) (database.Database, error) {
			if connectionID != "primary" {
				t.Fatalf("unexpected connection ID: %s", connectionID)
			}
			return fakeDB, nil
		},
	}

	service := NewDatabaseServiceWithDependencies(newSilentLogger(), manager, emitter)
	service.SetContext(context.Background())

	result, err := service.ExecuteQuery("primary", "SELECT 1", nil)
	if err != nil {
		t.Fatalf("ExecuteQuery returned error: %v", err)
	}

	if result.RowCount != 1 {
		t.Fatalf("expected RowCount 1, got %d", result.RowCount)
	}
	if _, ok := emitter.WaitFor("query:executed", 100*time.Millisecond); !ok {
		t.Fatalf("expected query:executed event")
	}
	if _, ok := emitter.WaitFor("query:editableMetadata", time.Second); !ok {
		t.Fatalf("expected query:editableMetadata event")
	}
	if fakeDB.editableMetadataArgs.query != "SELECT 1" {
		t.Fatalf("expected metadata job to receive query, got %s", fakeDB.editableMetadataArgs.query)
	}
	if len(fakeDB.editableMetadataArgs.columns) != 1 || fakeDB.editableMetadataArgs.columns[0] != "id" {
		t.Fatalf("unexpected columns passed to metadata job: %#v", fakeDB.editableMetadataArgs.columns)
	}
}

func TestDatabaseServiceExecuteQueryError(t *testing.T) {
	emitter := newRecordingEmitter()
	fakeDB := &fakeDatabase{
		executeErr: errors.New("boom"),
	}
	manager := &stubDatabaseManager{
		getConnectionFn: func(connectionID string) (database.Database, error) {
			return fakeDB, nil
		},
	}

	service := NewDatabaseServiceWithDependencies(newSilentLogger(), manager, emitter)
	service.SetContext(context.Background())

	if _, err := service.ExecuteQuery("primary", "SELECT 1", nil); err == nil {
		t.Fatalf("expected error from ExecuteQuery")
	}
	if emitter.Count("query:executed") != 0 {
		t.Fatalf("unexpected query:executed event on error path")
	}
}

func TestDatabaseServiceExecuteMultiDatabaseQuery(t *testing.T) {
	emitter := newRecordingEmitter()
	manager := &stubDatabaseManager{
		executeMultiQueryFn: func(ctx context.Context, query string, options *multiquery.Options) (*multiquery.Result, error) {
			if query != "select 1" {
				t.Fatalf("unexpected query: %s", query)
			}
			return &multiquery.Result{
				Columns:         []string{"result"},
				Rows:            [][]interface{}{{1}},
				RowCount:        1,
				Duration:        20 * time.Millisecond,
				ConnectionsUsed: []string{"primary"},
				Strategy:        multiquery.StrategyFederated,
			}, nil
		},
	}

	service := NewDatabaseServiceWithDependencies(newSilentLogger(), manager, emitter)
	service.SetContext(context.Background())

	resp, err := service.ExecuteMultiDatabaseQuery("select 1", nil)
	if err != nil {
		t.Fatalf("ExecuteMultiDatabaseQuery returned error: %v", err)
	}
	if resp.RowCount != 1 {
		t.Fatalf("expected row count 1, got %d", resp.RowCount)
	}

	if _, ok := emitter.WaitFor("multiquery:executed", 100*time.Millisecond); !ok {
		t.Fatalf("expected multiquery:executed event")
	}
}

func TestDatabaseServiceBeginTransactionEmitsEvent(t *testing.T) {
	emitter := newRecordingEmitter()
	fakeDB := &fakeDatabase{}
	manager := &stubDatabaseManager{
		getConnectionFn: func(connectionID string) (database.Database, error) {
			return fakeDB, nil
		},
	}

	service := NewDatabaseServiceWithDependencies(newSilentLogger(), manager, emitter)
	service.SetContext(context.Background())

	if _, err := service.BeginTransaction("primary"); err != nil {
		t.Fatalf("BeginTransaction returned error: %v", err)
	}
	if !fakeDB.beginTransactionCalled {
		t.Fatalf("expected BeginTransaction to call underlying database")
	}
	if _, ok := emitter.WaitFor("transaction:started", 100*time.Millisecond); !ok {
		t.Fatalf("expected transaction:started event")
	}
}

func TestDatabaseServiceClose(t *testing.T) {
	called := false
	manager := &stubDatabaseManager{
		closeFn: func() error {
			called = true
			return nil
		},
	}

	service := NewDatabaseServiceWithDependencies(newSilentLogger(), manager, newRecordingEmitter())
	if err := service.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected manager Close to be called")
	}
}
