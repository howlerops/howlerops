package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jbeck018/howlerops/pkg/ai"
	"github.com/jbeck018/howlerops/pkg/auth"
	"github.com/jbeck018/howlerops/pkg/federation/duckdb"
	"github.com/jbeck018/howlerops/pkg/rag"
	"github.com/jbeck018/howlerops/pkg/storage"
	"github.com/jbeck018/howlerops/pkg/storage/turso"
	"github.com/jbeck018/howlerops/services"
)

// AppLifecycle orchestrates application startup, shutdown, and service wiring.
// It is NOT a Wails service itself -- it creates and owns all the Wails service
// instances and the SharedDeps that bind them together.
type AppLifecycle struct {
	// Context supplied during OnStartup.
	ctx context.Context

	// Shared dependencies pointer. StorageManager and DuckDBEngine start nil
	// and are populated during OnStartup after storage initialisation.
	deps *SharedDeps

	// ---- Underlying service layer (non-Wails) ----
	logger            *logrus.Logger
	databaseService   *services.DatabaseService
	fileService       *services.FileService
	keyboardService   *services.KeyboardService
	credentialService *services.CredentialService
	passwordManager   *services.PasswordManager
	aiService         *ai.Service
	aiConfig          *ai.RuntimeConfig
	embeddingService  rag.EmbeddingService
	reportService     *services.ReportService
	storageManager    *storage.Manager
	storageMigration  *services.StorageMigrationService
	syntheticViews    *storage.SyntheticViewStorage
	duckdbEngine      *duckdb.Engine

	// OAuth / Auth
	githubOAuth     *auth.OAuth2Manager
	googleOAuth     *auth.OAuth2Manager
	secureStorage   *auth.SecureStorage
	webauthnManager *auth.WebAuthnManager
	credentialStore *auth.CredentialStore
	sessionStore    *auth.SessionStore

	// ---- Wails service wrappers ----
	connectionSvc *ConnectionService
	querySvc      *QueryService
	aiSvc         *WailsAIService
	authSvc       *WailsAuthService
	fileSvc       *WailsFileService
	keyboardSvc   *WailsKeyboardService
	reportSvc     *WailsReportService
	catalogSvc    *CatalogService
	schemaDiffSvc *SchemaDiffService
	storageSvc    *StorageService
	updateSvc     *UpdateService
}

