package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
)

// RateLimiter implements token bucket rate limiting using Redis
type RateLimiter struct {
	redis        *redis.Client
	defaultLimit int // requests per minute
	window       time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client, defaultLimit int) *RateLimiter {
	return &RateLimiter{
		redis:        redisClient,
		defaultLimit: defaultLimit,
		window:       time.Minute,
	}
}

// Handler wraps an HTTP handler with rate limiting
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract tenant ID from context
		tenantID, err := auth.ExtractTenantID(ctx)
		if err != nil {
			// If no tenant ID, skip rate limiting (for unauthenticated requests)
			next.ServeHTTP(w, r)
			return
		}

		// Check rate limit
		allowed, err := rl.checkLimit(ctx, tenantID)
		if err != nil {
			// Log error but don't block request
			fmt.Printf("Rate limit check error: %v\n", err)
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			rl.sendError(w, nil, protocol.RateLimitExceeded, "Rate limit exceeded for tenant")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// checkLimit checks if the tenant is within rate limits
func (rl *RateLimiter) checkLimit(ctx context.Context, tenantID string) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s:%d", tenantID, time.Now().Unix()/60)

	// Increment counter
	count, err := rl.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment counter: %w", err)
	}

	// Set expiration on first request
	if count == 1 {
		rl.redis.Expire(ctx, key, rl.window)
	}

	// Check against limit
	return count <= int64(rl.defaultLimit), nil
}

// sendError sends a JSON-RPC error response
func (rl *RateLimiter) sendError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	response := protocol.NewErrorResponse(id, code, message, map[string]interface{}{
		"retry_after": rl.window.Seconds(),
	})
	json.NewEncoder(w).Encode(response)
}
