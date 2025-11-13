package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sirupsen/logrus"

	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure"

	"github.com/sql-studio/backend-go/internal/ai"
	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/internal/config"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/internal/services"
)

// HTTPServer wraps the HTTP gateway server
type HTTPServer struct {
	server   *http.Server
	config   *config.Config
	logger   *logrus.Logger
	services *services.Services
}

// NewHTTPServer creates a new HTTP gateway server
func NewHTTPServer(cfg *config.Config, logger *logrus.Logger, svc *services.Services, authMiddleware *middleware.AuthMiddleware) (*HTTPServer, error) {
	// Create main router
	mainRouter := mux.NewRouter()

	// Create gRPC-Gateway mux for gRPC services
	grpcMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}),
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch key {
			case "Authorization", "Content-Type", "Accept":
				return key, true
			default:
				return key, false
			}
		}),
	)

	// Register Auth HTTP routes (public - no auth middleware)
	if svc.Auth != nil {
		logger.Info("Registering Auth HTTP routes")
		authHTTPHandler := auth.NewHandler(svc.Auth, logger)
		authHTTPHandler.RegisterRoutes(mainRouter)
		logger.Info("Auth HTTP routes registered successfully")
	} else {
		logger.Warn("Auth service is nil, skipping Auth route registration")
	}

	// Register AI HTTP routes
	if svc.AI != nil {
		logger.Info("Registering AI HTTP routes")
		aiHandler := ai.NewHTTPHandler(svc.AI, logger)
		aiRouter := mainRouter.PathPrefix("/api/ai").Subrouter()
		aiHandler.RegisterRoutes(aiRouter)
		logger.Info("AI HTTP routes registered successfully")
	} else {
		logger.Warn("AI service is nil, skipping AI route registration")
	}

	// Register Sync HTTP routes
	if svc.Sync != nil {
		logger.Info("Registering Sync HTTP routes")
		registerSyncRoutes(mainRouter, svc, logger)
		logger.Info("Sync HTTP routes registered successfully")
	} else {
		logger.Warn("Sync service is nil, skipping Sync route registration")
	}

	// Register Organization HTTP routes
	if svc.Organization != nil {
		logger.Info("Registering Organization HTTP routes")
		registerOrganizationRoutes(mainRouter, svc, authMiddleware, logger)
	} else {
		logger.Warn("Organization service is nil, skipping Organization route registration")
	}

	// Mount gRPC-Gateway mux
	mainRouter.PathPrefix("/api/grpc").Handler(grpcMux)

	// Add a simple health check endpoint
	mainRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status": "healthy", "service": "backend"}`)); err != nil {
			logger.WithError(err).Error("Failed to write health check response")
		}
	})

	// Register auth service
	// TODO: Uncomment when protobuf services are generated
	// opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	// grpcEndpoint := cfg.GetGRPCAddress()

	// Register auth service
	// err := authpb.RegisterAuthServiceHandlerFromEndpoint(ctx, grpcMux, grpcEndpoint, opts)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to register auth service: %w", err)
	// }

	// Register other services...
	// Similar registration for other services

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         cfg.GetHTTPAddress(),
		Handler:      corsHandler(mainRouter, cfg),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &HTTPServer{
		server:   httpServer,
		config:   cfg,
		logger:   logger,
		services: svc,
	}, nil
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.WithField("address", s.config.GetHTTPAddress()).Info("Starting HTTP gateway server")
	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP gateway server")
	return s.server.Shutdown(ctx)
}

// corsHandler adds CORS support
func corsHandler(h http.Handler, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.Security.EnableCORS {
			// Set CORS headers
			origin := r.Header.Get("Origin")
			if origin != "" {
				// In production, validate against allowed origins
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		h.ServeHTTP(w, r)
	})
}

// WebSocketServer handles WebSocket connections for real-time updates
type WebSocketServer struct {
	server   *http.Server
	config   *config.Config
	logger   *logrus.Logger
	services interface{} // Would be *services.Services
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(cfg *config.Config, logger *logrus.Logger, services interface{}) (*WebSocketServer, error) {
	mux := http.NewServeMux()

	// WebSocket endpoint for real-time updates
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, logger, services)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`)); err != nil {
			logger.WithError(err).Error("Failed to write health check response")
		}
	})

	server := &http.Server{
		Addr:         ":8081", // Different port for WebSocket
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &WebSocketServer{
		server:   server,
		config:   cfg,
		logger:   logger,
		services: services,
	}, nil
}

// Start starts the WebSocket server
func (s *WebSocketServer) Start() error {
	s.logger.WithField("address", s.server.Addr).Info("Starting WebSocket server")
	return s.server.ListenAndServe()
}

// Stop gracefully stops the WebSocket server
func (s *WebSocketServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping WebSocket server")
	return s.server.Shutdown(ctx)
}

// handleWebSocket handles WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request, logger *logrus.Logger, services interface{}) {
	// WebSocket upgrade logic would go here
	// This is a placeholder implementation

	logger.Info("WebSocket connection attempt")

	// In a real implementation, you would:
	// 1. Upgrade the HTTP connection to WebSocket
	// 2. Authenticate the client
	// 3. Handle subscription/unsubscription to real-time events
	// 4. Bridge gRPC streaming to WebSocket messages

	w.WriteHeader(http.StatusNotImplemented)
	if _, err := w.Write([]byte("WebSocket implementation coming soon")); err != nil {
		logger.WithError(err).Error("Failed to write WebSocket response")
	}
}
