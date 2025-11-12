package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/internal/config"
	"github.com/sql-studio/backend-go/internal/email"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/internal/organization"
	"github.com/sql-studio/backend-go/internal/server"
	"github.com/sql-studio/backend-go/internal/services"
	appsync "github.com/sql-studio/backend-go/internal/sync"
	"github.com/sql-studio/backend-go/pkg/database"
	"github.com/sql-studio/backend-go/pkg/logger"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
	"github.com/sql-studio/backend-go/pkg/updater"
	"github.com/sql-studio/backend-go/pkg/version"
)

// Version information (set via ldflags during build)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	setupVersion()
	cfg, appLogger := initializeConfig()
	logStartupInfo(cfg, appLogger)

	// Check for updates and validate environment
	if !cfg.IsProduction() {
		go checkForUpdates(appLogger)
	}
	if validateErr := validateEnvironment(cfg, appLogger); validateErr != nil {
		appLogger.WithError(validateErr).Fatal("Environment validation failed")
	}

	// Initialize database and services
	tursoClient := initializeDatabase(cfg, appLogger)
	svc, authMiddleware, dbManager := initializeServices(cfg, tursoClient, appLogger)

	// Create gRPC server
	grpcServer, err := server.NewGRPCServer(cfg, appLogger, svc)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to create gRPC server")
	}

	// Create HTTP gateway server
	httpServer, err := server.NewHTTPServer(cfg, appLogger, svc, authMiddleware)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to create HTTP server")
	}

	// Create metrics server
	var metricsServer *http.Server
	if cfg.Metrics.Enabled {
		metricsServer = &http.Server{
			Addr:    cfg.GetMetricsAddress(),
			Handler: promhttp.Handler(),
		}
		setupMetrics()
	}

	// Create WebSocket server for real-time updates
	wsServer, err := server.NewWebSocketServer(cfg, appLogger, svc)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to create WebSocket server")
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		appLogger.Info("Starting gRPC server")
		if err := grpcServer.Start(); err != nil {
			appLogger.WithError(err).Error("gRPC server failed")
		}
	}()

	// Start HTTP gateway server
	wg.Add(1)
	go func() {
		defer wg.Done()
		appLogger.Info("Starting HTTP gateway server")
		if startErr := httpServer.Start(); startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
			appLogger.WithError(startErr).Error("HTTP server failed")
		}
	}()

	// Start metrics server
	if metricsServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			appLogger.WithField("address", cfg.GetMetricsAddress()).Info("Starting metrics server")
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				appLogger.WithError(err).Error("Metrics server failed")
			}
		}()
	}

	// Start WebSocket server
	wg.Add(1)
	go func() {
		defer wg.Done()
		appLogger.Info("Starting WebSocket server")
		if err := wsServer.Start(); err != nil {
			appLogger.WithError(err).Error("WebSocket server failed")
		}
	}()

	// Start background tasks
	wg.Add(1)
	go func() {
		defer wg.Done()
		runBackgroundTasks(ctx, svc, appLogger)
	}()

	appLogger.Info("All servers started successfully")
	appLogger.WithFields(logrus.Fields{
		"http_url":    fmt.Sprintf("http://localhost:%d", cfg.Server.HTTPPort),
		"grpc_url":    fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort),
		"metrics_url": fmt.Sprintf("http://localhost:%d/metrics", cfg.Metrics.Port),
		"health_url":  fmt.Sprintf("http://localhost:%d/health", cfg.Server.HTTPPort),
		"ws_url":      "ws://localhost:8081/ws",
	}).Info("Server URLs")

	// Wait for shutdown signal
	<-sigChan
	appLogger.Info("Shutdown signal received, starting graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Stop all servers gracefully
	var shutdownWg sync.WaitGroup

	// Stop gRPC server
	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		if err := grpcServer.Stop(shutdownCtx); err != nil {
			appLogger.WithError(err).Error("Failed to stop gRPC server gracefully")
		}
	}()

	// Stop HTTP server
	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		if err := httpServer.Stop(shutdownCtx); err != nil {
			appLogger.WithError(err).Error("Failed to stop HTTP server gracefully")
		}
	}()

	// Stop metrics server
	if metricsServer != nil {
		shutdownWg.Add(1)
		go func() {
			defer shutdownWg.Done()
			if err := metricsServer.Shutdown(shutdownCtx); err != nil {
				appLogger.WithError(err).Error("Failed to stop metrics server gracefully")
			}
		}()
	}

	// Stop WebSocket server
	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		if err := wsServer.Stop(shutdownCtx); err != nil {
			appLogger.WithError(err).Error("Failed to stop WebSocket server gracefully")
		}
	}()

	// Stop background tasks
	cancel()

	// Wait for all servers to stop
	shutdownWg.Wait()

	// Close database connections
	if err := tursoClient.Close(); err != nil {
		appLogger.WithError(err).Error("Failed to close Turso database connection")
	}

	if err := dbManager.Close(); err != nil {
		appLogger.WithError(err).Error("Failed to close database manager connections")
	}

	// Wait for background tasks to finish
	wg.Wait()

	appLogger.Info("Graceful shutdown completed")
}

