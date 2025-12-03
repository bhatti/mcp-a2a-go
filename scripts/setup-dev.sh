#!/bin/bash
#
# Development environment setup script
# Sets up all dependencies and infrastructure for local development

set -e

echo "=== MCP & A2A Development Setup ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
echo "${YELLOW}Checking prerequisites...${NC}"

command -v docker >/dev/null 2>&1 || { echo "Error: docker is required but not installed."; exit 1; }

# Check for docker compose (plugin) or docker compose (standalone)
if docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    echo "Error: docker compose is required but not installed."
    echo "Install with: docker plugin install compose"
    exit 1
fi

command -v go >/dev/null 2>&1 || { echo "Error: Go is required but not installed."; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "Error: Python 3 is required but not installed."; exit 1; }

echo "${GREEN}✓ All prerequisites found${NC}"
echo "Using: $DOCKER_COMPOSE"
echo ""

# Start infrastructure
echo "${YELLOW}Starting infrastructure (Postgres, Redis, Jaeger, Prometheus)...${NC}"
$DOCKER_COMPOSE up -d

echo "Waiting for services to be ready..."
sleep 10

# Check if PostgreSQL is ready
echo "${YELLOW}Checking PostgreSQL connection...${NC}"
until docker exec mcp-postgres pg_isready -U mcp_user -d mcp_db >/dev/null 2>&1; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done
echo "${GREEN}✓ PostgreSQL is ready${NC}"

# Check if Redis is ready
echo "${YELLOW}Checking Redis connection...${NC}"
until docker exec mcp-redis redis-cli ping >/dev/null 2>&1; do
    echo "Waiting for Redis..."
    sleep 2
done
echo "${GREEN}✓ Redis is ready${NC}"

# Download Go dependencies
echo ""
echo "${YELLOW}Downloading Go dependencies...${NC}"
cd mcp-server && go mod download && cd ..
echo "${GREEN}✓ MCP server dependencies downloaded${NC}"

# Setup Python virtual environment
echo ""
echo "${YELLOW}Setting up Python environment...${NC}"
if [ ! -d "orchestration/venv" ]; then
    python3 -m venv orchestration/venv
    echo "${GREEN}✓ Python virtual environment created${NC}"
fi

echo ""
echo "${GREEN}=== Setup Complete! ===${NC}"
echo ""
echo "Infrastructure Status:"
echo "  • PostgreSQL: http://localhost:5432"
echo "  • Redis: http://localhost:6379"
echo "  • Jaeger UI: http://localhost:16686"
echo "  • Prometheus: http://localhost:9090"
echo "  • Grafana: http://localhost:3000 (admin/admin)"
echo ""
echo "Next steps:"
echo "  1. Run MCP server: ./scripts/run-mcp-server.sh"
echo "  2. Run tests: ./scripts/run-tests.sh"
echo "  3. View logs: docker compose logs -f"
echo ""
