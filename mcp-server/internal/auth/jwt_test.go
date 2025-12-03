package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to generate test RSA key pair
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return privateKey, string(publicKeyPEM)
}

func TestNewJWTValidator(t *testing.T) {
	_, publicKeyPEM := generateTestKeyPair(t)

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				PublicKeyPEM: publicKeyPEM,
				Issuer:       "test-issuer",
				Audience:     "test-audience",
			},
			wantErr: false,
		},
		{
			name: "invalid public key PEM",
			config: Config{
				PublicKeyPEM: "invalid pem",
				Issuer:       "test-issuer",
				Audience:     "test-audience",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewJWTValidator(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, validator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validator)
				assert.Equal(t, tt.config.Issuer, validator.issuer)
				assert.Equal(t, tt.config.Audience, validator.audience)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	privateKey, publicKeyPEM := generateTestKeyPair(t)

	validator, err := NewJWTValidator(Config{
		PublicKeyPEM: publicKeyPEM,
		Issuer:       "mcp-server-demo",  // Match GenerateDemoToken
		Audience:     "mcp-server",        // Match GenerateDemoToken
	})
	require.NoError(t, err)

	tests := []struct {
		name       string
		tokenFunc  func() string
		wantErr    bool
		errContains string
		validate   func(t *testing.T, claims *Claims)
	}{
		{
			name: "valid token",
			tokenFunc: func() string {
				token, _ := GenerateDemoToken(
					"tenant-123",
					"user-456",
					[]string{"read", "write"},
					privateKey,
				)
				return token
			},
			wantErr: false,
			validate: func(t *testing.T, claims *Claims) {
				assert.Equal(t, "tenant-123", claims.TenantID)
				assert.Equal(t, "user-456", claims.UserID)
				assert.Contains(t, claims.Scopes, "read")
				assert.Contains(t, claims.Scopes, "write")
			},
		},
		{
			name: "token with Bearer prefix",
			tokenFunc: func() string {
				token, _ := GenerateDemoToken(
					"tenant-123",
					"user-456",
					[]string{"read"},
					privateKey,
				)
				return "Bearer " + token
			},
			wantErr: false,
			validate: func(t *testing.T, claims *Claims) {
				assert.Equal(t, "tenant-123", claims.TenantID)
			},
		},
		{
			name: "expired token",
			tokenFunc: func() string {
				now := time.Now()
				claims := Claims{
					TenantID: "tenant-123",
					UserID:   "user-456",
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    "mcp-server-demo",
						Audience:  jwt.ClaimStrings{"mcp-server"},
						ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)), // Expired 1 hour ago
						IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString(privateKey)
				return tokenString
			},
			wantErr:     true,
			errContains: "expired",
		},
		{
			name: "wrong issuer",
			tokenFunc: func() string {
				now := time.Now()
				claims := Claims{
					TenantID: "tenant-123",
					UserID:   "user-456",
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    "wrong-issuer",
						Audience:  jwt.ClaimStrings{"mcp-server"},
						ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(now),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString(privateKey)
				return tokenString
			},
			wantErr:     true,
			errContains: "invalid issuer",
		},
		{
			name: "wrong audience",
			tokenFunc: func() string {
				now := time.Now()
				claims := Claims{
					TenantID: "tenant-123",
					UserID:   "user-456",
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    "mcp-server-demo",
						Audience:  jwt.ClaimStrings{"wrong-audience"},
						ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(now),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString(privateKey)
				return tokenString
			},
			wantErr:     true,
			errContains: "invalid audience",
		},
		{
			name: "missing tenant ID",
			tokenFunc: func() string {
				now := time.Now()
				claims := Claims{
					TenantID: "", // Missing tenant ID
					UserID:   "user-456",
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    "mcp-server-demo",
						Audience:  jwt.ClaimStrings{"mcp-server"},
						ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(now),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString(privateKey)
				return tokenString
			},
			wantErr:     true,
			errContains: "tenant_id claim is required",
		},
		{
			name: "invalid token format",
			tokenFunc: func() string {
				return "invalid.token.format"
			},
			wantErr:     true,
			errContains: "failed to parse token",
		},
		{
			name: "wrong signing method",
			tokenFunc: func() string {
				// Use HS256 instead of RS256
				claims := Claims{
					TenantID: "tenant-123",
					UserID:   "user-456",
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    "mcp-server-demo",
						Audience:  jwt.ClaimStrings{"mcp-server"},
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte("secret"))
				return tokenString
			},
			wantErr:     true,
			errContains: "unexpected signing method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := tt.tokenFunc()
			claims, err := validator.ValidateToken(tokenString)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				if tt.validate != nil {
					tt.validate(t, claims)
				}
			}
		})
	}
}

func TestExtractTenantID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
		wantErr  bool
	}{
		{
			name:     "valid tenant ID",
			ctx:      context.WithValue(context.Background(), ContextKeyTenantID, "tenant-123"),
			expected: "tenant-123",
			wantErr:  false,
		},
		{
			name:    "missing tenant ID",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name:    "invalid type",
			ctx:     context.WithValue(context.Background(), ContextKeyTenantID, 123),
			wantErr: true,
		},
		{
			name:    "empty tenant ID",
			ctx:     context.WithValue(context.Background(), ContextKeyTenantID, ""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantID, err := ExtractTenantID(tt.ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, tenantID)
			}
		})
	}
}

