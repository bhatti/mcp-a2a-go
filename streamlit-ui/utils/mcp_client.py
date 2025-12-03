"""MCP Client Wrapper for Streamlit UI"""
import requests
from typing import Dict, Any, Optional, List
import json


class MCPClient:
    """Client for interacting with MCP Server"""

    def __init__(self, base_url: str, token: Optional[str] = None):
        self.base_url = base_url.rstrip('/')
        self.token = token
        self.session = requests.Session()

        if token:
            self.session.headers.update({
                'Authorization': f'Bearer {token}'
            })

    def _make_request(self, method: str, params: Any = None, request_id: str = "1") -> Dict[str, Any]:
        """Make a JSON-RPC 2.0 request"""
        payload = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
        }

        if params is not None:
            payload["params"] = params

        response = self.session.post(
            f"{self.base_url}/mcp",
            json=payload,
            headers={'Content-Type': 'application/json'}
        )
        response.raise_for_status()
        return response.json()

    def initialize(self, client_name: str = "streamlit-ui", client_version: str = "1.0.0") -> Dict[str, Any]:
        """Initialize MCP session"""
        params = {
            "protocolVersion": "2024-11-05",
            "clientInfo": {
                "name": client_name,
                "version": client_version
            }
        }
        return self._make_request("initialize", params)

    def list_tools(self) -> List[Dict[str, Any]]:
        """List available MCP tools"""
        response = self._make_request("tools/list")
        if "result" in response:
            return response["result"].get("tools", [])
        return []

    def call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """Call an MCP tool"""
        params = {
            "name": tool_name,
            "arguments": arguments
        }
        return self._make_request("tools/call", params)

    def search_documents(self, query: str, limit: int = 10) -> Dict[str, Any]:
        """Search documents using MCP"""
        return self.call_tool("search_documents", {
            "query": query,
            "limit": limit
        })

    def retrieve_document(self, doc_id: str) -> Dict[str, Any]:
        """Retrieve a specific document"""
        return self.call_tool("retrieve_document", {
            "document_id": doc_id  # Match the parameter name expected by the tool
        })

    def list_documents(self, limit: int = 100, offset: int = 0) -> Dict[str, Any]:
        """List documents with pagination"""
        return self.call_tool("list_documents", {
            "limit": limit,
            "offset": offset
        })

    def hybrid_search(self, query: str, limit: int = 10,
                     bm25_weight: float = 0.5, vector_weight: float = 0.5) -> Dict[str, Any]:
        """Perform hybrid search (BM25 + Vector)"""
        return self.call_tool("hybrid_search", {
            "query": query,
            "limit": limit,
            "bm25_weight": bm25_weight,
            "vector_weight": vector_weight
        })

    def health_check(self) -> bool:
        """Check if MCP server is healthy"""
        try:
            response = requests.get(f"{self.base_url}/health", timeout=5)
            return response.status_code == 200
        except:
            return False
