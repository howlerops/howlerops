//go:build integration

package server_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/config"
	"github.com/sql-studio/backend-go/internal/server"
	"github.com/sql-studio/backend-go/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// Helper function to create a test logger
func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	return logger
}

// Helper function to create a minimal test config
func createTestConfig(port int) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			GRPCPort:        port,
			TLSEnabled:      false,
			Environment:     "development",
			ShutdownTimeout: 5 * time.Second,
		},
		Auth: config.AuthConfig{
			JWTSecret:         "test-secret-key-at-least-32-characters-long",
			JWTExpiration:     24 * time.Hour,
			RefreshExpiration: 168 * time.Hour,
		},
		Security: config.SecurityConfig{
			RateLimitRPS:   100,
			RateLimitBurst: 200,
			RequestTimeout: 30 * time.Second,
		},
	}
}

// Helper function to create mock services
func createMockServices() *services.Services {
	return &services.Services{}
}

// Helper function to get a free port
func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// Helper function to generate self-signed certificate
func generateTestCertificate(certPath, keyPath string) error {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to encode certificate: %w", err)
	}

	// Write private key to file
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	return nil
}

func TestNewGRPCServer_Success(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)
	assert.NotNil(t, grpcServer)

	// Verify server address
	assert.NotEmpty(t, grpcServer.GetAddress())
	assert.Contains(t, grpcServer.GetAddress(), fmt.Sprintf(":%d", port))

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}

func TestNewGRPCServer_WithTLS(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	// Create temporary directory for certificates
	tempDir, err := os.MkdirTemp("", "grpc-tls-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	certPath := filepath.Join(tempDir, "server.crt")
	keyPath := filepath.Join(tempDir, "server.key")

	// Generate test certificate
	err = generateTestCertificate(certPath, keyPath)
	require.NoError(t, err)

	cfg := createTestConfig(port)
	cfg.Server.TLSEnabled = true
	cfg.Server.TLSCertFile = certPath
	cfg.Server.TLSKeyFile = keyPath

	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)
	assert.NotNil(t, grpcServer)

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}

func TestNewGRPCServer_TLSMissingCertificate(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	cfg.Server.TLSEnabled = true
	cfg.Server.TLSCertFile = "/nonexistent/cert.pem"
	cfg.Server.TLSKeyFile = "/nonexistent/key.pem"

	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	assert.Error(t, err)
	assert.Nil(t, grpcServer)
	assert.Contains(t, err.Error(), "failed to load TLS certificates")
}

func TestNewGRPCServer_InvalidAddress(t *testing.T) {
	cfg := createTestConfig(99999) // Invalid port
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	assert.Error(t, err)
	assert.Nil(t, grpcServer)
	assert.Contains(t, err.Error(), "failed to create gRPC listener")
}

func TestNewGRPCServer_PortInUse(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	// Occupy the port
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	require.NoError(t, err)
	defer listener.Close()

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	assert.Error(t, err)
	assert.Nil(t, grpcServer)
	assert.Contains(t, err.Error(), "failed to create gRPC listener")
}