func TestExtractUserID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
		wantErr  bool
	}{
		{
			name:     "valid user ID",
			ctx:      context.WithValue(context.Background(), ContextKeyUserID, "user-456"),
			expected: "user-456",
			wantErr:  false,
		},
		{
			name:    "missing user ID",
			ctx:     context.Background(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := ExtractUserID(tt.ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, userID)
			}
		})
	}
}

func TestExtractScopes(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected []string
		wantErr  bool
	}{
		{
			name:     "valid scopes",
			ctx:      context.WithValue(context.Background(), ContextKeyScopes, []string{"read", "write"}),
			expected: []string{"read", "write"},
			wantErr:  false,
		},
		{
			name:     "empty scopes",
			ctx:      context.Background(),
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "invalid type",
			ctx:      context.WithValue(context.Background(), ContextKeyScopes, "not-a-slice"),
			expected: []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopes, err := ExtractScopes(tt.ctx)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, scopes)
		})
	}
}

func TestHasScope(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		scope    string
		expected bool
	}{
		{
			name:     "scope exists",
			ctx:      context.WithValue(context.Background(), ContextKeyScopes, []string{"read", "write"}),
			scope:    "read",
			expected: true,
		},
		{
			name:     "scope does not exist",
			ctx:      context.WithValue(context.Background(), ContextKeyScopes, []string{"read"}),
			scope:    "write",
			expected: false,
		},
		{
			name:     "no scopes in context",
			ctx:      context.Background(),
			scope:    "read",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasScope(tt.ctx, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithAuth(t *testing.T) {
	claims := &Claims{
		TenantID: "tenant-123",
		UserID:   "user-456",
		Scopes:   []string{"read", "write"},
	}

	ctx := WithAuth(context.Background(), claims)

	// Verify all values are set correctly
	tenantID, err := ExtractTenantID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "tenant-123", tenantID)

	userID, err := ExtractUserID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "user-456", userID)

	scopes, err := ExtractScopes(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []string{"read", "write"}, scopes)
}

func TestGenerateDemoToken(t *testing.T) {
	privateKey, publicKeyPEM := generateTestKeyPair(t)

	tests := []struct {
		name     string
		tenantID string
		userID   string
		scopes   []string
	}{
		{
			name:     "basic token",
			tenantID: "tenant-123",
			userID:   "user-456",
			scopes:   []string{"read", "write"},
		},
		{
			name:     "token with no scopes",
			tenantID: "tenant-789",
			userID:   "user-101",
			scopes:   []string{},
		},
		{
			name:     "token with many scopes",
			tenantID: "tenant-multi",
			userID:   "user-multi",
			scopes:   []string{"read", "write", "delete", "admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString, err := GenerateDemoToken(tt.tenantID, tt.userID, tt.scopes, privateKey)
			assert.NoError(t, err)
			assert.NotEmpty(t, tokenString)

			// Validate the generated token
			validator, err := NewJWTValidator(Config{
				PublicKeyPEM: publicKeyPEM,
				Issuer:       "mcp-server-demo",
				Audience:     "mcp-server",
			})
			require.NoError(t, err)

			claims, err := validator.ValidateToken(tokenString)
			assert.NoError(t, err)
			assert.Equal(t, tt.tenantID, claims.TenantID)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.ElementsMatch(t, tt.scopes, claims.Scopes)
		})
	}
}

// Benchmark tests
func BenchmarkValidateToken(b *testing.B) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(b, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(b, err)

	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}))

	validator, _ := NewJWTValidator(Config{
		PublicKeyPEM: publicKeyPEM,
		Issuer:       "test-issuer",
		Audience:     "test-audience",
	})

	tokenString, _ := GenerateDemoToken("tenant-123", "user-456", []string{"read"}, privateKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateToken(tokenString)
	}
}

func BenchmarkExtractTenantID(b *testing.B) {
	ctx := context.WithValue(context.Background(), ContextKeyTenantID, "tenant-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractTenantID(ctx)
	}
}

func BenchmarkHasScope(b *testing.B) {
	ctx := context.WithValue(context.Background(), ContextKeyScopes, []string{"read", "write", "delete"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HasScope(ctx, "write")
	}
}
