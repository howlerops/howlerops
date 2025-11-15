package middleware_test

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/jbeck018/howlerops/backend-go/internal/middleware"
)

func newSilentLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func newAuthMiddleware(secret string) *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware(secret, newSilentLogger())
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(metadata.MD)       {}
func (m *mockServerStream) SendMsg(interface{}) error    { return nil }
func (m *mockServerStream) RecvMsg(interface{}) error    { return nil }

func contextWithMetadata(md map[string]string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.New(md))
}

func authContext(token string) context.Context {
	header := map[string]string{
		"authorization": "",
	}
	if token != "" {
		header["authorization"] = "Bearer " + token
	}
	return contextWithMetadata(header)
}

type testAddr struct {
	addr string
}

func (a *testAddr) Network() string { return "tcp" }
func (a *testAddr) String() string  { return a.addr }

func peerContext(addr string) context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{Addr: &testAddr{addr: addr}})
}

func unaryHandlerSuccess(ctx context.Context, req interface{}) (interface{}, error) {
	return "success", nil
}

func unaryHandlerResponse(resp interface{}) grpc.UnaryHandler {
	return func(context.Context, interface{}) (interface{}, error) {
		return resp, nil
	}
}

func unaryHandlerError(err error) grpc.UnaryHandler {
	return func(context.Context, interface{}) (interface{}, error) {
		return nil, err
	}
}

func streamHandlerSuccess(srv interface{}, stream grpc.ServerStream) error {
	return nil
}

func streamHandlerError(err error) grpc.StreamHandler {
	return func(interface{}, grpc.ServerStream) error {
		return err
	}
}
