# OpenTelemetry Observability - Testing Guide

This guide shows how to manually test and verify the OpenTelemetry observability implementation.

## Prerequisites

1. Start all services:
```bash
docker-compose up -d
```

2. Wait for services to be healthy:
```bash
docker-compose ps
```

## 1. Testing Metrics Endpoints (CLI)

### MCP Server Metrics

```bash
# Check if metrics endpoint is available
curl http://localhost:8080/metrics

# Expected output: Prometheus-formatted metrics including:
# - mcp_request_count
# - mcp_request_duration_bucket
# - mcp_tool_execution_duration_bucket
# - mcp_db_query_duration_bucket
# - mcp_search_results_bucket
```

### A2A Server Metrics

```bash
# Check A2A server metrics
curl http://localhost:8081/metrics

# Expected output: Prometheus-formatted metrics including:
# - a2a_request_count
# - a2a_task_duration_bucket
# - a2a_cost_total
# - a2a_budget_remaining
# - a2a_sse_connections
```

### Generate Some Test Data

```bash
# Get a demo JWT token (from MCP server logs)
docker-compose logs mcp-server | grep "DEMO TOKEN"

# Make a test request to MCP server
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'

# Check metrics again - counters should have incremented
curl http://localhost:8080/metrics | grep mcp_request_count
```

## 2. Testing Prometheus (UI)

### Access Prometheus UI
```bash
# Open in browser
open http://localhost:9090
```

### Verify Targets Are Up

1. Go to **Status → Targets**
2. Verify these targets are **UP**:
   - `mcp-server:8080/metrics`
   - `a2a-server:8081/metrics`
   - `prometheus:9090/metrics`

### Run Sample Queries

In the Prometheus query interface, try these:

```promql
# MCP Server request rate (per second)
rate(mcp_request_count[1m])

# MCP Server request duration (95th percentile)
histogram_quantile(0.95, rate(mcp_request_duration_bucket[5m]))

# A2A Server task count by status
sum by (status) (a2a_task_count)

# A2A Server total cost by model
sum by (model) (a2a_cost_total)

# Active requests across all services
sum(mcp_request_active) + sum(a2a_request_active)
```

## 3. Testing Distributed Tracing (Jaeger UI)

### Access Jaeger UI
```bash
# Open in browser
open http://localhost:16686
```

### Generate Traces from Python

```bash
# Enter the orchestration directory
cd orchestration

# Install dependencies
pip install -r requirements.txt

# Set environment variables
export OTEL_ENABLE_TRACING=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318/v1/traces
export MCP_SERVER_URL=http://localhost:8080

# Run a sample RAG workflow (you'll need to create a test script)
# Or use the Streamlit UI (see next section)
```

### View Traces in Jaeger

1. **Select Service**: Choose `mcp-server`, `a2a-server`, or `rag-workflow`
2. **Find Traces**: Click "Find Traces"
3. **Inspect Trace**: Click on a trace to see the full span tree
4. **Look for**:
   - HTTP spans from Python → Go
   - Tool execution spans
   - Database query spans
   - End-to-end duration

### Example Trace Structure

```
rag-workflow: execute (1200ms)
├─ rag-workflow: mcp.hybrid_search (300ms)
│  └─ mcp-server: http.request (280ms)
│     └─ mcp-server: mcp.tool.call (250ms)
│        └─ mcp-server: mcp.db.hybrid_search (240ms)
├─ rag-workflow: llm.generate (800ms)
└─ rag-workflow: format.response (50ms)
```

## 4. Testing End-to-End Tracing

### Using Streamlit UI

1. **Start Streamlit**:
```bash
# In docker-compose
docker-compose logs -f streamlit

# Or locally
cd streamlit-ui
streamlit run app.py
```

2. **Open UI**: http://localhost:8501

3. **Execute RAG Query**:
   - Go to "MCP RAG" page
   - Enter a query (e.g., "machine learning")
   - Click "Search"
   - Note the response time

4. **View Trace in Jaeger**:
   - Open Jaeger: http://localhost:16686
   - Select service: `rag-workflow`
   - Find the trace with matching timestamp
   - Inspect the full trace tree

### Verify W3C Trace Context Propagation

```bash
# Make a request with trace context
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'

# The trace should appear in Jaeger with the specified trace ID
# Trace ID: 0af7651916cd43dd8448eb211c80319c
```

## 5. Testing Metrics Collection

### Check Prometheus is Scraping

