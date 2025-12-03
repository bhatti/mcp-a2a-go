#!/bin/bash
set -e

echo "🔄 Resetting PostgreSQL database with updated RLS configuration"
echo "================================================================"

# Check if Docker containers are running
if ! docker compose ps | grep -q "mcp-postgres"; then
    echo "❌ PostgreSQL container is not running!"
    echo "   Start services with: docker compose up -d"
    exit 1
fi

echo "⚠️  This will delete all data in the PostgreSQL database!"
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "❌ Operation cancelled"
    exit 0
fi

echo ""
echo "🛑 Stopping services..."
docker compose down

echo ""
echo "🗑️  Removing PostgreSQL volume..."
docker volume rm mcp-a2a-go_postgres_data || true

echo ""
echo "🚀 Starting services with fresh database..."
docker compose up -d postgres

echo ""
echo "⏳ Waiting for PostgreSQL to initialize (this may take 30-60 seconds)..."
sleep 10

until docker compose exec -T postgres pg_isready -U mcp_user -d mcp_db > /dev/null 2>&1; do
    echo "   Still initializing..."
    sleep 5
done

echo ""
echo "✅ PostgreSQL database has been reset with updated RLS configuration!"
echo ""
echo "📋 Verifying RLS setup..."
docker compose exec -T postgres psql -U mcp_user -d mcp_db -c "SELECT relname, relrowsecurity FROM pg_class WHERE relname = 'documents';"
docker compose exec -T postgres psql -U mcp_user -d mcp_db -c "SELECT rolname, rolsuper, rolbypassrls FROM pg_roles WHERE rolname IN ('mcp_user', 'app_user');"

echo ""
echo "📝 Note: Applications should use 'app_user' for proper RLS enforcement"
echo "   - app_user: NOBYPASSRLS (enforces tenant isolation)"
echo "   - mcp_user: SUPERUSER (for administrative tasks only)"
echo ""
echo "🔧 You can now start other services:"
echo "   docker compose up -d"
echo ""
echo "🧪 And run integration tests:"
echo "   ./scripts/run-integration-tests.sh"
