package services

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/sql-studio/backend-go/internal/ai"
	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/internal/config"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/internal/organization"
	"github.com/sql-studio/backend-go/internal/sync"
	"github.com/sql-studio/backend-go/pkg/database"
)

// Services holds all the application services
type Services struct {
	Auth         *auth.Service
	Sync         *sync.Service
	Organization organization.ServiceInterface
	Database     *database.Manager
	AI           ai.Service
	Query        *QueryService
	Table        *TableService
	Health       *HealthService
	Realtime     *RealtimeService
}

// Config holds service configuration
type Config struct {
	Database config.DatabaseConfig
	Auth     config.AuthConfig
	Security config.SecurityConfig
}

// NewServices creates a new services instance
// Note: Auth and Sync services are injected from main.go
// This function creates the supporting services that depend on database manager
func NewServices(
	cfg Config,
	dbManager *database.Manager,
	authMiddleware *middleware.AuthMiddleware,
	logger *logrus.Logger,
) (*Services, error) {
	// Create query service
	queryService := NewQueryService(dbManager, logger)

	// Create table service
	tableService := NewTableService(dbManager, logger)

	// Create health service
	healthService := NewHealthService(dbManager, logger)

	// Create realtime service
	realtimeService := NewRealtimeService(logger)

	// Create AI service
	aiConfig, err := ai.LoadConfig()
	if err != nil {
		return nil, err
	}
	aiService, err := ai.NewService(aiConfig, logger)
	if err != nil {
		return nil, err
	}

	return &Services{
		// Auth, Sync, and Organization will be set by main.go after Turso initialization
		Auth:         nil, // Injected from main.go
		Sync:         nil, // Injected from main.go
		Organization: nil, // Injected from main.go
		Database:     dbManager,
		AI:           aiService,
		Query:        queryService,
		Table:        tableService,
		Health:       healthService,
		Realtime:     realtimeService,
	}, nil
}

// RegisterGRPCServices registers all gRPC services
func (s *Services) RegisterGRPCServices(server *grpc.Server) {
	// Register auth service
	// authpb.RegisterAuthServiceServer(server, NewAuthGRPCService(s.Auth))

	// Register database service
	// databasepb.RegisterDatabaseServiceServer(server, NewDatabaseGRPCService(s.Database))

	// Register query service
	// querypb.RegisterQueryServiceServer(server, NewQueryGRPCService(s.Query))

	// Register table service
	// tablepb.RegisterTableServiceServer(server, NewTableGRPCService(s.Table))

	// Register health service
	// healthpb.RegisterHealthServiceServer(server, NewHealthGRPCService(s.Health))

	// Register realtime service
	// realtimepb.RegisterRealtimeServiceServer(server, NewRealtimeGRPCService(s.Realtime))
}

// Close gracefully closes all services
func (s *Services) Close(ctx context.Context) error {
	// Close database connections
	if err := s.Database.Close(); err != nil {
		return err
	}

	// Close realtime service
	if err := s.Realtime.Close(); err != nil {
		return err
	}

	return nil
}

// QueryService handles query operations
type QueryService struct {
	dbManager *database.Manager
	logger    *logrus.Logger
}

// NewQueryService creates a new query service
func NewQueryService(dbManager *database.Manager, logger *logrus.Logger) *QueryService {
	return &QueryService{
		dbManager: dbManager,
		logger:    logger,
	}
}

// TableService handles table operations
type TableService struct {
	dbManager *database.Manager
	logger    *logrus.Logger
}

// NewTableService creates a new table service
func NewTableService(dbManager *database.Manager, logger *logrus.Logger) *TableService {
	return &TableService{
		dbManager: dbManager,
		logger:    logger,
	}
}

// HealthService handles health checks
type HealthService struct {
	dbManager *database.Manager
	logger    *logrus.Logger
}

// NewHealthService creates a new health service
func NewHealthService(dbManager *database.Manager, logger *logrus.Logger) *HealthService {
	return &HealthService{
		dbManager: dbManager,
		logger:    logger,
	}
}

// RealtimeService handles real-time events
type RealtimeService struct {
	logger      *logrus.Logger
	subscribers map[string]chan interface{}
}

// NewRealtimeService creates a new realtime service
func NewRealtimeService(logger *logrus.Logger) *RealtimeService {
	return &RealtimeService{
		logger:      logger,
		subscribers: make(map[string]chan interface{}),
	}
}

// Close closes the realtime service
func (r *RealtimeService) Close() error {
	// Close all subscriber channels
	for _, ch := range r.subscribers {
		close(ch)
	}
	return nil
}
