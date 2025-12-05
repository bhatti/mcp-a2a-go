"""
OpenTelemetry Distributed Tracing with Jaeger
End-to-end request visibility across Python and Go services
"""
import streamlit as st
import os

st.set_page_config(page_title="OpenTelemetry Tracing", page_icon="🔍", layout="wide")

st.title("🔍 OpenTelemetry Distributed Tracing")

# Get Jaeger URL
jaeger_url = os.getenv('JAEGER_URL', 'http://localhost:16686')

# Overview
st.header("🎯 Overview")

st.info("""
**Full Stack OpenTelemetry Instrumentation:**
- ✅ Go Servers (MCP & A2A): HTTP middleware, tool execution, database queries
- ✅ Python Workflows: LangGraph nodes, MCP calls, LLM invocations
- ✅ Trace Propagation: W3C Trace Context via HTTP headers
- ✅ End-to-End Traces: Streamlit UI → Python → Go → Database → LLM
""")

# Quick Access
col1, col2 = st.columns(2)
with col1:
    st.markdown(f"### 🔗 Access Jaeger UI")
    st.markdown(f"**URL**: [{jaeger_url}]({jaeger_url})")
    st.markdown("View distributed traces, search by service, analyze performance")

with col2:
    st.markdown(f"### 📚 Documentation")
    st.markdown("[Testing Guide](../docs/TESTING_OBSERVABILITY.md)")
    st.markdown("[README - Observability](../README.md#-observability)")

# What is Distributed Tracing
st.header("📖 What is Distributed Tracing?")

st.markdown("""
Distributed tracing tracks requests as they flow through multiple services, helping you:

- 🔍 **Debug issues**: Find exactly where requests fail or slow down
- ⚡ **Optimize performance**: Identify bottlenecks at code-level granularity
- 📊 **Understand dependencies**: See how services interact in real-time
- 🐛 **Root cause analysis**: Trace errors to their origin across service boundaries
- 💡 **Capacity planning**: Understand resource utilization patterns
""")

# End-to-End Trace Example
st.header("🌐 End-to-End Trace Example")

st.markdown("""
### RAG Query Flow

Here's what happens when you make a RAG query through the Streamlit UI:

```
[User Query: "machine learning"]
    ↓
[Streamlit UI] user.query (1200ms)
    ↓ HTTP + W3C Trace Context
[Python] rag_workflow.execute (1150ms)
    ├─ [Python] mcp.hybrid_search (300ms)
    │   ↓ HTTP + traceparent header
    │   └─ [Go MCP] http.request (280ms)
    │       ├─ [Go MCP] mcp.auth.verify (10ms)
    │       ├─ [Go MCP] mcp.rate_limit.check (5ms)
    │       └─ [Go MCP] mcp.tool.call (250ms)
    │           └─ [Go MCP] mcp.db.hybrid_search (240ms)
    │               ├─ [Go MCP] mcp.db.bm25_search (100ms)
    │               └─ [Go MCP] mcp.db.vector_search (120ms)
    ├─ [Python] llm.generate (800ms)
    │   └─ [Langfuse] Tracks tokens, cost, prompt
    └─ [Python] format.response (50ms)
```

**Key Features:**
- **Single Trace ID**: All spans share the same trace ID across languages
- **Causal Relationships**: Parent-child relationships show call hierarchy
- **Precise Timing**: Understand exactly where time is spent
- **Context Propagation**: User ID, tenant ID flow through all spans
""")

# Trace Anatomy
st.header("🧬 Trace Anatomy")

tab1, tab2, tab3 = st.tabs(["Concepts", "Span Attributes", "Example Trace"])

