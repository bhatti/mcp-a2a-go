# Production-Grade MCP & A2A Implementation in Go

A comprehensive, tutorial-focused implementation demonstrating production-quality integration of **Model Context Protocol (MCP)** and **Agent-to-Agent (A2A)** protocols with complete observability, security, and cost control features.

Built with **Go servers** and an interactive **Streamlit UI** for hands-on exploration of all features.

## 🎯 Overview

This repository showcases two production-ready use cases:

1. **Multi-Tenant RAG Pipeline (MCP)**: Secure document search with hybrid search (BM25 + Vector), JWT authentication, rate limiting, and tenant isolation
2. **Cost-Controlled Research Assistant (A2A)**: Budget-aware research agent with task management, real-time streaming, and multi-tier cost tracking

## ✨ Key Features

### 🔐 Security & Multi-Tenancy
- **JWT Authentication**: RS256 tokens with tenant and user claims
- **Multi-tenant Isolation**: Row-Level Security (RLS) in PostgreSQL
- **Rate Limiting**: Redis-backed per-tenant request throttling
- **Scope-based Authorization**: Fine-grained access control

### 🔍 Search & Retrieval
- **Hybrid Search**: BM25 (keyword) + Vector (semantic) with Reciprocal Rank Fusion
- **pgvector**: Efficient similarity search with HNSW indexing
- **Document Management**: Full CRUD operations with tenant isolation
- **Pagination**: Efficient cursor-based pagination for large result sets

### 💰 Cost Control & Budgeting
- **Token Tracking**: Accurate per-request token counting for GPT-4, GPT-3.5, Claude
- **Budget Enforcement**: Pre-flight checks prevent exceeding limits
- **Multi-tier Plans**: Basic ($10), Pro ($50), Enterprise ($200) monthly budgets
- **Cost Attribution**: Per-user and per-task cost tracking

### 📊 Observability & Monitoring
- **Distributed Tracing**: OpenTelemetry + Jaeger for end-to-end request visibility
- **Metrics**: Prometheus-compatible metrics for all operations
- **Health Checks**: Readiness and liveness probes for all services
- **Structured Logging**: JSON logs with trace context propagation

### 🚀 Real-time Streaming
- **Server-Sent Events (SSE)**: Real-time task updates
- **Task Lifecycle**: Pending → Running → Completed/Failed/Cancelled
- **Event Broadcasting**: Pub/sub pattern for task state changes

### 🧪 Test Coverage
- **MCP Server**: 95% average coverage, 125+ unit tests
- **A2A Server**: 92.6% average coverage, 75+ unit tests
- **Integration Tests**: Redis, database, and middleware testing
- **Total**: 200+ passing tests

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Streamlit UI                         │
│  - Authentication & JWT token management                │
│  - MCP RAG testing (hybrid search, documents)           │
│  - A2A task creation & monitoring                       │
│  - Cost tracking & budget visualization                 │
│  - Metrics dashboards (Prometheus)                      │
│  - Distributed tracing (Jaeger)                         │
└─────────────────┬───────────────────────────────────────┘
                  │
      ┌───────────┴───────────┐
      │                       │
┌─────▼────────┐   ┌──────────▼──────────┐
│ MCP Server   │   │   A2A Server        │
│ Port 8080    │   │   Port 8081         │
│              │   │                     │
│ - JSON-RPC   │   │ - REST API          │
│ - JWT Auth   │   │ - Agent Cards       │
│ - Hybrid     │   │ - Task Management   │
│   Search     │   │ - SSE Streaming     │
│ - Rate Limit │   │ - Cost Tracking     │
└─────┬────────┘   └─────────┬───────────┘
      │                      │
  ┌───┴───┬────────┬─────────┴─────┬──────────┐
  │       │        │               │          │