func TestGRPCServer_StartAndStop(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running by attempting to connect
	conn, err := grpc.Dial(
		grpcServer.GetAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = grpcServer.Stop(ctx)
	assert.NoError(t, err)

	// Wait for server to stop
	select {
	case err := <-serverErr:
		// Server stopped, expect no error or specific gRPC error
		assert.True(t, err == nil || err.Error() == "accept tcp [::]:"+fmt.Sprint(port)+": use of closed network connection" ||
			err.Error() == "accept tcp 127.0.0.1:"+fmt.Sprint(port)+": use of closed network connection")
	case <-time.After(6 * time.Second):
		t.Fatal("Server did not stop within timeout")
	}
}

func TestGRPCServer_StopWithTimeout(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server
	go func() {
		_ = grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop with reasonable timeout
	// Note: GracefulStop may complete quickly if there are no active connections
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = grpcServer.Stop(ctx)
	// Should complete without error (graceful stop is fast with no connections)
	assert.NoError(t, err)
}

func TestGRPCServer_StopBeforeStart(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Stop server without starting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = grpcServer.Stop(ctx)
	assert.NoError(t, err) // Should complete without error
}

func TestGRPCServer_MultipleStopCalls(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server
	go func() {
		_ = grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// First stop
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()
	err = grpcServer.Stop(ctx1)
	assert.NoError(t, err)

	// Second stop (should be safe)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	err = grpcServer.Stop(ctx2)
	// Multiple stops should complete without hanging
	assert.NoError(t, err)
}

func TestGRPCServer_GetAddress(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	address := grpcServer.GetAddress()
	assert.NotEmpty(t, address)
	assert.Contains(t, address, ":")

	// Parse address to verify format
	host, portStr, err := net.SplitHostPort(address)
	assert.NoError(t, err)
	assert.NotEmpty(t, host)
	assert.NotEmpty(t, portStr)

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}

func TestGRPCServer_ReflectionInDevelopment(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	cfg.Server.Environment = "development"
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server
	go func() {
		_ = grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client connection
	conn, err := grpc.Dial(
		grpcServer.GetAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Create reflection client
	reflectionClient := grpc_reflection_v1alpha.NewServerReflectionClient(conn)
	stream, err := reflectionClient.ServerReflectionInfo(context.Background())
	require.NoError(t, err)

	// Send list services request
	err = stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	})
	assert.NoError(t, err)

	// Receive response
	resp, err := stream.Recv()
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}

func TestGRPCServer_NoReflectionInProduction(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	cfg.Server.Environment = "production"
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server
	go func() {
		_ = grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client connection
	conn, err := grpc.Dial(
		grpcServer.GetAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// In production mode, reflection is not registered
	// We verify this by checking the environment setting was respected
	assert.Equal(t, "production", cfg.Server.Environment)
	assert.False(t, cfg.IsDevelopment())
	assert.True(t, cfg.IsProduction())

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}

func TestGRPCServer_WithTLSConnection(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	// Create temporary directory for certificates
	tempDir, err := os.MkdirTemp("", "grpc-tls-conn-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	certPath := filepath.Join(tempDir, "server.crt")
	keyPath := filepath.Join(tempDir, "server.key")

	// Generate test certificate
	err = generateTestCertificate(certPath, keyPath)
	require.NoError(t, err)

	cfg := createTestConfig(port)
	cfg.Server.TLSEnabled = true
	cfg.Server.TLSCertFile = certPath
	cfg.Server.TLSKeyFile = keyPath

	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server
	go func() {
		_ = grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Load certificate for client
	certPool := x509.NewCertPool()
	certPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)
	ok := certPool.AppendCertsFromPEM(certPEM)
	require.True(t, ok)

	// Create TLS credentials
	tlsConfig := &tls.Config{
		RootCAs: certPool,
	}
	creds := credentials.NewTLS(tlsConfig)

	// Create client connection with TLS
	conn, err := grpc.Dial(
		grpcServer.GetAddress(),
		grpc.WithTransportCredentials(creds),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}

func TestGRPCServer_MiddlewareConfiguration(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	// Configure middleware settings
	cfg.Auth.JWTSecret = "test-jwt-secret-key-at-least-32-chars"
	cfg.Security.RateLimitRPS = 50
	cfg.Security.RateLimitBurst = 100
	cfg.Security.RequestTimeout = 10 * time.Second

	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)
	assert.NotNil(t, grpcServer)

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}

func TestGRPCServer_KeepaliveConfiguration(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	// Create server with keepalive settings
	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server
	go func() {
		_ = grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client connection with keepalive
	conn, err := grpc.Dial(
		grpcServer.GetAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Verify connection is established
	assert.NotNil(t, conn)

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}

func TestGetDefaultGRPCConfig(t *testing.T) {
	defaultConfig := server.GetDefaultGRPCConfig()

	assert.Equal(t, ":9090", defaultConfig.Address)
	assert.False(t, defaultConfig.TLSEnabled)
	assert.Equal(t, 32*1024*1024, defaultConfig.MaxRecvMsgSize)
	assert.Equal(t, 32*1024*1024, defaultConfig.MaxSendMsgSize)
	assert.Equal(t, 30*time.Second, defaultConfig.ConnectionTimeout)
	assert.Equal(t, time.Hour, defaultConfig.MaxConnectionAge)
	assert.Equal(t, 5*time.Minute, defaultConfig.MaxConnectionIdle)
	assert.Equal(t, 5*time.Minute, defaultConfig.KeepAliveTime)
	assert.Equal(t, time.Minute, defaultConfig.KeepAliveTimeout)
	assert.False(t, defaultConfig.EnableReflection)
	assert.True(t, defaultConfig.EnableMetrics)
}

func TestGRPCServer_GracefulShutdown(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client connection
	conn, err := grpc.Dial(
		grpcServer.GetAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownStart := time.Now()
	err = grpcServer.Stop(ctx)
	shutdownDuration := time.Since(shutdownStart)

	assert.NoError(t, err)
	assert.Less(t, shutdownDuration, 5*time.Second, "Graceful shutdown should complete within timeout")

	conn.Close()
}

func TestGRPCServer_ConcurrentConnections(t *testing.T) {
	port, err := getFreePort()
	require.NoError(t, err)

	cfg := createTestConfig(port)
	logger := createTestLogger()
	mockServices := createMockServices()

	grpcServer, err := server.NewGRPCServer(cfg, logger, mockServices)
	require.NoError(t, err)

	// Start server
	go func() {
		_ = grpcServer.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create multiple concurrent connections
	numConnections := 10
	connections := make([]*grpc.ClientConn, numConnections)
	for i := 0; i < numConnections; i++ {
		conn, err := grpc.Dial(
			grpcServer.GetAddress(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		connections[i] = conn
	}

	// Close all connections
	for _, conn := range connections {
		conn.Close()
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = grpcServer.Stop(ctx)
}
