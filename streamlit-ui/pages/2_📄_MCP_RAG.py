"""
MCP RAG Pipeline Testing
Test multi-tenant document search with hybrid search capabilities
"""
import streamlit as st
import os
import json
import pandas as pd
from utils.mcp_client import MCPClient

st.set_page_config(page_title="MCP RAG", page_icon="📄", layout="wide")

st.title("📄 MCP Multi-Tenant RAG Pipeline")

# Get MCP URL
mcp_url = os.getenv('MCP_SERVER_URL', 'http://localhost:8080')

# Check for token
if 'token' not in st.session_state or not st.session_state.token:
    st.warning("⚠️ No JWT token found. Please generate one on the **🔐 Authentication** page first.")
    st.stop()

# Initialize MCP client
client = MCPClient(mcp_url, st.session_state.token)

# Display current tenant
st.info(f"🏢 Active Tenant: **{st.session_state.get('current_tenant', 'Unknown')}**")

# Initialize MCP session
try:
    with st.spinner("Initializing MCP session..."):
        init_result = client.initialize()
        if "result" in init_result:
            st.success("✅ MCP session initialized")
            with st.expander("Server Info"):
                st.json(init_result["result"])
except Exception as e:
    st.error(f"❌ Failed to initialize MCP: {str(e)}")
    st.stop()

# List Available Tools
st.header("Available MCP Tools")

try:
    tools = client.list_tools()
    if tools:
        tool_names = [tool["name"] for tool in tools]
        st.success(f"Found {len(tools)} tools: {', '.join(tool_names)}")

        with st.expander("Tool Details"):
            for tool in tools:
                st.markdown(f"**{tool['name']}**")
                st.markdown(f"_{tool.get('description', 'No description')}_")
                if 'inputSchema' in tool:
                    st.json(tool['inputSchema'])
                st.markdown("---")
    else:
        st.warning("No tools available")
except Exception as e:
    st.error(f"Error listing tools: {str(e)}")

# Search Interface
st.header("🔍 Document Search")

tab1, tab2, tab3, tab4 = st.tabs(["Hybrid Search", "Simple Search", "List Documents", "Retrieve Document"])

with tab1:
    st.subheader("Hybrid Search (BM25 + Vector)")

    query = st.text_input("Search Query", value="machine learning algorithms", key="hybrid_query")

    col1, col2, col3 = st.columns(3)
    with col1:
        limit = st.slider("Max Results", 1, 50, 10, key="hybrid_limit")
    with col2:
        bm25_weight = st.slider("BM25 Weight", 0.0, 1.0, 0.5, 0.1, key="bm25_weight")
    with col3:
        vector_weight = st.slider("Vector Weight", 0.0, 1.0, 0.5, 0.1, key="vector_weight")

    if st.button("Search (Hybrid)", type="primary"):
        with st.spinner("Searching..."):
            try:
                result = client.hybrid_search(
                    query=query,
                    limit=limit,
                    bm25_weight=bm25_weight,
                    vector_weight=vector_weight
                )

                if "result" in result and "content" in result["result"]:
                    content = result["result"]["content"]
                    if isinstance(content, list) and content:
                        results_data = json.loads(content[0].get("text", "[]"))

                        if results_data:
                            st.success(f"Found {len(results_data)} results")

                            # Display results
                            for i, doc in enumerate(results_data, 1):
                                with st.expander(f"Result {i}: {doc.get('title', 'Untitled')} (Score: {doc.get('score', 0):.4f})"):
                                    st.markdown(f"**Document ID**: {doc.get('doc_id', 'N/A')}")
                                    st.markdown(f"**Tenant**: {doc.get('tenant_id', 'N/A')}")
                                    st.markdown(f"**Hybrid Score**: {doc.get('score', 0):.4f}")
                                    st.markdown(f"**BM25 Rank**: {doc.get('bm25_rank', 'N/A')}")
                                    st.markdown(f"**Vector Rank**: {doc.get('vector_rank', 'N/A')}")
                                    if doc.get('content'):
                                        st.markdown("**Content Preview**:")
                                        st.text(doc['content'][:500] + "..." if len(doc['content']) > 500 else doc['content'])
                        else:
                            st.info("No results found")
                    else:
                        st.warning("Unexpected response format")
                else:
                    st.error("Error in response")
                    st.json(result)
            except Exception as e:
                st.error(f"Search failed: {str(e)}")

with tab2:
    st.subheader("Simple Text Search")

    simple_query = st.text_input("Search Query", value="neural networks", key="simple_query")
    simple_limit = st.slider("Max Results", 1, 50, 10, key="simple_limit")

    if st.button("Search (Simple)", type="primary"):
        with st.spinner("Searching..."):
            try:
                result = client.search_documents(query=simple_query, limit=simple_limit)

                if "result" in result:
                    st.success("Search completed")
                    st.json(result["result"])
                else:
                    st.error("Error in response")
                    st.json(result)
            except Exception as e:
                st.error(f"Search failed: {str(e)}")

with tab3:
    st.subheader("List All Documents")

    col1, col2 = st.columns(2)
    with col1:
        list_limit = st.number_input("Limit", 1, 100, 20, key="list_limit")
    with col2:
        list_offset = st.number_input("Offset", 0, 1000, 0, key="list_offset")

    if st.button("List Documents", type="primary"):
        with st.spinner("Fetching documents..."):
            try:
                result = client.list_documents(limit=list_limit, offset=list_offset)

                if "result" in result:
                    st.success("Documents retrieved")
                    st.json(result["result"])
                else:
                    st.error("Error in response")
                    st.json(result)
            except Exception as e:
                st.error(f"Failed to list documents: {str(e)}")

with tab4:
    st.subheader("Retrieve Specific Document")

    doc_id = st.text_input("Document ID", value="", placeholder="Enter document ID")

    if st.button("Retrieve", type="primary"):
        if not doc_id:
            st.warning("Please enter a document ID")
        else:
            with st.spinner("Retrieving document..."):
                try:
                    result = client.retrieve_document(doc_id=doc_id)

                    if "result" in result:
                        st.success("Document retrieved")
                        st.json(result["result"])
                    else:
                        st.error("Document not found or error occurred")
                        st.json(result)
                except Exception as e:
                    st.error(f"Failed to retrieve document: {str(e)}")

# Multi-tenancy Demo
st.header("🏢 Multi-Tenancy Isolation")

st.markdown("""
**How it works:**
1. Each tenant has isolated data via PostgreSQL Row-Level Security (RLS)
2. JWT token contains `tenant_id` claim
3. Database automatically filters queries based on tenant
4. Tenant A cannot see Tenant B's documents

**Try this:**
1. Generate a token for **acme-corp** tenant
2. Search for documents - you'll only see acme-corp's data
3. Generate a token for **globex** tenant
4. Same search shows different results!
""")

# Performance Metrics
st.header("📊 Search Performance")

st.markdown("""
**Hybrid Search Benefits:**
- **BM25**: Excellent for exact keyword matching
- **Vector Search**: Captures semantic meaning
- **RRF (Reciprocal Rank Fusion)**: Combines both approaches

Adjust weights to tune results:
- More BM25 weight → Better for exact terms
- More vector weight → Better for concepts/meaning
""")

# Rate Limiting Info
st.header("⏱️ Rate Limiting")

st.info("""
**Current Configuration:**
- 100 requests per minute per tenant
- Token bucket algorithm with Redis
- Exceeding limit returns 429 Too Many Requests

**Try this:**
Make multiple rapid searches to see rate limiting in action
(refresh the page between attempts)
""")