// NewAppLifecycle creates the lifecycle orchestrator together with all
// underlying services and Wails service wrappers. Services that depend on
// storage will receive a populated SharedDeps pointer once OnStartup runs.
func NewAppLifecycle() *AppLifecycle {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Core services
	databaseService := services.NewDatabaseService(logger)
	fileService := services.NewFileService(logger)
	keyboardService := services.NewKeyboardService(logger)
	credentialService := services.NewCredentialService(logger)
	reportService := services.NewReportService(logger, databaseService)

	// OAuth managers (credentials from environment variables)
	var githubOAuth, googleOAuth *auth.OAuth2Manager

	githubClientID := os.Getenv("GH_CLIENT_ID")
	githubClientSecret := os.Getenv("GH_CLIENT_SECRET")
	if githubClientID != "" && githubClientSecret != "" {
		githubOAuth, _ = auth.NewOAuth2Manager("github", githubClientID, githubClientSecret)
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if googleClientID != "" && googleClientSecret != "" {
		googleOAuth, _ = auth.NewOAuth2Manager("google", googleClientID, googleClientSecret)
	}

	// WebAuthn components
	credStore := auth.NewCredentialStore()
	sessStore := auth.NewSessionStore()
	webauthnManager, err := auth.NewWebAuthnManager(credStore, sessStore)
	if err != nil {
		logger.Warnf("Failed to initialize WebAuthn: %v", err)
	}

	aiConfig := ai.DefaultRuntimeConfig()

	// SharedDeps -- StorageManager and DuckDBEngine start nil; set in OnStartup.
	deps := &SharedDeps{
		Logger:          logger,
		DatabaseService: databaseService,
	}

	// Build lifecycle struct (services that depend on storage can tolerate nil
	// StorageManager until OnStartup populates it via the SharedDeps pointer).
	lc := &AppLifecycle{
		deps:              deps,
		logger:            logger,
		databaseService:   databaseService,
		fileService:       fileService,
		keyboardService:   keyboardService,
		credentialService: credentialService,
		aiConfig:          aiConfig,
		reportService:     reportService,
		githubOAuth:       githubOAuth,
		googleOAuth:       googleOAuth,
		secureStorage:     auth.NewSecureStorage(),
		webauthnManager:   webauthnManager,
		credentialStore:   credStore,
		sessionStore:      sessStore,
	}

	// ---- Create Wails service wrappers ----
	// ConnectionService: storageManager field is set from deps.StorageManager
	// (nil now, will be updated in OnStartup).
	lc.connectionSvc = NewConnectionService(deps, credentialService, nil)
	lc.querySvc = NewQueryService(deps)
	lc.aiSvc = NewWailsAIService(deps)
	lc.authSvc = NewWailsAuthService(deps, githubOAuth, googleOAuth,
		lc.secureStorage, webauthnManager, credStore, sessStore)
	// WailsFileService: passwordManager is nil until storage init.
	lc.fileSvc = NewWailsFileService(deps, fileService, credentialService, nil)
	lc.keyboardSvc = NewWailsKeyboardService(deps, keyboardService)
	lc.reportSvc = NewWailsReportService(deps, reportService)
	lc.catalogSvc = NewCatalogService(deps)
	lc.schemaDiffSvc = NewSchemaDiffService(deps)
	// StorageService: storageMigration and syntheticViews are nil until
	// OnStartup populates them.
	lc.storageSvc = NewStorageService(deps, nil, nil, context.Background())
	lc.updateSvc = NewUpdateService(deps)

	return lc
}

// SetApplication propagates the Wails application reference into SharedDeps
// and any services that need it.
func (lc *AppLifecycle) SetApplication(app *application.App) {
	lc.deps.App = app
	if lc.fileService != nil {
		lc.fileService.SetApplication(app)
	}
}

// SetMainWindow stores the main window reference in SharedDeps for dialog use.
func (lc *AppLifecycle) SetMainWindow(window application.Window) {
	lc.deps.MainWindow = window
}

// GetServices returns all Wails service wrappers for registration with the
// application. Each entry should be wrapped with application.NewService(...)
// by the caller.
func (lc *AppLifecycle) GetServices() []application.Service {
	return []application.Service{
		application.NewService(lc.connectionSvc),
		application.NewService(lc.querySvc),
		application.NewService(lc.aiSvc),
		application.NewService(lc.authSvc),
		application.NewService(lc.fileSvc),
		application.NewService(lc.keyboardSvc),
		application.NewService(lc.reportSvc),
		application.NewService(lc.catalogSvc),
		application.NewService(lc.schemaDiffSvc),
		application.NewService(lc.storageSvc),
		application.NewService(lc.updateSvc),
	}
}

// ---------------------------------------------------------------------------
// Lifecycle hooks
// ---------------------------------------------------------------------------

// OnStartup is called after the Wails application has started. It initialises
// storage, populates SharedDeps, and propagates dependencies to services that
// need them post-storage-init.
func (lc *AppLifecycle) OnStartup() {
	ctx := context.Background()
	lc.ctx = ctx

	// Set context on services that require it.
	lc.databaseService.SetContext(ctx)
	lc.fileService.SetContext(ctx)
	lc.keyboardService.SetContext(ctx)
	lc.credentialService.SetContext(ctx)
	if lc.reportService != nil {
		lc.reportService.SetContext(ctx)
	}

	// Initialise storage manager (populates deps.StorageManager, DuckDBEngine, etc.)
	if err := lc.initializeStorageManager(ctx); err != nil {
		lc.logger.WithError(err).Error("Failed to initialize storage manager")
		// Continue without storage -- graceful degradation.
	}

	// Propagate storage-dependent resources to Wails services that cached
	// copies of fields from deps during construction.
	lc.propagatePostStartupDeps()

	lc.logger.Info("HowlerOps desktop application started")

	// Emit app ready event.
	lc.deps.emitEvent("app:startup-complete", map[string]interface{}{"status": "ready"})
}

// OnShutdown is called when the application is shutting down.
func (lc *AppLifecycle) OnShutdown() {
	lc.logger.Info("HowlerOps desktop application shutting down")

	if lc.storageManager != nil {
		if err := lc.storageManager.Close(); err != nil {
			lc.logger.WithError(err).Error("Failed to close storage manager")
		}
	}

	if lc.reportService != nil {
		lc.reportService.Shutdown()
	}

	if lc.aiService != nil {
		ctx := context.Background()
		if err := lc.aiService.Stop(ctx); err != nil {
			lc.logger.WithError(err).Error("Failed to stop AI service")
		}
	}

	if lc.databaseService != nil {
		lc.databaseService.Close()
	}

	lc.deps.emitEvent("app:shutdown", map[string]interface{}{"status": "shutdown"})
}

// OnUrlOpen handles deep-link / URL scheme callbacks (e.g. OAuth on macOS).
func (lc *AppLifecycle) OnUrlOpen(url string) {
	lc.logger.WithField("url", url).Info("Received URL open event")

	if !strings.Contains(url, "howlerops://auth/callback") {
		lc.logger.WithField("url", url).Debug("Not an OAuth callback URL, ignoring")
		return
	}

	// Parse URL query parameters.
	queryStart := strings.Index(url, "?")
	if queryStart == -1 {
		lc.logger.Error("OAuth callback URL missing query parameters")
		lc.deps.emitEvent("auth:error", "Invalid callback URL: missing parameters")
		return
	}

	query := url[queryStart+1:]
	params := make(map[string]string)
	for _, pair := range strings.Split(query, "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parts[1]
		}
	}

	code, hasCode := params["code"]
	state, hasState := params["state"]

	if !hasCode {
		lc.logger.Error("OAuth callback missing authorization code")
		lc.deps.emitEvent("auth:error", "Missing authorization code")
		return
	}
	if !hasState {
		lc.logger.Error("OAuth callback missing state parameter")
		lc.deps.emitEvent("auth:error", "Missing state parameter")
		return
	}

	lc.handleOAuthCallback(code, state)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// handleOAuthCallback exchanges an OAuth code for a token across providers.
func (lc *AppLifecycle) handleOAuthCallback(code, state string) {
	lc.logger.Info("Processing OAuth callback")

	var user *auth.OAuthUser
	var provider string
	var err error

	// Try GitHub first.
	if lc.githubOAuth != nil {
		user, err = lc.githubOAuth.ExchangeCodeForToken(code, state)
		if err == nil {
			provider = "github"
		} else if !strings.Contains(err.Error(), "invalid state") {
			lc.logger.WithError(err).Error("GitHub OAuth exchange failed")
		}
	}

	// If GitHub didn't work, try Google.
	if user == nil && lc.googleOAuth != nil {
		user, err = lc.googleOAuth.ExchangeCodeForToken(code, state)
		if err == nil {
			provider = "google"
		} else if !strings.Contains(err.Error(), "invalid state") {
			lc.logger.WithError(err).Error("Google OAuth exchange failed")
		}
	}

	if user == nil {
		lc.logger.WithError(err).Error("OAuth code exchange failed for all providers")
		errMsg := "Authentication failed"
		if err != nil {
			errMsg += ": " + err.Error()
		}
		lc.deps.emitEvent("auth:error", errMsg)
		return
	}

	// Store token securely.
	storedToken := &auth.StoredToken{
		AccessToken: user.AccessToken,
		Provider:    user.Provider,
		UserID:      user.ID,
		Email:       user.Email,
		ExpiresAt:   user.ExpiresAt,
	}

	if err := lc.secureStorage.StoreToken(provider, storedToken); err != nil {
		lc.logger.WithError(err).Error("Failed to store OAuth token")
		lc.deps.emitEvent("auth:error", "Failed to store authentication token")
		return
	}

	lc.logger.WithFields(logrus.Fields{
		"provider": provider,
		"userId":   user.ID,
		"email":    user.Email,
	}).Info("OAuth authentication successful")

	userData := map[string]interface{}{
		"provider": user.Provider,
		"id":       user.ID,
		"login":    user.Login,
		"email":    user.Email,
		"name":     user.Name,
	}
	if user.AvatarURL != "" {
		userData["avatarUrl"] = user.AvatarURL
	}

	lc.deps.emitEvent("auth:success", userData)
}

// initializeStorageManager sets up the storage layer (SQLite/Turso), password
// manager, synthetic views, reports storage, and the DuckDB federation engine.
func (lc *AppLifecycle) initializeStorageManager(ctx context.Context) error {
	userID := getEnvOrDefault("HOWLEROPS_USER_ID", "local-user")

	vectorStoreType := strings.ToLower(getEnvOrDefault("VECTOR_STORE_TYPE", "sqlite"))
	mysqlVectorDSN := os.Getenv("MYSQL_VECTOR_DSN")
	mysqlVectorSize := getEnvOrDefault("MYSQL_VECTOR_SIZE", "1536")
	vectorSize := 1536
	if parsed, err := strconv.Atoi(mysqlVectorSize); err == nil {
		vectorSize = parsed
	}

	storageConfig := &storage.Config{
		Mode: storage.ModeSolo,
		Local: storage.LocalStorageConfig{
			DataDir:         getEnvOrDefault("HOWLEROPS_DATA_DIR", "~/.howlerops"),
			Database:        "local.db",
			VectorsDB:       "vectors.db",
			UserID:          userID,
			VectorSize:      vectorSize,
			VectorStoreType: vectorStoreType,
		},
		UserID: userID,
	}

	if vectorStoreType == "mysql" && mysqlVectorDSN != "" {
		storageConfig.Local.MySQLVector = &storage.MySQLVectorConfig{
			DSN:        mysqlVectorDSN,
			VectorSize: vectorSize,
		}
	}

	// Team mode with Turso.
	if os.Getenv("HOWLEROPS_MODE") == "team" && os.Getenv("TURSO_URL") != "" {
		storageConfig.Mode = storage.ModeTeam
		storageConfig.Team = &storage.TursoConfig{
			Enabled:        true,
			URL:            os.Getenv("TURSO_URL"),
			AuthToken:      os.Getenv("TURSO_AUTH_TOKEN"),
			LocalReplica:   getEnvOrDefault("TURSO_LOCAL_REPLICA", "~/.howlerops/team-replica.db"),
			SyncInterval:   getEnvOrDefault("TURSO_SYNC_INTERVAL", "30s"),
			ShareHistory:   true,
			ShareQueries:   true,
			ShareLearnings: true,
			TeamID:         os.Getenv("TEAM_ID"),
		}
	}

	manager, err := storage.NewManager(ctx, storageConfig, lc.logger)
	if err != nil {
		return fmt.Errorf("failed to create storage manager: %w", err)
	}
	lc.storageManager = manager
	lc.deps.StorageManager = manager

	// Storage migration service.
	lc.storageMigration = services.NewStorageMigrationService(manager, lc.logger)

	// Password manager (hybrid dual-read: keychain + encrypted DB).
	db := manager.GetDB()
	if db != nil {
		credentialStore := turso.NewCredentialStore(db, lc.logger)
		connectionStore := turso.NewConnectionStore(db, lc.logger)
		lc.passwordManager = services.NewPasswordManager(
			lc.credentialService,
			credentialStore,
			connectionStore,
			lc.logger,
		)
		lc.logger.Info("Password manager initialized with hybrid storage")
	} else {
		lc.logger.Warn("Database not available, password manager will use keychain only")
	}

	// Synthetic views storage.
	syntheticViewsStorage := storage.NewSyntheticViewStorage(manager.GetDB(), lc.logger)
	if err := syntheticViewsStorage.CreateTable(); err != nil {
		lc.logger.WithError(err).Warn("Failed to create synthetic views table")
	}
	lc.syntheticViews = syntheticViewsStorage

	// Report storage.
	reportStorage := storage.NewReportStorage(manager.GetDB(), lc.logger)
	if err := reportStorage.EnsureSchema(); err != nil {
		lc.logger.WithError(err).Warn("Failed to ensure reports table")
	}
	if lc.reportService != nil {
		lc.reportService.SetStorage(reportStorage)
	}

	// DuckDB federation engine.
	engine := duckdb.NewEngine(lc.logger, lc.databaseService.GetManager())
	if err := engine.Initialize(ctx); err != nil {
		lc.logger.WithError(err).Warn("Failed to initialize DuckDB federation engine")
	}
	lc.duckdbEngine = engine
	lc.deps.DuckDBEngine = engine

	lc.logger.WithFields(logrus.Fields{
		"mode":    string(manager.GetMode()),
		"user_id": manager.GetUserID(),
	}).Info("Storage manager initialized")

	return nil
}

// propagatePostStartupDeps pushes storage-dependent resources into Wails
// service wrappers that captured nil copies during construction.
func (lc *AppLifecycle) propagatePostStartupDeps() {
	// ConnectionService caches storageManager directly.
	if lc.connectionSvc != nil && lc.storageManager != nil {
		lc.connectionSvc.storageManager = lc.storageManager
	}

	// Update ConnectionService embedding service if we built one.
	if lc.connectionSvc != nil && lc.embeddingService != nil {
		lc.connectionSvc.embeddingService = lc.embeddingService
	}

	// WailsFileService needs the password manager.
	if lc.fileSvc != nil && lc.passwordManager != nil {
		lc.fileSvc.passwordManager = lc.passwordManager
	}

	// StorageService needs storageMigration, syntheticViews, and a valid context.
	if lc.storageSvc != nil {
		if lc.storageMigration != nil {
			lc.storageSvc.storageMigration = lc.storageMigration
		}
		if lc.syntheticViews != nil {
			lc.storageSvc.syntheticViews = lc.syntheticViews
		}
		if lc.ctx != nil {
			lc.storageSvc.ctx = lc.ctx
		}
	}
}

// ---------------------------------------------------------------------------
// AI helper functions
// ---------------------------------------------------------------------------

// hasConfiguredProvider returns true if the current AI config has at least one
// provider with credentials.
func (lc *AppLifecycle) hasConfiguredProvider() bool {
	if lc.aiConfig == nil {
		return false
	}
	if lc.aiConfig.OpenAI.APIKey != "" {
		return true
	}
	if lc.aiConfig.Anthropic.APIKey != "" {
		return true
	}
	if lc.aiConfig.Codex.APIKey != "" {
		return true
	}
	if lc.aiConfig.Ollama.Endpoint != "" {
		return true
	}
	if lc.aiConfig.HuggingFace.Endpoint != "" {
		return true
	}
	if lc.aiConfig.ClaudeCode.ClaudePath != "" {
		return true
	}
	return false
}

// applyAIConfiguration rebuilds the AI service using the current runtime
// configuration stored in lc.aiConfig.
func (lc *AppLifecycle) applyAIConfiguration() error {
	if lc.ctx == nil {
		return fmt.Errorf("application context not initialised")
	}

	if lc.aiConfig == nil {
		lc.aiConfig = ai.DefaultRuntimeConfig()
	}

	if !lc.hasConfiguredProvider() {
		lc.logger.Info("No AI providers configured; AI features remain disabled")
		if lc.aiService != nil {
			if err := lc.aiService.Stop(lc.ctx); err != nil {
				lc.logger.WithError(err).Warn("Failed to stop existing AI service")
			}
			lc.aiService = nil
		}
		lc.embeddingService = nil
		if lc.reportService != nil {
			lc.reportService.SetAIService(nil)
		}
		return fmt.Errorf("no AI providers configured")
	}

	// Shut down any existing instance before reconfiguring.
	if lc.aiService != nil {
		if err := lc.aiService.Stop(lc.ctx); err != nil {
			lc.logger.WithError(err).Warn("Failed to stop existing AI service during reconfiguration")
		}
		lc.aiService = nil
	}

	service, err := ai.NewServiceWithConfig(lc.aiConfig, lc.logger)
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	if err := service.Start(lc.ctx); err != nil {
		return fmt.Errorf("failed to start AI service: %w", err)
	}

	lc.aiService = service
	if lc.reportService != nil {
		lc.reportService.SetAIService(service)
	}
	lc.logger.Info("AI service configured successfully")

	lc.rebuildEmbeddingService()
	return nil
}

// rebuildEmbeddingService establishes (or re-establishes) the embedding
// service. It prefers a local ONNX provider with an optional OpenAI fallback.
func (lc *AppLifecycle) rebuildEmbeddingService() {
	lc.embeddingService = nil

	// Primary: local ONNX provider (works offline).
	onnxModelPath := getEnvOrDefault("HOWLEROPS_EMBEDDING_MODEL", "")
	provider := rag.NewONNXEmbeddingProvider(onnxModelPath, lc.logger)

	// Optional fallback: OpenAI embedding API.
	if lc.aiConfig != nil && strings.TrimSpace(lc.aiConfig.OpenAI.APIKey) != "" {
		openAIModel := "text-embedding-3-small"
		openAIProv := rag.NewOpenAIEmbeddingProvider(lc.aiConfig.OpenAI.APIKey, openAIModel, lc.logger)
		provider = rag.NewFallbackEmbeddingProvider(provider, openAIProv)
	}

	lc.embeddingService = rag.NewEmbeddingService(provider, lc.logger)
	lc.logger.WithField("model", provider.GetModel()).Info("Embedding service initialised")

	// Propagate to ConnectionService.
	if lc.connectionSvc != nil {
		lc.connectionSvc.embeddingService = lc.embeddingService
	}
}

// NOTE: getEnvOrDefault is defined in app.go (and will remain as a package-level
// helper after the full decomposition). This file relies on that function.
