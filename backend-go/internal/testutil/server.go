package testutil

import (
\t"context"
\t"net"
\t"testing"

\t"google.golang.org/grpc"
\t"google.golang.org/grpc/test/bufconn"
)

// NewTestGRPCServer creates a test gRPC server using bufconn
func NewTestGRPCServer(t *testing.T) (*grpc.Server, *bufconn.Listener) {
\tt.Helper()

\t// Create a listener with 1MB buffer
\tlistener := bufconn.Listen(1024 * 1024)

\t// Create gRPC server
\tserver := grpc.NewServer()

\t// Cleanup
\tt.Cleanup(func() {
\t\tserver.Stop()
\t\tlistener.Close()
\t})

\treturn server, listener
}

// NewTestGRPCClient creates a test gRPC client connection
func NewTestGRPCClient(t *testing.T, listener *bufconn.Listener) *grpc.ClientConn {
\tt.Helper()

\tconn, err := grpc.Dial(
\t\t"bufnet",
\t\tgrpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
\t\t\treturn listener.Dial()
\t\t}),
\t\tgrpc.WithInsecure(),
\t)
\tif err != nil {
\t\tt.Fatalf("failed to create test gRPC client: %v", err)
\t}

\tt.Cleanup(func() {
\t\tconn.Close()
\t})

\treturn conn
}

// StartTestGRPCServer starts the test gRPC server in background
func StartTestGRPCServer(t *testing.T, server *grpc.Server, listener *bufconn.Listener) {
\tt.Helper()

\tgo func() {
\t\tif err := server.Serve(listener); err != nil {
\t\t\tt.Logf("test gRPC server error: %v", err)
\t\t}
\t}()
}
