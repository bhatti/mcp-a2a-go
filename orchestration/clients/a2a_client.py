"""A2A Client Wrapper for Streamlit UI"""
import requests
from typing import Dict, Any, Optional, List
import sseclient


class A2AClient:
    """Client for interacting with A2A Server"""

    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()

    def get_agent_card(self) -> Dict[str, Any]:
        """Get agent card with capabilities"""
        response = self.session.get(f"{self.base_url}/agent")
        response.raise_for_status()
        return response.json()

    def create_task(self, user_id: str, agent_id: str,
                   capability: str, input_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new task"""
        payload = {
            "user_id": user_id,
            "agent_id": agent_id,
            "capability": capability,
            "input": input_data
        }
        response = self.session.post(
            f"{self.base_url}/tasks",
            json=payload
        )
        response.raise_for_status()
        return response.json()

    def get_task(self, task_id: str) -> Dict[str, Any]:
        """Get task by ID"""
        response = self.session.get(f"{self.base_url}/tasks/{task_id}")
        response.raise_for_status()
        return response.json()

    def list_tasks(self, agent_id: Optional[str] = None,
                   limit: int = 100, offset: int = 0) -> List[Dict[str, Any]]:
        """List tasks with optional filtering"""
        params = {"limit": limit, "offset": offset}
        if agent_id:
            params["agent_id"] = agent_id

        response = self.session.get(
            f"{self.base_url}/tasks",
            params=params
        )
        response.raise_for_status()
        return response.json()

    def cancel_task(self, task_id: str) -> Dict[str, Any]:
        """Cancel a task"""
        response = self.session.delete(f"{self.base_url}/tasks/{task_id}")
        response.raise_for_status()
        return response.json()

    def stream_task_events(self, task_id: str):
        """Stream task events via SSE"""
        response = self.session.get(
            f"{self.base_url}/tasks/{task_id}/events",
            stream=True,
            headers={'Accept': 'text/event-stream'}
        )
        response.raise_for_status()

        client = sseclient.SSEClient(response)
        for event in client.events():
            if event.data:
                yield event.data

    def health_check(self) -> bool:
        """Check if A2A server is healthy"""
        try:
            response = requests.get(f"{self.base_url}/health", timeout=5)
            return response.status_code == 200
        except:
            return False
