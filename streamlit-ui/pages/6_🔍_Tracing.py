"""
Distributed Tracing with Jaeger
Visualize request flows across MCP and A2A services
"""
import streamlit as st
import os

st.set_page_config(page_title="Tracing", page_icon="🔍", layout="wide")

st.title("🔍 Distributed Tracing with Jaeger")

# Get Jaeger URL
jaeger_url = os.getenv('JAEGER_URL', 'http://localhost:16686')

st.markdown(f"""
**Jaeger UI**: [{jaeger_url}]({jaeger_url})

OpenTelemetry distributed tracing provides end-to-end visibility across all services.
""")

# What is Distributed Tracing
st.header("🎯 What is Distributed Tracing?")

st.markdown("""
Distributed tracing tracks requests as they flow through multiple services, helping you:
- 🔍 **Debug issues**: Find where requests fail or slow down
- ⚡ **Optimize performance**: Identify bottlenecks
- 📊 **Understand dependencies**: See how services interact
- 🐛 **Root cause analysis**: Trace errors to their origin
""")

# Trace Anatomy
st.header("🧬 Trace Anatomy")

st.markdown("""
```
┌─────────────────────────────────────────────────────────────┐
│ Trace: Search Request (trace_id: abc123)                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Span 1: HTTP POST /mcp (200ms)                            │
│  ├─ Span 2: JWT Validation (5ms)                           │
│  ├─ Span 3: Rate Limit Check (3ms)                         │
│  └─ Span 4: Tool Execution (192ms)                         │
│      ├─ Span 5: BM25 Search (45ms)                         │
│      ├─ Span 6: Vector Search (120ms)                      │
│      │   └─ Span 7: Database Query (118ms)                 │
│      └─ Span 8: Result Fusion (27ms)                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Key Concepts:**
- **Trace**: End-to-end request journey
- **Span**: Individual operation within a trace
- **Parent-Child**: Spans form a tree structure
- **Timing**: Duration of each operation
""")

# Sample Traces
st.header("📋 Sample Traces")

trace_examples = [
    {
        "operation": "MCP Search",
        "service": "mcp-server",
        "duration": "245ms",
        "spans": 8,
        "status": "✅ Success"
    },
    {
        "operation": "A2A Task Creation",
        "service": "a2a-server",
        "duration": "42ms",
        "spans": 4,
        "status": "✅ Success"
    },
    {
        "operation": "MCP Hybrid Search",
        "service": "mcp-server",
        "duration": "523ms",
        "spans": 12,
        "status": "⚠️ Slow"
    },
    {
        "operation": "A2A Budget Check",
        "service": "a2a-server",
        "duration": "15ms",
        "spans": 2,
        "status": "✅ Success"
    }
]

for trace in trace_examples:
    with st.expander(f"{trace['operation']} - {trace['duration']}", expanded=False):
        col1, col2, col3, col4 = st.columns(4)

        with col1:
            st.metric("Service", trace['service'])
        with col2:
            st.metric("Duration", trace['duration'])
        with col3:
            st.metric("Spans", trace['spans'])
        with col4:
            st.write(trace['status'])

        st.markdown("**View in Jaeger**: Click the link below to see the full trace")
        st.code(f"{jaeger_url}/trace/mock-trace-id-{hash(trace['operation'])}", language=None)

# Common Tracing Scenarios
st.header("🎬 Common Tracing Scenarios")

tab1, tab2, tab3 = st.tabs(["Performance Debugging", "Error Analysis", "Dependency Mapping"])

with tab1:
    st.subheader("Performance Debugging")
    st.markdown("""
    **Scenario**: Search request is slow (>1s)

    **Steps**:
    1. Open Jaeger UI
    2. Search for service: `mcp-server`
    3. Filter by operation: `tools/call`
    4. Look for traces >1000ms
    5. Examine span timeline to find bottleneck

    **Common Issues**:
    - 🐌 Database query taking too long
    - 🐌 Vector embedding generation slow
    - 🐌 Network latency to external services
    - 🐌 Inefficient algorithm (O(n²) instead of O(n log n))
    """)

    st.code("""
# Example slow trace structure:
POST /mcp/tools/call (1200ms)
├─ Auth middleware (5ms) ✅
├─ Rate limiter (3ms) ✅
└─ hybrid_search (1192ms) ❌ SLOW
    ├─ BM25 search (45ms) ✅
    └─ Vector search (1147ms) ❌ SLOW
        └─ postgres_query (1145ms) ❌ BOTTLENECK
            # Missing index on vector column!
""", language=None)

with tab2:
    st.subheader("Error Analysis")
    st.markdown("""
    **Scenario**: Requests failing with 500 errors

    **Steps**:
    1. Filter traces by error status
    2. Examine error tags and logs
    3. Trace back to root cause span
    4. Check span attributes for error details

    **Error Information**:
    - Stack traces in span logs
    - Error messages in span tags
    - HTTP status codes
    - Exception types
    """)

    st.code("""
# Example error trace:
POST /tasks (500 Internal Server Error)
├─ Budget check (10ms) ✅
└─ Create task (15ms) ❌ ERROR
    └─ Database insert (12ms) ❌ FAILED
        error.type: "database_error"
        error.message: "duplicate key violation"
        db.statement: "INSERT INTO tasks..."
""", language=None)

