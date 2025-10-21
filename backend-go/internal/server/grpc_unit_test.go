package server

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTimeoutInterceptorAddsDeadline(t *testing.T) {
	interceptor := timeoutInterceptor(50 * time.Millisecond)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected deadline to be set")
		}
		if time.Until(deadline) <= 0 {
			t.Fatalf("deadline already expired")
		}
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestValidateAuthAllowsPublicMethods(t *testing.T) {
	methods := []string{
		"/sqlstudio.auth.AuthService/Login",
		"/sqlstudio.health.HealthService/Check",
		"/sqlstudio.health.HealthService/Watch",
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
	}

	for _, method := range methods {
		if err := validateAuth(context.Background(), method); err != nil {
			t.Fatalf("expected no error for public method %s, got %v", method, err)
		}
	}
}

func TestValidateAuthRequiresToken(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(nil))
	err := validateAuth(ctx, "/secured.Method")
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}
}

func TestValidateAuthAcceptsBearerToken(t *testing.T) {
	md := metadata.New(map[string]string{
		"authorization": "Bearer token123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	if err := validateAuth(ctx, "/secured.Method"); err != nil {
		t.Fatalf("expected token to be accepted, got %v", err)
	}
}

func TestExtractUserFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", "abc-123")
	userID, err := extractUserFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "abc-123" {
		t.Fatalf("expected user id abc-123, got %s", userID)
	}
}

func TestExtractUserFromContextRequiresUser(t *testing.T) {
	_, err := extractUserFromContext(context.Background())
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}
}

func TestGetDefaultGRPCConfig(t *testing.T) {
	cfg := GetDefaultGRPCConfig()
	if cfg.Address != ":9090" {
		t.Fatalf("unexpected address: %s", cfg.Address)
	}
	if cfg.EnableMetrics != true {
		t.Fatalf("expected metrics enabled by default")
	}
	if cfg.MaxRecvMsgSize != 32*1024*1024 {
		t.Fatalf("unexpected recv size %d", cfg.MaxRecvMsgSize)
	}
}
