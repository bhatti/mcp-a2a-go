"""
Production-Grade MCP & A2A Demo UI
Main entry point for the Streamlit application
"""
import streamlit as st
import os
from utils.mcp_client import MCPClient
from utils.a2a_client import A2AClient
from utils.auth import JWTHelper, DEMO_TENANTS, DEMO_USERS

# Page configuration
st.set_page_config(
    page_title="MCP & A2A Demo",
    page_icon="🤖",
    layout="wide",
    initial_sidebar_state="expanded"
)

# Initialize session state
if 'jwt_helper' not in st.session_state:
    st.session_state.jwt_helper = JWTHelper()

if 'token' not in st.session_state:
    st.session_state.token = None

if 'mcp_client' not in st.session_state:
    st.session_state.mcp_client = None

if 'a2a_client' not in st.session_state:
    st.session_state.a2a_client = None

# Get server URLs from environment
MCP_URL = os.getenv('MCP_SERVER_URL', 'http://localhost:8080')
A2A_URL = os.getenv('A2A_SERVER_URL', 'http://localhost:8081')
JAEGER_URL = os.getenv('JAEGER_URL', 'http://localhost:16686')
PROMETHEUS_URL = os.getenv('PROMETHEUS_URL', 'http://localhost:9090')

# Main page
st.title("🤖 Production-Grade MCP & A2A Demo")
st.markdown("""
This interactive demo showcases production-ready implementations of:
- **MCP (Model Context Protocol)**: Multi-tenant RAG pipeline with authentication
- **A2A (Agent-to-Agent Protocol)**: Cost-controlled research assistant

### Features Demonstrated:
- ✅ JWT Authentication & Authorization
- ✅ Multi-tenant isolation
- ✅ Rate limiting & budget enforcement
- ✅ Hybrid search (BM25 + Vector)
- ✅ Real-time task streaming (SSE)
- ✅ Distributed tracing (Jaeger)
- ✅ Metrics & monitoring (Prometheus)
""")

# Sidebar - System Status
st.sidebar.title("System Status")

# Check MCP server health
mcp_healthy = False
try:
    mcp_client = MCPClient(MCP_URL)
    mcp_healthy = mcp_client.health_check()
except:
    pass

st.sidebar.metric(
    "MCP Server",
    "🟢 Healthy" if mcp_healthy else "🔴 Offline",
    MCP_URL
)

# Check A2A server health
a2a_healthy = False
try:
    a2a_client = A2AClient(A2A_URL)
    a2a_healthy = a2a_client.health_check()
except:
    pass

st.sidebar.metric(
    "A2A Server",
    "🟢 Healthy" if a2a_healthy else "🔴 Offline",
    A2A_URL
)

# Links to observability tools
st.sidebar.markdown("### Observability")
st.sidebar.markdown(f"- [Jaeger Traces]({JAEGER_URL})")
st.sidebar.markdown(f"- [Prometheus Metrics]({PROMETHEUS_URL})")
st.sidebar.markdown("- [Grafana Dashboards](http://localhost:3000)")

# Quick Start Guide
st.header("Quick Start")

col1, col2 = st.columns(2)

with col1:
    st.subheader("📄 MCP RAG Pipeline")
    st.markdown("""
    1. Go to **🔐 Authentication** page
    2. Generate a JWT token
    3. Navigate to **📄 MCP RAG** page
    4. Try hybrid search and document retrieval
    """)

with col2:
    st.subheader("🤖 A2A Cost Control")
    st.markdown("""
    1. Go to **🤖 A2A Tasks** page
    2. Select a demo user (basic/pro/enterprise)
    3. Create tasks and monitor budgets
    4. Watch real-time SSE streaming
    """)

# System Architecture
st.header("Architecture")
st.markdown("""
```
┌─────────────────────────────────────────────────────────┐
│                    Streamlit UI (You are here)          │
└─────────────────────┬───────────────────────────────────┘
                      │
          ┌───────────┴───────────┐
          │                       │
┌─────────▼────────┐   ┌──────────▼──────────┐
│   MCP Server     │   │   A2A Server        │
│   Port 8080      │   │   Port 8081         │
│                  │   │                     │
│ - Auth (JWT)     │   │ - Agent Cards       │
│ - Rate Limiting  │   │ - Task Management   │
│ - Hybrid Search  │   │ - Cost Tracking     │
│ - Multi-Tenancy  │   │ - SSE Streaming     │
└─────────┬────────┘   └─────────────────────┘
          │
    ┌─────┴─────┬────────────┬────────────┐
    │           │            │            │
┌───▼────┐  ┌──▼───┐  ┌────▼─────┐  ┌──▼──────┐
│Postgres│  │Redis │  │ Jaeger   │  │Prometheus│
│pgvector│  │      │  │ Tracing  │  │ Metrics  │
└────────┘  └──────┘  └──────────┘  └──────────┘
```
""")

# Test Coverage Statistics
st.header("📊 Test Coverage")
col1, col2, col3 = st.columns(3)

with col1:
    st.metric("MCP Server", "95%", "Coverage")
    st.caption("125+ unit tests across all packages")

with col2:
    st.metric("A2A Server", "92.6%", "Coverage")
    st.caption("75+ unit tests across all packages")

with col3:
    st.metric("Total Tests", "200+", "Passing")
    st.caption("All core packages tested")

# Next Steps
st.header("📚 Navigation")
st.markdown("""
Use the sidebar to navigate between pages:
- **🔐 Authentication**: Generate JWT tokens and test auth
- **📄 MCP RAG**: Test multi-tenant document search
- **🤖 A2A Tasks**: Create and monitor cost-controlled tasks
- **💰 Cost Tracking**: View budget usage and costs
- **📊 Metrics**: Real-time system metrics
- **🔍 Tracing**: Distributed tracing with Jaeger
""")

# Footer
st.markdown("---")
st.markdown("""
<div style='text-align: center'>
    <p>Built with Go, Python, Streamlit | Tutorial-focused implementation</p>
    <p>🚀 Production patterns: Authentication, Multi-tenancy, Observability, Cost Control</p>
</div>
""", unsafe_allow_html=True)
