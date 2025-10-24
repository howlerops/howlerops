package main

import (
	"context"
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
	// Set version info in package
	version.Version = Version
	version.Commit = Commit
	version.BuildDate = BuildDate

	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		info := version.GetInfo()
		fmt.Println(info.String())
		os.Exit(0)
	}

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

	appLogger.WithFields(logrus.Fields{
		"version":     Version,
		"commit":      Commit,
		"build_date":  BuildDate,
		"environment": cfg.GetEnv(),
		"grpc_port":   cfg.Server.GRPCPort,
		"http_port":   cfg.Server.HTTPPort,
	}).Info("Starting SQL Studio Backend (Phase 2)")

	// Check for updates (non-blocking, non-intrusive)
	if !cfg.IsProduction() {
		go checkForUpdates(appLogger)
	}

	// Validate environment in production
	if err := validateEnvironment(cfg, appLogger); err != nil {
		appLogger.WithError(err).Fatal("Environment validation failed")
	}

	// Initialize Turso database connection
	appLogger.Info("Connecting to Turso database...")
	tursoClient, err := turso.NewClient(&turso.Config{
		URL:       cfg.Turso.URL,
		AuthToken: cfg.Turso.AuthToken,
		MaxConns:  cfg.Turso.MaxConnections,
	}, appLogger)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to Turso database")
	}

	// Initialize schema (creates tables if they don't exist)
	appLogger.Info("Initializing database schema...")
	if err := turso.InitializeSchema(tursoClient, appLogger); err != nil {
		appLogger.WithError(err).Fatal("Failed to initialize schema")
	}

	// Run database migrations
	appLogger.Info("Running database migrations...")
	if err := turso.RunMigrations(tursoClient, appLogger); err != nil {
		appLogger.WithError(err).Fatal("Failed to run migrations")
	}

	// Create Turso storage implementations
	userStore := turso.NewTursoUserStore(tursoClient, appLogger)
	sessionStore := turso.NewTursoSessionStore(tursoClient, appLogger)
	loginAttemptStore := turso.NewTursoLoginAttemptStore(tursoClient, appLogger)
	syncStore := turso.NewSyncStoreAdapter(tursoClient, appLogger)
	organizationStore := turso.NewOrganizationStore(tursoClient, appLogger)

	appLogger.Info("Storage layer initialized with Turso")

	// Create email service
	var emailService email.EmailService
	if cfg.Email.APIKey != "" {
		emailService, err = email.NewResendEmailService(
			cfg.Email.APIKey,
			cfg.Email.FromEmail,
			cfg.Email.FromName,
			appLogger,
		)
		if err != nil {
			appLogger.WithError(err).Fatal("Failed to create email service")
		}
		appLogger.Info("Email service initialized (Resend)")
	} else {
		emailService = email.NewMockEmailService(appLogger)
		appLogger.Warn("Email service initialized (Mock - emails logged only)")
	}

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.JWTSecret, appLogger)

	// Create auth service
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
		appLogger,
	)

	// Wire up email service to auth
	authService.SetEmailService(emailService)

	appLogger.Info("Auth service initialized")

	// Create sync service
	syncConfig := appsync.Config{
		MaxUploadSize:      cfg.Sync.MaxUploadSize,
		ConflictStrategy:   appsync.ConflictResolutionStrategy(cfg.Sync.ConflictStrategy),
		RetentionDays:      cfg.Sync.RetentionDays,
		MaxHistoryItems:    cfg.Sync.MaxHistoryItems,
		EnableSanitization: cfg.Sync.EnableSanitization,
	}

	syncService := appsync.NewService(
		syncStore,
		syncConfig,
		appLogger,
	)

	appLogger.Info("Sync service initialized")

	// Create organization service
	organizationService := organization.NewService(
		organizationStore,
		appLogger,
	)

	appLogger.Info("Organization service initialized")

	// Create database manager (for query execution on user-connected databases)
	dbManager := database.NewManager(appLogger)

	// Create legacy services (existing database operations)
	serviceConfig := services.Config{
		Database: cfg.Database,
		Auth:     cfg.Auth,
		Security: cfg.Security,
	}

	svc, err := services.NewServices(serviceConfig, dbManager, authMiddleware, appLogger)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to create services")
	}

	// Wire up new services to replace in-memory implementations
	svc.Auth = authService
	svc.Sync = syncService
	svc.Database = dbManager
	svc.Organization = organizationService

	appLogger.Info("All services wired up successfully")

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
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Error("HTTP server failed")
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
			if svc.Database != nil {
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

			logger.Debug("Background cleanup tasks completed successfully")
		}
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
