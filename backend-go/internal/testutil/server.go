package testutil

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// NewTestGRPCServer creates a test gRPC server using bufconn
func NewTestGRPCServer(t *testing.T) (*grpc.Server, *bufconn.Listener) {
	t.Helper()

	// Create a listener with 1MB buffer
	listener := bufconn.Listen(1024 * 1024)

	// Create gRPC server
	server := grpc.NewServer()

	// Cleanup
	t.Cleanup(func() {
		server.Stop()
		listener.Close()
	})

	return server, listener
}

// NewTestGRPCClient creates a test gRPC client connection
func NewTestGRPCClient(t *testing.T, listener *bufconn.Listener) *grpc.ClientConn {
	t.Helper()

	conn, err := grpc.Dial(
		"bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("failed to create test gRPC client: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

// StartTestGRPCServer starts the test gRPC server in background
func StartTestGRPCServer(t *testing.T, server *grpc.Server, listener *bufconn.Listener) {
	t.Helper()

	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("test gRPC server error: %v", err)
		}
	}()
}
