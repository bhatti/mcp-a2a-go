#!/bin/bash
#
# Run the MCP server locally

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "${YELLOW}Starting MCP Server...${NC}"
echo ""

# Set environment variables
export PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=mcp_user
export DB_PASSWORD=mcp_password
export DB_NAME=mcp_db
export DB_SSLMODE=disable
export REDIS_ADDR=localhost:6379
export JAEGER_URL=http://localhost:14268/api/traces
export RATE_LIMIT=100

echo "Environment:"
echo "  PORT=$PORT"
echo "  DB_HOST=$DB_HOST:$DB_PORT"
echo "  REDIS=$REDIS_ADDR"
echo "  JAEGER=$JAEGER_URL"
echo ""

cd mcp-server

# Build and run
echo "${YELLOW}Building server...${NC}"
go build -o bin/mcp-server cmd/server/main.go

echo "${GREEN}✓ Build successful${NC}"
echo ""
echo "${YELLOW}Starting server on port $PORT...${NC}"
echo "  MCP endpoint: http://localhost:$PORT/mcp"
echo "  Health check: http://localhost:$PORT/health"
echo ""
echo "Press Ctrl+C to stop"
echo ""

./bin/mcp-server
