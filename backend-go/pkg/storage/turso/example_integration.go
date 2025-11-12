package turso

// This file provides integration examples for SQL Studio backend
// DO NOT import this in production - it's documentation only

/*

EXAMPLE 1: Basic Integration in main.go
========================================

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 1. Initialize Turso client
	tursoConfig := &turso.Config{
		URL:       os.Getenv("TURSO_URL"),       // libsql://your-db.turso.io
		AuthToken: os.Getenv("TURSO_AUTH_TOKEN"), // Your Turso auth token
		MaxConns:  10,
	}

	db, err := turso.NewClient(tursoConfig, logger)
	if err != nil {
		log.Fatal("Failed to connect to Turso:", err)
	}
	defer db.Close()

	// 2. Initialize schema (creates tables if needed)
	if err := turso.InitializeSchema(db, logger); err != nil {
		log.Fatal("Failed to initialize schema:", err)
	}

	// 3. Create store instances
	userStore := turso.NewTursoUserStore(db, logger)
	sessionStore := turso.NewTursoSessionStore(db, logger)
	loginAttemptStore := turso.NewTursoLoginAttemptStore(db, logger)
	appDataStore := turso.NewTursoAppDataStore(db, logger)

	// 4. Create auth service
	jwtSecret := os.Getenv("JWT_SECRET")
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret, logger)

	authService := auth.NewService(
		userStore,
		sessionStore,
		loginAttemptStore,
		authMiddleware,
		auth.Config{
			BcryptCost:        12,
			JWTExpiration:     24 * time.Hour,
			RefreshExpiration: 7 * 24 * time.Hour,
			MaxLoginAttempts:  5,
			LockoutDuration:   15 * time.Minute,
		},
		logger,
	)

	// 5. Start background cleanup tasks
	go startCleanupTasks(context.Background(), sessionStore, loginAttemptStore, logger)

	// 6. Set up HTTP handlers
	setupHTTPHandlers(authService, appDataStore, logger)

	logger.Info("SQL Studio backend started with Turso storage")
}

func startCleanupTasks(ctx context.Context, sessionStore *turso.TursoSessionStore, attemptStore *turso.TursoLoginAttemptStore, logger *logrus.Logger) {
	// Cleanup expired sessions every hour
	sessionTicker := time.NewTicker(1 * time.Hour)
	defer sessionTicker.Stop()

	// Cleanup old login attempts every 6 hours
	attemptTicker := time.NewTicker(6 * time.Hour)
	defer attemptTicker.Stop()

	for {
		select {
		case <-sessionTicker.C:
			if err := sessionStore.CleanupExpiredSessions(ctx); err != nil {
				logger.WithError(err).Error("Failed to cleanup expired sessions")
			}
		case <-attemptTicker.C:
			before := time.Now().Add(-24 * time.Hour)
			if err := attemptStore.CleanupOldAttempts(ctx, before); err != nil {
				logger.WithError(err).Error("Failed to cleanup old login attempts")
			}
		case <-ctx.Done():
			return
		}

EXAMPLE 2: User Registration Handler
=====================================

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/sql-studio/backend-go/internal/auth"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Create user
	:= &auth.User{
		ID:       uuid.New().String(),
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // Will be hashed by auth service
		Role:     "user",
		Active:   true,
	}

	if err := h.authService.CreateUser(r.Context(), user); err != nil {
		h.logger.WithError(err).Error("Failed to create user")
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":      user.ID,
		"message": "User created successfully",
	})
}

EXAMPLE 3: Login Handler
=========================

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RememberMe bool   `json:"remember_me"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get IP and user agent
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	// Attempt login
	loginResp, err := h.authService.Login(r.Context(), &auth.LoginRequest{
		Username:   req.Username,
		Password:   req.Password,
		RememberMe: req.RememberMe,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Return tokens
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":          loginResp.User,
		"token":         loginResp.Token,
		"refresh_token": loginResp.RefreshToken,
		"expires_at":    loginResp.ExpiresAt,
	})
}

EXAMPLE 4: Sync Connection Templates
====================================

func (h *Handler) SyncConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userID := r.Context().Value("user_id").(string)

	switch r.Method {
	case http.MethodGet:
		// Get all connection templates for user
		templates, err := h.appDataStore.GetConnectionTemplates(r.Context(), userID, false)
		if err != nil {
			http.Error(w, "Failed to get connections", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(templates)

	case http.MethodPost:
		// Save connection template
		var conn turso.ConnectionTemplate
		if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		conn.UserID = userID
		if err := h.appDataStore.SaveConnectionTemplate(r.Context(), &conn); err != nil {
			http.Error(w, "Failed to save connection", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(conn)

	case http.MethodDelete:
		// Delete connection template
		connID := r.URL.Query().Get("id")
		if connID == "" {
			http.Error(w, "Missing connection ID", http.StatusBadRequest)
			return
		}

		if err := h.appDataStore.DeleteConnectionTemplate(r.Context(), connID); err != nil {
			http.Error(w, "Failed to delete connection", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

EXAMPLE 5: Sync Saved Queries
==============================

func (h *Handler) SyncQueriesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	switch r.Method {
	case http.MethodGet:
		queries, err := h.appDataStore.GetSavedQueries(r.Context(), userID, false)
		if err != nil {
			http.Error(w, "Failed to get queries", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(queries)

	case http.MethodPost:
		var query turso.SavedQuerySync
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		query.UserID = userID
		if err := h.appDataStore.SaveQuerySync(r.Context(), &query); err != nil {
			http.Error(w, "Failed to save query", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(query)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

EXAMPLE 6: Query History (Sanitized)
=====================================

import "regexp"

// sanitizeQuery removes literal values from SQL
func sanitizeQuery(query string) string {
	// Replace string literals
	re1 := regexp.MustCompile(`'[^']*'`)
	query = re1.ReplaceAllString(query, "?")

	// Replace numeric literals
	re2 := regexp.MustCompile(`\b\d+\b`)
	query = re2.ReplaceAllString(query, "?")

	return query
}

func (h *Handler) SaveQueryHistory(ctx context.Context, userID, query, connectionID string, duration time.Duration, rows int64, success bool, errMsg string) error {
	history := &turso.QueryHistorySync{
		UserID:         userID,
		QuerySanitized: sanitizeQuery(query), // Remove data literals!
		ConnectionID:   connectionID,
		ExecutedAt:     time.Now(),
		DurationMS:     duration.Milliseconds(),
		RowsReturned:   rows,
		Success:        success,
		ErrorMessage:   errMsg,
		SyncVersion:    1,
	}

	return h.appDataStore.SaveQueryHistory(ctx, history)
}

EXAMPLE 7: Health Check Endpoint
=================================

func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check Turso connection
	if err := h.db.Ping(); err != nil {
		h.logger.WithError(err).Error("Health check failed: database unreachable")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  "database unreachable",
		})
		return
	}

	// Check user count (basic query test)
	var userCount int
	err := h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		h.logger.WithError(err).Error("Health check failed: query error")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  "query failed",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"users":  userCount,
	})
}

EXAMPLE 8: Configuration from Environment
==========================================

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Turso struct {
		URL       string
		AuthToken string
		MaxConns  int
	}
	JWT struct {
		Secret            string
		Expiration        time.Duration
		RefreshExpiration time.Duration
	}
	Auth struct {
		BcryptCost       int
		MaxLoginAttempts int
		LockoutDuration  time.Duration
	}

func LoadConfig() (*Config, error) {
	var cfg Config

	// Turso
	cfg.Turso.URL = os.Getenv("TURSO_URL")
	cfg.Turso.AuthToken = os.Getenv("TURSO_AUTH_TOKEN")
	cfg.Turso.MaxConns = getEnvInt("TURSO_MAX_CONNS", 10)

	// JWT
	cfg.JWT.Secret = os.Getenv("JWT_SECRET")
	cfg.JWT.Expiration = getEnvDuration("JWT_EXPIRATION", 24*time.Hour)
	cfg.JWT.RefreshExpiration = getEnvDuration("REFRESH_EXPIRATION", 7*24*time.Hour)

	// Auth
	cfg.Auth.BcryptCost = getEnvInt("BCRYPT_COST", 12)
	cfg.Auth.MaxLoginAttempts = getEnvInt("MAX_LOGIN_ATTEMPTS", 5)
	cfg.Auth.LockoutDuration = getEnvDuration("LOCKOUT_DURATION", 15*time.Minute)

	// Validate
	if cfg.Turso.URL == "" {
		return nil, fmt.Errorf("TURSO_URL is required")
	}
	if cfg.Turso.AuthToken == "" {
		return nil, fmt.Errorf("TURSO_AUTH_TOKEN is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return &cfg, nil
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	return defaultVal
}

EXAMPLE 9: Docker Compose Setup
================================

version: '3.8'

services:
  backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      - TURSO_URL=libsql://your-db.turso.io
      - TURSO_AUTH_TOKEN=${TURSO_AUTH_TOKEN}
      - TURSO_MAX_CONNS=10
      - JWT_SECRET=${JWT_SECRET}
      - JWT_EXPIRATION=24h
      - REFRESH_EXPIRATION=168h
      - BCRYPT_COST=12
      - MAX_LOGIN_ATTEMPTS=5
      - LOCKOUT_DURATION=15m
      - LOG_LEVEL=info
    restart: unless-stopped

EXAMPLE 10: .env File Template
===============================

# Turso Configuration
TURSO_URL=libsql://your-db-name.turso.io
TURSO_AUTH_TOKEN=your-auth-token-here
TURSO_MAX_CONNS=10

# JWT Configuration
JWT_SECRET=your-random-secret-key-min-32-chars
JWT_EXPIRATION=24h
REFRESH_EXPIRATION=168h

# Auth Configuration
BCRYPT_COST=12
MAX_LOGIN_ATTEMPTS=5
LOCKOUT_DURATION=15m

# Server Configuration
SERVER_PORT=8080
LOG_LEVEL=info

*/