┌─▼──┐ ┌─▼───┐ ┌──▼──────┐  ┌─────▼────┐ ┌──▼──────┐
│PG  │ │Redis│ │ Jaeger  │  │Prometheus│ │ Grafana │
│+pgv│ │     │ │ Tracing │  │ Metrics  │ │Dashboard│
└────┘ └─────┘ └─────────┘  └──────────┘ └─────────┘
```

## 📁 Project Structure

```
.
├── mcp-server/                    # Go MCP server (95% coverage)
│   ├── cmd/server/main.go         # Entry point
│   ├── internal/
│   │   ├── auth/                  # JWT validation (93.1% coverage)
│   │   ├── database/              # PostgreSQL + pgvector
│   │   ├── protocol/              # JSON-RPC 2.0 (100% coverage)
│   │   ├── tools/                 # MCP tools (97.4% coverage)
│   │   ├── middleware/            # Logging, rate limiting (92.9%)
│   │   └── server/                # HTTP server (94.8% coverage)
│   ├── Dockerfile                 # Multi-stage build
│   └── go.mod
│
├── a2a-server/                    # Go A2A server (92.6% coverage)
│   ├── cmd/server/main.go         # Entry point with 3 capabilities
│   ├── internal/
│   │   ├── protocol/              # A2A types (100% coverage)
│   │   ├── agentcard/             # Agent Card store (100% coverage)
│   │   ├── tasks/                 # Task lifecycle (98.3% coverage)
│   │   ├── cost/                  # Cost tracking (91.5% coverage)
│   │   └── server/                # HTTP + SSE server (81.8%)
│   ├── Dockerfile
│   └── go.mod
│
├── streamlit-ui/                  # Interactive testing UI
│   ├── app.py                     # Main dashboard
│   ├── pages/
│   │   ├── 1_🔐_Authentication.py # JWT generation & testing
│   │   ├── 2_📄_MCP_RAG.py        # Hybrid search, documents
│   │   ├── 3_🤖_A2A_Tasks.py      # Task creation & monitoring
│   │   ├── 4_💰_Cost_Tracking.py  # Budget analytics
│   │   ├── 5_📊_Metrics.py        # Prometheus dashboards
│   │   └── 6_🔍_Tracing.py        # Jaeger integration guide
│   ├── utils/
│   │   ├── mcp_client.py          # JSON-RPC 2.0 client
│   │   ├── a2a_client.py          # REST + SSE client
│   │   └── auth.py                # JWT token generation
│   ├── Dockerfile
│   └── requirements.txt
│
├── docker compose.yml             # Complete local stack
├── README.md                      # This file
├── DESIGN.md                      # Detailed architecture (coming next)
└── go.work                        # Go workspace
```

## 🚀 Quick Start

### Prerequisites

- **Docker** & **Docker Compose** (required)
- **Go 1.23+** (for local development)
- **Python 3.11+** (for Streamlit UI development)

### Start Everything with Docker Compose

```bash
# Clone the repository
git clone <repo-url>
cd mcp-a2a-go

# Start all services (PostgreSQL, Redis, Jaeger, Prometheus, MCP, A2A, Streamlit)
docker compose up --build

