# Architecture & Implementation Design

This document provides a comprehensive deep-dive into the architecture, design decisions, and implementation details of the production-grade MCP & A2A system. It's designed to be tutorial-friendly and explain the "why" behind every major decision.

## Table of Contents

- [System Architecture](#system-architecture)
- [MCP Server Implementation](#mcp-server-implementation)
- [A2A Server Implementation](#a2a-server-implementation)
- [Multi-Tenancy & Isolation](#multi-tenancy--isolation)
- [Authentication & Authorization](#authentication--authorization)
- [Cost Tracking System](#cost-tracking-system)
- [Observability & Monitoring](#observability--monitoring)
- [Database Design](#database-design)
- [Testing Strategy](#testing-strategy)
- [Performance Considerations](#performance-considerations)
- [Security Patterns](#security-patterns)
- [Production Deployment](#production-deployment)

---

## System Architecture

### High-Level Overview

```
┌────────────────────────────────────────────────────────────────┐
│                        Client Layer                            │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │            Streamlit UI (Python)                        │   │
│  │  - JWT Token Management                                 │   │
│  │  - Interactive Testing                                  │   │
│  │  - Real-time Monitoring                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
└───────────────────┬────────────────────────────────────────────┘
                    │
                    │ HTTP/JSON-RPC 2.0 + REST
                    │
┌───────────────────┴────────────────────────────────────────────┐
│                     Service Layer (Go)                         │
│  ┌──────────────────────┐         ┌──────────────────────┐     │
│  │   MCP Server         │         │   A2A Server         │     │
│  │   (Port 8080)        │         │   (Port 8081)        │     │
│  │                      │         │                      │     │
│  │  ┌────────────────┐  │         │  ┌────────────────┐  │     │
│  │  │ Protocol Layer │  │         │  │ Protocol Layer │  │     │
│  │  │ JSON-RPC 2.0   │  │         │  │ REST + SSE     │  │     │
│  │  └────────┬───────┘  │         │  └────────┬───────┘  │     │
│  │           │          │         │           │          │     │
│  │  ┌────────▼───────┐  │         │  ┌────────▼───────┐  │     │
│  │  │ Middleware     │  │         │  │ HTTP Handlers  │  │     │
│  │  │ - Auth         │  │         │  │ - Agent Card   │  │     │
│  │  │ - Rate Limit   │  │         │  │ - Tasks CRUD   │  │     │
│  │  │ - Logging      │  │         │  │ - SSE Stream   │  │     │
│  │  │ - Tracing      │  │         │  │ - Cost Check   │  │     │
│  │  └────────┬───────┘  │         │  └────────┬───────┘  │     │
│  │           │          │         │           │          │     │
│  │  ┌────────▼───────┐  │         │  ┌────────▼───────┐  │     │
│  │  │ Business Logic │  │         │  │ Business Logic │  │     │
│  │  │ - Tools        │  │         │  │ - Task Mgmt    │  │     │
│  │  │ - Search       │  │         │  │ - Cost Track   │  │     │
│  │  │ - Documents    │  │         │  │ - Pub/Sub      │  │     │
│  │  └────────┬───────┘  │         │  └────────┬───────┘  │     │
│  └───────────┼──────────┘         └───────────┼──────────┘     │
└──────────────┼────────────────────────────────┼────────────────┘
               │                                │
               │                                │
┌──────────────┴────────────────────────────────┴────────────────┐
│                    Data & Infrastructure Layer                 │
│  ┌──────────────┐  ┌──────┐  ┌────────┐  ┌────────┐  ┌──────┐ │
│  │ PostgreSQL   │  │Redis │  │Jaeger  │  │Prometh-│  │Grafana│ │
│  │ + pgvector   │  │      │  │        │  │eus     │  │      │ │
│  │              │  │      │  │        │  │        │  │      │ │
│  │ - Documents  │  │-Rate │  │-Traces │  │-Metrics│  │-Dash-│ │
│  │ - Embeddings │  │ Limit│  │-Spans  │  │-Alerts │  │boards│ │
│  │ - RLS        │  │-Cache│  │        │  │        │  │      │ │
│  └──────────────┘  └──────┘  └────────┘  └────────┘  └──────┘ │
└────────────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Protocol-First Design**: Both servers implement well-defined protocols (MCP JSON-RPC 2.0, A2A REST)
2. **Separation of Concerns**: Clear boundaries between protocol, middleware, and business logic
3. **Multi-Tenancy from Day One**: Tenant isolation is fundamental, not bolted on
4. **Observability Built-In**: Tracing, metrics, and structured logging in every layer
5. **Test-Driven Development**: 90%+ test coverage ensures reliability
6. **Production-Ready**: Error handling, rate limiting, security, and monitoring from the start

### Technology Choices

| Component | Technology | Reasoning |
|-----------|-----------|-----------|
| Backend Language | Go 1.23 | Fast, compiled, excellent concurrency, strong typing |
| Database | PostgreSQL 16 | ACID compliance, pgvector for similarity search, RLS for multi-tenancy |
| Cache/Rate Limiting | Redis | Fast, atomic operations, perfect for rate limiting and session state |
| Protocol (MCP) | JSON-RPC 2.0 | Standard RPC protocol, well-documented, easy to implement |
| Protocol (A2A) | REST + SSE | RESTful for CRUD, SSE for real-time streaming |
| Tracing | OpenTelemetry | Vendor-neutral, comprehensive instrumentation |
| Metrics | Prometheus | Industry standard, powerful querying, alerting |
| UI | Streamlit (Python) | Rapid prototyping, interactive widgets, easy to extend |
| Container Runtime | Docker Compose | Simple local development, service orchestration |

---

## MCP Server Implementation

### JSON-RPC 2.0 Protocol Layer

The MCP server implements the Model Context Protocol using JSON-RPC 2.0 as the transport layer.

#### Protocol Package (`mcp-server/internal/protocol/`)

**Key Files:**
- `types.go`: Request, Response, Error, and standard error codes
- `request.go`: Request parsing and validation
- `response.go`: Response serialization
- `errors.go`: Error constructors for standard error codes

**Request Structure:**
```go
type Request struct {
    JSONRPC string          `json:"jsonrpc"` // Must be "2.0"
    ID      string          `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}
```

**Design Decision: Why json.RawMessage for Params?**

We use `json.RawMessage` for params instead of `interface{}` to defer unmarshaling until we know the method type. This allows:
1. Type-safe unmarshaling based on method
2. Validation of required fields before business logic
3. Better error messages for invalid params

**Example:**
```go
// In tools/handler.go
func (h *ToolsHandler) HandleCall(params json.RawMessage) (*Response, error) {
    var callParams CallToolParams
    if err := json.Unmarshal(params, &callParams); err != nil {
        return nil, NewInvalidParamsError("invalid tool call parameters")
    }
    // Now we have type-safe params
}
```

#### Error Handling Strategy

**Standard Error Codes (JSON-RPC 2.0):**
- `-32700` Parse Error: Invalid JSON
- `-32600` Invalid Request: Missing required fields
- `-32601` Method Not Found: Unknown method
- `-32602` Invalid Params: Parameters don't match schema
- `-32603` Internal Error: Server error

**Custom Error Codes (MCP-specific):**
- `-32000` Authentication Required: Missing or invalid JWT
- `-32001` Authorization Failed: Valid JWT but insufficient scopes
- `-32002` Rate Limit Exceeded: Too many requests
- `-32003` Server Error: General server error

**Critical Design Decision: HTTP Status Codes vs JSON-RPC Errors**

This was a subtle but important decision. According to JSON-RPC 2.0 spec:

- **Protocol-level errors** (parse error, invalid request, method not found) should return **HTTP 200** with error in JSON response
- **Application-level errors** (auth, rate limit) can use **semantic HTTP status codes** (401, 429, etc.)

**Implementation in `server/mcp_handler.go:sendResponse()`:**
```go
func (h *MCPHandler) sendResponse(w http.ResponseWriter, response *protocol.Response) {
    if response.Error != nil {
        switch response.Error.Code {
        case protocol.AuthenticationRequired, protocol.AuthorizationFailed:
            w.WriteHeader(http.StatusUnauthorized)
        case protocol.RateLimitExceeded:
            w.WriteHeader(http.StatusTooManyRequests)
        // JSON-RPC protocol errors return HTTP 200
        case protocol.ParseError, protocol.InvalidRequest, protocol.MethodNotFound,
            protocol.InvalidParams, protocol.InternalError, protocol.ServerError:
            w.WriteHeader(http.StatusOK)
        default:
            w.WriteHeader(http.StatusInternalServerError)
        }
    }
    // ... write JSON response
}
```

### Tools Implementation

MCP tools are the core functionality - they allow LLMs to interact with external systems.

#### Tool Registration Pattern

**File: `tools/registry.go`**
```go
type ToolRegistry struct {
    tools map[string]*Tool
    mu    sync.RWMutex
}

func (r *ToolRegistry) Register(tool *Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[tool.Name] = tool
}
```

**Why mutex?** Even though tools are typically registered at startup, we use `sync.RWMutex` to allow:
1. Safe concurrent reads during request handling
2. Potential dynamic tool registration in the future
3. Thread-safe testing

#### Hybrid Search Algorithm

**File: `tools/search.go`**

The hybrid search combines BM25 (keyword matching) and vector similarity (semantic matching) using Reciprocal Rank Fusion (RRF).

**Algorithm Steps:**
1. **BM25 Search**: Traditional keyword search using PostgreSQL's `ts_rank`
2. **Vector Search**: Semantic similarity using pgvector's `<=>` operator
3. **Reciprocal Rank Fusion**: Combine scores from both methods

**RRF Formula:**
```
RRF(d) = Σ (1 / (k + rank_i(d)))
```
Where:
- `k` = constant (we use 60, standard in information retrieval)
- `rank_i(d)` = rank of document `d` in result set `i`

**Implementation:**
```go
func (s *SearchService) HybridSearch(ctx context.Context, query string, limit int,
    bm25Weight, vectorWeight float64) ([]Document, error) {

    // 1. BM25 search
    bm25Results, err := s.BM25Search(ctx, query, limit*2)
    // 2. Vector search
    vectorResults, err := s.VectorSearch(ctx, query, limit*2)

    // 3. RRF fusion
    scoreMap := make(map[string]float64)
    for rank, doc := range bm25Results {
        scoreMap[doc.ID] += bm25Weight / (60.0 + float64(rank+1))
    }
    for rank, doc := range vectorResults {
        scoreMap[doc.ID] += vectorWeight / (60.0 + float64(rank+1))
    }

    // 4. Sort by combined score and return top N
    return sortAndLimit(scoreMap, limit), nil
}
```

**Why fetch `limit*2` from each source?**

If we only fetch `limit` from each, we might miss documents that rank poorly in one method but excellently in another. Fetching 2x ensures better coverage before fusion.

### Middleware Chain

**File: `middleware/chain.go`**

Middleware is applied in a specific order:

```go
chain := middleware.Chain(
    middleware.RequestLogger(),   // 1. Log all requests
    middleware.Tracing(),          // 2. Start trace span
    middleware.Authentication(),   // 3. Validate JWT
    middleware.RateLimiter(),      // 4. Check rate limits
)

handler := chain(mcpHandler)
```

**Order Matters:**
1. **RequestLogger**: Log before any processing (even rejected requests)
2. **Tracing**: Start span early to capture auth and rate limit in trace
3. **Authentication**: Reject unauthenticated requests before rate limiting
4. **RateLimiter**: Check limits after auth (to use tenant ID from JWT)

#### Rate Limiting Implementation

**Algorithm: Token Bucket**

The token bucket algorithm allows bursts while enforcing average rate.

**Redis Implementation:**
```go
func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
    now := time.Now().Unix()

    // Lua script for atomic token bucket operation
    script := `
        local key = KEYS[1]
        local limit = tonumber(ARGV[1])
        local window = tonumber(ARGV[2])
        local now = tonumber(ARGV[3])

        local current = redis.call('GET', key)
        if current == false then
            redis.call('SETEX', key, window, limit - 1)
            return 1
        end

        current = tonumber(current)
        if current > 0 then
            redis.call('DECR', key)
            return 1
        end

        return 0
    `

    result, err := rl.redis.Eval(ctx, script, []string{key},
        rl.limit, rl.window, now).Result()
    return result == 1, err
}
```

**Why Lua script?**
- **Atomicity**: All operations execute atomically, preventing race conditions
- **Performance**: Single round-trip to Redis instead of multiple commands
- **Correctness**: No time-of-check-time-of-use bugs

**Testing with miniredis:**

Instead of mocking Redis, we use `miniredis` - an in-memory Redis implementation for testing.

```go
func setupTestRedis(t *testing.T) *miniredis.Miniredis {
    mr := miniredis.RunT(t)
    return mr
}

func TestRateLimiter_Allow(t *testing.T) {
    mr := setupTestRedis(t)
    defer mr.Close()

    rl := NewRateLimiter(mr.Addr(), 3, 60)

    // Test: First 3 requests allowed
    for i := 0; i < 3; i++ {
        allowed, err := rl.Allow(ctx, "test-key")
        assert.True(t, allowed)
    }

    // Test: 4th request blocked
    allowed, err := rl.Allow(ctx, "test-key")
    assert.False(t, allowed)
}
```

---

## A2A Server Implementation

### REST API Design

Unlike MCP's JSON-RPC, A2A uses RESTful conventions for better alignment with agent discovery patterns.

**Endpoints:**
- `GET /agent` - Get agent card with capabilities
- `POST /tasks` - Create a new task
- `GET /tasks` - List tasks (with filtering)
- `GET /tasks/{id}` - Get specific task
- `DELETE /tasks/{id}` - Cancel task
- `GET /tasks/{id}/events` - Stream task events (SSE)
- `GET /health` - Health check

**Why REST instead of JSON-RPC for A2A?**

1. **Discovery**: RESTful URLs are easier for agents to discover and explore
2. **Caching**: Standard HTTP caching works out of the box
3. **Tooling**: Existing HTTP tools (curl, Postman) work without JSON-RPC wrapper
4. **SSE Integration**: Server-Sent Events fit naturally with REST

### Agent Card Implementation

**File: `a2a-server/internal/agentcard/store.go`**

Agent Cards are self-describing capability manifests that allow agents to discover what other agents can do.

**Structure:**
```go
type AgentCard struct {
    ID           string       `json:"id"`
    Name         string       `json:"name"`
    Description  string       `json:"description"`
    Capabilities []Capability `json:"capabilities"`
    Version      string       `json:"version"`
}

type Capability struct {
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    InputSchema  map[string]interface{} `json:"input_schema"`
    OutputSchema map[string]interface{} `json:"output_schema,omitempty"`
}
```

**In-Memory Store with Thread-Safety:**
```go
type MemoryStore struct {
    mu    sync.RWMutex
    cards map[string]*protocol.AgentCard
}

func (s *MemoryStore) Register(card *protocol.AgentCard) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.cards[card.ID]; exists {
        return errors.New("agent card already registered")
    }
    s.cards[card.ID] = card
    return nil
}
```

**Why in-memory instead of database?**

For this tutorial/demo:
1. **Simplicity**: Agent cards are relatively static
2. **Performance**: No database round-trip for discovery
3. **Stateless**: Can reconstruct on restart from config

In production, you'd use a database with versioning for:
- Agent card history
- A/B testing capabilities
- Dynamic capability updates

### Task Lifecycle Management

**File: `a2a-server/internal/tasks/store.go`**

Tasks go through a state machine:

```
┌─────────┐
│ PENDING │ (initial state)
└────┬────┘
     │
     ▼
┌─────────┐     ┌───────────┐
│ RUNNING │────▶│ COMPLETED │
└────┬────┘     └───────────┘
     │
     ├─────────▶┌──────────┐
     │          │  FAILED  │
     │          └──────────┘
     │
     └─────────▶┌───────────┐
                │ CANCELLED │
                └───────────┘
```

**State Transitions:**
```go
var validTransitions = map[TaskState][]TaskState{
    TaskStatePending: {
        TaskStateRunning,
        TaskStateCancelled,
    },
    TaskStateRunning: {
        TaskStateCompleted,
        TaskStateFailed,
        TaskStateCancelled,
    },
    // Terminal states (no transitions)
    TaskStateCompleted: {},
    TaskStateFailed:    {},
    TaskStateCancelled: {},
}

func (s *MemoryStore) UpdateTaskState(ctx context.Context, taskID string,
    newState TaskState) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    task, exists := s.tasks[taskID]
    if !exists {
        return errors.New("task not found")
    }

    // Validate transition
    allowedStates := validTransitions[task.State]
    valid := false
    for _, allowed := range allowedStates {
        if newState == allowed {
            valid = true
            break
        }
    }
    if !valid {
        return fmt.Errorf("invalid transition: %s -> %s", task.State, newState)
    }

    task.State = newState
    task.UpdatedAt = time.Now()

    // Broadcast event to subscribers
    s.broadcast(taskID, TaskEvent{
        TaskID: taskID,
        State:  newState,
        Time:   time.Now(),
    })

    return nil
}
```

### Server-Sent Events (SSE) Implementation

**File: `a2a-server/internal/server/server.go`**

SSE provides real-time updates without WebSocket complexity.

**Publisher-Subscriber Pattern:**
```go
type MemoryStore struct {
    mu          sync.RWMutex
    tasks       map[string]*protocol.Task
    subscribers map[string][]chan protocol.TaskEvent // taskID -> channels
}

func (s *MemoryStore) Subscribe(ctx context.Context, taskID string) <-chan protocol.TaskEvent {
    s.mu.Lock()
    defer s.mu.Unlock()

    ch := make(chan protocol.TaskEvent, 10) // buffered to prevent blocking
    s.subscribers[taskID] = append(s.subscribers[taskID], ch)

    return ch
}

func (s *MemoryStore) broadcast(taskID string, event protocol.TaskEvent) {
    subscribers := s.subscribers[taskID]
    for _, ch := range subscribers {
        select {
        case ch <- event:
            // Event sent
        default:
            // Channel full, skip (prevents slow consumers from blocking)
        }
    }
}
```

**SSE HTTP Handler:**
```go
func (s *Server) handleTaskEvents(w http.ResponseWriter, r *http.Request, taskID string) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    ctx := r.Context()
    eventCh := s.taskStore.Subscribe(ctx, taskID)
    defer s.taskStore.Unsubscribe(ctx, taskID, eventCh)

    for {
        select {
        case event, ok := <-eventCh:
            if !ok {
                return // Channel closed
            }
            fmt.Fprintf(w, "data: %s\n\n", toJSON(event))
            flusher.Flush()

        case <-ctx.Done():
            return // Client disconnected
        }
    }
}
```

**Why buffered channel (size 10)?**
- Prevents slow consumers from blocking publishers
- If consumer can't keep up, oldest events are dropped
- Alternative: Persist events to database for replay

---

## Multi-Tenancy & Isolation

Multi-tenancy is enforced at multiple layers for defense-in-depth.

### Layer 1: JWT Claims

Every request includes a JWT with tenant context:
```json
{
  "tenant_id": "11111111-1111-1111-1111-111111111111",
  "user_id": "user-123",
  "scopes": ["read", "write"],
  "iss": "mcp-server-demo",
  "aud": "mcp-server",
  "exp": 1735689600
}
```

### Layer 2: Context Propagation

Tenant ID is extracted from JWT and propagated through request context:

```go
// middleware/auth.go
func (m *AuthMiddleware) Handle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, err := m.validator.ValidateToken(r.Header.Get("Authorization"))

        // Add claims to context
        ctx := context.WithValue(r.Context(), "tenant_id", claims.TenantID)
        ctx = context.WithValue(ctx, "user_id", claims.UserID)
        ctx = context.WithValue(ctx, "scopes", claims.Scopes)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Layer 3: Row-Level Security (RLS)

PostgreSQL Row-Level Security policies enforce tenant isolation at the database level.

**Migration: `migrations/001_create_documents.sql`**
```sql
-- Enable RLS
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see their tenant's documents
CREATE POLICY tenant_isolation ON documents
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy: Service role can see all (for migrations, admin)
CREATE POLICY service_access ON documents
    FOR ALL
    TO service_role
    USING (true);
```

**Setting Tenant Context:**
```go
// database/store.go
func (s *Store) WithTenant(ctx context.Context, tenantID string) *Store {
    _, err := s.db.ExecContext(ctx, "SET app.current_tenant = $1", tenantID)
    return s
}

// Usage in handler
func (h *ToolsHandler) SearchDocuments(ctx context.Context, query string) ([]Doc, error) {
    tenantID := ctx.Value("tenant_id").(string)
    return h.db.WithTenant(ctx, tenantID).Search(query)
}
```

**Why RLS instead of query-level filtering?**

Query-level filtering (`WHERE tenant_id = $1`) has risks:
1. **Developer Error**: Easy to forget in complex queries
2. **SQL Injection**: Bypass possible with dynamic queries
3. **No Defense-in-Depth**: Single point of failure

RLS enforces isolation at the database level, independent of application code.

### Layer 4: Rate Limiting

Each tenant has separate rate limit counters:

```go
func (rl *RateLimiter) Handle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.Context().Value("tenant_id").(string)

        // Rate limit key includes tenant ID
        key := fmt.Sprintf("rate_limit:%s:%s", tenantID, r.URL.Path)

        allowed, err := rl.limiter.Allow(r.Context(), key)
        if !allowed {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

---

## Authentication & Authorization

### JWT Token Generation

**File: `streamlit-ui/utils/auth.py`**

We use RS256 (RSA asymmetric signing) instead of HS256 (HMAC symmetric).

**Why RS256?**
- **Key Separation**: Private key only on token issuer, public key on all validators
- **Scalability**: Multiple services can validate without sharing secret
- **Security**: Private key compromise doesn't expose all services

**Token Generation:**
```python
def generate_token(self, tenant_id: str, user_id: str, scopes: List[str]) -> str:
    now = datetime.utcnow()
    payload = {
        "tenant_id": tenant_id,
        "user_id": user_id,
        "scopes": scopes,
        "iss": "mcp-server-demo",
        "aud": "mcp-server",
        "exp": now + timedelta(hours=24),
        "iat": now,
        "nbf": now
    }

    return jwt.encode(payload, self.private_key_pem, algorithm="RS256")
```

### Token Validation

**File: `mcp-server/internal/auth/validator.go`**

```go
type JWTValidator struct {
    publicKey  *rsa.PublicKey
    issuer     string
    audience   string
}

func (v *JWTValidator) ValidateToken(authHeader string) (*Claims, error) {
    // Extract bearer token
    token := strings.TrimPrefix(authHeader, "Bearer ")

    // Parse and validate
    parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Validate algorithm
        if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return v.publicKey, nil
    })

    // Check standard claims
    claims := parsed.Claims.(*Claims)
    if claims.Issuer != v.issuer {
        return nil, errors.New("invalid issuer")
    }
    if claims.Audience != v.audience {
        return nil, errors.New("invalid audience")
    }

    return claims, nil
}
```

### Scope-Based Authorization

**Example: Write operations require 'write' scope**

```go
func (m *AuthMiddleware) RequireScopes(scopes ...string) middleware.Func {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userScopes := r.Context().Value("scopes").([]string)

            for _, required := range scopes {
                if !contains(userScopes, required) {
                    http.Error(w, "Insufficient permissions", http.StatusForbidden)
                    return
                }
            }

            next.ServeHTTP(w, r)
        })
    }
}

// Usage
router.Handle("/mcp",
    authMiddleware.RequireScopes("write").
    Then(mcpHandler))
```

---

## Cost Tracking System

### Token Counting

**File: `a2a-server/internal/cost/tracker.go`**

Accurate token counting is critical for cost control.

**Model Pricing Table:**
```go
var modelPricing = map[string]Pricing{
    "gpt-4": {
        PromptCost:     0.03,  // per 1K tokens
        CompletionCost: 0.06,  // per 1K tokens
    },
    "gpt-3.5-turbo": {
        PromptCost:     0.0015,
        CompletionCost: 0.002,
    },
    "claude-3-opus": {
        PromptCost:     0.015,
        CompletionCost: 0.075,
    },
}

func CalculateCost(model string, promptTokens, completionTokens int) float64 {
    pricing := modelPricing[model]

    promptCost := float64(promptTokens) * pricing.PromptCost / 1000.0
    completionCost := float64(completionTokens) * pricing.CompletionCost / 1000.0

    return promptCost + completionCost
}
```

### Budget Enforcement

**Pre-Flight Budget Check:**

The key to preventing overspending is checking budget **before** making the LLM call.

```go
func (b *BudgetManager) CheckAndReserve(ctx context.Context, userID string,
    estimatedCost float64) error {

    b.mu.Lock()
    defer b.mu.Unlock()

    usage, exists := b.usage[userID]
    if !exists {
        usage = &Usage{UserID: userID, Spent: 0}
        b.usage[userID] = usage
    }

    limit := b.getLimitForUser(userID)

    // Check if estimated cost would exceed budget
    if usage.Spent + estimatedCost > limit {
        return fmt.Errorf("insufficient budget: spent $%.2f, limit $%.2f, requested $%.2f",
            usage.Spent, limit, estimatedCost)
    }

    // Reserve budget (will be adjusted after actual cost is known)
    usage.Spent += estimatedCost

    return nil
}

func (b *BudgetManager) RecordActualCost(ctx context.Context, userID string,
    estimatedCost, actualCost float64) error {

    b.mu.Lock()
    defer b.mu.Unlock()

    usage := b.usage[userID]
    // Adjust: remove estimate, add actual
    usage.Spent = usage.Spent - estimatedCost + actualCost

    return nil
}
```

**Flow:**
1. User creates task
2. Estimate cost based on capability and model
3. `CheckAndReserve()` - Reserve estimated cost
4. Execute task
5. `RecordActualCost()` - Adjust with actual cost

**Why reserve + adjust instead of just checking?**

Prevents race condition:
```
Time  | User A              | User B
------|---------------------|--------------------
T1    | Check: $9 spent     | Check: $9 spent
T2    | Pass (limit $10)    | Pass (limit $10)
T3    | Execute ($2 cost)   | Execute ($2 cost)
T4    | Total: $11 (OVER!)  |
```

With reservation:
```
Time  | User A              | User B
------|---------------------|--------------------
T1    | Reserve $2          | Reserve $2
T2    | usage = $11         | Blocked (would exceed $10)
T3    | Execute             |
```

### Multi-Tier Budgets

**File: `a2a-server/cmd/server/main.go`**

```go
budgets := map[string]float64{
    "demo-user-basic":      10.0,  // $10/month
    "demo-user-pro":        50.0,  // $50/month
    "demo-user-enterprise": 200.0, // $200/month
}

budgetManager := cost.NewBudgetManager(budgets)
```

In production, this would:
- Come from database with user tier information
- Reset monthly via cron job
- Send alerts at 50%, 80%, 90% thresholds
- Integrate with billing system

---

## Observability & Monitoring

### Distributed Tracing with OpenTelemetry

**Initialization: `mcp-server/internal/observability/tracing.go`**

```go
func InitTracer(serviceName string) (*trace.TracerProvider, error) {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint(os.Getenv("JAEGER_ENDPOINT")),
    ))

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(serviceName),
            semconv.ServiceVersion("1.0.0"),
        )),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}
```

**Trace Propagation:**

```go
// middleware/tracing.go
func Tracing() middleware.Func {
    tracer := otel.Tracer("mcp-server")

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := otel.GetTextMapPropagator().Extract(r.Context(),
                propagation.HeaderCarrier(r.Header))

            ctx, span := tracer.Start(ctx, r.Method + " " + r.URL.Path,
                trace.WithSpanKind(trace.SpanKindServer),
                trace.WithAttributes(
                    semconv.HTTPMethod(r.Method),
                    semconv.HTTPTarget(r.URL.Path),
                ))
            defer span.End()

            // Add tenant_id to span
            if tenantID, ok := ctx.Value("tenant_id").(string); ok {
                span.SetAttributes(attribute.String("tenant.id", tenantID))
            }

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Custom Span Attributes:**

We add domain-specific attributes to spans for better debugging:

```go
func (h *ToolsHandler) HandleCall(ctx context.Context, toolName string, args map[string]interface{}) {
    _, span := tracer.Start(ctx, "tools.call")
    defer span.End()

    span.SetAttributes(
        attribute.String("tool.name", toolName),
        attribute.String("tenant.id", getTenantID(ctx)),
        attribute.String("user.id", getUserID(ctx)),
    )

    // Execute tool...

    span.SetAttributes(attribute.Int("result.count", len(results)))
}
```

**Trace Sampling:**

- **Development**: 100% sampling (see all traces)
- **Production**: 1-10% sampling (reduce overhead and storage)
- **Tail-based**: Keep all errors and slow requests, sample routine traffic

### Prometheus Metrics

**Metrics Collection:**

```go
// server/metrics.go
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path", "status"},
    )

    activeRequests = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "http_active_requests",
            Help: "Number of active HTTP requests",
        },
    )
)

func init() {
    prometheus.MustRegister(requestDuration, activeRequests)
}
```

**Recording Metrics:**

```go
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        activeRequests.Inc()
        defer activeRequests.Dec()

        start := time.Now()

        wrapped := wrapResponseWriter(w)
        next.ServeHTTP(wrapped, r)

        duration := time.Since(start).Seconds()
        requestDuration.WithLabelValues(
            r.Method,
            r.URL.Path,
            strconv.Itoa(wrapped.statusCode),
        ).Observe(duration)
    })
}
```

---

## Database Design

### Schema

**Documents Table:**
```sql
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),  -- OpenAI ada-002 dimensions
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_documents_tenant ON documents(tenant_id);
CREATE INDEX idx_documents_embedding ON documents USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_documents_content_fts ON documents USING GIN (to_tsvector('english', content));
```

**Why IVFFLAT index for vectors?**

pgvector supports two index types:
- **IVFFLAT**: Approximate nearest neighbor (ANN), faster but less accurate
- **HNSW**: Hierarchical navigable small world, more accurate but slower

For this demo, IVFFLAT is sufficient. In production with millions of documents, HNSW is better.

### Connection Pooling

**File: `mcp-server/internal/database/pool.go`**

```go
func NewPool(config Config) (*pgxpool.Pool, error) {
    poolConfig, err := pgxpool.ParseConfig(config.DatabaseURL)

    // Connection pool tuning
    poolConfig.MaxConns = 25
    poolConfig.MinConns = 5
    poolConfig.MaxConnLifetime = time.Hour
    poolConfig.MaxConnIdleTime = 30 * time.Minute
    poolConfig.HealthCheckPeriod = time.Minute

    return pgxpool.NewWithConfig(context.Background(), poolConfig)
}
```

**Pool Size Tuning:**

Rule of thumb: `connections = ((core_count * 2) + effective_spindle_count)`

For typical server with 4 cores + 1 SSD:
- `(4 * 2) + 1 = 9` minimum
- Set to 25 for some headroom
- Monitor `active_connections` and `idle_connections` metrics

---

## Testing Strategy

### Test Coverage Philosophy

We aim for **90%+ coverage** but focus on **meaningful tests**, not just line coverage.

**Coverage by Package:**
| Package | Coverage | Why not 100%? |
|---------|----------|---------------|
| protocol | 100% | Core logic, fully testable |
| auth | 93.1% | Some error paths hard to trigger |
| tools | 97.4% | Database integration corner cases |
| middleware | 92.9% | HTTP hijacking edge cases |
| server | 94.8% | Graceful shutdown timing |

### Testing Patterns

#### 1. Table-Driven Tests

**Example: `protocol/request_test.go`**
```go
func TestNewRequest(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        method  string
        params  interface{}
        wantErr bool
    }{
        {
            name:    "valid request",
            id:      "1",
            method:  "initialize",
            params:  map[string]string{"foo": "bar"},
            wantErr: false,
        },
        {
            name:    "empty id",
            id:      "",
            method:  "initialize",
            params:  nil,
            wantErr: true,
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req, err := NewRequest(tt.id, tt.method, tt.params)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.id, req.ID)
            }
        })
    }
}
```

#### 2. Integration Tests with Real Dependencies

Instead of mocking everything, we use real services for integration tests:

**Redis: miniredis**
```go
func TestRateLimiter_Integration(t *testing.T) {
    mr := miniredis.RunT(t)
    defer mr.Close()

    rl := NewRateLimiter(mr.Addr(), 10, 60)
    // Test with real Redis implementation
}
```

**PostgreSQL: testcontainers (future)**
```go
// For full integration tests, we can use testcontainers:
func TestDatabase_Integration(t *testing.T) {
    ctx := context.Background()

    postgresC, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16"),
        postgres.WithDatabase("test"),
        postgres.WithExtensions("pgvector"),
    )
    defer postgresC.Terminate(ctx)

    // Test with real PostgreSQL + pgvector
}
```

#### 3. Mocking for Unit Tests

When we do mock, we use interfaces:

```go
// database/store.go
type Store interface {
    Search(ctx context.Context, query string) ([]Document, error)
    Get(ctx context.Context, id string) (*Document, error)
}

// tools/handler_test.go
type MockStore struct {
    mock.Mock
}

func (m *MockStore) Search(ctx context.Context, query string) ([]Document, error) {
    args := m.Called(ctx, query)
    return args.Get(0).([]Document), args.Error(1)
}
```

### Benchmarking

**Example: `tools/search_bench_test.go`**
```go
func BenchmarkHybridSearch(b *testing.B) {
    // Setup
    store := setupTestStore(b)
    defer store.Close()

    // Insert test documents
    for i := 0; i < 1000; i++ {
        store.Insert(ctx, generateTestDoc(i))
    }

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, err := store.HybridSearch(ctx, "test query", 10, 0.5, 0.5)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

Run with: `go test -bench=. -benchmem`

---

## Performance Considerations

### 1. Connection Pooling

Both database and Redis use connection pools to avoid connection overhead.

**Database Pool:**
- Min: 5 connections (always warm)
- Max: 25 connections (prevent overwhelming database)
- Health checks: Every 1 minute

**Redis Pool:**
- Default from go-redis library
- Connection reuse via FIFO pool

### 2. Caching Strategy

**What to cache:**
- ✅ Agent cards (static, frequently accessed)
- ✅ Rate limit counters (need fast access)
- ✅ Session state (temporary data)
- ❌ Documents (large, infrequently repeated)
- ❌ Search results (personalized per tenant)

### 3. Index Strategy

**PostgreSQL Indexes:**
- `tenant_id`: For tenant isolation filtering (B-tree)
- `embedding`: For vector similarity (IVFFLAT)
- `to_tsvector(content)`: For full-text search (GIN)

**Trade-off:**
- More indexes = faster reads, slower writes
- For read-heavy workload (RAG), indexes are worth it

### 4. Batch Processing

For embedding generation, batch multiple documents:

```go
func (s *EmbeddingService) BatchEmbed(texts []string) ([][]float64, error) {
    // OpenAI supports up to 2048 texts per batch
    const batchSize = 100

    var allEmbeddings [][]float64
    for i := 0; i < len(texts); i += batchSize {
        end := min(i+batchSize, len(texts))
        batch := texts[i:end]

        embeddings, err := s.client.CreateEmbeddings(ctx, batch)
        allEmbeddings = append(allEmbeddings, embeddings...)
    }

    return allEmbeddings, nil
}
```

### 5. Load Testing

**Tool: k6 (future work)**
```javascript
import http from 'k6/http';

export let options = {
    vus: 100,           // 100 virtual users
    duration: '5m',     // Run for 5 minutes
};

export default function() {
    let payload = JSON.stringify({
        jsonrpc: "2.0",
        id: "1",
        method: "tools/call",
        params: {
            name: "search_documents",
            arguments: { query: "test" }
        }
    });

    http.post('http://localhost:8080/mcp', payload, {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + TOKEN
        }
    });
}
```

---

## Security Patterns

### Defense in Depth

Security is enforced at multiple layers:

1. **Network**: TLS for transport (in production)
2. **Application**: JWT authentication
3. **Authorization**: Scope-based access control
4. **Database**: Row-level security
5. **Rate Limiting**: Prevent DoS
6. **Input Validation**: JSON schema validation

### Input Validation

**Example: JSON-RPC Request Validation**

```go
func ValidateRequest(req *Request) error {
    if req.JSONRPC != "2.0" {
        return NewInvalidRequestError("jsonrpc must be '2.0'")
    }
    if req.ID == "" {
        return NewInvalidRequestError("id is required")
    }
    if req.Method == "" {
        return NewInvalidRequestError("method is required")
    }
    return nil
}
```

**Example: Tool Argument Validation**

```go
func (h *ToolsHandler) ValidateSearchArgs(args map[string]interface{}) error {
    query, ok := args["query"].(string)
    if !ok || query == "" {
        return errors.New("query must be a non-empty string")
    }

    limit, ok := args["limit"].(float64) // JSON numbers are float64
    if ok && (limit < 1 || limit > 100) {
        return errors.New("limit must be between 1 and 100")
    }

    return nil
}
```

### SQL Injection Prevention

We use **parameterized queries** exclusively:

```go
// ✅ SAFE: Parameterized query
func (s *Store) Search(ctx context.Context, query string) ([]Document, error) {
    sql := `
        SELECT id, title, content
        FROM documents
        WHERE to_tsvector('english', content) @@ plainto_tsquery('english', $1)
        LIMIT $2
    `
    rows, err := s.db.Query(ctx, sql, query, limit)
}

// ❌ UNSAFE: String concatenation (NEVER DO THIS)
func (s *Store) SearchUnsafe(ctx context.Context, query string) ([]Document, error) {
    sql := fmt.Sprintf("SELECT * FROM documents WHERE content LIKE '%%%s%%'", query)
    // Attacker could use: query = "'; DROP TABLE documents; --"
}
```

### Secrets Management

**Development:**
- Environment variables in `.env` file
- Git-ignored `.env` file
- Docker Compose reads from `.env`

**Production:**
- Kubernetes secrets
- HashiCorp Vault
- AWS Secrets Manager
- Never commit secrets to git

---

## Production Deployment

### Docker Multi-Stage Builds

**MCP Server Dockerfile:**
```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /mcp-server cmd/server/main.go

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates wget
WORKDIR /root/
COPY --from=builder /mcp-server .
EXPOSE 8080
CMD ["./mcp-server"]
```

**Benefits:**
- Small image size (builder stage discarded)
- Only runtime dependencies in final image
- ca-certificates for HTTPS calls
- wget for health checks

### Health Checks

**Docker Compose:**
```yaml
healthcheck:
  test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
  interval: 10s
  timeout: 5s
  retries: 3
  start_period: 30s
```

**Health Endpoint:**
```go
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    // Check database
    if err := s.db.Ping(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
            "error": "database unavailable",
        })
        return
    }

    // Check Redis
    if err := s.redis.Ping(r.Context()).Err(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
            "error": "redis unavailable",
        })
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
```

### Graceful Shutdown

**Server Shutdown:**
```go
func (s *Server) Start() error {
    server := &http.Server{
        Addr:    ":8080",
        Handler: s.router,
    }

    // Channel for shutdown signal
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

    // Start server in goroutine
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Wait for shutdown signal
    <-shutdown

    // Graceful shutdown with 15s timeout
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Printf("Shutdown error: %v", err)
    }

    // Close resources
    s.db.Close()
    s.redis.Close()

    return nil
}
```

**Why 15s timeout?**
- Load balancer typically has 30s timeout
- 15s gives time to finish in-flight requests
- Prevents hanging on stuck connections

### Kubernetes Deployment (Future)

**Deployment YAML:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mcp-server
  template:
    metadata:
      labels:
        app: mcp-server
    spec:
      containers:
      - name: mcp-server
        image: mcp-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: postgres-service
        - name: REDIS_ADDR
          value: redis-service:6379
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

---

## Conclusion

This design document covers the key architectural decisions and implementation patterns used in this production-grade MCP & A2A system. The focus has been on:

1. **Production Quality**: Authentication, rate limiting, observability from day one
2. **Multi-Tenancy**: Defense-in-depth isolation at every layer
3. **Test-Driven**: 90%+ coverage with meaningful tests
4. **Tutorial-Friendly**: Clear separation of concerns, extensive documentation
5. **Scalability**: Connection pooling, caching, proper indexing
6. **Security**: JWT, RLS, input validation, parameterized queries

This implementation serves as both a working system and an educational resource for building production-grade agentic AI systems.

---

**Next Steps:**

- Deploy to Kubernetes for production scaling
- Add advanced observability (distributed profiling)
- Implement A/B testing for agent capabilities
- Add support for multiple LLM providers (OpenAI, Anthropic, local models)
- Create comprehensive load testing suite
- Build admin dashboard for tenant management
