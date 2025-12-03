package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// ContextKeyTenantID is the context key for tenant ID
	ContextKeyTenantID ContextKey = "tenant_id"
	// ContextKeyUserID is the context key for user ID
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyScopes is the context key for authorization scopes
	ContextKeyScopes ContextKey = "scopes"
)

// Claims represents JWT claims for our MCP server
type Claims struct {
	TenantID string   `json:"tenant_id"`
	UserID   string   `json:"user_id"`
	Email    string   `json:"email,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// JWTValidator validates JWT tokens
type JWTValidator struct {
	publicKey *rsa.PublicKey
	issuer    string
	audience  string
}

// Config holds JWT validator configuration
type Config struct {
	PublicKeyPEM string // RSA public key in PEM format
	Issuer       string // Expected token issuer
	Audience     string // Expected token audience
}

// NewJWTValidator creates a new JWT validator
func NewJWTValidator(cfg Config) (*JWTValidator, error) {
	// Parse RSA public key from PEM
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cfg.PublicKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &JWTValidator{
		publicKey: publicKey,
		issuer:    cfg.Issuer,
		audience:  cfg.Audience,
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (v *JWTValidator) ValidateToken(tokenString string) (*Claims, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate issuer
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, claims.Issuer)
	}

	// Validate audience
	validAudience := false
	for _, aud := range claims.Audience {
		if aud == v.audience {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return nil, fmt.Errorf("invalid audience")
	}

	// Validate expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	// Validate tenant ID is present
	if claims.TenantID == "" {
		return nil, fmt.Errorf("tenant_id claim is required")
	}

	return claims, nil
}

// ExtractTenantID extracts tenant ID from context
func ExtractTenantID(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(ContextKeyTenantID).(string)
	if !ok || tenantID == "" {
		return "", fmt.Errorf("tenant_id not found in context")
	}
	return tenantID, nil
}

// ExtractUserID extracts user ID from context
func ExtractUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("user_id not found in context")
	}
	return userID, nil
}

// ExtractScopes extracts scopes from context
func ExtractScopes(ctx context.Context) ([]string, error) {
	scopes, ok := ctx.Value(ContextKeyScopes).([]string)
	if !ok {
		return []string{}, nil
	}
	return scopes, nil
}

// HasScope checks if a specific scope exists
func HasScope(ctx context.Context, requiredScope string) bool {
	scopes, err := ExtractScopes(ctx)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if scope == requiredScope {
			return true
		}
	}
	return false
}

// WithAuth adds authentication claims to context
func WithAuth(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, ContextKeyTenantID, claims.TenantID)
	ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
	ctx = context.WithValue(ctx, ContextKeyScopes, claims.Scopes)
	return ctx
}

// GenerateDemoToken generates a demo JWT token for testing (DO NOT USE IN PRODUCTION)
// This is useful for local development and testing
func GenerateDemoToken(tenantID, userID string, scopes []string, privateKey *rsa.PrivateKey) (string, error) {
	return GenerateDemoTokenWithExpiry(tenantID, userID, scopes, privateKey, 24*time.Hour)
}

// GenerateDemoTokenWithExpiry generates a JWT token with custom expiry duration (for testing)
func GenerateDemoTokenWithExpiry(tenantID, userID string, scopes []string, privateKey *rsa.PrivateKey, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		TenantID: tenantID,
		UserID:   userID,
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "mcp-server-demo",
			Audience:  jwt.ClaimStrings{"mcp-server"},
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
