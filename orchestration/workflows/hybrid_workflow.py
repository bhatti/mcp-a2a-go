"""Hybrid workflow combining MCP RAG and A2A research capabilities."""

import os
from typing import TypedDict, List, Optional, Dict, Any
from langgraph.graph import StateGraph, END
from langchain_openai import ChatOpenAI
from langchain.schema import HumanMessage, SystemMessage
# Optional: Langfuse for observability
try:
    from langfuse import Langfuse
    from langfuse.decorators import observe, langfuse_context
    LANGFUSE_AVAILABLE = True
except ImportError:
    LANGFUSE_AVAILABLE = False
    # Create no-op decorator
    def observe(*args, **kwargs):
        def decorator(func):
            return func
        return decorator if not args or not callable(args[0]) else decorator(args[0])
    langfuse_context = None


from ..clients import MCPClient, A2AClient, JWTHelper


class HybridState(TypedDict):
    """State for hybrid workflow."""
    query: str
    tenant_id: str
    user_id: str
    budget_tier: str
    internal_docs: List[dict]
    external_research: List[dict]
    combined_context: str
    final_answer: str
    cost: float
    error: Optional[str]


class HybridWorkflow:
    """
    Hybrid workflow that combines:
    1. MCP RAG for internal knowledge base
    2. A2A research for external information
    3. LLM synthesis of both sources
    4. LangFuse observability

    Use Case: Answer questions using both internal docs and external research,
              with cost controls and multi-tenant isolation.
    """

    def __init__(
        self,
        mcp_url: str = "http://localhost:8080",
        a2a_url: str = "http://localhost:8081",
        tenant_id: str = "acme-corp",
        user_id: str = "demo-user-pro",
        budget_tier: str = "pro",
        model: str = "gpt-4",
        langfuse_public_key: Optional[str] = None,
        langfuse_secret_key: Optional[str] = None,
    ):
        """Initialize hybrid workflow."""
        self.mcp_url = mcp_url
        self.a2a_url = a2a_url
        self.tenant_id = tenant_id
        self.user_id = user_id
        self.budget_tier = budget_tier
        self.model = model

        # Initialize JWT helper
        self.jwt_helper = JWTHelper()

        # Generate token for MCP access
        self.token = self.jwt_helper.generate_token(
            tenant_id=tenant_id,
            user_id=user_id,
            scopes=["read"],
            expires_in_hours=24
        )

        # Initialize clients
        self.mcp_client = MCPClient(mcp_url, self.token)
        self.a2a_client = A2AClient(a2a_url)

        # Initialize LLM
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
        workflow = StateGraph(HybridState)

        # Add nodes
        workflow.add_node("search_internal", self._search_internal)
        workflow.add_node("research_external", self._research_external)
        workflow.add_node("combine_sources", self._combine_sources)
        workflow.add_node("generate_answer", self._generate_answer)

        # Add edges - internal and external run in parallel conceptually
        # but LangGraph executes sequentially
        workflow.set_entry_point("search_internal")
        workflow.add_edge("search_internal", "research_external")
        workflow.add_edge("research_external", "combine_sources")
        workflow.add_edge("combine_sources", "generate_answer")
        workflow.add_edge("generate_answer", END)

        return workflow.compile()

    @observe(name="search_internal_docs")
    def _search_internal(self, state: HybridState) -> HybridState:
        """Search internal knowledge base via MCP."""
        try:
            result = self.mcp_client.hybrid_search(
                query=state["query"],
                limit=3,
                bm25_weight=0.5,
                vector_weight=0.5
            )

            if "result" in result:
                documents = result["result"].get("documents", [])
                state["internal_docs"] = documents

                # Log to LangFuse
                if self.langfuse:
                    langfuse_context.update_current_observation(
                        metadata={
                            "source": "internal",
                            "tenant_id": state["tenant_id"],
                            "num_docs": len(documents)
                        }
                    )
            else:
                state["internal_docs"] = []

        except Exception as e:
            state["error"] = f"Internal search error: {e}"
            state["internal_docs"] = []

        return state

    @observe(name="research_external_sources")
    def _research_external(self, state: HybridState) -> HybridState:
        """Research external sources via A2A."""
        if state.get("error"):
            state["external_research"] = []
            return state

        try:
            # Create research task
            task_result = self.a2a_client.create_task(
                user_id=state["user_id"],
                agent_id="research-assistant",
                capability="search_papers",  # Use appropriate capability
                input_data={
                    "query": state["query"],
                    "limit": 3
                }
            )

            if "error" in task_result:
                state["error"] = task_result["error"]
                state["external_research"] = []
                return state

            task_id = task_result.get("task_id")
            if task_id:
                # In production, use SSE streaming
                # For demo, just get final result
                import time
                for _ in range(30):
                    time.sleep(1)
                    status = self.a2a_client.get_task(task_id)
                    if status.get("state") == "completed":
                        state["external_research"] = [status.get("result", {})]
                        state["cost"] = status.get("cost", 0.0)
                        break

                # Log to LangFuse
                if self.langfuse:
                    langfuse_context.update_current_observation(
                        metadata={
                            "source": "external",
                            "task_id": task_id,
                            "cost": state["cost"]
                        }
                    )
            else:
                state["external_research"] = []

        except Exception as e:
            state["error"] = f"External research error: {e}"
            state["external_research"] = []

        return state

    def _combine_sources(self, state: HybridState) -> HybridState:
        """Combine internal and external sources."""
        context_parts = []

        # Add internal docs
        if state["internal_docs"]:
            context_parts.append("=== Internal Knowledge Base ===\n")
            for i, doc in enumerate(state["internal_docs"], 1):
                title = doc.get("title", "Untitled")
                content = doc.get("content", "")
                context_parts.append(f"{i}. {title}\n{content}\n")

        # Add external research
        if state["external_research"]:
            context_parts.append("\n=== External Research ===\n")
            for i, research in enumerate(state["external_research"], 1):
                context_parts.append(f"{i}. {research}\n")

        state["combined_context"] = "\n\n".join(context_parts)
        return state

    @observe(name="generate_final_answer")
    def _generate_answer(self, state: HybridState) -> HybridState:
        """Generate final answer combining all sources."""
        if state.get("error"):
            state["final_answer"] = f"Error: {state['error']}"
            return state

        try:
            system_prompt = """You are a comprehensive research assistant with access to both
internal company knowledge and external research. Synthesize information from both sources
to provide complete, well-rounded answers. Always cite your sources clearly."""

            user_prompt = f"""Question: {state['query']}

Available Information:
{state['combined_context']}

Please provide a comprehensive answer that:
1. Addresses the question directly
2. Incorporates relevant information from both internal and external sources
3. Clearly cites which sources you're using
4. Notes any limitations or gaps in the available information"""

            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=user_prompt)
            ]

            response = self.llm.invoke(messages)
            state["final_answer"] = response.content

            # Log to LangFuse
            if self.langfuse:
                langfuse_context.update_current_observation(
                    input=user_prompt,
                    output=response.content,
                    metadata={
                        "model": self.model,
                        "internal_docs": len(state["internal_docs"]),
                        "external_research": len(state["external_research"]),
                        "total_cost": state["cost"]
                    }
                )

        except Exception as e:
            state["error"] = str(e)
            state["final_answer"] = f"Error generating answer: {e}"

        return state

    @observe(name="hybrid_query")
    def query(
        self,
        query: str,
        tenant_id: Optional[str] = None,
        user_id: Optional[str] = None,
        budget_tier: Optional[str] = None
    ) -> dict:
        """
        Execute hybrid query combining internal and external sources.

        Args:
            query: User question
            tenant_id: Optional tenant ID (overrides default)
            user_id: Optional user ID (overrides default)
            budget_tier: Optional budget tier (overrides default)

        Returns:
            dict with answer, sources, cost, and metadata
        """
        initial_state: HybridState = {
            "query": query,
            "tenant_id": tenant_id or self.tenant_id,
            "user_id": user_id or self.user_id,
            "budget_tier": budget_tier or self.budget_tier,
            "internal_docs": [],
            "external_research": [],
            "combined_context": "",
            "final_answer": "",
            "cost": 0.0,
            "error": None
        }

        # Execute workflow
        final_state = self.workflow.invoke(initial_state)

        return {
            "query": final_state["query"],
            "answer": final_state["final_answer"],
            "sources": {
                "internal": final_state["internal_docs"],
                "external": final_state["external_research"]
            },
            "cost": final_state["cost"],
            "error": final_state.get("error"),
            "metadata": {
                "tenant_id": final_state["tenant_id"],
                "user_id": final_state["user_id"],
                "budget_tier": final_state["budget_tier"],
                "num_internal_docs": len(final_state["internal_docs"]),
                "num_external_sources": len(final_state["external_research"]),
                "model": self.model
            }
        }

    async def aquery(
        self,
        query: str,
        tenant_id: Optional[str] = None,
        user_id: Optional[str] = None,
        budget_tier: Optional[str] = None
    ) -> dict:
        """Async version of query."""
        return self.query(query, tenant_id, user_id, budget_tier)


# Example usage
if __name__ == "__main__":
    import json

    # Initialize workflow
    workflow = HybridWorkflow(
        mcp_url="http://localhost:8080",
        a2a_url="http://localhost:8081",
        tenant_id="acme-corp",
        user_id="demo-user-pro",
        model="gpt-4"
    )

    # Execute hybrid query
    result = workflow.query(
        "What are the latest developments in transformer architectures, "
        "and how do they relate to our internal ML strategy?"
    )

    print(json.dumps(result, indent=2))