# Wait for services to be healthy (check logs)
# When you see "Streamlit UI available at http://localhost:8501"
```

**Access the services:**

- **Streamlit UI**: http://localhost:8501 (Interactive testing dashboard)
- **MCP Server**: http://localhost:8080 (JSON-RPC endpoint: /mcp)
- **A2A Server**: http://localhost:8081 (REST API)
- **Jaeger**: http://localhost:16686 (Distributed tracing)
- **Prometheus**: http://localhost:9090 (Metrics)
- **Grafana**: http://localhost:3000 (Dashboards - admin/admin)

### Using the Streamlit UI

The Streamlit UI provides a complete interactive environment for testing all features:

#### 1. **🔐 Authentication Page**
- Generate JWT tokens for 3 demo tenants (acme-corp, globex, initech)
- Configure scopes (read, write, admin)
- Test token validation
- View decoded token claims

#### 2. **📄 MCP RAG Page**
- Initialize MCP session
- List available tools
- Test hybrid search with adjustable BM25/Vector weights
- List documents with pagination
- Retrieve specific documents
- Verify multi-tenant isolation

#### 3. **🤖 A2A Tasks Page**
- View agent card with capabilities
- Create tasks for 3 capabilities:
  - `search_papers`: Search academic papers
  - `analyze_code`: Analyze code repositories
  - `summarize_research`: Summarize research topics
- Select budget tier (Basic $10, Pro $50, Enterprise $200)
- Monitor real-time task progress via SSE
- View task lifecycle and status

#### 4. **💰 Cost Tracking Page**
- View budget overview by tier
- Monitor cost by model (GPT-4, GPT-3.5, Claude)
- Analyze usage timeline
- Track token consumption
- Compare model pricing

#### 5. **📊 Metrics Page**
- View system health metrics
- Request rate and error analysis
- Response time distribution
- Rate limiting status
- Database and cache metrics

#### 6. **🔍 Tracing Page**
- Learn about distributed tracing
- View sample traces
- Understand span hierarchy
- Performance debugging guide
- Link to Jaeger UI

### Manual Testing (Without UI)

#### Test MCP Server

```bash
# Generate a JWT token (you'll need to create a test token)
# See streamlit-ui/utils/auth.py for token generation

# 1. Initialize MCP session
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "clientInfo": {"name": "test-client", "version": "1.0.0"}
    }
  }'

# 2. List tools
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "id": "2",
    "method": "tools/list"
  }'

# 3. Hybrid search
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "id": "3",
    "method": "tools/call",
    "params": {
      "name": "hybrid_search",
      "arguments": {
        "query": "machine learning",
        "limit": 10,
        "bm25_weight": 0.5,
        "vector_weight": 0.5
      }
    }
  }'
```

#### Test A2A Server

```bash
# 1. Get agent card
curl http://localhost:8081/agent

# 2. Create a task
curl -X POST http://localhost:8081/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "demo-user-pro",
    "agent_id": "research-assistant",
    "capability": "search_papers",
    "input": {
      "query": "transformer architecture",
      "limit": 5
    }
  }'

# 3. Get task status
curl http://localhost:8081/tasks/{task_id}

# 4. Stream task events (SSE)
curl -N http://localhost:8081/tasks/{task_id}/events
```

## 🧪 Running Tests

### All Tests

```bash
# From project root
./scripts/run-tests.sh
```

### MCP Server Tests

```bash
cd mcp-server

# All tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Specific package
go test ./internal/auth/...
go test ./internal/tools/...
go test ./internal/middleware/...
```

### A2A Server Tests

```bash
cd a2a-server

# All tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Specific package
go test ./internal/protocol/...
go test ./internal/tasks/...
go test ./internal/cost/...
```

### Test Coverage Summary

| Package | Coverage | Tests |
|---------|----------|-------|
| mcp-server/internal/protocol | 100% | 25+ |
| mcp-server/internal/auth | 93.1% | 20+ |
| mcp-server/internal/tools | 97.4% | 30+ |
| mcp-server/internal/middleware | 92.9% | 25+ |
| mcp-server/internal/server | 94.8% | 25+ |
| **MCP Server Average** | **95%** | **125+** |
| a2a-server/internal/protocol | 100% | 12 |
| a2a-server/internal/agentcard | 100% | 10 |
| a2a-server/internal/tasks | 98.3% | 13 |
| a2a-server/internal/cost | 91.5% | 13 |
| a2a-server/internal/server | 81.8% | 27 |
| **A2A Server Average** | **92.6%** | **75+** |
| **Total** | **94%** | **200+** |

## 📚 Use Case Walkthroughs

### Use Case 1: Multi-Tenant RAG Pipeline

**Scenario**: You have multiple teams (acme-corp, globex, initech) that need isolated document search.

**Steps**:
1. Go to **Authentication** page in Streamlit
2. Generate token for `acme-corp` tenant with `read` and `write` scopes
3. Go to **MCP RAG** page
4. Initialize session with the token
5. Test hybrid search with query: "security policy"
6. Adjust BM25/Vector weights to see different results
7. List documents to see tenant-isolated data
8. Generate token for different tenant and verify isolation

**Key Learnings**:
- JWT claims enforce tenant isolation at database level
- Hybrid search combines keyword and semantic matching
- Rate limiting prevents abuse
- All requests are traced in Jaeger

### Use Case 3: Cost-Controlled Research Assistant

**Scenario**: Research team needs AI assistance with budget constraints.

**Steps**:
1. Go to **A2A Tasks** page in Streamlit
2. View agent card showing 3 capabilities
3. Select budget tier: Basic ($10/month), Pro ($50/month), or Enterprise ($200/month)
4. Create task: "Search papers on transformer architecture"
5. Watch real-time SSE events as task executes
6. Go to **Cost Tracking** page to see budget usage
7. Try creating tasks until budget is exceeded
8. See budget enforcement in action

**Key Learnings**:
- Pre-flight budget checks prevent overspending
- Different models have different costs (GPT-4 vs GPT-3.5)
- SSE provides real-time task updates
- Cost attribution tracks spending by user and task

## 🔧 Local Development (Without Docker)

### Prerequisites
- PostgreSQL 16 with pgvector extension
- Redis 7+
- Go 1.23+

### Setup PostgreSQL

```bash
# Install pgvector extension
psql -U postgres -c "CREATE EXTENSION IF NOT EXISTS vector;"

