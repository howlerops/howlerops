package services

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/internal/ai"
	"github.com/sql-studio/backend-go/internal/config"
	"github.com/sql-studio/backend-go/pkg/database"
	"github.com/sql-studio/backend-go/internal/middleware"
)

// Services holds all the application services
type Services struct {
	Auth     *auth.Service
	Database *database.Manager
	AI       ai.Service
	Query    *QueryService
	Table    *TableService
	Health   *HealthService
	Realtime *RealtimeService
}

// Config holds service configuration
type Config struct {
	Database config.DatabaseConfig
	Auth     config.AuthConfig
	Security config.SecurityConfig
}

// NewServices creates a new services instance
func NewServices(
	cfg Config,
	dbManager *database.Manager,
	authMiddleware *middleware.AuthMiddleware,
	logger *logrus.Logger,
) (*Services, error) {
	// Create auth service with in-memory stores for simplicity
	// In production, you'd use persistent storage
	authService := auth.NewService(
		&InMemoryUserStore{},
		&InMemorySessionStore{},
		&InMemoryLoginAttemptStore{},
		authMiddleware,
		auth.Config{
			BcryptCost:        cfg.Auth.BcryptCost,
			JWTExpiration:     cfg.Auth.JWTExpiration,
			RefreshExpiration: cfg.Auth.RefreshExpiration,
			MaxLoginAttempts:  cfg.Auth.MaxLoginAttempts,
			LockoutDuration:   cfg.Auth.LockoutDuration,
		},
		logger,
	)

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
		Auth:     authService,
		Database: dbManager,
		AI:       aiService,
		Query:    queryService,
		Table:    tableService,
		Health:   healthService,
		Realtime: realtimeService,
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