with tab1:
    st.markdown("""
    ### Core Concepts

    **Trace**: Complete end-to-end request journey
    - Has unique `trace_id` (e.g., `0af7651916cd43dd8448eb211c80319c`)
    - Contains multiple spans in a tree structure
    - Spans from different services all share the same trace ID

    **Span**: Individual operation within a trace
    - Has unique `span_id`
    - Has `parent_id` (except root span)
    - Contains timing, status, and attributes

    **Attributes**: Key-value pairs attached to spans
    - Standard: `http.method`, `http.status_code`, `db.system`
    - Custom: `tenant.id`, `user.id`, `tool.name`
    - Help filter and analyze traces

    **W3C Trace Context**: Standard for propagating trace context
    - Header: `traceparent: 00-{trace_id}-{span_id}-{flags}`
    - Automatically propagated by OpenTelemetry
    - Works across languages and frameworks
    """)

with tab2:
    st.markdown("""
    ### Span Attributes

    #### MCP Server Spans
    ```
    http.request
    ├─ http.method: POST
    ├─ http.url: /mcp
    ├─ http.status_code: 200
    ├─ tenant.id: acme-corp
    └─ user.id: demo-user

    mcp.tool.call
    ├─ tool.name: hybrid_search
    ├─ rpc.method: tools/call
    └─ request.id: 1

    mcp.db.hybrid_search
    ├─ db.system: postgresql
    ├─ db.operation: hybrid_search
    ├─ query.type: hybrid
    ├─ search.type: hybrid
    ├─ result.count: 5
    └─ limit: 5
    ```

    #### A2A Server Spans
    ```
    a2a.task.execute
    ├─ task.id: task-123
    ├─ task.type: search_papers
    ├─ task.priority: normal
    └─ user.id: demo-user

    a2a.budget.check
    ├─ budget.tier: pro
    ├─ budget.remaining: 42.50
    └─ cost.estimated: 0.05
    ```

    #### Python Workflow Spans
    ```
    mcp.hybrid_search
    ├─ query: machine learning
    ├─ top_k: 5
    ├─ user.id: demo-user
    ├─ tenant.id: acme-corp
    └─ results.count: 5

    llm.generate
    ├─ llm.model: gpt-4
    ├─ llm.temperature: 0.7
    ├─ llm.max_tokens: 2000
    └─ response.length: 487
    ```
    """)

with tab3:
    st.markdown("""
    ### Real Trace Example

    This is what you'll see in Jaeger:

    ```
    Trace: rag-workflow: execute
    Duration: 1200ms
    Services: 2 (rag-workflow, mcp-server)
    Spans: 12

    Timeline:
    ────────────────────────────────────────────────────────────
    0ms      300ms    600ms     900ms     1200ms
    │──────────┼─────────┼─────────┼─────────┤
    │rag-workflow: execute                     │ 1200ms
    ├─┤mcp.hybrid_search                │ 300ms
    │ ├┤http.request                 │ 280ms
    │ │├┤mcp.tool.call              │ 250ms
    │ │││mcp.db.hybrid_search     │ 240ms
    │ │││├┤mcp.db.bm25_search   │ 100ms
    │ │││└─┤mcp.db.vector_search│ 120ms
    │ └────────┤llm.generate          │ 800ms
    │          └┤format.response       │ 50ms
    ────────────────────────────────────────────────────────────
    ```

    **Insights from this trace:**
    - Total time: 1200ms
    - MCP search: 300ms (25%)
    - LLM generation: 800ms (67%) ← Main bottleneck
    - Database queries: 240ms total
      - Vector search slower than BM25 (120ms vs 100ms)
    - Formatting negligible: 50ms
    """)

# How to Use Jaeger
st.header("🎮 How to Use Jaeger")