# Create database
psql -U postgres -c "CREATE DATABASE mcp_dev;"

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=mcp_dev
export DB_SSLMODE=disable
```

### Setup Redis

```bash
# Start Redis
redis-server

# Set environment variable
export REDIS_ADDR=localhost:6379
```

### Run MCP Server

```bash
cd mcp-server

# Install dependencies
go mod download

# Run server
go run cmd/server/main.go

# Server starts on http://localhost:8080
```

### Run A2A Server

```bash
cd a2a-server

# Install dependencies
go mod download

# Run server
go run cmd/server/main.go

# Server starts on http://localhost:8081
```

### Run Streamlit UI

```bash
cd streamlit-ui

# Install dependencies
pip install -r requirements.txt

# Set environment variables
export MCP_SERVER_URL=http://localhost:8080
export A2A_SERVER_URL=http://localhost:8081
export JAEGER_URL=http://localhost:16686
export PROMETHEUS_URL=http://localhost:9090

# Run Streamlit
streamlit run app.py

# UI starts on http://localhost:8501
```

## 🔐 Security Features

### Authentication & Authorization

- **JWT Tokens**: RS256 algorithm with public/private key pairs
- **Token Claims**: `tenant_id`, `user_id`, `scopes`, `exp`, `iat`, `nbf`
- **Scope Validation**: Endpoints require specific scopes (read, write, admin)
- **Token Expiry**: Configurable expiration with automatic validation

### Multi-Tenancy

- **Row-Level Security (RLS)**: PostgreSQL policies enforce tenant isolation
- **Context Propagation**: Tenant ID from JWT flows through all operations
- **Isolated Rate Limits**: Each tenant has separate rate limit counters
- **Data Isolation**: Queries automatically filtered by tenant_id

### Rate Limiting

- **Algorithm**: Token bucket with Redis backend
- **Configuration**: Per-tenant and per-endpoint limits
- **Response**: HTTP 429 with `Retry-After` header
- **Monitoring**: Prometheus metrics for rate limit hits

## 📊 Observability

### Distributed Tracing (Jaeger)

- **Instrumentation**: OpenTelemetry SDK in both servers
- **Trace Propagation**: W3C Trace Context headers
- **Span Attributes**: Custom attributes for tenant_id, user_id, tool names
- **Sampling**: 100% in dev, configurable for production

**View Traces**: http://localhost:16686

### Metrics (Prometheus)

- **HTTP Metrics**: Request count, duration, status codes
- **Database Metrics**: Connection pool, query duration
- **Redis Metrics**: Hit/miss rate, operation latency
- **Custom Metrics**: Tool execution time, token usage

**View Metrics**: http://localhost:9090

### Logs

- **Format**: Structured JSON with trace context
- **Fields**: timestamp, level, msg, trace_id, span_id, tenant_id
- **Correlation**: Logs linked to traces via trace_id

## 💡 Configuration

### Environment Variables

#### MCP Server

```bash
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=mcp_dev
DB_SSLMODE=disable

