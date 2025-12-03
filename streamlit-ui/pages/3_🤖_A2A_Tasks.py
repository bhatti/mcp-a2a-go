"""
A2A Task Management
Create and monitor cost-controlled tasks with real-time SSE streaming
"""
import streamlit as st
import os
import json
import time
from utils.a2a_client import A2AClient
from utils.auth import DEMO_USERS

st.set_page_config(page_title="A2A Tasks", page_icon="🤖", layout="wide")

st.title("🤖 A2A Cost-Controlled Research Assistant")

# Get A2A URL
a2a_url = os.getenv('A2A_SERVER_URL', 'http://localhost:8081')

# Initialize A2A client
client = A2AClient(a2a_url)

# Get Agent Card
st.header("Agent Capabilities")

try:
    agent_card = client.get_agent_card()
    st.success(f"🤖 Agent: **{agent_card['name']}** v{agent_card['version']}")
    st.markdown(f"_{agent_card['description']}_")

    with st.expander("View Full Agent Card"):
        st.json(agent_card)

    # Display capabilities
    st.subheader("Available Capabilities")
    capabilities = agent_card.get('capabilities', [])

    for cap in capabilities:
        with st.expander(f"📋 {cap['name']}"):
            st.markdown(f"**Description**: {cap['description']}")
            if 'input_schema' in cap:
                st.markdown("**Input Schema**:")
                st.json(cap['input_schema'])

except Exception as e:
    st.error(f"Failed to get agent card: {str(e)}")
    st.stop()

# User Selection
st.header("💰 Select User (Budget Tier)")

user_id = st.selectbox(
    "Demo User",
    list(DEMO_USERS.keys()),
    help="Each user has a different monthly budget"
)

user_info = DEMO_USERS[user_id]
col1, col2 = st.columns(2)
with col1:
    st.metric("Budget Tier", user_info['tier'])
with col2:
    st.metric("Monthly Budget", f"${user_info['budget']}")

st.info("💡 Budget enforcement: Tasks are pre-checked against remaining budget. Create multiple tasks to test budget limits!")

# Create Task
st.header("📝 Create Task")

capability = st.selectbox(
    "Select Capability",
    [cap['name'] for cap in capabilities],
    help="Choose which capability to invoke"
)

# Dynamic input based on capability
st.subheader("Task Input")

if capability == "search_papers":
    query = st.text_input("Search Query", value="transformer architecture")
    max_results = st.slider("Max Results", 1, 50, 10)
    task_input = {"query": query, "max_results": max_results}

elif capability == "analyze_code":
    code = st.text_area("Source Code", value='def hello():\n    print("Hello, World!")', height=200)
    language = st.selectbox("Language", ["python", "javascript", "go", "java"])
    task_input = {"code": code, "language": language}

elif capability == "summarize_document":
    document = st.text_area("Document Text", value="Enter document text here...", height=200)
    max_length = st.slider("Max Summary Length (words)", 50, 500, 200)
    task_input = {"document": document, "max_length": max_length}

else:
    task_input = {}

if st.button("Create Task", type="primary"):
    with st.spinner("Creating task..."):
        try:
            task = client.create_task(
                user_id=user_id,
                agent_id=agent_card['id'],
                capability=capability,
                input_data=task_input
            )

            st.success(f"✅ Task created: {task['id']}")
            st.session_state.last_task_id = task['id']

            with st.expander("Task Details"):
                st.json(task)

        except Exception as e:
            if "402" in str(e) or "Payment Required" in str(e):
                st.error("❌ Budget Exceeded! User has insufficient remaining budget for this task.")
                st.info("Try selecting a user with a higher budget tier (Pro or Enterprise)")
            else:
                st.error(f"❌ Failed to create task: {str(e)}")

# List Tasks
st.header("📋 Task List")

col1, col2 = st.columns([3, 1])
with col1:
    filter_agent = st.checkbox("Filter by current agent", value=True)
with col2:
    if st.button("🔄 Refresh"):
        st.rerun()

