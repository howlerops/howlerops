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

	servicesauth "github.com/sql-studio/sql-studio/services/auth"
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
	performStartupChecks(cfg, appLogger)

	// Initialize database and services
	tursoClient := initializeDatabase(cfg, appLogger)
	svc, authMiddleware, dbManager := initializeServices(cfg, tursoClient, appLogger)

	// Create all servers
	servers := createServers(cfg, appLogger, svc, authMiddleware)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start all servers
	wg := startAllServers(ctx, servers, svc, appLogger)

	// Log server URLs
	logServerURLs(cfg, appLogger)

	// Wait for shutdown signal
	waitForShutdownSignal(appLogger)

	// Perform graceful shutdown
	performGracefulShutdown(ctx, cancel, cfg, servers, tursoClient, dbManager, appLogger)

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
	if !cfg.IsProduction() {
		return nil
	}

	// Production-specific validations
	if err := validateProductionSecurity(cfg); err != nil {
		return err
	}

	logProductionWarnings(cfg, logger)

	return nil
}

// validateProductionSecurity validates security requirements for production
func validateProductionSecurity(cfg *config.Config) error {
	if cfg.Auth.JWTSecret == "change-me-in-production" {
		return fmt.Errorf("JWT secret must be changed in production")
	}

	if cfg.Turso.URL == "" {
		return fmt.Errorf("turso URL must be configured in production")
	}

	if cfg.Turso.AuthToken == "" {
		return fmt.Errorf("turso auth token must be configured in production")
	}

	return nil
}

// logProductionWarnings logs warnings for production configuration issues
func logProductionWarnings(cfg *config.Config, logger *logrus.Logger) {
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
	}).Info("Starting Howlerops Backend (Phase 2)")
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
	masterKeyStore := turso.NewMasterKeyStore(tursoClient, logger)
	logger.Info("Storage layer initialized with Turso")

	// Create email service
	emailService := createEmailService(cfg, logger)

	// Create auth middleware and service
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.JWTSecret, logger)
	authService := createAuthService(cfg, userStore, sessionStore, loginAttemptStore, masterKeyStore, authMiddleware, emailService, logger)

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

	// Initialize OAuth managers
	githubOAuth, googleOAuth := initializeOAuthManagers(cfg, logger)
	svc.GitHubOAuth = githubOAuth
	svc.GoogleOAuth = googleOAuth

	// Initialize WebAuthn manager
	webauthnManager := initializeWebAuthnManager(logger)
	svc.WebAuthnManager = webauthnManager

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
	masterKeyStore auth.MasterKeyStore,
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
		masterKeyStore,
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

// serverCollection holds all server instances
type serverCollection struct {
	grpc      *server.GRPCServer
	http      *server.HTTPServer
	websocket *server.WebSocketServer
	metrics   *http.Server
}

// performStartupChecks performs initial environment checks
func performStartupChecks(cfg *config.Config, logger *logrus.Logger) {
	if !cfg.IsProduction() {
		go checkForUpdates(logger)
	}
	if validateErr := validateEnvironment(cfg, logger); validateErr != nil {
		logger.WithError(validateErr).Fatal("Environment validation failed")
	}
}

// createServers creates all server instances
func createServers(cfg *config.Config, logger *logrus.Logger, svc *services.Services, authMiddleware *middleware.AuthMiddleware) *serverCollection {
	// Create gRPC server
	grpcServer, err := server.NewGRPCServer(cfg, logger, svc)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create gRPC server")
	}

	// Create HTTP gateway server
	httpServer, err := server.NewHTTPServer(cfg, logger, svc, authMiddleware)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create HTTP server")
	}

	// Create metrics server
	var metricsServer *http.Server
	if cfg.Metrics.Enabled {
		// #nosec G112 - metrics server is internal/trusted, ReadHeaderTimeout not critical for prometheus scraping
		metricsServer = &http.Server{
			Addr:    cfg.GetMetricsAddress(),
			Handler: promhttp.Handler(),
		}
		setupMetrics()
	}

	// Create WebSocket server for real-time updates
	wsServer, err := server.NewWebSocketServer(cfg, logger, svc)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create WebSocket server")
	}

	return &serverCollection{
		grpc:      grpcServer,
		http:      httpServer,
		websocket: wsServer,
		metrics:   metricsServer,
	}
}

// startAllServers starts all servers in goroutines
func startAllServers(ctx context.Context, servers *serverCollection, svc *services.Services, logger *logrus.Logger) *sync.WaitGroup {
	var wg sync.WaitGroup

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting gRPC server")
		if err := servers.grpc.Start(); err != nil {
			logger.WithError(err).Error("gRPC server failed")
		}
	}()

	// Start HTTP gateway server
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting HTTP gateway server")
		if startErr := servers.http.Start(); startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
			logger.WithError(startErr).Error("HTTP server failed")
		}
	}()

	// Start metrics server
	if servers.metrics != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.WithField("address", servers.metrics.Addr).Info("Starting metrics server")
			if err := servers.metrics.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Metrics server failed")
			}
		}()
	}

	// Start WebSocket server
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting WebSocket server")
		if err := servers.websocket.Start(); err != nil {
			logger.WithError(err).Error("WebSocket server failed")
		}
	}()

	// Start background tasks
	wg.Add(1)
	go func() {
		defer wg.Done()
		runBackgroundTasks(ctx, svc, logger)
	}()

	return &wg
}

