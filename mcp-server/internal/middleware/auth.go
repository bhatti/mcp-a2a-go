package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
)

// AuthMiddleware validates JWT tokens and adds auth context
type AuthMiddleware struct {
	validator *auth.JWTValidator
	// allowUnauthenticated allows requests without auth for certain methods
	allowUnauthenticated map[string]bool
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(validator *auth.JWTValidator) *AuthMiddleware {
	return &AuthMiddleware{
		validator: validator,
		allowUnauthenticated: map[string]bool{
			protocol.MethodInitialize: true, // Initialize is always allowed
		},
	}
}

// Handler wraps an HTTP handler with authentication
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.sendError(w, nil, protocol.AuthenticationRequired, "Authorization header required")
			return
		}

		// Validate token
		claims, err := m.validator.ValidateToken(authHeader)
		if err != nil {
			m.sendError(w, nil, protocol.AuthenticationRequired, "Invalid token: "+err.Error())
			return
		}

		// Add auth context to request
		ctx := auth.WithAuth(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalHandler wraps an HTTP handler with optional authentication
// Allows unauthenticated access to certain methods (like initialize)
func (m *AuthMiddleware) OptionalHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to extract and validate token if present
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			claims, err := m.validator.ValidateToken(authHeader)
			if err == nil {
				// Valid token - add context
				ctx := auth.WithAuth(r.Context(), claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// Invalid token but present - this is an error
			m.sendError(w, nil, protocol.AuthenticationRequired, "Invalid token: "+err.Error())
			return
		}

		// No token - proceed without auth context
		next.ServeHTTP(w, r)
	})
}

// sendError sends a JSON-RPC error response
func (m *AuthMiddleware) sendError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	response := protocol.NewErrorResponse(id, code, message, nil)
	json.NewEncoder(w).Encode(response)
}

// ContextHandler wraps a context-aware handler
type ContextHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request)

// WithContext converts a ContextHandler to http.Handler
func WithContext(handler ContextHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(r.Context(), w, r)
	})
}
