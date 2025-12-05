"""
OpenTelemetry Metrics & Monitoring
Real-time Prometheus metrics from MCP and A2A servers
"""
import streamlit as st
import os
import requests
import pandas as pd
import plotly.graph_objects as go
from datetime import datetime, timedelta

st.set_page_config(page_title="OpenTelemetry Metrics", page_icon="📊", layout="wide")

st.title("📊 OpenTelemetry Metrics & Monitoring")

# Get URLs
prometheus_url = os.getenv('PROMETHEUS_URL', 'http://localhost:9090')
mcp_url = os.getenv('MCP_SERVER_URL', 'http://localhost:8080')
a2a_url = os.getenv('A2A_SERVER_URL', 'http://localhost:8081')

# Helper function to query Prometheus
def query_prometheus(query, time_range='5m'):
    """Query Prometheus API"""
    try:
        url = f"{prometheus_url}/api/v1/query"
        response = requests.get(url, params={'query': query}, timeout=5)
        if response.status_code == 200:
            data = response.json()
            if data['status'] == 'success':
                return data['data']['result']
        return None
    except Exception as e:
        st.error(f"Failed to query Prometheus: {e}")
        return None

def query_prometheus_range(query, start, end, step='15s'):
    """Query Prometheus API for range data"""
    try:
        url = f"{prometheus_url}/api/v1/query_range"
        response = requests.get(url, params={
            'query': query,
            'start': start,
            'end': end,
            'step': step
        }, timeout=10)
        if response.status_code == 200:
            data = response.json()
            if data['status'] == 'success':
                return data['data']['result']
        return None
    except Exception as e:
        st.error(f"Failed to query Prometheus range: {e}")
        return None

# Overview Section
st.header("🎯 OpenTelemetry Implementation")

st.info("""
**Dual Observability Strategy:**
- **OpenTelemetry**: Service-to-service tracing, infrastructure metrics
- **Langfuse**: LLM-specific observability (prompts, tokens, costs)

This page shows real-time metrics from the OpenTelemetry Prometheus exporter.
""")

# Quick Access Links
col1, col2, col3, col4 = st.columns(4)
with col1:
    st.markdown(f"[🔗 Prometheus UI]({prometheus_url})")
with col2:
    st.markdown(f"[📊 MCP Metrics]({mcp_url}/metrics)")
with col3:
    st.markdown(f"[📈 A2A Metrics]({a2a_url}/metrics)")
with col4:
    st.markdown("[📚 Testing Guide](../docs/TESTING_OBSERVABILITY.md)")

# System Status
st.header("🚥 Service Health")

col1, col2, col3 = st.columns(3)

# Check if services are up via Prometheus
targets_up = query_prometheus('up{job=~"mcp-server|a2a-server"}')

with col1:
    mcp_status = "🟢 Healthy" if targets_up else "🔴 Unknown"
    st.metric("MCP Server", mcp_status, mcp_url.split('//')[1])

    # Try to get actual metrics
    try:
        resp = requests.get(f"{mcp_url}/metrics", timeout=2)
        if resp.status_code == 200:
            st.success("✅ Metrics endpoint accessible")
        else:
            st.warning(f"⚠️ Metrics returned {resp.status_code}")
    except:
        st.error("❌ Cannot reach metrics endpoint")

with col2:
    a2a_status = "🟢 Healthy" if targets_up else "🔴 Unknown"
    st.metric("A2A Server", a2a_status, a2a_url.split('//')[1])

    try:
        resp = requests.get(f"{a2a_url}/metrics", timeout=2)
        if resp.status_code == 200:
            st.success("✅ Metrics endpoint accessible")
        else:
            st.warning(f"⚠️ Metrics returned {resp.status_code}")
    except:
        st.error("❌ Cannot reach metrics endpoint")

with col3:
    prom_status = "🟢 Healthy" if targets_up is not None else "🔴 Down"
    st.metric("Prometheus", prom_status, prometheus_url.split('//')[1])

    if targets_up is not None:
        st.success("✅ Prometheus API accessible")
    else:
        st.error("❌ Cannot reach Prometheus")

# Request Metrics
st.header("📈 Request Metrics")

tab1, tab2, tab3 = st.tabs(["MCP Server", "A2A Server", "Combined"])

with tab1:
    st.subheader("MCP Server Request Metrics")

    # Request count
    mcp_req_count = query_prometheus('sum(mcp_request_count)')
    if mcp_req_count:
        total = float(mcp_req_count[0]['value'][1])
        st.metric("Total Requests", f"{int(total):,}")
    else:
        st.info("No request data available yet. Make some requests to generate metrics.")

    # Request rate
    st.code("""
    # Prometheus Query Examples

    # Request rate (per second)
    rate(mcp_request_count[1m])

    # Request duration (95th percentile)
    histogram_quantile(0.95, rate(mcp_request_duration_bucket[5m]))

    # Active requests
    mcp_request_active
    """)

