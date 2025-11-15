package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/jbeck018/howlerops/backend-go/internal/config"
	"github.com/jbeck018/howlerops/backend-go/internal/middleware"
	"github.com/jbeck018/howlerops/backend-go/internal/services"
)

// GRPCServer wraps the gRPC server with additional functionality
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	config   *config.Config
	logger   *logrus.Logger
	services *services.Services
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(cfg *config.Config, logger *logrus.Logger, services *services.Services) (*GRPCServer, error) {
	// Create listener
	listener, err := net.Listen("tcp", cfg.GetGRPCAddress())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC listener: %w", err)
	}

	// Configure server options
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge:      time.Hour,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Minute,
			Timeout:               1 * time.Minute,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxRecvMsgSize(32 * 1024 * 1024), // 32MB
		grpc.MaxSendMsgSize(32 * 1024 * 1024), // 32MB
	}

	// Add TLS if enabled
	if cfg.Server.TLSEnabled {
		cert, err := tls.LoadX509KeyPair(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
		}

		// #nosec G402 - MinVersion defaults to TLS 1.2 in Go 1.18+, explicit setting not required
		creds := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		opts = append(opts, grpc.Creds(creds))
	}

	// Configure middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.JWTSecret, logger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(cfg.Security.RateLimitRPS, cfg.Security.RateLimitBurst)

	// Setup interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_logrus.UnaryServerInterceptor(logrus.NewEntry(logger)),
		grpc_prometheus.UnaryServerInterceptor,
		grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandler(recoveryHandler)),
		authMiddleware.UnaryInterceptor,
		rateLimitMiddleware.UnaryInterceptor,
		timeoutInterceptor(cfg.Security.RequestTimeout),
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_ctxtags.StreamServerInterceptor(),
		grpc_logrus.StreamServerInterceptor(logrus.NewEntry(logger)),
		grpc_prometheus.StreamServerInterceptor,
		grpc_recovery.StreamServerInterceptor(grpc_recovery.WithRecoveryHandler(recoveryHandler)),
		authMiddleware.StreamInterceptor,
		rateLimitMiddleware.StreamInterceptor,
	}

	opts = append(opts,
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)),
	)

	// Create gRPC server
	server := grpc.NewServer(opts...)

	// Register services
	services.RegisterGRPCServices(server)

	// Enable reflection in development
	if cfg.IsDevelopment() {
		reflection.Register(server)
	}

	// Initialize Prometheus metrics
	grpc_prometheus.Register(server)

	return &GRPCServer{
		server:   server,
		listener: listener,
		config:   cfg,
		logger:   logger,
		services: services,
	}, nil
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	s.logger.WithFields(logrus.Fields{
		"address": s.config.GetGRPCAddress(),
		"tls":     s.config.Server.TLSEnabled,
	}).Info("Starting gRPC server")

	return s.server.Serve(s.listener)
}

// Stop gracefully stops the gRPC server
func (s *GRPCServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping gRPC server")

	// Create a channel to signal when graceful stop is complete
	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	// Wait for graceful stop or context timeout
	select {
	case <-done:
		s.logger.Info("gRPC server stopped gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("gRPC server stop timed out, forcing shutdown")
		s.server.Stop()
		return ctx.Err()
	}
}

// GetAddress returns the server address
func (s *GRPCServer) GetAddress() string {
	return s.listener.Addr().String()
}

// recoveryHandler handles panics in gRPC handlers
func recoveryHandler(p interface{}) error {
	return status.Errorf(codes.Internal, "Internal server error: %v", p)
}

// timeoutInterceptor adds request timeout to gRPC calls
func timeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return handler(ctx, req)
	}
}

// validateAuth validates authentication for protected methods
func validateAuth(ctx context.Context, method string) error {
	// List of methods that don't require authentication
	unprotectedMethods := map[string]bool{
		"/sqlstudio.auth.AuthService/Login":                              true,
		"/sqlstudio.health.HealthService/Check":                          true,
		"/sqlstudio.health.HealthService/Watch":                          true,
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
	}

	if unprotectedMethods[method] {
		return nil
	}

	// Check for authentication token
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "missing or invalid authentication token")
	}

	if token == "" {
		return status.Errorf(codes.Unauthenticated, "authentication token is required")
	}

	return nil
}

// extractUserFromContext extracts user information from context
func extractUserFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return "", status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	return userID, nil
}

// GRPCServerConfig holds gRPC server configuration
type GRPCServerConfig struct {
	Address           string
	TLSEnabled        bool
	TLSCertFile       string
	TLSKeyFile        string
	MaxRecvMsgSize    int
	MaxSendMsgSize    int
	ConnectionTimeout time.Duration
	MaxConnectionAge  time.Duration
	MaxConnectionIdle time.Duration
	KeepAliveTime     time.Duration
	KeepAliveTimeout  time.Duration
	EnableReflection  bool
	EnableMetrics     bool
}

// GetDefaultGRPCConfig returns default gRPC server configuration
func GetDefaultGRPCConfig() GRPCServerConfig {
	return GRPCServerConfig{
		Address:           ":9090",
		TLSEnabled:        false,
		MaxRecvMsgSize:    32 * 1024 * 1024, // 32MB
		MaxSendMsgSize:    32 * 1024 * 1024, // 32MB
		ConnectionTimeout: 30 * time.Second,
		MaxConnectionAge:  time.Hour,
		MaxConnectionIdle: 5 * time.Minute,
		KeepAliveTime:     5 * time.Minute,
		KeepAliveTimeout:  time.Minute,
		EnableReflection:  false,
		EnableMetrics:     true,
	}
}
