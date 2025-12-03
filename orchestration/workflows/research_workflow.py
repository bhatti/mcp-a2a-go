"""Research workflow using LangGraph and A2A server with cost controls."""

import os
import time
import json
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


from ..clients import A2AClient


class ResearchState(TypedDict):
    """State for research workflow."""
    topic: str
    user_id: str
    budget_tier: str
    tasks: List[dict]
    results: List[dict]
    summary: str
    cost: float
    error: Optional[str]


class ResearchWorkflow:
    """
    Research workflow with cost controls using A2A server.

    Features:
    - Budget-aware task execution
    - Real-time cost tracking
    - Multi-step research pipeline
    - LangFuse observability
    """

    def __init__(
        self,
        a2a_url: str = "http://localhost:8081",
        user_id: str = "demo-user-pro",
        budget_tier: str = "pro",
        model: str = "gpt-3.5-turbo",
        langfuse_public_key: Optional[str] = None,
        langfuse_secret_key: Optional[str] = None,
    ):
        """Initialize research workflow."""
        self.a2a_url = a2a_url
        self.user_id = user_id
        self.budget_tier = budget_tier
        self.model = model

        # Initialize A2A client
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
        workflow = StateGraph(ResearchState)

        # Add nodes
        workflow.add_node("plan_research", self._plan_research)
        workflow.add_node("execute_tasks", self._execute_tasks)
        workflow.add_node("synthesize_results", self._synthesize_results)

        # Add edges
        workflow.set_entry_point("plan_research")
        workflow.add_edge("plan_research", "execute_tasks")
        workflow.add_edge("execute_tasks", "synthesize_results")
        workflow.add_edge("synthesize_results", END)

        return workflow.compile()

    @observe(name="plan_research")
    def _plan_research(self, state: ResearchState) -> ResearchState:
        """Plan research tasks based on topic."""
        try:
            # Get agent card to see available capabilities
            agent_card = self.a2a_client.get_agent_card()

            if not agent_card:
                state["error"] = "Failed to get agent card"
                state["tasks"] = []
                return state

            # For demo, create tasks for each capability
            capabilities = agent_card.get("capabilities", [])
            tasks = []

            for cap in capabilities[:3]:  # Limit to 3 for cost control
                task = {
                    "capability": cap["name"],
                    "description": cap["description"],
                    "input": {
                        "query": state["topic"],
                        "limit": 5
                    }
                }
                tasks.append(task)

            state["tasks"] = tasks

            # Log to LangFuse
            if self.langfuse:
                langfuse_context.update_current_observation(
                    metadata={
                        "topic": state["topic"],
                        "num_tasks": len(tasks),
                        "budget_tier": state["budget_tier"]
                    }
                )

        except Exception as e:
            state["error"] = str(e)
            state["tasks"] = []

        return state

    @observe(name="execute_tasks")
    def _execute_tasks(self, state: ResearchState) -> ResearchState:
        """Execute research tasks via A2A server."""
        if state.get("error") or not state["tasks"]:
            state["results"] = []
            return state

        results = []
        total_cost = 0.0

        try:
            for task_spec in state["tasks"]:
                # Create task
                task_result = self.a2a_client.create_task(
                    user_id=state["user_id"],
                    agent_id="research-assistant",
                    capability=task_spec["capability"],
                    input_data=task_spec["input"]
                )

                if "error" in task_result:
                    # Budget exceeded or other error
                    state["error"] = task_result["error"]
                    break

                task_id = task_result.get("task_id")
                if not task_id:
                    continue

                # Poll for completion (in production, use SSE streaming)
                max_attempts = 30
                for attempt in range(max_attempts):
                    time.sleep(1)
                    status = self.a2a_client.get_task(task_id)

                    if status.get("state") == "completed":
                        results.append({
                            "capability": task_spec["capability"],
                            "result": status.get("result", {}),
                            "cost": status.get("cost", 0.0)
                        })
                        total_cost += status.get("cost", 0.0)
                        break
                    elif status.get("state") in ["failed", "cancelled"]:
                        results.append({
                            "capability": task_spec["capability"],
                            "error": status.get("error", "Task failed")
                        })
                        break

            state["results"] = results
            state["cost"] = total_cost

            # Log to LangFuse
            if self.langfuse:
                langfuse_context.update_current_observation(
                    metadata={
                        "num_results": len(results),
                        "total_cost": total_cost,
                        "user_id": state["user_id"]
                    }
                )

        except Exception as e:
            state["error"] = str(e)
            state["results"] = results
            state["cost"] = total_cost

        return state

    @observe(name="synthesize_results")
    def _synthesize_results(self, state: ResearchState) -> ResearchState:
        """Synthesize research results into summary."""
        if state.get("error") or not state["results"]:
            state["summary"] = f"Research incomplete. Error: {state.get('error', 'No results')}"
            return state

        try:
            # Format results for LLM
            results_text = []
            for i, result in enumerate(state["results"], 1):
                cap = result.get("capability", "unknown")
                res_data = result.get("result", result.get("error", "No data"))
                results_text.append(f"{i}. {cap}:\n{json.dumps(res_data, indent=2)}\n")

            results_str = "\n\n".join(results_text)

            system_prompt = """You are a research synthesizer. Your job is to create a comprehensive summary
from multiple research sources. Be concise but thorough."""

            user_prompt = f"""Topic: {state['topic']}

Research Results:
{results_str}

Please provide a comprehensive synthesis of the research findings above. Highlight key insights
and note any gaps or limitations."""

            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=user_prompt)
            ]

            response = self.llm.invoke(messages)
            state["summary"] = response.content

            # Log to LangFuse
            if self.langfuse:
                langfuse_context.update_current_observation(
                    input=user_prompt,
                    output=response.content,
                    metadata={
                        "model": self.model,
                        "total_cost": state["cost"]
                    }
                )

        except Exception as e:
            state["error"] = str(e)
            state["summary"] = f"Error synthesizing results: {e}"

        return state

    @observe(name="research_topic")
    def research(self, topic: str, user_id: Optional[str] = None, budget_tier: Optional[str] = None) -> dict:
        """
        Execute research workflow.

        Args:
            topic: Research topic
            user_id: Optional user ID (overrides default)
            budget_tier: Optional budget tier (basic, pro, enterprise)

        Returns:
            dict with summary, results, cost, and metadata
        """
        initial_state: ResearchState = {
            "topic": topic,
            "user_id": user_id or self.user_id,
            "budget_tier": budget_tier or self.budget_tier,
            "tasks": [],
            "results": [],
            "summary": "",
            "cost": 0.0,
            "error": None
        }

        # Execute workflow
        final_state = self.workflow.invoke(initial_state)

        return {
            "topic": final_state["topic"],
            "summary": final_state["summary"],
            "results": final_state["results"],
            "cost": final_state["cost"],
            "error": final_state.get("error"),
            "metadata": {
                "user_id": final_state["user_id"],
                "budget_tier": final_state["budget_tier"],
                "num_tasks": len(final_state["tasks"]),
                "num_results": len(final_state["results"]),
                "model": self.model
            }
        }

    async def aresearch(self, topic: str, user_id: Optional[str] = None, budget_tier: Optional[str] = None) -> dict:
        """Async version of research."""
        # For now, just wrap synchronous version
        return self.research(topic, user_id, budget_tier)


# Example usage
if __name__ == "__main__":
    import json

    # Initialize workflow
    workflow = ResearchWorkflow(
        a2a_url="http://localhost:8081",
        user_id="demo-user-pro",
        budget_tier="pro"
    )

    # Execute research
    result = workflow.research("transformer architecture improvements")

    print(json.dumps(result, indent=2))