// runBackgroundTasks runs periodic background tasks
func runBackgroundTasks(ctx context.Context, svc *services.Services, logger *logrus.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	logger.Info("Background tasks started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Background tasks stopped")
			return
		case <-ticker.C:
			runCleanupTasks(ctx, svc, logger)
		}
	}
}

// runCleanupTasks performs all periodic cleanup operations
func runCleanupTasks(ctx context.Context, svc *services.Services, logger *logrus.Logger) {
	// Cleanup expired sessions
	if err := svc.Auth.CleanupExpiredSessions(ctx); err != nil {
		logger.WithError(err).Error("Failed to cleanup expired sessions")
	} else {
		logger.Debug("Expired sessions cleaned up")
	}

	// Cleanup old login attempts
	if err := svc.Auth.CleanupOldLoginAttempts(ctx); err != nil {
		logger.WithError(err).Error("Failed to cleanup old login attempts")
	} else {
		logger.Debug("Old login attempts cleaned up")
	}

	// Health check all user database connections
	checkDatabaseHealth(ctx, svc, logger)

	logger.Debug("Background cleanup tasks completed successfully")
}

// checkDatabaseHealth performs health checks on all database connections
func checkDatabaseHealth(ctx context.Context, svc *services.Services, logger *logrus.Logger) {
	if svc.Database == nil {
		return
	}

	healthStatuses := svc.Database.HealthCheckAll(ctx)
	unhealthyCount := 0

	for connID, status := range healthStatuses {
		if status.Status != "healthy" {
			unhealthyCount++
			logger.WithFields(logrus.Fields{
				"connection_id": connID,
				"status":        status.Status,
				"message":       status.Message,
			}).Warn("Unhealthy user database connection detected")
		}
	}

	if unhealthyCount > 0 {
		logger.WithField("unhealthy_connections", unhealthyCount).Warn("Some user database connections are unhealthy")
	} else {
		logger.Debug("All user database connections healthy")
	}
}

// setupMetrics sets up Prometheus metrics
func setupMetrics() {
	// Custom metrics setup would go here
	// For example: connection pool metrics, query duration metrics, etc.
}

// validateEnvironment validates the environment configuration
func validateEnvironment(cfg *config.Config, logger *logrus.Logger) error {
	if cfg.IsProduction() {
		// Production-specific validations
		if cfg.Auth.JWTSecret == "change-me-in-production" {
			return fmt.Errorf("JWT secret must be changed in production")
		}

		if cfg.Turso.URL == "" {
			return fmt.Errorf("Turso URL must be configured in production")
		}

		if cfg.Turso.AuthToken == "" {
			return fmt.Errorf("Turso auth token must be configured in production")
		}

		if !cfg.Server.TLSEnabled {
			logger.Warn("TLS is disabled in production environment")
		}

		if cfg.Log.Level == "debug" || cfg.Log.Level == "trace" {
			logger.Warn("Debug/trace logging is enabled in production")
		}

		if cfg.Email.APIKey == "" {
			logger.Warn("Email API key not configured - using mock email service")
		}
	}

	return nil
}

// setupVersion sets version information in the version package
func setupVersion() {
	version.Version = Version
	version.Commit = Commit
	version.BuildDate = BuildDate

	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		info := version.GetInfo()
		fmt.Println(info.String())
		os.Exit(0)
	}
}

// initializeConfig loads configuration and creates logger
func initializeConfig() (*config.Config, *logrus.Logger) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create logger
	logConfig := logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		File:       cfg.Log.File,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	}

	appLogger, err := logger.NewLogger(logConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	return cfg, appLogger
}

// logStartupInfo logs startup information
func logStartupInfo(cfg *config.Config, logger *logrus.Logger) {
	logger.WithFields(logrus.Fields{
		"version":     Version,
		"commit":      Commit,
		"build_date":  BuildDate,
		"environment": cfg.GetEnv(),
		"grpc_port":   cfg.Server.GRPCPort,
		"http_port":   cfg.Server.HTTPPort,
	}).Info("Starting SQL Studio Backend (Phase 2)")
}

// initializeDatabase connects to database and runs migrations
func initializeDatabase(cfg *config.Config, logger *logrus.Logger) *sql.DB {
	logger.Info("Connecting to Turso database...")
	tursoClient, err := turso.NewClient(&turso.Config{
		URL:       cfg.Turso.URL,
		AuthToken: cfg.Turso.AuthToken,
		MaxConns:  cfg.Turso.MaxConnections,
	}, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to Turso database")
	}

	logger.Info("Initializing database schema...")
	if schemaErr := turso.InitializeSchema(tursoClient, logger); schemaErr != nil {
		logger.WithError(schemaErr).Fatal("Failed to initialize schema")
	}

	logger.Info("Running database migrations...")
	if migErr := turso.RunMigrations(tursoClient, logger); migErr != nil {
		logger.WithError(migErr).Fatal("Failed to run migrations")
	}

	return tursoClient
}

