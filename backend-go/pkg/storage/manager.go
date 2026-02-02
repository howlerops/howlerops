package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	internalrag "github.com/jbeck018/howlerops/backend-go/internal/rag"
	pkgrag "github.com/jbeck018/howlerops/backend-go/pkg/rag"
	"github.com/sirupsen/logrus"
)

// Config holds storage manager configuration
type Config struct {
	Mode   Mode               `json:"mode"`
	Local  LocalStorageConfig `json:"local"`
	Team   *TursoConfig       `json:"team,omitempty"`
	UserID string             `json:"user_id"`
}

// TursoConfig holds Turso team storage configuration
type TursoConfig struct {
	Enabled        bool   `json:"enabled"`
	URL            string `json:"url"`
	AuthToken      string `json:"auth_token"`
	LocalReplica   string `json:"local_replica"`
	SyncInterval   string `json:"sync_interval"`
	ShareHistory   bool   `json:"share_history"`
	ShareQueries   bool   `json:"share_queries"`
	ShareLearnings bool   `json:"share_learnings"`
	TeamID         string `json:"team_id"`
}

// Manager manages storage operations and mode switching
type Manager struct {
	mu         sync.RWMutex
	mode       Mode
	storage    Storage
	localStore *LocalSQLiteStorage // Always present for local operations
	teamStore  Storage             // nil in solo mode, TursoTeamStorage in team mode
	userID     string
	dataDir    string // Local data directory path
	logger     *logrus.Logger
}

// NewManager creates a new storage manager
func NewManager(ctx context.Context, config *Config, logger *logrus.Logger) (*Manager, error) {
	if config.UserID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Always create local storage
	localStore, err := NewLocalStorage(&config.Local, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create local storage: %w", err)
	}

	// Expand home directory for dataDir (same logic as NewLocalStorage)
	dataDir := os.ExpandEnv(config.Local.DataDir)
	if strings.HasPrefix(dataDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(home, dataDir[2:])
	}

	manager := &Manager{
		mode:       config.Mode,
		localStore: localStore,
		userID:     config.UserID,
		dataDir:    dataDir,
		logger:     logger,
	}

	// Determine active storage based on mode
	switch config.Mode {
	case ModeSolo:
		manager.storage = localStore
		logger.Info("Storage manager initialized in solo mode")

		// If cloud sync is configured (remote URL present), enable individual-tier local-first sync
		if config.Team != nil && config.Team.URL != "" {
			db, err := sql.Open("sqlite3", config.Team.URL)
			if err != nil {
				logger.WithError(err).Warn("Failed to open remote DB for individual sync; continuing local-only")
			} else {
				remote := internalrag.NewTursoRemoteVectorStore(db, logger)
				if err := remote.Initialize(context.Background()); err != nil {
					logger.WithError(err).Warn("Failed to initialize remote vector store; continuing local-only")
				} else {
					adaptive := pkgrag.NewAdaptiveVectorStore("individual", localStore.getVectorStore(), remote, true)
					pkgrag.StartSyncWorker(context.Background(), adaptive, 200_000_000) // 200ms
					localStore.setVectorStore(adaptive)
					logger.Info("Solo mode with cloud user: enabled individual-tier local-first sync")
				}
			}
		}

	case ModeTeam:
		if config.Team == nil || !config.Team.Enabled {
			logger.Warn("Team mode requested but team config not enabled, falling back to solo mode")
			manager.mode = ModeSolo
			manager.storage = localStore
		} else {
			// Local-first with remote sync enabled by default for logged-in cloud users (team mode)
			// Wrap local vector store with adaptive sync if remote URL is provided
			if config.Team.URL != "" {
				db, err := sql.Open("sqlite3", config.Team.URL)
				if err != nil {
					logger.WithError(err).Warn("Failed to open Turso remote DB; continuing local-only")
					manager.mode = ModeSolo
					manager.storage = localStore
				} else {
					remote := internalrag.NewTursoRemoteVectorStore(db, logger)
					if err := remote.Initialize(context.Background()); err != nil {
						logger.WithError(err).Warn("Failed to initialize Turso remote vector store; continuing local-only")
						manager.mode = ModeSolo
						manager.storage = localStore
					} else {
						adaptive := pkgrag.NewAdaptiveVectorStore("team", localStore.getVectorStore(), remote, true)
						// conservative heartbeat; per-doc backoff governs actual sync
						pkgrag.StartSyncWorker(context.Background(), adaptive, 200_000_000) // 200ms
						localStore.setVectorStore(adaptive)
						manager.mode = ModeTeam
						manager.storage = localStore
						logger.Info("Team mode enabled: local-first with remote sync")
					}
				}
			} else {
				// No remote provided; fall back to local-only semantics
				logger.Warn("Team mode requested without remote URL; using local storage")
				manager.mode = ModeSolo
				manager.storage = localStore
			}

			// Future implementation:
			// teamStore, err := NewTursoStorage(config.Team, config.UserID, logger)
			// if err != nil {
			//     logger.WithError(err).Error("Failed to initialize team storage, falling back to local")
			//     manager.mode = ModeSolo
			//     manager.storage = localStore
			// } else {
			//     manager.teamStore = teamStore
			//     manager.storage = teamStore
			//     logger.Info("Storage manager initialized in team mode")
			// }
		}

	default:
		return nil, fmt.Errorf("unknown storage mode: %s", config.Mode)
	}

	return manager, nil
}