with st.expander("Step-by-Step Guide"):
    st.markdown("""
    ### 1. Access Jaeger UI
    Open [Jaeger UI]({jaeger_url}) in your browser.

    ### 2. Select a Service
    - From the "Service" dropdown, select:
      - `mcp-server` for MCP traces
      - `a2a-server` for A2A traces
      - `rag-workflow` for end-to-end RAG traces

    ### 3. Find Traces
    - Click "Find Traces" to see recent traces
    - Or use filters:
      - **Operation**: Select specific operations (e.g., `mcp.tool.call`)
      - **Tags**: Filter by attributes (e.g., `user.id=demo-user`)
      - **Duration**: Find slow requests (e.g., `>500ms`)

    ### 4. Inspect a Trace
    Click on a trace to see:
    - **Timeline**: Visual representation of spans
    - **Span Details**: Click spans to see attributes
    - **Critical Path**: Highlighted spans on critical path
    - **Service Diagram**: Services involved in the trace

    ### 5. Analyze Performance
    Look for:
    - **Long spans**: Operations taking most time
    - **Error spans**: Spans with error status
    - **Gaps**: Idle time between operations
    - **Parallelization**: Operations that could run in parallel
    """.format(jaeger_url=jaeger_url))

# Common Trace Patterns
st.header("🔍 Common Trace Patterns")

col1, col2 = st.columns(2)

with col1:
    st.markdown("""
    ### Successful MCP Request
    ```
    mcp-server: http.request [200ms]
    ├─ mcp.auth.verify [10ms] ✅
    ├─ mcp.rate_limit.check [5ms] ✅
    └─ mcp.tool.call [185ms] ✅
        └─ mcp.db.hybrid_search [180ms] ✅
    ```
    **Status**: All spans green ✅
    **Total**: 200ms
    """)

    st.markdown("""
    ### Failed Authentication
    ```
    mcp-server: http.request [15ms]
    ├─ mcp.auth.verify [10ms] ❌
    │   Error: "Invalid JWT signature"
    └─ [Request rejected]
    ```
    **Status**: Auth span red ❌
    **Total**: 15ms (fast failure)
    """)

with col2:
    st.markdown("""
    ### Slow Database Query
    ```
    mcp-server: http.request [1200ms]
    ├─ mcp.auth.verify [10ms] ✅
    ├─ mcp.rate_limit.check [5ms] ✅
    └─ mcp.tool.call [1185ms] ⚠️
        └─ mcp.db.hybrid_search [1180ms] ⚠️
            └─ mcp.db.vector_search [1150ms] ⚠️
    ```
    **Issue**: Vector search taking 1150ms
    **Action**: Check database indexes
    """)

    st.markdown("""
    ### Rate Limited Request
    ```
    mcp-server: http.request [8ms]
    ├─ mcp.auth.verify [5ms] ✅
    └─ mcp.rate_limit.check [3ms] ❌
        Error: "Rate limit exceeded: 100/min"
    ```
    **Status**: Rate limit span red ❌
    **Response**: HTTP 429
    """)

# Testing Tips
st.header("🧪 Testing & Debugging Tips")

col1, col2 = st.columns(2)

with col1:
    st.markdown("""
    ### Generate Test Traces

    1. **Make RAG Queries**
       - Go to "MCP RAG" page
       - Enter queries to generate traces
       - Check Jaeger for `rag-workflow` traces

    2. **Create A2A Tasks**
       - Go to "A2A Tasks" page
       - Create tasks to see task lifecycle traces
       - Check Jaeger for `a2a-server` traces

    3. **Use CLI**
       ```bash
       # Make a direct MCP call
       curl -X POST http://localhost:8080/mcp \\
         -H "Content-Type: application/json" \\
         -d '{
           "jsonrpc": "2.0",
           "id": 1,
           "method": "tools/list"
         }'
       ```

    4. **Inject Custom Trace ID**
       ```bash
       # Specify trace ID in header
       curl -X POST http://localhost:8080/mcp \\
         -H "traceparent: 00-trace123-span123-01" \\
         -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
       ```
    """)

