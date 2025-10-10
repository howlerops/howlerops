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

	"github.com/sql-studio/backend-go/internal/config"
	"github.com/sql-studio/backend-go/pkg/database"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/internal/server"
	"github.com/sql-studio/backend-go/internal/services"
	"github.com/sql-studio/backend-go/pkg/logger"
)

func main() {
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
		"version":     "1.0.0",
		"environment": cfg.GetEnv(),
		"grpc_port":   cfg.Server.GRPCPort,
		"http_port":   cfg.Server.HTTPPort,
	}).Info("Starting HowlerOps Backend")

	// Create database manager
	dbManager := database.NewManager(appLogger)

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.JWTSecret, appLogger)

	// Create services
	serviceConfig := services.Config{
		Database: cfg.Database,
		Auth:     cfg.Auth,
		Security: cfg.Security,
	}

	svc, err := services.NewServices(serviceConfig, dbManager, authMiddleware, appLogger)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to create services")
	}

	// Create gRPC server
	grpcServer, err := server.NewGRPCServer(cfg, appLogger, svc)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to create gRPC server")
	}

	// Create HTTP gateway server
	httpServer, err := server.NewHTTPServer(cfg, appLogger, svc)
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
	if err := dbManager.Close(); err != nil {
		appLogger.WithError(err).Error("Failed to close database connections")
	}

	// Wait for background tasks to finish
	wg.Wait()

	appLogger.Info("Graceful shutdown completed")
}

// runBackgroundTasks runs periodic background tasks
func runBackgroundTasks(ctx context.Context, svc *services.Services, logger *logrus.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Background tasks stopped")
			return
		case <-ticker.C:
			// Cleanup expired sessions
			if err := svc.Auth.CleanupExpiredSessions(ctx); err != nil {
				logger.WithError(err).Error("Failed to cleanup expired sessions")
			}

			// Cleanup old login attempts
			if err := svc.Auth.CleanupOldLoginAttempts(ctx); err != nil {
				logger.WithError(err).Error("Failed to cleanup old login attempts")
			}

			// Health check all database connections
			healthStatuses := svc.Database.HealthCheckAll(ctx)
			unhealthyCount := 0
			for connID, status := range healthStatuses {
				if status.Status != "healthy" {
					unhealthyCount++
					logger.WithFields(logrus.Fields{
						"connection_id": connID,
						"status":        status.Status,
						"message":       status.Message,
					}).Warn("Unhealthy database connection detected")
				}
			}

			if unhealthyCount > 0 {
				logger.WithField("unhealthy_connections", unhealthyCount).Warn("Some database connections are unhealthy")
			}

			logger.Debug("Background tasks completed successfully")
		}
	}
}

// initializeDefaultData initializes default data if needed
func initializeDefaultData(svc *services.Services, logger *logrus.Logger) error {
	// Create default admin user if it doesn't exist
	// This is a simplified implementation - in production, you'd want more robust user management
	logger.Info("Checking for default admin user...")

	// Implementation would go here
	// ctx := context.Background() // Will be used when implementing actual initialization

	return nil
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

		if !cfg.Server.TLSEnabled {
			logger.Warn("TLS is disabled in production environment")
		}

		if cfg.Log.Level == "debug" {
			logger.Warn("Debug logging is enabled in production")
		}
	}

	return nil
}