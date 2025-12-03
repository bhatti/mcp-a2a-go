package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestAuth creates a test auth validator and generates keys
func setupTestAuth(t *testing.T) (*auth.JWTValidator, *rsa.PrivateKey, string) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Export public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}))

	// Create validator with same issuer/audience as GenerateDemoToken
	validator, err := auth.NewJWTValidator(auth.Config{
		PublicKeyPEM: publicKeyPEM,
		Issuer:       "mcp-server-demo",
		Audience:     "mcp-server",
	})
	require.NoError(t, err)

	return validator, privateKey, publicKeyPEM
}

func TestNewAuthMiddleware(t *testing.T) {
	validator, _, _ := setupTestAuth(t)
	middleware := NewAuthMiddleware(validator)

	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.validator)
	assert.NotNil(t, middleware.allowUnauthenticated)
	assert.True(t, middleware.allowUnauthenticated[protocol.MethodInitialize])
}

func TestAuthMiddleware_Handler_ValidToken(t *testing.T) {
	validator, privateKey, _ := setupTestAuth(t)

	// Generate a valid token
	token, err := auth.GenerateDemoToken("tenant-123", "user-456", []string{"admin"}, privateKey)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(validator)

	// Create test handler that checks context
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify auth context was added
		tenantID, err := auth.ExtractTenantID(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "tenant-123", tenantID)

		userID, err := auth.ExtractUserID(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "user-456", userID)

		w.WriteHeader(http.StatusOK)
	})

	// Create request with Authorization header
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	// Execute
	handler := middleware.Handler(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_Handler_MissingToken(t *testing.T) {
	validator, _, _ := setupTestAuth(t)
	middleware := NewAuthMiddleware(validator)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	})

	// Create request without Authorization header
	req := httptest.NewRequest("POST", "/mcp", nil)
	rr := httptest.NewRecorder()

	// Execute
	handler := middleware.Handler(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify error response
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var response protocol.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.AuthenticationRequired, response.Error.Code)
	assert.Contains(t, response.Error.Message, "Authorization header required")
}

func TestAuthMiddleware_Handler_InvalidToken(t *testing.T) {
	validator, _, _ := setupTestAuth(t)
	middleware := NewAuthMiddleware(validator)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	})

	// Create request with invalid token
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-format")
	rr := httptest.NewRecorder()

	// Execute
	handler := middleware.Handler(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify error response
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var response protocol.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.AuthenticationRequired, response.Error.Code)
	assert.Contains(t, response.Error.Message, "Invalid token")
}

func TestAuthMiddleware_Handler_ExpiredToken(t *testing.T) {
	validator, privateKey, _ := setupTestAuth(t)
	middleware := NewAuthMiddleware(validator)

	// Generate an expired token
	expiredToken, err := auth.GenerateDemoTokenWithExpiry("tenant-123", "user-456", []string{"admin"}, privateKey, -time.Hour)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called for expired token")
	})

	// Create request with expired token
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	rr := httptest.NewRecorder()

	// Execute
	handler := middleware.Handler(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify error response
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_OptionalHandler_ValidToken(t *testing.T) {
	validator, privateKey, _ := setupTestAuth(t)

	// Generate a valid token
	token, err := auth.GenerateDemoToken("tenant-123", "user-456", []string{"admin"}, privateKey)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(validator)

	// Create test handler
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify auth context was added
		tenantID, err := auth.ExtractTenantID(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "tenant-123", tenantID)

		w.WriteHeader(http.StatusOK)
	})

	// Create request with Authorization header
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	// Execute
	handler := middleware.OptionalHandler(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_OptionalHandler_NoToken(t *testing.T) {
	validator, _, _ := setupTestAuth(t)
	middleware := NewAuthMiddleware(validator)

	// Create test handler
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify no auth context
		_, err := auth.ExtractTenantID(r.Context())
		assert.Error(t, err) // Should error since no auth context

		w.WriteHeader(http.StatusOK)
	})

	// Create request without Authorization header
	req := httptest.NewRequest("POST", "/mcp", nil)
	rr := httptest.NewRecorder()

	// Execute
	handler := middleware.OptionalHandler(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify handler was called and succeeded
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_OptionalHandler_InvalidToken(t *testing.T) {
	validator, _, _ := setupTestAuth(t)
	middleware := NewAuthMiddleware(validator)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called for invalid token")
	})

	// Create request with invalid token
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-format")
	rr := httptest.NewRecorder()

	// Execute
	handler := middleware.OptionalHandler(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify error response (invalid token present is an error)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var response protocol.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.AuthenticationRequired, response.Error.Code)
}

func TestWithContext(t *testing.T) {
	// Create context handler
	handlerCalled := false
	contextHandler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		assert.NotNil(t, ctx)
		assert.Equal(t, r.Context(), ctx)
		w.WriteHeader(http.StatusOK)
	}

	// Convert to http.Handler
	httpHandler := WithContext(contextHandler)
	assert.NotNil(t, httpHandler)

	// Create request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute
	httpHandler.ServeHTTP(rr, req)

	// Verify
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// Benchmark tests
func BenchmarkAuthMiddleware_Handler(b *testing.B) {
	validator, privateKey, _ := setupTestAuth(&testing.T{})
	token, _ := auth.GenerateDemoToken("tenant-123", "user-456", []string{"admin"}, privateKey)

	middleware := NewAuthMiddleware(validator)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(testHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