with col2:
    st.markdown("""
    ### Debugging Workflows

    1. **Find Slow Requests**
       - In Jaeger, filter by duration: `>500ms`
       - Identify bottleneck spans
       - Optimize slow operations

    2. **Trace Errors**
       - Filter by tags: `error=true`
       - Find error spans (marked red)
       - Check error messages in span logs

    3. **Compare Traces**
       - Select multiple traces
       - Compare timeline and durations
       - Identify performance variations

    4. **Service Dependencies**
       - Use "System Architecture" tab
       - See service call graph
       - Understand dependencies
    """)

# Available Services
st.header("📊 Available Services")

services_data = {
    "Service": ["rag-workflow", "mcp-server", "a2a-server"],
    "Language": ["Python", "Go", "Go"],
    "Operations": [
        "execute, mcp.hybrid_search, llm.generate, format.response",
        "http.request, mcp.tool.call, mcp.db.hybrid_search, mcp.auth.verify",
        "a2a.task.execute, a2a.budget.check, a2a.cost.calculate, a2a.sse.publish"
    ],
    "Port": ["N/A", "8080", "8081"]
}

import pandas as pd
st.table(pd.DataFrame(services_data))

# Troubleshooting
st.header("🔧 Troubleshooting")

with st.expander("No traces appearing in Jaeger"):
    st.markdown("""
    1. **Check if tracing is enabled**:
       ```bash
       docker-compose exec mcp-server env | grep OTEL_ENABLE_TRACING
       # Should show: OTEL_ENABLE_TRACING=true
       ```

    2. **Verify Jaeger is receiving traces**:
       ```bash
       docker-compose logs jaeger | grep -i otlp
       # Should show OTLP receiver logs
       ```

    3. **Check OTLP endpoint configuration**:
       ```bash
       docker-compose exec mcp-server env | grep OTLP
       # Should show: OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4318
       ```

    4. **Test connectivity**:
       ```bash
       docker-compose exec mcp-server ping -c 1 jaeger
       # Should succeed
       ```
    """)

with st.expander("Traces not propagating between services"):
    st.markdown("""
    1. **Check W3C Trace Context headers**:
       - Python clients should inject `traceparent` header
       - Go services should extract trace context
       - Look for "traceparent" in HTTP requests

    2. **Verify middleware order**:
       - Tracing middleware should be outermost
       - Check: Tracing → Auth → Rate Limit → Handler

    3. **Check Python instrumentation**:
       ```python
       from opentelemetry.propagate import inject
       headers = {}
       inject(headers)  # Should add traceparent header
       ```
    """)

# Best Practices
st.header("💡 Best Practices")

st.markdown("""
### For Development
- ✅ Use 100% sampling to see all traces
- ✅ Add descriptive span names
- ✅ Include relevant attributes (user_id, tenant_id)
- ✅ Log errors in spans

### For Production
- ✅ Use sampling (e.g., 10%) to reduce overhead
- ✅ Always sample on errors
- ✅ Monitor trace storage costs
- ✅ Set up alerts on slow traces
- ✅ Use trace IDs in logs for correlation

### For Debugging
- ✅ Search by trace ID from logs
- ✅ Filter by error status
- ✅ Compare slow vs fast traces
- ✅ Look for patterns in failures
""")

# Next Steps
st.header("🚀 Next Steps")

col1, col2, col3 = st.columns(3)

with col1:
    st.markdown("""
    ### 📊 Metrics
    - View **Metrics** page for Prometheus data
    - Correlate metrics with traces
    - Set up alerts
    """)

with col2:
    st.markdown("""
    ### 📈 Grafana
    - Create custom dashboards
    - Visualize trace data
    - Set up service graphs
    """)

with col3:
    st.markdown("""
    ### 📚 Learn More
    - [OpenTelemetry Docs](https://opentelemetry.io/docs/)
    - [Jaeger Docs](https://www.jaegertracing.io/docs/)
    - [Testing Guide](../docs/TESTING_OBSERVABILITY.md)
    """)

# Footer
st.divider()
st.caption(f"Jaeger UI: {jaeger_url} | Tracing enabled via OpenTelemetry")
