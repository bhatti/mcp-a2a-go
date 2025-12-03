# Testing Guide

This guide provides comprehensive testing instructions for the MCP-A2A system.

## Quick Start

```bash
# 1. Start all services
docker compose up -d

# 2. Wait for services to be ready (30-60 seconds)
docker compose logs -f

# 3. Run unit tests
cd mcp-server && go test ./...

# 4. Run integration tests (requires running PostgreSQL)
./scripts/run-integration-tests.sh
```

## Unit Tests

### MCP Server Unit Tests

All unit tests use mocks and don't require external dependencies:

```bash
cd mcp-server

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test -v ./internal/tools/...
go test -v ./internal/database/...
go test -v ./internal/auth/...

# Run specific test
go test -v ./internal/tools/... -run TestRetrieveToolExecute

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Test Coverage:**
- ✅ Authentication & JWT handling
- ✅ All MCP tools (search, retrieve, list, hybrid_search)
- ✅ Protocol handling (JSON-RPC 2.0)
- ✅ Middleware (auth, rate limiting)
- ✅ Server handlers

## Integration Tests

Integration tests verify the system works correctly against a real PostgreSQL database with pgvector.

### Prerequisites

1. Docker Compose services must be running:
   ```bash
   docker compose up -d
   ```

2. PostgreSQL must be initialized with the schema from `scripts/init-db.sql`

### Running Integration Tests

```bash
# Automated test runner (recommended)
./scripts/run-integration-tests.sh
```

Or manually:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=mcp_user
export DB_PASSWORD=mcp_secure_pass
export DB_NAME=mcp_db
export DB_SSLMODE=disable

cd mcp-server
go test -tags=integration -v ./internal/database/
```

### What Integration Tests Verify

**✅ NULL Embedding Handling**
- Documents without embeddings can be inserted and retrieved
- GetDocument works with NULL embeddings (no scan errors)
- ListDocuments handles mixed embedding states
- SearchDocuments works with NULL embeddings
- VectorSearch only returns documents with embeddings
- HybridSearch handles NULL embeddings gracefully

**✅ Tenant Isolation**
- Row-Level Security (RLS) enforces tenant boundaries
- Cross-tenant access is blocked
- Tenant context is properly set in transactions

**✅ Sample Data Validation**
- All 10 sample documents from init-db.sql can be retrieved
- Documents have correct titles, content, and metadata
- Embedding presence is correctly tracked

**✅ Concurrency**
- Multiple simultaneous retrievals work correctly
- No race conditions or deadlocks
- Connection pool handles concurrent load

## Manual Testing via UI

### 1. Authentication Page

**URL:** http://localhost:8501/Authentication

**Test Steps:**
1. Generate a JWT token with desired tenant/user ID
2. Copy token to clipboard (verify clipboard functionality works)
3. Use token in subsequent API calls

**Expected Result:**
- Token is generated successfully
- Token can be decoded to show claims
- Copy to clipboard works in your browser

### 2. MCP RAG Search

**URL:** http://localhost:8501/MCP_RAG

**Test Cases:**

#### A. List Documents
1. Click "List All Documents"
2. Verify you see 10+ documents
3. Check that document titles, IDs, and metadata are displayed

**Expected:** All sample documents from init-db.sql are listed

#### B. Simple Search
1. Enter query: "security"
2. Click "Search"
3. Verify results include "Q4 Security Policy" document

**Expected:** Text search finds documents with matching keywords

#### C. Hybrid Search
1. Enter query: "machine learning"
2. Click "Hybrid Search"
3. Verify results show BM25 and vector scores
4. Check that results are ranked by combined score

**Expected:** Hybrid results combine text matching with semantic similarity

#### D. Retrieve Document by ID
1. Copy a document ID from the list results
2. Paste into "Document ID" field
3. Click "Retrieve Document"
4. Verify full document details are shown

**Expected:** Document is retrieved with all fields (title, content, metadata, timestamps)

**NOTE:** This was the bug you reported - it should now work without the NULL embedding scan error!

### 3. A2A Task Streaming

**URL:** http://localhost:8501/A2A_Tasks

**Test Steps:**
1. Submit a research task (e.g., "Research machine learning best practices")
2. Click "Start Streaming Events"
3. Watch real-time progress updates

**Expected:**
- Task is created with a task ID
- SSE stream shows status updates
- Progress appears in real-time
- Final result is displayed when complete

**Troubleshooting:**
- If streaming doesn't work, check browser console for SSE errors
- Verify A2A server is running: `docker compose ps a2a-server`
- Check A2A logs: `docker compose logs a2a-server`

### 4. Cost Tracking

**URL:** http://localhost:8501/Cost_Tracking

**Test Steps:**
1. Review budget overview for demo users
2. Check cost distribution charts
3. Click "Export CSV" - verify download works
4. Click "Export JSON" - verify download works
5. Click "Generate Report" - verify text report downloads

**Expected:**
- All three export buttons download files
- CSV contains user budget data
- JSON contains structured data with timestamps
- Report is human-readable text format

## Verifying The GetDocument NULL Embedding Fix

**Bug:** `can't scan into dest[5]: unsupported data type: <nil>`

**Test via UI:**
```
1. Go to http://localhost:8501/MCP_RAG
2. Click "List All Documents"
3. Copy document ID: 8293f7c9-ba23-41b4-9fac-635537eea6a0
4. Paste into "Document ID" field
5. Click "Retrieve Document"
6. Should show document details WITHOUT error
```

**Test via Integration Tests:**
```bash
./scripts/run-integration-tests.sh
```

Look for these passing tests:
- ✅ TestGetDocument_WithNullEmbedding
- ✅ TestGetDocument_WithEmbedding
- ✅ TestGetDocument_FromInitialSampleData

## Test Data

The system comes with 10 sample documents for testing:

1. **Q4 Security Policy** - Security and MFA requirements
2. **Remote Work Guidelines** - Remote work policies
3. **API Design Standards** - RESTful API guidelines
4. **Machine Learning Model Development** - ML best practices
5. **Data Privacy and GDPR Compliance** - Privacy requirements
6. **Microservices Architecture Guidelines** - Architecture patterns
7. **Code Review Best Practices** - Code review process
8. **Database Performance Optimization** - DB tuning
9. **Incident Response Procedures** - Operations procedures
10. **AI Ethics and Responsible AI** - AI ethics guidelines

## Success Criteria

All tests should pass if:

✅ Unit tests: All 24 test cases pass
✅ Integration tests: All 13 database tests pass
✅ UI: All pages load without errors
✅ UI: All export buttons download files
✅ No NULL scan errors in any operation
✅ Document retrieval by ID works for all sample documents

## Troubleshooting

### PostgreSQL Connection Issues

```bash
# Check if PostgreSQL is running
docker compose ps postgres

# Test connection
docker compose exec postgres pg_isready -U mcp_user -d mcp_db

# Verify sample data
docker compose exec postgres psql -U mcp_user -d mcp_db -c "SELECT COUNT(*) FROM documents;"
```

### MCP Server Issues

```bash
# Check MCP server status
docker compose ps mcp-server

# View logs
docker compose logs -f mcp-server

# Rebuild and restart
docker compose down
docker compose build mcp-server
docker compose up -d mcp-server
```
