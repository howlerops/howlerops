package middleware_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jbeck018/howlerops/backend-go/internal/middleware"
)

func TestRateLimitMiddleware_AllowsRequestsUnderLimit(t *testing.T) {
	rl := middleware.NewRateLimitMiddleware(10, 20)

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	ctx := peerContext("10.0.0.5:1234")

	resp, err := rl.UnaryInterceptor(ctx, "payload", info, unaryHandlerSuccess)
	require.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestRateLimitMiddleware_BlocksAfterBurst(t *testing.T) {
	rl := middleware.NewRateLimitMiddleware(1, 1)

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	ctx := peerContext("10.0.0.5:1234")

	_, err := rl.UnaryInterceptor(ctx, "payload", info, unaryHandlerSuccess)
	require.NoError(t, err)

	_, err = rl.UnaryInterceptor(ctx, "payload", info, unaryHandlerSuccess)
	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestRateLimitMiddleware_StreamInterceptor(t *testing.T) {
	rl := middleware.NewRateLimitMiddleware(1, 1)

	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/Stream"}
	stream := &mockServerStream{ctx: peerContext("10.0.0.1:1111")}

	require.NoError(t, rl.StreamInterceptor(nil, stream, info, streamHandlerSuccess))
	err := rl.StreamInterceptor(nil, stream, info, streamHandlerSuccess)
	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestRateLimitMiddleware_SeparatesClientsByIP(t *testing.T) {
	rl := middleware.NewRateLimitMiddleware(1, 1)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	ctxA := peerContext("10.0.0.1:1000")
	ctxB := peerContext("10.0.0.2:1000")

	_, err := rl.UnaryInterceptor(ctxA, "payload", info, unaryHandlerSuccess)
	require.NoError(t, err)

	_, err = rl.UnaryInterceptor(ctxB, "payload", info, unaryHandlerSuccess)
	require.NoError(t, err, "different IP should have independent limiter")
}

func TestPerUserRateLimiter(t *testing.T) {
	rl := middleware.NewPerUserRateLimiter(1, 1)

	assert.True(t, rl.CheckLimit("alice"))
	assert.False(t, rl.CheckLimit("alice"))
	assert.True(t, rl.CheckLimit("bob"), "separate limiter per user")
}

func TestPerMethodRateLimiter(t *testing.T) {
	config := map[string]middleware.MethodRateConfig{
		"/svc.Method": {RPS: 1, Burst: 2},
	}
	rl := middleware.NewPerMethodRateLimiter(config)

	assert.True(t, rl.CheckLimit("/svc.Method", "client1"))
	assert.True(t, rl.CheckLimit("/svc.Method", "client1"))
	assert.False(t, rl.CheckLimit("/svc.Method", "client1"))
	assert.True(t, rl.CheckLimit("/svc.Other", "client1"), "uses default config when method missing")
}

func TestAdaptiveRateLimiter_AdjustsLimiters(t *testing.T) {
	adaptive := middleware.NewAdaptiveRateLimiter(2, 2, 1, 4)

	adaptive.UpdateLoadFactor(2.0)

	assert.True(t, adaptive.CheckLimit("client"))
	assert.True(t, adaptive.CheckLimit("client"))
	assert.False(t, adaptive.CheckLimit("client"))
}

func TestConcurrentRequests(t *testing.T) {
	rl := middleware.NewRateLimitMiddleware(5, 5)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	failures := 0

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := peerContext("10.0.0.99:0")
			_, err := rl.UnaryInterceptor(ctx, "payload", info, unaryHandlerSuccess)
			mu.Lock()
			if err != nil {
				failures++
			} else {
				success++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	assert.Greater(t, success, 0)
	assert.Greater(t, failures, 0)
	assert.Equal(t, 20, success+failures)
}

func TestGetClientIPFallsBackWhenPeerMissing(t *testing.T) {
	rl := middleware.NewRateLimitMiddleware(1, 1)
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	_, err := rl.UnaryInterceptor(ctx, "payload", info, unaryHandlerSuccess)
	require.NoError(t, err)
}
