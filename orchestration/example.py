"""Example demonstrating all LangGraph workflows with LangFuse observability."""

import os
import json
from dotenv import load_dotenv
from workflows import RAGWorkflow, ResearchWorkflow, HybridWorkflow

# Load environment variables
load_dotenv()

def main():
    """Demonstrate all three workflows."""

    # Configuration
    MCP_URL = os.getenv("MCP_SERVER_URL", "http://localhost:8080")
    A2A_URL = os.getenv("A2A_SERVER_URL", "http://localhost:8081")
    TENANT_ID = "acme-corp"
    USER_ID = "demo-user-pro"

    # LangFuse configuration (optional)
    LANGFUSE_PUBLIC_KEY = os.getenv("LANGFUSE_PUBLIC_KEY")
    LANGFUSE_SECRET_KEY = os.getenv("LANGFUSE_SECRET_KEY")

    print("=" * 80)
    print("LangGraph Workflows Demo")
    print("=" * 80)
    print()

    # Example 1: RAG Workflow (MCP only)
    print("1. RAG Workflow - Internal Knowledge Base Search")
    print("-" * 80)
    rag_workflow = RAGWorkflow(
        mcp_url=MCP_URL,
        tenant_id=TENANT_ID,
        user_id=USER_ID,
        model="gpt-3.5-turbo",  # Using cheaper model for demo
        langfuse_public_key=LANGFUSE_PUBLIC_KEY,
        langfuse_secret_key=LANGFUSE_SECRET_KEY
    )

    rag_result = rag_workflow.query("What are our security policies?")
    print(f"Query: {rag_result.get('metadata', {}).get('query', 'N/A')}")
    print(f"Answer: {rag_result.get('answer', 'N/A')[:200]}...")
    print(f"Documents Found: {rag_result.get('metadata', {}).get('num_documents', 0)}")
    print()

    # Example 2: Research Workflow (A2A only)
    print("2. Research Workflow - External Research with Cost Controls")
    print("-" * 80)
    research_workflow = ResearchWorkflow(
        a2a_url=A2A_URL,
        user_id=USER_ID,
        budget_tier="pro",
        model="gpt-3.5-turbo",
        langfuse_public_key=LANGFUSE_PUBLIC_KEY,
        langfuse_secret_key=LANGFUSE_SECRET_KEY
    )

    research_result = research_workflow.research("transformer architecture improvements")
    print(f"Topic: {research_result.get('topic', 'N/A')}")
    print(f"Summary: {research_result.get('summary', 'N/A')[:200]}...")
    print(f"Cost: ${research_result.get('cost', 0.0):.4f}")
    print(f"Tasks Executed: {research_result.get('metadata', {}).get('num_results', 0)}")
    print()

    # Example 3: Hybrid Workflow (MCP + A2A)
    print("3. Hybrid Workflow - Internal + External Sources")
    print("-" * 80)
    hybrid_workflow = HybridWorkflow(
        mcp_url=MCP_URL,
        a2a_url=A2A_URL,
        tenant_id=TENANT_ID,
        user_id=USER_ID,
        budget_tier="pro",
        model="gpt-3.5-turbo",
        langfuse_public_key=LANGFUSE_PUBLIC_KEY,
        langfuse_secret_key=LANGFUSE_SECRET_KEY
    )

    hybrid_result = hybrid_workflow.query(
        "What are the latest ML model improvements and how do they relate to our strategy?"
    )
    print(f"Query: {hybrid_result.get('query', 'N/A')[:80]}...")
    print(f"Answer: {hybrid_result.get('answer', 'N/A')[:200]}...")
    print(f"Internal Docs: {hybrid_result.get('metadata', {}).get('num_internal_docs', 0)}")
    print(f"External Sources: {hybrid_result.get('metadata', {}).get('num_external_sources', 0)}")
    print(f"Total Cost: ${hybrid_result.get('cost', 0.0):.4f}")
    print()

    print("=" * 80)
    print("Demo Complete!")
    print("=" * 80)
    print()

    if LANGFUSE_PUBLIC_KEY:
        print("✅ LangFuse tracing enabled - view traces at https://cloud.langfuse.com")
    else:
        print("ℹ️  LangFuse not configured - set LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY")
    print()

    # Save full results to file
    results = {
        "rag": rag_result,
        "research": research_result,
        "hybrid": hybrid_result
    }

    with open("workflow_results.json", "w") as f:
        json.dump(results, f, indent=2)

    print("📄 Full results saved to: workflow_results.json")


if __name__ == "__main__":
    main()