// logServerURLs logs all server URLs
func logServerURLs(cfg *config.Config, logger *logrus.Logger) {
	logger.Info("All servers started successfully")
	logger.WithFields(logrus.Fields{
		"http_url":    fmt.Sprintf("http://localhost:%d", cfg.Server.HTTPPort),
		"grpc_url":    fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort),
		"metrics_url": fmt.Sprintf("http://localhost:%d/metrics", cfg.Metrics.Port),
		"health_url":  fmt.Sprintf("http://localhost:%d/health", cfg.Server.HTTPPort),
		"ws_url":      "ws://localhost:8081/ws",
	}).Info("Server URLs")
}

// waitForShutdownSignal waits for OS shutdown signal
func waitForShutdownSignal(logger *logrus.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	logger.Info("Shutdown signal received, starting graceful shutdown...")
}

// performGracefulShutdown performs graceful shutdown of all services
func performGracefulShutdown(_ context.Context, cancel context.CancelFunc, cfg *config.Config, servers *serverCollection, tursoClient *sql.DB, dbManager *database.Manager, logger *logrus.Logger) {
	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Stop all servers gracefully
	stopAllServers(shutdownCtx, servers, logger)

	// Stop background tasks
	cancel()

	// Close database connections
	closeConnections(tursoClient, dbManager, logger)
}

// stopAllServers stops all servers gracefully
func stopAllServers(ctx context.Context, servers *serverCollection, logger *logrus.Logger) {
	var shutdownWg sync.WaitGroup

	// Stop gRPC server
	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		if err := servers.grpc.Stop(ctx); err != nil {
			logger.WithError(err).Error("Failed to stop gRPC server gracefully")
		}
	}()

	// Stop HTTP server
	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		if err := servers.http.Stop(ctx); err != nil {
			logger.WithError(err).Error("Failed to stop HTTP server gracefully")
		}
	}()

	// Stop metrics server
	if servers.metrics != nil {
		shutdownWg.Add(1)
		go func() {
			defer shutdownWg.Done()
			if err := servers.metrics.Shutdown(ctx); err != nil {
				logger.WithError(err).Error("Failed to stop metrics server gracefully")
			}
		}()
	}

	// Stop WebSocket server
	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		if err := servers.websocket.Stop(ctx); err != nil {
			logger.WithError(err).Error("Failed to stop WebSocket server gracefully")
		}
	}()

	// Wait for all servers to stop
	shutdownWg.Wait()
}

// closeConnections closes all database connections
func closeConnections(tursoClient *sql.DB, dbManager *database.Manager, logger *logrus.Logger) {
	if err := tursoClient.Close(); err != nil {
		logger.WithError(err).Error("Failed to close Turso database connection")
	}

	if err := dbManager.Close(); err != nil {
		logger.WithError(err).Error("Failed to close database manager connections")
	}
}

// initializeOAuthManagers initializes GitHub and Google OAuth managers
func initializeOAuthManagers(cfg *config.Config, logger *logrus.Logger) (*servicesauth.OAuth2Manager, *servicesauth.OAuth2Manager) {
	var githubOAuth, googleOAuth *servicesauth.OAuth2Manager

	// Initialize GitHub OAuth if configured
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if githubClientID != "" && githubClientSecret != "" {
		manager, err := servicesauth.NewOAuth2Manager("github", githubClientID, githubClientSecret)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize GitHub OAuth")
		} else {
			githubOAuth = manager
			logger.Info("GitHub OAuth initialized successfully")
		}
	} else {
		logger.Debug("GitHub OAuth not configured (GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET missing)")
	}

	// Initialize Google OAuth if configured
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if googleClientID != "" && googleClientSecret != "" {
		manager, err := servicesauth.NewOAuth2Manager("google", googleClientID, googleClientSecret)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize Google OAuth")
		} else {
			googleOAuth = manager
			logger.Info("Google OAuth initialized successfully")
		}
	} else {
		logger.Debug("Google OAuth not configured (GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET missing)")
	}

	return githubOAuth, googleOAuth
}

// initializeWebAuthnManager initializes WebAuthn manager
func initializeWebAuthnManager(logger *logrus.Logger) *servicesauth.WebAuthnManager {
	// Initialize WebAuthn components
	credentialStore := servicesauth.NewCredentialStore()
	sessionStore := servicesauth.NewSessionStore()

	manager, err := servicesauth.NewWebAuthnManager(credentialStore, sessionStore)
	if err != nil {
		logger.WithError(err).Warn("Failed to initialize WebAuthn manager")
		return nil
	}

	logger.Info("WebAuthn manager initialized successfully")
	return manager
}
