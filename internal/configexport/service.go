package configexport

import (
	"context"

	"github.com/jbeck018/howlerops/pkg/export"
	"github.com/jbeck018/howlerops/pkg/storage"
	"github.com/sirupsen/logrus"
)

// ConnectionStore defines the interface for connection storage
type ConnectionStore interface {
	GetByUserID(ctx context.Context, userID string) ([]*storage.Connection, error)
	GetByName(ctx context.Context, userID, name string) (*storage.Connection, error)
	Create(ctx context.Context, conn *storage.Connection) error
	Update(ctx context.Context, conn *storage.Connection) error
}

// QueryStore defines the interface for query storage
type QueryStore interface {
	GetByUserID(ctx context.Context, userID string) ([]*storage.SavedQuery, error)
	GetByName(ctx context.Context, userID, name string) (*storage.SavedQuery, error)
	Create(ctx context.Context, query *storage.SavedQuery) error
	Update(ctx context.Context, query *storage.SavedQuery) error
}

// HistoryStore defines the interface for query history storage
type HistoryStore interface {
	GetByUserID(ctx context.Context, userID string, limit int) ([]*storage.QueryHistory, error)
}

// Service handles config export/import business logic
type Service struct {
	connStore    ConnectionStore
	queryStore   QueryStore
	historyStore HistoryStore
	exporter     *export.ConfigExporter
	importer     *export.ConfigImporter
	logger       *logrus.Logger
	appVersion   string
}

// NewService creates a new config export service
func NewService(
	connStore ConnectionStore,
	queryStore QueryStore,
	historyStore HistoryStore,
	logger *logrus.Logger,
	appVersion string,
) *Service {
	return &Service{
		connStore:    connStore,
		queryStore:   queryStore,
		historyStore: historyStore,
		exporter:     export.NewConfigExporter(logger, appVersion),
		importer:     export.NewConfigImporter(logger),
		logger:       logger,
		appVersion:   appVersion,
	}
}

// Export creates an exported config for the user
func (s *Service) Export(
	ctx context.Context,
	userID string,
	userEmail string,
	options export.ConfigExportOptions,
) (*export.ExportedConfig, error) {
	// Fetch all user data
	var connections []storage.Connection
	var queries []storage.SavedQuery
	var history []storage.QueryHistory

	// Get connections
	if options.IncludeConnections {
		conns, err := s.connStore.GetByUserID(ctx, userID)
		if err != nil {
			s.logger.WithError(err).Error("Failed to fetch connections for export")
			return nil, err
		}
		for _, c := range conns {
			if c != nil {
				connections = append(connections, *c)
			}
		}
	}

	// Get saved queries
	if options.IncludeSavedQueries {
		qs, err := s.queryStore.GetByUserID(ctx, userID)
		if err != nil {
			s.logger.WithError(err).Error("Failed to fetch queries for export")
			return nil, err
		}
		for _, q := range qs {
			if q != nil {
				queries = append(queries, *q)
			}
		}
	}

	// Get query history
	if options.IncludeQueryHistory && s.historyStore != nil {
		h, err := s.historyStore.GetByUserID(ctx, userID, 1000) // Limit history
		if err != nil {
			s.logger.WithError(err).Warn("Failed to fetch history for export, continuing without")
		} else {
			for _, item := range h {
				if item != nil {
					history = append(history, *item)
				}
			}
		}
	}

	return s.exporter.Export(ctx, userEmail, connections, queries, history, options)
}

// Import imports a config for the user
func (s *Service) Import(
	ctx context.Context,
	config *export.ExportedConfig,
	userID string,
	options export.ConfigImportOptions,
) (*export.ImportResult, error) {
	// Create adapter stores that wrap our stores to match the importer interface
	connAdapter := &connectionStoreAdapter{store: s.connStore}
	queryAdapter := &queryStoreAdapter{store: s.queryStore}

	return s.importer.Import(ctx, config, userID, connAdapter, queryAdapter, options)
}

// Adapters to match the import interfaces

type connectionStoreAdapter struct {
	store ConnectionStore
}

func (a *connectionStoreAdapter) GetByName(ctx context.Context, userID, name string) (*storage.Connection, error) {
	return a.store.GetByName(ctx, userID, name)
}

func (a *connectionStoreAdapter) Create(ctx context.Context, conn *storage.Connection) error {
	return a.store.Create(ctx, conn)
}

func (a *connectionStoreAdapter) Update(ctx context.Context, conn *storage.Connection) error {
	return a.store.Update(ctx, conn)
}

type queryStoreAdapter struct {
	store QueryStore
}

func (a *queryStoreAdapter) GetByName(ctx context.Context, userID, name string) (*storage.SavedQuery, error) {
	return a.store.GetByName(ctx, userID, name)
}

func (a *queryStoreAdapter) Create(ctx context.Context, query *storage.SavedQuery) error {
	return a.store.Create(ctx, query)
}

func (a *queryStoreAdapter) Update(ctx context.Context, query *storage.SavedQuery) error {
	return a.store.Update(ctx, query)
}
