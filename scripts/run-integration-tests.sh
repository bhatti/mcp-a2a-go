#!/bin/bash -x
set -e

echo "🧪 Running MCP Server Integration Tests"
echo "========================================"

# Check if Docker containers are running
if ! docker compose ps | grep -q "postgres"; then
    echo "❌ PostgreSQL container is not running!"
    echo "   Start services with: docker compose up -d"
    exit 1
fi

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
until docker compose exec -T postgres pg_isready -U mcp_user -d mcp_db > /dev/null 2>&1; do
    sleep 1
done
echo "✅ PostgreSQL is ready"

# Set environment variables for test
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=mcp_user
export DB_PASSWORD=mcp_secure_pass
export DB_NAME=mcp_db
export DB_SSLMODE=disable

echo ""
echo "📊 Running integration tests..."
echo ""

# Run integration tests
cd mcp-server
go test -tags=integration -v ./internal/database/ -run "TestGetDocument|TestListDocuments|TestSearchDocuments|TestHybridSearch|TestVectorSearch|TestTenantIsolation|TestConcurrent"

echo ""
echo "✅ Integration tests completed!"