// GetStorage returns the active storage implementation
func (m *Manager) GetStorage() Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storage
}

// GetMode returns the current storage mode
func (m *Manager) GetMode() Mode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mode
}

// GetUserID returns the current user ID
func (m *Manager) GetUserID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userID
}

// GetDataDir returns the local data directory path
func (m *Manager) GetDataDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dataDir
}

// GetDB returns the database connection for direct access
func (m *Manager) GetDB() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.localStore != nil {
		return m.localStore.GetDB()
	}
	return nil
}

// GetVectorStore returns the active vector store (always local-first)
func (m *Manager) GetVectorStore() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.localStore != nil {
		return m.localStore.getVectorStore()
	}
	return nil
}

// SwitchToTeamMode switches from solo to team mode
func (m *Manager) SwitchToTeamMode(ctx context.Context, teamConfig *TursoConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mode == ModeTeam {
		return fmt.Errorf("already in team mode")
	}

	// TODO: Implement team mode switching
	// 1. Initialize Turso storage
	// 2. Optionally migrate local data to team
	// 3. Switch active storage

	return fmt.Errorf("team mode not yet implemented")
}

// SwitchToSoloMode switches from team to solo mode
func (m *Manager) SwitchToSoloMode(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mode == ModeSolo {
		return fmt.Errorf("already in solo mode")
	}

	// Switch to local storage
	m.storage = m.localStore
	m.mode = ModeSolo

	// Close team storage if present
	if m.teamStore != nil {
		if err := m.teamStore.Close(); err != nil {
			m.logger.WithError(err).Warn("Failed to close team storage")
		}
		m.teamStore = nil
	}

	m.logger.Info("Switched to solo mode")
	return nil
}

// Close closes all storage connections
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error

	if m.localStore != nil {
		if err := m.localStore.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close local storage: %w", err))
		}
	}

	if m.teamStore != nil {
		if err := m.teamStore.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close team storage: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing storage: %v", errors)
	}

	return nil
}

// Delegate methods to active storage

// getStorage returns the active storage with read lock
func (m *Manager) getStorage() Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storage
}

func (m *Manager) SaveConnection(ctx context.Context, conn *Connection) error {
	return m.getStorage().SaveConnection(ctx, conn)
}

