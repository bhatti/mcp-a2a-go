package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/middleware"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/observability"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/server"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/tools"
)

const (
	defaultPort      = "8080"
	defaultDBHost    = "localhost"
	defaultDBPort    = 5432
	defaultRedisAddr = "localhost:6379"
	defaultRateLimit = 100 // requests per minute
)

func main() {
	ctx := context.Background()

	// Load configuration from environment
	cfg := loadConfig()

	// Initialize database
	log.Println("Connecting to database...")
	db, err := database.NewDB(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Database connected successfully")

	// Initialize Redis
	log.Println("Connecting to Redis...")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Redis connected successfully")

	// Initialize observability
	log.Println("Setting up OpenTelemetry...")
	telemetry, err := observability.NewTelemetry(ctx, observability.Config{
		ServiceName:    "mcp-server",
		ServiceVersion: "1.0.0",
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		SamplingRate:   cfg.SamplingRate,
		EnableTracing:  cfg.EnableTracing,
		EnableMetrics:  cfg.EnableMetrics,
	})
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down telemetry: %v", err)
		}
	}()
	log.Println("OpenTelemetry initialized successfully")

	// Initialize JWT validator
	log.Println("Setting up authentication...")
	jwtValidator, publicKeyPEM, err := setupAuth()
	if err != nil {
		log.Fatalf("Failed to setup auth: %v", err)
	}
	log.Println("Authentication setup complete")
	log.Printf("Demo Public Key:\n%s", publicKeyPEM)

	// Initialize tool registry
	log.Println("Registering MCP tools...")
	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(tools.NewSearchTool(db))
	toolRegistry.Register(tools.NewRetrieveTool(db))
	toolRegistry.Register(tools.NewListTool(db))
	toolRegistry.Register(tools.NewHybridSearchTool(db))
	log.Printf("Registered %d tools", len(toolRegistry.List()))

	// Create MCP handler with telemetry
	mcpHandler := server.NewMCPHandler(toolRegistry, telemetry)

	// Setup middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtValidator)
	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimit)
	tracingMiddleware := middleware.NewTracingMiddleware(telemetry)

	// Create HTTP server with middleware stack
	mux := http.NewServeMux()

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint for Prometheus (no auth required)
	if cfg.EnableMetrics {
		mux.Handle("/metrics", promhttp.Handler())
		log.Printf("Metrics endpoint: http://localhost:%s/metrics", cfg.Port)
	}

	// MCP endpoint with full middleware stack (tracing -> auth -> rate limiting -> handler)
	mux.Handle("/mcp",
		tracingMiddleware.Handler(
			authMiddleware.OptionalHandler(
				rateLimiter.Handler(mcpHandler),
			),
		),
	)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting MCP server on port %s...", cfg.Port)
		log.Printf("MCP endpoint: http://localhost:%s/mcp", cfg.Port)
		log.Printf("Health check: http://localhost:%s/health", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Config holds application configuration
type Config struct {
	Port          string
	Database      database.Config
	RedisAddr     string
	RateLimit     int
	Environment   string
	OTLPEndpoint  string
	SamplingRate  float64
	EnableTracing bool
	EnableMetrics bool
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	return Config{
		Port: getEnv("PORT", defaultPort),
		Database: database.Config{
			Host:     getEnv("DB_HOST", defaultDBHost),
			Port:     getEnvInt("DB_PORT", defaultDBPort),
			User:     getEnv("DB_USER", "mcp_user"),
			Password: getEnv("DB_PASSWORD", "mcp_password"),
			DBName:   getEnv("DB_NAME", "mcp_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxConns: int32(getEnvInt("DB_MAX_CONNS", 25)),
			MinConns: int32(getEnvInt("DB_MIN_CONNS", 5)),
		},
		RedisAddr:     getEnv("REDIS_ADDR", defaultRedisAddr),
		RateLimit:     getEnvInt("RATE_LIMIT", defaultRateLimit),
		Environment:   getEnv("ENVIRONMENT", "development"),
		OTLPEndpoint:  getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger:4318"),
		SamplingRate:  getEnvFloat("OTEL_TRACES_SAMPLER_ARG", 1.0),
		EnableTracing: getEnvBool("OTEL_ENABLE_TRACING", true),
		EnableMetrics: getEnvBool("OTEL_ENABLE_METRICS", true),
	}
}

// setupAuth sets up authentication with demo keys for development
func setupAuth() (*auth.JWTValidator, string, error) {
	// In production, load keys from secure storage (e.g., vault, k8s secrets)
	// For demo, generate RSA key pair
	log.Println("Generating demo RSA key pair (DO NOT USE IN PRODUCTION)...")

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Export public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Export private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Save keys to shared directory for UI access (demo only!)
	keysDir := getEnv("DEMO_KEYS_DIR", "/tmp/demo-keys")
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		log.Printf("Warning: Failed to create keys directory: %v", err)
	} else {
		// Save public key
		if err := os.WriteFile(keysDir+"/public_key.pem", publicKeyPEM, 0644); err != nil {
			log.Printf("Warning: Failed to save public key: %v", err)
		} else {
			log.Printf("Public key saved to %s/public_key.pem", keysDir)
		}

		// Save private key
		if err := os.WriteFile(keysDir+"/private_key.pem", privateKeyPEM, 0600); err != nil {
			log.Printf("Warning: Failed to save private key: %v", err)
		} else {
			log.Printf("Private key saved to %s/private_key.pem", keysDir)
		}
	}

	// Create JWT validator
	validator, err := auth.NewJWTValidator(auth.Config{
		PublicKeyPEM: string(publicKeyPEM),
		Issuer:       "mcp-server-demo",
		Audience:     "mcp-server",
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create JWT validator: %w", err)
	}

	// Generate a demo token for testing
	demoToken, err := auth.GenerateDemoToken(
		"11111111-1111-1111-1111-111111111111", // acme-corp tenant
		"demo-user",
		[]string{"read", "write"},
		privateKey,
	)
	if err != nil {
		log.Printf("Warning: Failed to generate demo token: %v", err)
	} else {
		log.Printf("\n=== DEMO TOKEN (Valid for 24 hours) ===\n%s\n", demoToken)
		log.Println("Use this token in the Authorization header: Bearer <token>")
		log.Println("=========================================")
	}

	return validator, string(publicKeyPEM), nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvFloat retrieves a float environment variable or returns a default value
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var floatValue float64
		if _, err := fmt.Sscanf(value, "%f", &floatValue); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvBool retrieves a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if value == "true" || value == "1" || value == "yes" {
			return true
		}
		if value == "false" || value == "0" || value == "no" {
			return false
		}
	}
	return defaultValue
}
