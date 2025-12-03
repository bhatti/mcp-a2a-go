package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/agentcard"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/cost"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/server"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/tasks"
)

const (
	defaultPort = "8081"
	serverName  = "cost-controlled-research-agent"
	serverVersion = "1.0.0"
)

func main() {
	ctx := context.Background()

	// Load configuration
	port := getEnv("PORT", defaultPort)

	log.Println("Initializing A2A Cost-Controlled Research Assistant...")

	// Initialize stores
	taskStore := tasks.NewMemoryStore()
	agentStore := agentcard.NewStore()
	costTracker := cost.NewTracker()
	budgetManager := cost.NewBudgetManager()

	// Create agent card
	agentCard := protocol.NewAgentCard(
		serverName,
		"Cost-Controlled Research Assistant",
		serverVersion,
		"An AI research assistant with cost tracking and budget enforcement",
	)

	// Add capabilities
	agentCard.AddCapability(protocol.Capability{
		Name:        "search_papers",
		Description: "Search academic papers and research documents",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for academic papers",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return",
					"default":     10,
				},
			},
			"required": []string{"query"},
		},
	})

	agentCard.AddCapability(protocol.Capability{
		Name:        "analyze_code",
		Description: "Analyze source code for patterns and issues",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code": map[string]interface{}{
					"type":        "string",
					"description": "Source code to analyze",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Programming language",
				},
			},
			"required": []string{"code"},
		},
	})

	agentCard.AddCapability(protocol.Capability{
		Name:        "summarize_document",
		Description: "Generate concise summaries of research documents",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"document": map[string]interface{}{
					"type":        "string",
					"description": "Document text to summarize",
				},
				"max_length": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum summary length in words",
					"default":     200,
				},
			},
			"required": []string{"document"},
		},
	})

	// Register agent
	if err := agentStore.Register(ctx, agentCard); err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}
	log.Printf("Registered agent: %s v%s", agentCard.Name, agentCard.Version)

	// Set up demo budgets
	setupDemoBudgets(ctx, budgetManager)

	// Create server
	srv := server.NewServer(taskStore, agentStore, costTracker, budgetManager, agentCard)

	// Start task processor for background task execution
	processor := server.NewTaskProcessor(taskStore, 1*time.Second)
	processor.Start(ctx)
	defer processor.Stop()
	log.Println("Task processor initialized")

	// Start server in goroutine
	addr := ":" + port
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Starting A2A server on %s", addr)
		log.Printf("Agent Card available at: http://localhost:%s/agent", port)
		log.Printf("Tasks endpoint: http://localhost:%s/tasks", port)
		log.Printf("Health check: http://localhost:%s/health", port)
		errCh <- srv.Start(addr)
	}()

	// Wait for interrupt signal or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigCh:
		log.Printf("Received signal: %v. Shutting down gracefully...", sig)
	}

	log.Println("A2A server shutdown complete")
}

// setupDemoBudgets configures demo budgets for testing
func setupDemoBudgets(ctx context.Context, manager *cost.BudgetManager) {
	// Demo users with different budget tiers
	budgets := map[string]float64{
		"demo-user-basic":      10.0,  // $10/month
		"demo-user-pro":        50.0,  // $50/month
		"demo-user-enterprise": 200.0, // $200/month
	}

	for userID, limit := range budgets {
		if err := manager.SetBudget(ctx, userID, limit); err != nil {
			log.Printf("Warning: Failed to set budget for %s: %v", userID, err)
		} else {
			log.Printf("Set budget for %s: $%.2f/month", userID, limit)
		}
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