func (m *Manager) GetConnections(ctx context.Context, filters *ConnectionFilters) ([]*Connection, error) {
	return m.getStorage().GetConnections(ctx, filters)
}

func (m *Manager) GetConnection(ctx context.Context, id string) (*Connection, error) {
	return m.getStorage().GetConnection(ctx, id)
}

func (m *Manager) UpdateConnection(ctx context.Context, conn *Connection) error {
	return m.getStorage().UpdateConnection(ctx, conn)
}

func (m *Manager) DeleteConnection(ctx context.Context, id string) error {
	return m.getStorage().DeleteConnection(ctx, id)
}

func (m *Manager) GetAvailableEnvironments(ctx context.Context) ([]string, error) {
	return m.getStorage().GetAvailableEnvironments(ctx)
}

func (m *Manager) SaveQuery(ctx context.Context, query *SavedQuery) error {
	return m.getStorage().SaveQuery(ctx, query)
}

func (m *Manager) GetQueries(ctx context.Context, filters *QueryFilters) ([]*SavedQuery, error) {
	return m.getStorage().GetQueries(ctx, filters)
}

func (m *Manager) GetQuery(ctx context.Context, id string) (*SavedQuery, error) {
	return m.getStorage().GetQuery(ctx, id)
}

func (m *Manager) UpdateQuery(ctx context.Context, query *SavedQuery) error {
	return m.getStorage().UpdateQuery(ctx, query)
}

func (m *Manager) DeleteQuery(ctx context.Context, id string) error {
	return m.getStorage().DeleteQuery(ctx, id)
}

func (m *Manager) SaveQueryHistory(ctx context.Context, history *QueryHistory) error {
	return m.getStorage().SaveQueryHistory(ctx, history)
}

func (m *Manager) GetQueryHistory(ctx context.Context, filters *HistoryFilters) ([]*QueryHistory, error) {
	return m.getStorage().GetQueryHistory(ctx, filters)
}

func (m *Manager) DeleteQueryHistory(ctx context.Context, id string) error {
	return m.getStorage().DeleteQueryHistory(ctx, id)
}

func (m *Manager) IndexDocument(ctx context.Context, doc *Document) error {
	return m.getStorage().IndexDocument(ctx, doc)
}

func (m *Manager) SearchDocuments(ctx context.Context, embedding []float32, filters *DocumentFilters) ([]*Document, error) {
	return m.getStorage().SearchDocuments(ctx, embedding, filters)
}

func (m *Manager) GetDocument(ctx context.Context, id string) (*Document, error) {
	return m.getStorage().GetDocument(ctx, id)
}

func (m *Manager) DeleteDocument(ctx context.Context, id string) error {
	return m.getStorage().DeleteDocument(ctx, id)
}

func (m *Manager) CacheSchema(ctx context.Context, connID string, schema *SchemaCache) error {
	return m.getStorage().CacheSchema(ctx, connID, schema)
}

func (m *Manager) GetCachedSchema(ctx context.Context, connID string) (*SchemaCache, error) {
	return m.getStorage().GetCachedSchema(ctx, connID)
}

func (m *Manager) InvalidateSchemaCache(ctx context.Context, connID string) error {
	return m.getStorage().InvalidateSchemaCache(ctx, connID)
}

func (m *Manager) GetSetting(ctx context.Context, key string) (string, error) {
	return m.getStorage().GetSetting(ctx, key)
}

func (m *Manager) SetSetting(ctx context.Context, key, value string) error {
	return m.getStorage().SetSetting(ctx, key, value)
}

func (m *Manager) DeleteSetting(ctx context.Context, key string) error {
	return m.getStorage().DeleteSetting(ctx, key)
}

func (m *Manager) GetTeam(ctx context.Context) (*Team, error) {
	return m.getStorage().GetTeam(ctx)
}

func (m *Manager) GetTeamMembers(ctx context.Context) ([]*TeamMember, error) {
	return m.getStorage().GetTeamMembers(ctx)
}