with tab3:
    st.subheader("Dependency Mapping")
    st.markdown("""
    **Scenario**: Understanding service dependencies

    **Steps**:
    1. Go to Jaeger → "System Architecture"
    2. View service dependency graph
    3. See request flow between services
    4. Identify critical paths

    **Service Graph**:
    ```
    streamlit-ui
    ├─→ mcp-server
    │   ├─→ postgres
    │   ├─→ redis
    │   └─→ jaeger
    └─→ a2a-server
        ├─→ redis
        └─→ jaeger
    ```
    """)

# Span Attributes
st.header("🏷️ Span Attributes")

st.markdown("""
Spans contain rich metadata for debugging:

**Standard Attributes**:
- `http.method`: GET, POST, etc.
- `http.status_code`: 200, 404, 500, etc.
- `http.url`: Request URL
- `db.system`: Database type (postgres, redis)
- `db.statement`: SQL query or command

**Custom Attributes** (MCP/A2A specific):
- `tenant.id`: Multi-tenant identifier
- `user.id`: User making the request
- `tool.name`: MCP tool being called
- `task.id`: A2A task identifier
- `budget.remaining`: Remaining user budget
""")

# Sampling
st.header("🎲 Trace Sampling")

st.markdown("""
**Why Sample?**
In high-traffic systems, tracing every request is expensive.

**Sampling Strategies**:
1. **Head-based sampling** (configured):
   - Sample X% of requests upfront
   - Current: 100% (demo/dev mode)
   - Production: 1-10%

2. **Tail-based sampling**:
   - Sample after seeing full trace
   - Keep all errors and slow requests
   - Discard routine successes

3. **Adaptive sampling**:
   - Increase sampling when errors occur
   - Decrease during normal operation
""")

col1, col2 = st.columns(2)

with col1:
    st.metric("Current Sampling Rate", "100%", "Dev Mode")
with col2:
    st.metric("Recommended Production", "1-5%", "High traffic")

# Integration with Other Tools
st.header("🔗 Integration")

st.markdown("""
**Jaeger integrates with**:
- **Grafana**: Link traces from metrics dashboards
- **Prometheus**: Alert on slow traces
- **Logs**: Correlate traces with log entries
- **Errors**: Link exceptions to traces

**Context Propagation**:
Traces work by propagating context across service boundaries:
1. Service A adds `trace_id` to headers
2. Service B extracts `trace_id` from headers
3. Both services report spans with same `trace_id`
4. Jaeger reconstructs the full trace
""")

# Best Practices
st.header("✅ Best Practices")

col1, col2 = st.columns(2)

with col1:
    st.markdown("""
    **DO**:
    - ✅ Add meaningful span names
    - ✅ Include relevant attributes
    - ✅ Capture errors in spans
    - ✅ Keep spans focused (one operation)
    - ✅ Use parent-child relationships
    """)

with col2:
    st.markdown("""
    **DON'T**:
    - ❌ Create too many spans (overhead)
    - ❌ Include sensitive data in attributes
    - ❌ Forget to propagate context
    - ❌ Create spans for trivial operations
    - ❌ Leave spans open (memory leak)
    """)

# Quick Links
st.header("🚀 Quick Links")

col1, col2, col3 = st.columns(3)

with col1:
    st.markdown(f"""
    **[Open Jaeger UI]({jaeger_url})**

    View all traces and service dependencies
    """)

with col2:
    st.markdown("""
    **Search Tips**:
    - Service: `mcp-server` or `a2a-server`
    - Operation: `POST /mcp` or `POST /tasks`
    - Tags: `http.status_code=500`
    """)

with col3:
    st.markdown("""
    **Common Queries**:
    - Errors: `error=true`
    - Slow: `duration>1s`
    - Tenant: `tenant.id=<uuid>`
    """)

# Tutorial
st.header("📚 Tutorial: Trace a Request")

st.markdown("""
**Exercise**: Trace a hybrid search request

1. Go to **📄 MCP RAG** page
2. Perform a hybrid search
3. Copy the request time
4. Open [Jaeger UI]({}) (new tab)
5. Search for:
   - Service: `mcp-server`
   - Operation: `POST /mcp`
   - Lookback: Last 15 minutes
6. Find your request (match timestamp)
7. Click to see full trace
8. Examine:
   - Total duration
   - Number of spans
   - Database query time
   - BM25 vs Vector search time

**What to look for**:
- Which takes longer: BM25 or vector search?
- How much time is spent in PostgreSQL?
- What's the overhead of middleware (auth + rate limit)?
""".format(jaeger_url))
