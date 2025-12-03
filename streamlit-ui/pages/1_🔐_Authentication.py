"""
Authentication & Authorization Testing
Generate and test JWT tokens for MCP server
"""
import streamlit as st
import json
from utils.auth import JWTHelper, DEMO_TENANTS, DEMO_USERS
from utils.mcp_client import MCPClient

st.set_page_config(page_title="Authentication", page_icon="🔐", layout="wide")

st.title("🔐 Authentication & Authorization")

# Initialize JWT helper
if 'jwt_helper' not in st.session_state:
    st.session_state.jwt_helper = JWTHelper()

jwt_helper = st.session_state.jwt_helper

# Token Generation Section
st.header("Generate JWT Token")

col1, col2 = st.columns(2)

with col1:
    st.subheader("MCP Server Authentication")

    tenant = st.selectbox(
        "Select Tenant",
        list(DEMO_TENANTS.keys()),
        help="Multi-tenant isolation for MCP RAG"
    )

    user_id = st.text_input("User ID", value="demo-user", help="User identifier")

    scopes = st.multiselect(
        "Scopes",
        ["read", "write", "admin"],
        default=["read", "write"],
        help="Permission scopes for the token"
    )

    expires_in = st.slider("Expires in (hours)", 1, 168, 24)

    if st.button("Generate MCP Token", type="primary"):
        try:
            token = jwt_helper.generate_token(
                tenant_id=DEMO_TENANTS[tenant],
                user_id=user_id,
                scopes=scopes,
                expires_in_hours=expires_in
            )
            st.session_state.token = token
            st.session_state.current_tenant = tenant
            st.success("✅ Token generated successfully!")
        except Exception as e:
            st.error(f"❌ Error generating token: {str(e)}")

with col2:
    st.subheader("A2A Server Users")

    a2a_user = st.selectbox(
        "Select A2A User",
        list(DEMO_USERS.keys()),
        help="Users with different budget tiers"
    )

    user_info = DEMO_USERS[a2a_user]
    st.info(f"**Tier**: {user_info['tier']}")
    st.info(f"**Monthly Budget**: ${user_info['budget']}")

    st.session_state.a2a_user_id = a2a_user

    st.markdown("""
    **Note**: A2A server doesn't require JWT tokens in this demo.
    Budget enforcement is based on user_id.
    """)

# Display Current Token
st.header("Current Token")

if 'token' in st.session_state and st.session_state.token:
    st.success(f"🔑 Active token for tenant: **{st.session_state.get('current_tenant', 'Unknown')}**")

    # Token display
    with st.expander("View Token", expanded=False):
        st.text_area("Token (click to select all, then copy)",
                     st.session_state.token,
                     height=100,
                     help="Select all text and copy with Ctrl+C / Cmd+C")

    # Decode token
    with st.expander("Decoded Token", expanded=True):
        try:
            decoded = jwt_helper.decode_token(st.session_state.token)
            st.json(decoded)
        except Exception as e:
            st.error(f"Error decoding token: {str(e)}")

    # Test token
    st.subheader("Test Token")

    import os
    mcp_url = os.getenv('MCP_SERVER_URL', 'http://localhost:8080')

    if st.button("Test Token with MCP Server"):
        with st.spinner("Testing token..."):
            try:
                client = MCPClient(mcp_url, st.session_state.token)
                result = client.initialize()

                if "result" in result:
                    st.success("✅ Token is valid!")
                    st.json(result["result"])
                else:
                    st.error("❌ Token validation failed")
                    st.json(result)
            except Exception as e:
                st.error(f"❌ Error: {str(e)}")
else:
    st.warning("⚠️ No active token. Generate one above to continue.")

# Authentication Flow Diagram
st.header("Authentication Flow")

st.markdown("""
```
┌─────────┐                           ┌─────────────┐
│ Client  │                           │ MCP Server  │
│ (UI)    │                           │             │
└────┬────┘                           └──────┬──────┘
     │                                       │
     │ 1. Request with JWT token             │
     │──────────────────────────────────────>│
     │    Authorization: Bearer <token>      │
     │                                       │
     │                                    2. Validate
     │                                    ┌──┴───┐
     │                                    │ JWT  │
     │                                    │Check │
     │                                    └──┬───┘
     │                                       │
     │ 3. Response with tenant context       │
     │<──────────────────────────────────────│
     │    - tenant_id extracted              │
     │    - RLS policies applied             │
     │                                       │
```
""")

# Security Best Practices
st.header("🔒 Security Features")

col1, col2 = st.columns(2)

with col1:
    st.markdown("""
    **Implemented Security:**
    - ✅ RS256 (RSA) signing algorithm
    - ✅ Token expiration validation
    - ✅ Issuer and audience validation
    - ✅ Multi-tenant isolation (tenant_id in claims)
    - ✅ Scope-based authorization
    - ✅ Rate limiting per tenant
    """)

with col2:
    st.markdown("""
    **Production Recommendations:**
    - 🔐 Store private keys in secure vault (HashiCorp Vault, AWS KMS)
    - 🔐 Use short-lived tokens (1-24 hours)
    - 🔐 Implement token refresh mechanism
    - 🔐 Rotate signing keys regularly
    - 🔐 Use HTTPS only in production
    - 🔐 Implement API key rotation
    """)

# Quick Test Section
st.header("Quick Test")

st.markdown("""
**Try this**:
1. Generate a token above with `read` and `write` scopes
2. Go to **📄 MCP RAG** page
3. Use the token to perform searches
4. Try removing the token to see authentication errors
""")