// initializeServices creates and wires up all application services
func initializeServices(cfg *config.Config, tursoClient *sql.DB, logger *logrus.Logger) (*services.Services, *middleware.AuthMiddleware, *database.Manager) {
	// Create storage implementations
	userStore := turso.NewTursoUserStore(tursoClient, logger)
	sessionStore := turso.NewTursoSessionStore(tursoClient, logger)
	loginAttemptStore := turso.NewTursoLoginAttemptStore(tursoClient, logger)
	syncStore := turso.NewSyncStoreAdapter(tursoClient, logger)
	organizationStore := turso.NewOrganizationStore(tursoClient, logger)
	logger.Info("Storage layer initialized with Turso")

	// Create email service
	emailService := createEmailService(cfg, logger)

	// Create auth middleware and service
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.JWTSecret, logger)
	authService := createAuthService(cfg, userStore, sessionStore, loginAttemptStore, authMiddleware, emailService, logger)

	// Create sync and organization services
	syncService := createSyncService(cfg, syncStore, logger)
	organizationService := organization.NewService(organizationStore, logger)
	logger.Info("Organization service initialized")

	// Create database manager and wire up services
	dbManager := database.NewManager(logger)
	serviceConfig := services.Config{
		Database: cfg.Database,
		Auth:     cfg.Auth,
		Security: cfg.Security,
	}

	svc, err := services.NewServices(serviceConfig, dbManager, authMiddleware, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create services")
	}

	svc.Auth = authService
	svc.Sync = syncService
	svc.Database = dbManager
	svc.Organization = organizationService
	logger.Info("All services wired up successfully")

	return svc, authMiddleware, dbManager
}

// createEmailService creates and configures email service
func createEmailService(cfg *config.Config, logger *logrus.Logger) email.EmailService {
	if cfg.Email.APIKey != "" {
		emailService, err := email.NewResendEmailService(
			cfg.Email.APIKey,
			cfg.Email.FromEmail,
			cfg.Email.FromName,
			logger,
		)
		if err != nil {
			logger.WithError(err).Fatal("Failed to create email service")
		}
		logger.Info("Email service initialized (Resend)")
		return emailService
	}

	logger.Warn("Email service initialized (Mock - emails logged only)")
	return email.NewMockEmailService(logger)
}

// createAuthService creates and configures authentication service
func createAuthService(
	cfg *config.Config,
	userStore auth.UserStore,
	sessionStore auth.SessionStore,
	loginAttemptStore auth.LoginAttemptStore,
	authMiddleware *middleware.AuthMiddleware,
	emailService email.EmailService,
	logger *logrus.Logger,
) *auth.Service {
	authConfig := auth.Config{
		BcryptCost:        cfg.Auth.BcryptCost,
		JWTExpiration:     cfg.Auth.JWTExpiration,
		RefreshExpiration: cfg.Auth.RefreshExpiration,
		MaxLoginAttempts:  cfg.Auth.MaxLoginAttempts,
		LockoutDuration:   cfg.Auth.LockoutDuration,
	}

	authService := auth.NewService(
		userStore,
		sessionStore,
		loginAttemptStore,
		authMiddleware,
		authConfig,
		logger,
	)

	authService.SetEmailService(emailService)
	logger.Info("Auth service initialized")

	return authService
}

// createSyncService creates and configures sync service
func createSyncService(cfg *config.Config, syncStore appsync.Store, logger *logrus.Logger) *appsync.Service {
	syncConfig := appsync.Config{
		MaxUploadSize:      cfg.Sync.MaxUploadSize,
		ConflictStrategy:   appsync.ConflictResolutionStrategy(cfg.Sync.ConflictStrategy),
		RetentionDays:      cfg.Sync.RetentionDays,
		MaxHistoryItems:    cfg.Sync.MaxHistoryItems,
		EnableSanitization: cfg.Sync.EnableSanitization,
	}

	syncService := appsync.NewService(syncStore, syncConfig, logger)
	logger.Info("Sync service initialized")

	return syncService
}

// checkForUpdates checks for available updates in the background
func checkForUpdates(logger *logrus.Logger) {
	// Get config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.WithError(err).Debug("Failed to get home directory for update check")
		return
	}

	configDir := fmt.Sprintf("%s/.sqlstudio", homeDir)

	// Create updater
	u := updater.NewUpdater(configDir)

	// Check if we should check for updates (only once per 24 hours)
	if !u.ShouldCheckForUpdate() {
		logger.Debug("Skipping update check (checked recently)")
		return
	}

	// Check for updates with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	updateInfo, err := u.CheckForUpdate(ctx)
	if err != nil {
		logger.WithError(err).Debug("Failed to check for updates")
		return
	}

	// Record that we checked
	if err := u.RecordUpdateCheck(); err != nil {
		logger.WithError(err).Debug("Failed to record update check")
	}

	// Show notification if update is available
	if updateInfo.Available {
		logger.WithFields(logrus.Fields{
			"current_version": updateInfo.CurrentVersion,
			"latest_version":  updateInfo.LatestVersion,
		}).Info("New version available! Run 'sqlstudio update' to upgrade")
	} else {
		logger.Debug("No updates available")
	}
}
