"""RAG workflow using LangGraph and MCP server."""

import os
from typing import TypedDict, List, Optional, Any
from langgraph.graph import StateGraph, END
from langchain_openai import ChatOpenAI, OpenAIEmbeddings
from langchain_ollama import ChatOllama
from langchain.schema import HumanMessage, SystemMessage

# Optional: Langfuse for observability
try:
    from langfuse import Langfuse
    from langfuse.decorators import observe, langfuse_context
    LANGFUSE_AVAILABLE = True
except ImportError:
    LANGFUSE_AVAILABLE = False
    print("⚠️  Langfuse not installed - observability disabled")
    print("   Install with: pip install langfuse")
    # Create no-op decorator
    def observe(*args, **kwargs):
        def decorator(func):
            return func
        return decorator if not args or not callable(args[0]) else decorator(args[0])
    langfuse_context = None

from ..clients import MCPClient, JWTHelper


class RAGState(TypedDict):
    """State for RAG workflow."""
    query: str
    tenant_id: str
    user_id: str
    documents: List[dict]
    context: str
    answer: str
    error: Optional[str]
    trace_id: Optional[str]


class RAGWorkflow:
    """
    RAG workflow that combines:
    1. MCP server for hybrid search
    2. OpenAI for answer generation
    3. LangFuse for observability
    """

    def __init__(
        self,
        mcp_url: str = "http://localhost:8080",
        tenant_id: str = "acme-corp",
        user_id: str = "demo-user",
        model: str = "gpt-4",
        use_ollama: bool = None,
        ollama_url: str = "http://localhost:11434",
        langfuse_public_key: Optional[str] = None,
        langfuse_secret_key: Optional[str] = None,
    ):
        """
        Initialize RAG workflow.

        Args:
            mcp_url: MCP server URL
            tenant_id: Tenant ID for multi-tenancy
            user_id: User ID
            model: Model name (e.g., 'gpt-4', 'llama3', 'mistral')
            use_ollama: Use Ollama instead of OpenAI (default: check USE_OLLAMA env)
            ollama_url: Ollama server URL
            langfuse_public_key: LangFuse API key (optional)
            langfuse_secret_key: LangFuse secret key (optional)
        """
        self.mcp_url = mcp_url
        self.tenant_id = tenant_id
        self.user_id = user_id
        self.model = model

        # Determine whether to use Ollama
        if use_ollama is None:
            use_ollama = os.getenv("USE_OLLAMA", "false").lower() == "true"
        self.use_ollama = use_ollama

        # Initialize JWT helper
        self.jwt_helper = JWTHelper()

        # Generate token for MCP access
        self.token = self.jwt_helper.generate_token(
            tenant_id=tenant_id,
            user_id=user_id,
            scopes=["read"],
            expires_in_hours=24
        )

        # Initialize MCP client
        self.mcp_client = MCPClient(mcp_url, self.token)

        # Initialize LLM (Ollama or OpenAI)
        if self.use_ollama:
            self.llm = ChatOllama(
                model=model if model not in ["gpt-4", "gpt-3.5-turbo"] else "llama3",
                base_url=os.getenv("OLLAMA_URL", ollama_url),
                temperature=0.7,
            )
        else:
            self.llm = ChatOpenAI(
                model=model,
                temperature=0.7,
                openai_api_key=os.getenv("OPENAI_API_KEY")
            )

        # Initialize LangFuse
        self.langfuse = None
        if langfuse_public_key and langfuse_secret_key:
            self.langfuse = Langfuse(
                public_key=langfuse_public_key,
                secret_key=langfuse_secret_key,
                host=os.getenv("LANGFUSE_HOST", "https://cloud.langfuse.com")
            )

        # Build workflow graph
        self.workflow = self._build_workflow()

    def _build_workflow(self):
        """Build LangGraph workflow."""
        workflow = StateGraph(RAGState)

        # Add nodes
        workflow.add_node("search", self._search_documents)
        workflow.add_node("format_context", self._format_context)
        workflow.add_node("generate_answer", self._generate_answer)

        # Add edges
        workflow.set_entry_point("search")
        workflow.add_edge("search", "format_context")
        workflow.add_edge("format_context", "generate_answer")
        workflow.add_edge("generate_answer", END)

        return workflow.compile()

    @observe(name="search_documents")
    def _search_documents(self, state: RAGState) -> RAGState:
        """Search for relevant documents using MCP hybrid search."""
        try:
            # Call MCP hybrid search
            result = self.mcp_client.hybrid_search(
                query=state["query"],
                limit=5,
                bm25_weight=0.5,
                vector_weight=0.5
            )

            if "result" in result:
                documents = result["result"].get("documents", [])
                state["documents"] = documents

                # Log to LangFuse
                if self.langfuse:
                    langfuse_context.update_current_observation(
                        metadata={
                            "tenant_id": state["tenant_id"],
                            "user_id": state["user_id"],
                            "num_documents": len(documents),
                            "search_type": "hybrid"
                        }
                    )
            else:
                state["error"] = result.get("error", {}).get("message", "Unknown error")
                state["documents"] = []

        except Exception as e:
            state["error"] = str(e)
            state["documents"] = []

        return state

    def _format_context(self, state: RAGState) -> RAGState:
        """Format documents into context string."""
        if not state["documents"]:
            state["context"] = "No relevant documents found."
            return state

        context_parts = []
        for i, doc in enumerate(state["documents"], 1):
            title = doc.get("title", "Untitled")
            content = doc.get("content", "")
            score = doc.get("score", 0.0)

            context_parts.append(
                f"Document {i} (relevance: {score:.2f}):\n"
                f"Title: {title}\n"
                f"Content: {content}\n"
            )

        state["context"] = "\n\n".join(context_parts)
        return state

    @observe(name="generate_answer")
    def _generate_answer(self, state: RAGState) -> RAGState:
        """Generate answer using LLM with retrieved context."""
        if state.get("error"):
            state["answer"] = f"Error: {state['error']}"
            return state

        try:
            system_prompt = """You are a helpful assistant that answers questions based on the provided context.
If the context doesn't contain relevant information, say so clearly.
Always cite which documents you're using in your answer."""

            user_prompt = f"""Context:
{state['context']}

Question: {state['query']}

Please provide a comprehensive answer based on the context above."""

            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=user_prompt)
            ]

            response = self.llm.invoke(messages)
            state["answer"] = response.content

            # Log to LangFuse
            if self.langfuse:
                langfuse_context.update_current_observation(
                    input=user_prompt,
                    output=response.content,
                    metadata={
                        "model": self.model,
                        "tenant_id": state["tenant_id"],
                        "num_documents": len(state["documents"])
                    },
                    usage={
                        "input": len(user_prompt),
                        "output": len(response.content)
                    }
                )

        except Exception as e:
            state["error"] = str(e)
            state["answer"] = f"Error generating answer: {e}"

        return state

    @observe(name="rag_query")
    def query(self, query: str, tenant_id: Optional[str] = None, user_id: Optional[str] = None) -> dict:
        """
        Execute RAG query.

        Args:
            query: User question
            tenant_id: Optional tenant ID (overrides default)
            user_id: Optional user ID (overrides default)

        Returns:
            dict with answer, documents, and metadata
        """
        initial_state: RAGState = {
            "query": query,
            "tenant_id": tenant_id or self.tenant_id,
            "user_id": user_id or self.user_id,
            "documents": [],
            "context": "",
            "answer": "",
            "error": None,
            "trace_id": None
        }

        # Execute workflow
        final_state = self.workflow.invoke(initial_state)

        return {
            "answer": final_state["answer"],
            "documents": final_state["documents"],
            "context": final_state["context"],
            "error": final_state.get("error"),
            "metadata": {
                "tenant_id": final_state["tenant_id"],
                "user_id": final_state["user_id"],
                "num_documents": len(final_state["documents"]),
                "model": self.model
            }
        }

    async def aquery(self, query: str, tenant_id: Optional[str] = None, user_id: Optional[str] = None) -> dict:
        """Async version of query."""
        # For now, just wrap synchronous version
        # In production, use async LangChain methods
        return self.query(query, tenant_id, user_id)


# Example usage
if __name__ == "__main__":
    import json

    # Initialize workflow
    workflow = RAGWorkflow(
        mcp_url="http://localhost:8080",
        tenant_id="acme-corp",
        model="gpt-4"
    )

    # Execute query
    result = workflow.query("What are our security policies?")

    print(json.dumps(result, indent=2))