with tab2:
    st.subheader("A2A Server Request Metrics")

    # Task count
    a2a_task_count = query_prometheus('sum(a2a_task_count)')
    if a2a_task_count:
        total = float(a2a_task_count[0]['value'][1])
        st.metric("Total Tasks", f"{int(total):,}")
    else:
        st.info("No task data available yet.")

    st.code("""
    # A2A Prometheus Queries

    # Task count by status
    sum by (status) (a2a_task_count)

    # Task duration (95th percentile)
    histogram_quantile(0.95, rate(a2a_task_duration_bucket[5m]))

    # Active tasks
    a2a_task_active
    """)

with tab3:
    st.subheader("Combined Metrics")

    col1, col2 = st.columns(2)
    with col1:
        # Total active requests
        active_result = query_prometheus('sum(mcp_request_active) + sum(a2a_request_active)')
        if active_result:
            active = int(float(active_result[0]['value'][1]))
            st.metric("Active Requests", active)
        else:
            st.metric("Active Requests", "N/A")

    with col2:
        # Error count
        error_result = query_prometheus('sum(mcp_error_count + a2a_error_count)')
        if error_result:
            errors = int(float(error_result[0]['value'][1]))
            st.metric("Total Errors", errors)
        else:
            st.metric("Total Errors", "N/A")

# Tool Execution Metrics (MCP)
st.header("🔧 Tool Execution Metrics")

col1, col2 = st.columns(2)

with col1:
    st.subheader("Tool Execution Count")

    # Query tool execution count by tool name
    tool_count = query_prometheus('sum by (tool_name) (mcp_tool_execution_count)')

    if tool_count:
        tool_data = []
        for result in tool_count:
            tool_name = result['metric'].get('tool_name', 'unknown')
            count = float(result['value'][1])
            tool_data.append({'Tool': tool_name, 'Count': int(count)})

        df = pd.DataFrame(tool_data)
        st.dataframe(df, use_container_width=True)
    else:
        st.info("No tool execution data yet. Call some tools to see metrics.")

with col2:
    st.subheader("Sample Query")
    st.code("""
    # Tool execution time by tool name
    histogram_quantile(0.95,
      sum by (tool_name, le) (
        rate(mcp_tool_execution_duration_bucket[5m])
      )
    )

    # Tool execution count
    sum by (tool_name) (mcp_tool_execution_count)
    """)

# Cost Tracking Metrics (A2A)
st.header("💰 Cost Tracking Metrics")

col1, col2, col3 = st.columns(3)

with col1:
    cost_result = query_prometheus('sum(a2a_cost_total)')
    if cost_result:
        total_cost = float(cost_result[0]['value'][1])
        st.metric("Total Cost", f"${total_cost:.2f}")
    else:
        st.metric("Total Cost", "$0.00")

with col2:
    tokens_result = query_prometheus('sum(a2a_tokens_total)')
    if tokens_result:
        total_tokens = int(float(tokens_result[0]['value'][1]))
        st.metric("Total Tokens", f"{total_tokens:,}")
    else:
        st.metric("Total Tokens", "0")

with col3:
    st.subheader("Cost by Model")
    cost_by_model = query_prometheus('sum by (model) (a2a_cost_total)')

    if cost_by_model:
        for result in cost_by_model:
            model = result['metric'].get('model', 'unknown')
            cost = float(result['value'][1])
            st.metric(model, f"${cost:.2f}")
    else:
        st.info("No cost data available")

# Database Metrics
st.header("🗄️ Database Metrics")

col1, col2 = st.columns(2)

with col1:
    st.subheader("Query Performance")
    st.code("""
    # Database query duration (99th percentile)
    histogram_quantile(0.99,
      rate(mcp_db_query_duration_bucket[5m])
    )

    # Query count by type
    sum by (query_type) (mcp_db_query_count)
    """)

with col2:
    st.subheader("Connection Pool")

    active_conn = query_prometheus('mcp_db_connection_pool_active')
    idle_conn = query_prometheus('mcp_db_connection_pool_idle')

    if active_conn and idle_conn:
        active = int(float(active_conn[0]['value'][1]))
        idle = int(float(idle_conn[0]['value'][1]))
        st.metric("Active Connections", active)
        st.metric("Idle Connections", idle)
    else:
        st.info("Connection pool metrics not available yet")