```bash
# Query Prometheus to see latest metrics
curl 'http://localhost:9090/api/v1/query?query=up'

# Should show mcp-server and a2a-server as "1" (up)
```

### Verify Metric Labels

```promql
# In Prometheus UI, query:
mcp_request_count

# Should show labels:
# - method (e.g., "tools/list", "tools/call")
# - status (e.g., "success", "error")
# - service="mcp-server"
```

## 6. Testing Configuration

### Verify Environment Variables

```bash
# Check MCP server config
docker-compose exec mcp-server env | grep OTEL

# Expected output:
# OTEL_ENABLE_TRACING=true
# OTEL_ENABLE_METRICS=true
# OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4318
# OTEL_TRACES_SAMPLER_ARG=1.0
```

### Test with Tracing Disabled

```bash
# Edit docker-compose.yml, set:
# OTEL_ENABLE_TRACING=false

# Restart services
docker-compose restart mcp-server

# Metrics should still work
curl http://localhost:8080/metrics

# But no traces should appear in Jaeger
```

## 7. Load Testing

### Generate Load to See Metrics

```bash
# Install Apache Bench (if needed)
# apt-get install apache2-utils

# Generate 100 requests
for i in {1..100}; do
  curl -s -X POST http://localhost:8080/mcp \
    -H "Content-Type: application/json" \
    -d '{
      "jsonrpc": "2.0",
      "id": '$i',
      "method": "tools/list"
    }' > /dev/null
  echo "Request $i completed"
done

# View metrics
curl http://localhost:8080/metrics | grep mcp_request_count

# View in Prometheus
# Query: rate(mcp_request_count[1m])
# Should show increased request rate
```

## 8. Troubleshooting

### No Metrics Appearing

```bash
# Check if metrics endpoint is accessible
curl -v http://localhost:8080/metrics

# Check Prometheus targets
open http://localhost:9090/targets

# Check Prometheus logs
docker-compose logs prometheus
```

### No Traces in Jaeger

```bash
# Check Jaeger is receiving traces
docker-compose logs jaeger | grep -i otlp

# Verify OTLP endpoint is correct
docker-compose exec mcp-server env | grep OTLP

# Check if services can reach Jaeger
docker-compose exec mcp-server ping -c 1 jaeger
```

### Traces Not Propagating

```bash
# Verify W3C Trace Context headers
# Add logging in mcp_handler.go to see incoming headers

# Check Python client is injecting headers
# Look for "traceparent" header in requests
```

## 9. Example Queries for Different Scenarios

### Performance Analysis

```promql
# Slowest tool executions (95th percentile)
histogram_quantile(0.95, rate(mcp_tool_execution_duration_bucket[5m]))

# Database query performance
histogram_quantile(0.99, rate(mcp_db_query_duration_bucket[5m]))

# Request latency by method
rate(mcp_request_duration_sum[5m]) / rate(mcp_request_duration_count[5m])
```

### Cost Tracking

```promql
# Total cost in last hour
increase(a2a_cost_total[1h])

# Cost per model
sum by (model) (a2a_cost_total)

# Tokens used per model
sum by (model) (a2a_tokens_total)

# Budget remaining per tier
a2a_budget_remaining
```

### Error Tracking

```promql
# Error rate
rate(mcp_error_count[5m])

# Failed requests percentage
rate(mcp_request_count{status="error"}[5m]) / rate(mcp_request_count[5m])

# Failed tasks by type
sum by (task_type) (a2a_task_count{status="error"})
```

## 10. Success Criteria

✅ **Metrics**:
- [ ] `/metrics` endpoints return data
- [ ] Prometheus shows targets as UP
- [ ] Metrics update after requests
- [ ] All expected metric names present

✅ **Tracing**:
- [ ] Jaeger UI accessible
- [ ] Traces appear after requests
- [ ] Trace propagation works (Python → Go)
- [ ] Span attributes include tenant_id, user_id

✅ **Integration**:
- [ ] End-to-end traces visible
- [ ] Trace IDs match across services
- [ ] Metrics correlate with trace data
- [ ] Configuration changes take effect

## Next Steps

1. **Set up Grafana Dashboards**: Import pre-built dashboards for visualization
2. **Configure Alerts**: Set up Prometheus alerting rules
3. **Add More Instrumentation**: Instrument additional code paths
4. **Performance Tuning**: Adjust sampling rates for production

---

For more information, see:
- Jaeger UI: http://localhost:16686
- Prometheus UI: http://localhost:9090
- Grafana UI: http://localhost:3000 (default login: admin/admin)