try:
    agent_filter = agent_card['id'] if filter_agent else None
    tasks = client.list_tasks(agent_id=agent_filter, limit=50)

    if tasks:
        st.success(f"Found {len(tasks)} tasks")

        # Create DataFrame for better display
        import pandas as pd
        task_data = []
        for task in tasks:
            task_data.append({
                "Task ID": task['id'][:8] + "...",
                "Capability": task['capability'],
                "State": task['state'],
                "Created": task['created_at'][:19],
                "Full ID": task['id']
            })

        df = pd.DataFrame(task_data)
        st.dataframe(df[["Task ID", "Capability", "State", "Created"]], use_container_width=True)

        # Task Details
        selected_task_id = st.selectbox(
            "Select task to view details",
            [task['Full ID'] for task in task_data],
            format_func=lambda x: x[:8] + "..."
        )

        if selected_task_id:
            col1, col2 = st.columns([3, 1])

            with col1:
                if st.button("Get Task Details"):
                    try:
                        task_details = client.get_task(selected_task_id)
                        st.json(task_details)
                    except Exception as e:
                        st.error(f"Error: {str(e)}")

            with col2:
                if st.button("Cancel Task", type="secondary"):
                    try:
                        cancelled = client.cancel_task(selected_task_id)
                        st.success("Task cancelled")
                        st.json(cancelled)
                        time.sleep(1)
                        st.rerun()
                    except Exception as e:
                        if "409" in str(e):
                            st.error("Task is already in terminal state")
                        else:
                            st.error(f"Error: {str(e)}")

    else:
        st.info("No tasks found. Create one above!")

except Exception as e:
    st.error(f"Failed to list tasks: {str(e)}")

# SSE Streaming Demo
st.header("📡 Real-Time Task Events (SSE)")

st.markdown("""
Server-Sent Events (SSE) provide real-time updates on task progress without polling.
""")

if 'last_task_id' in st.session_state:
    task_id = st.session_state.last_task_id

    st.info(f"Monitoring task: {task_id[:16]}...")

    if st.button("Start Streaming Events"):
        event_placeholder = st.empty()
        event_count = 0

        try:
            for event_data in client.stream_task_events(task_id):
                event_count += 1
                event = json.loads(event_data)

                with event_placeholder.container():
                    st.markdown(f"**Event #{event_count}**")
                    st.json(event)

                # Stop after 10 events for demo
                if event_count >= 10:
                    st.info("Stopped after 10 events (demo limit)")
                    break

                # Stop if terminal state
                if event.get('state') in ['completed', 'failed', 'cancelled']:
                    st.success(f"Task reached terminal state: {event.get('state')}")
                    break

        except Exception as e:
            st.error(f"Streaming error: {str(e)}")
else:
    st.info("Create a task above to enable streaming")

# Budget Enforcement Demo
st.header("💸 Budget Enforcement Demo")

st.markdown("""
**How it works:**
1. Each user has a monthly budget ($10, $50, or $200)
2. Each task has an estimated cost ($0.01 in this demo)
3. Budget is checked BEFORE task creation
4. If insufficient budget → 402 Payment Required

**Try this:**
1. Select **demo-user-basic** ($10 budget)
2. Create 1000 tasks rapidly
3. Around task #1000, you'll hit the budget limit
4. Switch to **demo-user-enterprise** to continue
""")

# Task States
st.header("🔄 Task Lifecycle")

st.markdown("""
```
┌─────────┐
│ pending │ ← Created
└────┬────┘
     │
     ▼
┌─────────┐
│ running │ ← Processing
└────┬────┘
     │
     ├──────────┬──────────┐
     ▼          ▼          ▼
┌───────────┐ ┌──────┐ ┌───────────┐
│ completed │ │failed│ │ cancelled │
└───────────┘ └──────┘ └───────────┘
     ▲          ▲          ▲
     └──────────┴──────────┘
         Terminal States
```

**States:**
- **pending**: Task created, awaiting execution
- **running**: Task is being processed
- **completed**: Task finished successfully
- **failed**: Task encountered an error
- **cancelled**: Task was manually cancelled
""")