# Redis
REDIS_ADDR=redis:6379

# Server
MCP_PORT=8080
MCP_LOG_LEVEL=info

# JWT
JWT_PUBLIC_KEY_PATH=/path/to/public.pem
JWT_ISSUER=mcp-server-demo
JWT_AUDIENCE=mcp-server

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60s

# Observability
OTEL_EXPORTER_JAEGER_ENDPOINT=http://jaeger:14268/api/traces
```

#### A2A Server

```bash
# Redis
REDIS_ADDR=redis:6379

# Server
A2A_PORT=8081
A2A_LOG_LEVEL=info

# Cost Limits (monthly budgets in USD)
BUDGET_BASIC=10.0
BUDGET_PRO=50.0
BUDGET_ENTERPRISE=200.0

# Observability
OTEL_EXPORTER_JAEGER_ENDPOINT=http://jaeger:14268/api/traces
```

## 🐛 Troubleshooting

### MCP Server won't start

```bash
# Check database connection
psql -h localhost -U postgres -d mcp_dev -c "SELECT 1;"

# Check pgvector extension
psql -h localhost -U postgres -d mcp_dev -c "SELECT * FROM pg_extension WHERE extname='vector';"

# Check Redis connection
redis-cli ping
```

### JWT tokens rejected

- Verify token hasn't expired (check `exp` claim)
- Ensure `issuer` matches server configuration
- Ensure `audience` matches server configuration
- Check token signature (must be RS256)

### Rate limiting too strict

- Adjust `RATE_LIMIT_REQUESTS` and `RATE_LIMIT_WINDOW`
- Check Redis for stuck counters: `redis-cli KEYS "rate_limit:*"`

### SSE streaming not working

- Verify task ID is correct
- Check CORS headers if accessing from browser
- Ensure connection timeout is long enough
- Check server logs for subscription errors

## 📖 Further Reading

- **DESIGN.md**: Detailed architecture and implementation guide (coming next)
- **MCP Specification**: https://spec.modelcontextprotocol.io/
- **OpenTelemetry Go**: https://opentelemetry.io/docs/languages/go/
- **pgvector**: https://github.com/pgvector/pgvector

## 🤝 Contributing

This is a tutorial project demonstrating production patterns. Contributions welcome!

1. Fork the repository
2. Create a feature branch
3. Add tests (maintain 90%+ coverage)
4. Ensure all tests pass
5. Submit a pull request

## 📝 License

MIT License - see LICENSE file for details

## ✍️ Blog Series

This code accompanies a detailed blog series on production-grade agentic AI:

- **Part 1**: Multi-Tenant RAG with MCP - Authentication, authorization, and hybrid search
- **Part 2**: Cost-Controlled AI with A2A - Budget enforcement and real-time streaming
- **Part 3**: Observability Patterns - Tracing, metrics, and debugging distributed systems
- **Part 4**: Production Deployment - Kubernetes, security hardening, and scaling

## 🙏 Acknowledgments

Built with:
- [Go](https://golang.org/) - Backend services
- [PostgreSQL](https://www.postgresql.org/) + [pgvector](https://github.com/pgvector/pgvector) - Vector database
- [Redis](https://redis.io/) - Rate limiting and caching
- [OpenTelemetry](https://opentelemetry.io/) - Observability
- [Jaeger](https://www.jaegertracing.io/) - Distributed tracing
- [Prometheus](https://prometheus.io/) - Metrics
- [Streamlit](https://streamlit.io/) - Interactive UI
- [Docker](https://www.docker.com/) - Containerization

---

**Built with ❤️ for the agentic AI community. Happy learning!**
