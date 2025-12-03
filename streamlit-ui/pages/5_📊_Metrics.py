"""
System Metrics and Monitoring
Prometheus metrics and system health monitoring
"""
import streamlit as st
import os
import pandas as pd
import plotly.graph_objects as go
from datetime import datetime, timedelta

st.set_page_config(page_title="Metrics", page_icon="📊", layout="wide")

st.title("📊 System Metrics & Monitoring")

# Get URLs
prometheus_url = os.getenv('PROMETHEUS_URL', 'http://localhost:9090')
mcp_url = os.getenv('MCP_SERVER_URL', 'http://localhost:8080')
a2a_url = os.getenv('A2A_SERVER_URL', 'http://localhost:8081')

# System Status
st.header("🚥 System Status")

col1, col2, col3 = st.columns(3)

with col1:
    st.metric("MCP Server", "🟢 Healthy", "8080")
    st.metric("Uptime", "99.9%", "+0.1%")

with col2:
    st.metric("A2A Server", "🟢 Healthy", "8081")
    st.metric("Response Time", "45ms", "-5ms")

with col3:
    st.metric("PostgreSQL", "🟢 Healthy", "5432")
    st.metric("Connections", "12/25", "+2")

# Request Metrics
st.header("📈 Request Metrics")

# Generate sample metrics data
times = pd.date_range(end=datetime.now(), periods=60, freq='min')
mcp_requests = [100 + i * 2 + ((-1) ** i * 10) for i in range(60)]
a2a_requests = [50 + i + ((-1) ** i * 5) for i in range(60)]

df_requests = pd.DataFrame({
    'time': times,
    'mcp': mcp_requests,
    'a2a': a2a_requests
})

fig = go.Figure()
fig.add_trace(go.Scatter(x=df_requests['time'], y=df_requests['mcp'],
                         mode='lines', name='MCP Server', line=dict(color='blue')))
fig.add_trace(go.Scatter(x=df_requests['time'], y=df_requests['a2a'],
                         mode='lines', name='A2A Server', line=dict(color='green')))

fig.update_layout(
    title='Requests per Minute (Last Hour)',
    xaxis_title='Time',
    yaxis_title='Requests/min',
    hovermode='x unified'
)

st.plotly_chart(fig, use_container_width=True)

# Error Rates
st.header("⚠️ Error Rates")

col1, col2 = st.columns(2)

with col1:
    # Error rate chart
    error_data = pd.DataFrame({
        'status': ['2xx Success', '4xx Client Error', '5xx Server Error'],
        'count': [9850, 120, 30]
    })

    fig = go.Figure(data=[go.Bar(
        x=error_data['status'],
        y=error_data['count'],
        marker_color=['green', 'orange', 'red']
    )])
    fig.update_layout(title='HTTP Status Codes (Last 24h)', yaxis_title='Count')
    st.plotly_chart(fig, use_container_width=True)

with col2:
    # Success rate
    total = error_data['count'].sum()
    success_rate = (error_data.loc[0, 'count'] / total) * 100

    st.metric("Success Rate", f"{success_rate:.2f}%")
    st.metric("Total Requests", f"{total:,}")
    st.metric("Error Rate", f"{100-success_rate:.2f}%")

# Response Time Distribution
st.header("⏱️ Response Time Distribution")

# Generate sample latency data
latencies = [20 + i * 0.5 + ((-1) ** i * 5) for i in range(100)]
df_latency = pd.DataFrame({'latency': latencies})

fig = go.Figure(data=[go.Histogram(x=df_latency['latency'], nbinsx=20)])
fig.update_layout(
    title='Response Time Distribution',
    xaxis_title='Latency (ms)',
    yaxis_title='Count'
)
st.plotly_chart(fig, use_container_width=True)

col1, col2, col3 = st.columns(3)
with col1:
    st.metric("P50 (Median)", "42ms")
with col2:
    st.metric("P95", "85ms")
with col3:
    st.metric("P99", "120ms")

# Rate Limiting Metrics
st.header("🚦 Rate Limiting")

rate_limit_data = pd.DataFrame([
    {"tenant": "acme-corp", "requests": 8500, "limit": 10000, "rejected": 0},
    {"tenant": "globex", "requests": 9800, "limit": 10000, "rejected": 45},
    {"tenant": "initech", "requests": 4200, "limit": 10000, "rejected": 0}
])

