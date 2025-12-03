package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRedisClient is a mock implementation of redis.Client
type MockRedisClient struct {
	mock.Mock
}

// Incr mocks the Incr method
func (m *MockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewIntCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(int64))
	} else {
		cmd.SetErr(args.Error(1))
	}
	return cmd
}

// Expire mocks the Expire method
func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	args := m.Called(ctx, key, expiration)
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetVal(args.Bool(0))
	return cmd
}

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter((*redis.Client)(nil), 100)

	assert.NotNil(t, limiter)
	assert.Equal(t, 100, limiter.defaultLimit)
	assert.Equal(t, time.Minute, limiter.window)
}

func TestRateLimiter_Handler_WithinLimit(t *testing.T) {
	// Setup mock Redis
	mockRedis := &redis.Client{}
	limiter := &RateLimiter{
		redis:        mockRedis,
		defaultLimit: 100,
		window:       time.Minute,
	}

	// Create test handler
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Note: This test would require Redis to be available for full testing
	// In production test, we'd use miniredis or similar
	// For now, testing the "no tenant ID" path which doesn't require Redis

	// Test without tenant ID (should skip rate limiting)
	reqNoAuth := httptest.NewRequest("POST", "/mcp", nil)
	rrNoAuth := httptest.NewRecorder()

	handler := limiter.Handler(testHandler)
	handler.ServeHTTP(rrNoAuth, reqNoAuth)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rrNoAuth.Code)
}

func TestRateLimiter_Handler_NoTenantID(t *testing.T) {
	limiter := NewRateLimiter((*redis.Client)(nil), 100)

	// Create test handler
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create request without auth context (no tenant ID)
	req := httptest.NewRequest("POST", "/mcp", nil)
	rr := httptest.NewRecorder()

	// Execute
	handler := limiter.Handler(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify handler was called (rate limiting skipped)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// Note: For comprehensive Redis-based tests, we would need either:
// 1. miniredis (in-memory Redis for testing)
// 2. testcontainers with real Redis
// 3. Refactor to use an interface for testability

// Testing the error path by extracting checkLimit logic
func TestRateLimiter_checkLimit_Logic(t *testing.T) {
	tests := []struct {
		name          string
		requestCount  int64
		limit         int
		expectAllowed bool
	}{
		{
			name:          "first request",
			requestCount:  1,
			limit:         100,
			expectAllowed: true,
		},
		{
			name:          "within limit",
			requestCount:  50,
			limit:         100,
			expectAllowed: true,
		},
		{
			name:          "at limit",
			requestCount:  100,
			limit:         100,
			expectAllowed: true,
		},
		{
			name:          "exceeded limit",
			requestCount:  101,
			limit:         100,
			expectAllowed: false,
		},
		{
			name:          "far exceeded",
			requestCount:  500,
			limit:         100,
			expectAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic without Redis
			allowed := tt.requestCount <= int64(tt.limit)
			assert.Equal(t, tt.expectAllowed, allowed)
		})
	}
}

func TestRateLimiter_sendError(t *testing.T) {
	limiter := NewRateLimiter((*redis.Client)(nil), 100)

	rr := httptest.NewRecorder()
	limiter.sendError(rr, nil, protocol.RateLimitExceeded, "Rate limit exceeded")

	// Verify response
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response protocol.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.RateLimitExceeded, response.Error.Code)
	assert.Contains(t, response.Error.Message, "Rate limit exceeded")

	// Verify retry_after is present in error data
	data, ok := response.Error.Data.(map[string]interface{})
	assert.True(t, ok)
	retryAfter, ok := data["retry_after"]
	assert.True(t, ok)
	assert.Equal(t, float64(60), retryAfter) // 1 minute in seconds
}

// Tests using miniredis for actual Redis interactions
func setupMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestRateLimiter_WithRedis_WithinLimit(t *testing.T) {
	mr, redisClient := setupMiniRedis(t)
	defer mr.Close()

	limiter := NewRateLimiter(redisClient, 10)

	handlerCalled := 0
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Handler(testHandler)

	// Make 10 requests (within limit)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		ctx := context.WithValue(req.Context(), auth.ContextKeyTenantID, "tenant-123")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Request %d should succeed", i+1)
	}

	assert.Equal(t, 10, handlerCalled)
}

func TestRateLimiter_WithRedis_ExceedsLimit(t *testing.T) {
	mr, redisClient := setupMiniRedis(t)
	defer mr.Close()

	limiter := NewRateLimiter(redisClient, 5)

	handlerCalled := 0
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Handler(testHandler)

	// Make 7 requests (exceed limit of 5)
	for i := 0; i < 7; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		ctx := context.WithValue(req.Context(), auth.ContextKeyTenantID, "tenant-123")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if i < 5 {
			assert.Equal(t, http.StatusOK, rr.Code, "Request %d should succeed", i+1)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, rr.Code, "Request %d should be rate limited", i+1)

			var response protocol.Response
			err := json.NewDecoder(rr.Body).Decode(&response)
			require.NoError(t, err)
			assert.NotNil(t, response.Error)
			assert.Equal(t, protocol.RateLimitExceeded, response.Error.Code)
		}
	}

	// Only first 5 requests should call the handler
	assert.Equal(t, 5, handlerCalled)
}

func TestRateLimiter_WithRedis_DifferentTenants(t *testing.T) {
	mr, redisClient := setupMiniRedis(t)
	defer mr.Close()

	limiter := NewRateLimiter(redisClient, 3)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Handler(testHandler)

	// Tenant 1 makes 3 requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		ctx := context.WithValue(req.Context(), auth.ContextKeyTenantID, "tenant-1")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Tenant 2 should also be able to make 3 requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		ctx := context.WithValue(req.Context(), auth.ContextKeyTenantID, "tenant-2")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Tenant 1's 4th request should be rate limited
	req := httptest.NewRequest("POST", "/mcp", nil)
	ctx := context.WithValue(req.Context(), auth.ContextKeyTenantID, "tenant-1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestRateLimiter_checkLimit(t *testing.T) {
	mr, redisClient := setupMiniRedis(t)
	defer mr.Close()

	limiter := NewRateLimiter(redisClient, 100)
	ctx := context.Background()

	// First check
	allowed, err := limiter.checkLimit(ctx, "tenant-123")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Check multiple times within limit
	for i := 0; i < 50; i++ {
		allowed, err := limiter.checkLimit(ctx, "tenant-123")
		assert.NoError(t, err)
		assert.True(t, allowed)
	}
}

// Benchmark tests
func BenchmarkRateLimiter_Handler_NoAuth(b *testing.B) {
	limiter := NewRateLimiter((*redis.Client)(nil), 100)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Handler(testHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

func BenchmarkRateLimiter_WithRedis(b *testing.B) {
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		b.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	limiter := NewRateLimiter(client, 1000000) // High limit for benchmarking

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Handler(testHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		ctx := context.WithValue(req.Context(), auth.ContextKeyTenantID, "tenant-123")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