# Available Metrics Reference
st.header("📋 Available Metrics Reference")

with st.expander("MCP Server Metrics"):
    st.markdown("""
    ### Request Metrics
    - `mcp_request_count` - Total number of requests (labels: method, status)
    - `mcp_request_duration` - Request duration histogram (milliseconds)
    - `mcp_request_active` - Number of active requests (gauge)

    ### Tool Metrics
    - `mcp_tool_execution_count` - Tool execution count (labels: tool_name, status)
    - `mcp_tool_execution_duration` - Tool execution duration histogram (milliseconds)

    ### Database Metrics
    - `mcp_db_query_duration` - Database query duration histogram (milliseconds)
    - `mcp_db_query_count` - Database query count (labels: query_type, status)
    - `mcp_db_connection_pool_active` - Active database connections (gauge)
    - `mcp_db_connection_pool_idle` - Idle database connections (gauge)

    ### Search Metrics
    - `mcp_search_results` - Number of search results histogram
    - `mcp_hybrid_search_score` - Hybrid search relevance scores
    - `mcp_documents_retrieved` - Total documents retrieved counter

    ### Error Metrics
    - `mcp_error_count` - Total error count (labels: error_type, operation)
    """)

with st.expander("A2A Server Metrics"):
    st.markdown("""
    ### Request Metrics
    - `a2a_request_count` - Total number of requests (labels: http_path, http_method, status)
    - `a2a_request_duration` - Request duration histogram (milliseconds)
    - `a2a_request_active` - Number of active requests (gauge)

    ### Task Metrics
    - `a2a_task_count` - Task count (labels: task_type, status)
    - `a2a_task_duration` - Task execution duration histogram (milliseconds)
    - `a2a_task_queue_depth` - Tasks in queue by priority (gauge)
    - `a2a_task_active` - Number of actively executing tasks (gauge)

    ### Cost Metrics
    - `a2a_cost_total` - Total cost in USD (labels: model)
    - `a2a_tokens_total` - Total tokens used (labels: model, token_type)
    - `a2a_budget_remaining` - Remaining budget in USD (labels: budget_tier)
    - `a2a_budget_utilization` - Budget utilization percentage histogram

    ### SSE Metrics
    - `a2a_sse_connections` - Active SSE connections (gauge)
    - `a2a_sse_events_sent` - Total SSE events sent (labels: event_type)

    ### Capability Metrics
    - `a2a_capability_execution_count` - Capability execution count (labels: capability_name, status)
    - `a2a_capability_execution_duration` - Capability execution duration histogram

    ### Error Metrics
    - `a2a_error_count` - Total error count (labels: error_type, operation)
    """)

# Configuration
st.header("⚙️ Configuration")

with st.expander("OpenTelemetry Configuration"):
    st.markdown("""
    ### Environment Variables

    ```bash
    # Enable/disable observability
    OTEL_ENABLE_TRACING=true
    OTEL_ENABLE_METRICS=true

    # OTLP endpoint (Jaeger)
    OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4318

    # Sampling rate (0.0 to 1.0)
    OTEL_TRACES_SAMPLER_ARG=1.0  # 100% sampling

    # Environment
    ENVIRONMENT=development  # or production
    ```

    ### Current Configuration
    """)

    config_data = {
        "Setting": ["Prometheus URL", "MCP Server URL", "A2A Server URL"],
        "Value": [prometheus_url, mcp_url, a2a_url]
    }
    st.table(pd.DataFrame(config_data))

# How to Use
st.header("📖 How to Use This Page")

st.markdown("""
1. **Check Service Health**: Ensure all services show 🟢 Healthy status
2. **Generate Metrics**: Use the MCP RAG or A2A Tasks pages to generate requests
3. **View Real-time Data**: Metrics update automatically as you use the system
4. **Explore Prometheus**: Click the links above to access Prometheus UI for advanced queries
5. **Correlate with Traces**: Use trace IDs to correlate metrics with distributed traces

### Testing Tips

- Make some RAG queries to generate MCP metrics
- Create A2A tasks to see cost tracking metrics
- Check Prometheus UI for historical data
- Use the sample queries provided above

### Next Steps

- 🔍 Visit the **Tracing** page to see distributed traces in Jaeger
- 📊 Open **Prometheus UI** for custom queries and visualizations
- 📈 Set up **Grafana dashboards** for advanced monitoring
""")

# Footer
st.divider()
st.caption(f"Metrics endpoints: {mcp_url}/metrics | {a2a_url}/metrics | Prometheus: {prometheus_url}")