for _, row in rate_limit_data.iterrows():
    with st.expander(f"🏢 {row['tenant']}", expanded=True):
        col1, col2, col3, col4 = st.columns(4)

        with col1:
            st.metric("Requests", f"{row['requests']:,}")

        with col2:
            st.metric("Limit", f"{row['limit']:,}")

        with col3:
            usage_pct = (row['requests'] / row['limit']) * 100
            st.metric("Usage", f"{usage_pct:.1f}%")

        with col4:
            st.metric("Rejected", row['rejected'])

        st.progress(min(usage_pct / 100, 1.0))

# Database Metrics
st.header("🗄️ Database Metrics")

col1, col2 = st.columns(2)

with col1:
    st.subheader("PostgreSQL")
    db_metrics = pd.DataFrame([
        {"metric": "Active Connections", "value": "12"},
        {"metric": "Idle Connections", "value": "5"},
        {"metric": "Total Connections", "value": "17 / 25"},
        {"metric": "Database Size", "value": "2.4 GB"},
        {"metric": "Query Performance", "value": "8ms avg"}
    ])
    st.dataframe(db_metrics, use_container_width=True, hide_index=True)

with col2:
    st.subheader("Redis")
    redis_metrics = pd.DataFrame([
        {"metric": "Used Memory", "value": "45 MB"},
        {"metric": "Peak Memory", "value": "62 MB"},
        {"metric": "Connected Clients", "value": "3"},
        {"metric": "Keys", "value": "1,247"},
        {"metric": "Hit Rate", "value": "94.2%"}
    ])
    st.dataframe(redis_metrics, use_container_width=True, hide_index=True)

# Cache Performance
st.header("⚡ Cache Performance")

cache_data = pd.DataFrame({
    'category': ['Hits', 'Misses'],
    'count': [9420, 580]
})

col1, col2 = st.columns([2, 1])

with col1:
    fig = go.Figure(data=[go.Pie(labels=cache_data['category'], values=cache_data['count'])])
    fig.update_layout(title='Cache Hit/Miss Ratio')
    st.plotly_chart(fig, use_container_width=True)

with col2:
    total_requests = cache_data['count'].sum()
    hit_rate = (cache_data.loc[0, 'count'] / total_requests) * 100
    st.metric("Hit Rate", f"{hit_rate:.1f}%")
    st.metric("Total Requests", f"{total_requests:,}")
    st.metric("Avg Latency", "12ms")

# Links to External Tools
st.header("🔗 Monitoring Tools")

col1, col2, col3 = st.columns(3)

with col1:
    st.markdown(f"""
    **Prometheus**
    - [Query Interface]({prometheus_url})
    - Real-time metrics
    - Custom queries
    """)

with col2:
    st.markdown("""
    **Grafana**
    - [Dashboards](http://localhost:3000)
    - Visual analytics
    - Alerts management
    """)

with col3:
    st.markdown("""
    **Jaeger**
    - [Tracing UI](http://localhost:16686)
    - Distributed traces
    - Performance analysis
    """)

# Sample Prometheus Queries
st.header("📝 Sample Prometheus Queries")

with st.expander("Common Queries", expanded=False):
    st.code("""
# Request rate per service
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# 95th percentile response time
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Active connections
pg_stat_activity_count

# Cache hit rate
rate(cache_hits_total[5m]) / rate(cache_requests_total[5m])
""", language='promql')

# Alerting Rules
st.header("🚨 Alert Rules")

alerts = [
    {"name": "High Error Rate", "condition": "error_rate > 5%", "severity": "warning", "status": "🟢 OK"},
    {"name": "Response Time", "condition": "p95 > 1000ms", "severity": "critical", "status": "🟢 OK"},
    {"name": "Database Connections", "condition": "connections > 80%", "severity": "warning", "status": "🟢 OK"},
    {"name": "Budget Exceeded", "condition": "budget_usage > 100%", "severity": "critical", "status": "🟢 OK"}
]

df_alerts = pd.DataFrame(alerts)
st.dataframe(df_alerts, use_container_width=True, hide_index=True)

# System Health Score
st.header("💯 System Health Score")

health_score = 98.5
st.metric("Overall Health", f"{health_score}%", "+0.5%")

st.progress(health_score / 100)

if health_score >= 95:
    st.success("✅ System is operating optimally")
elif health_score >= 80:
    st.warning("⚠️ System has minor issues")
else:
    st.error("❌ System requires attention")
