"""
Cost Tracking and Budget Management
Monitor token usage, costs, and budget enforcement
"""
import streamlit as st
import pandas as pd
import plotly.graph_objects as go
import plotly.express as px
from datetime import datetime, timedelta

st.set_page_config(page_title="Cost Tracking", page_icon="💰", layout="wide")

st.title("💰 Cost Tracking & Budget Management")

# Simulated cost data (in production, this would come from the A2A server)
st.info("💡 This page shows the cost tracking capabilities. In production, data would be fetched from the A2A server's cost tracking API.")

# User Budget Overview
st.header("Budget Overview")

users_data = [
    {"user": "demo-user-basic", "budget": 10.00, "spent": 2.35, "tier": "Basic"},
    {"user": "demo-user-pro", "budget": 50.00, "spent": 23.45, "tier": "Pro"},
    {"user": "demo-user-enterprise", "budget": 200.00, "spent": 78.90, "tier": "Enterprise"}
]

for user_data in users_data:
    with st.expander(f"👤 {user_data['user']} - {user_data['tier']} Tier", expanded=True):
        col1, col2, col3, col4 = st.columns(4)

        with col1:
            st.metric("Monthly Budget", f"${user_data['budget']:.2f}")

        with col2:
            st.metric("Spent", f"${user_data['spent']:.2f}")

        with col3:
            remaining = user_data['budget'] - user_data['spent']
            st.metric("Remaining", f"${remaining:.2f}")

        with col4:
            percent = (user_data['spent'] / user_data['budget']) * 100
            st.metric("Usage", f"{percent:.1f}%")

        # Progress bar
        progress_color = "normal" if percent < 80 else "inverse"
        st.progress(min(percent / 100, 1.0))

        if percent >= 100:
            st.error("⚠️ Budget exceeded! Tasks will be rejected.")
        elif percent >= 80:
            st.warning("⚠️ Approaching budget limit")

# Cost by Model
st.header("💸 Cost by Model")

model_data = pd.DataFrame([
    {"model": "GPT-4", "requests": 45, "tokens": 45000, "cost": 1.35},
    {"model": "GPT-3.5-Turbo", "requests": 230, "tokens": 230000, "cost": 0.58},
    {"model": "Claude-3-Opus", "requests": 12, "tokens": 12000, "cost": 0.42}
])

col1, col2 = st.columns(2)

with col1:
    # Pie chart for cost distribution
    fig = px.pie(model_data, values='cost', names='model', title='Cost Distribution by Model')
    st.plotly_chart(fig, use_container_width=True)

with col2:
    # Bar chart for token usage
    fig = px.bar(model_data, x='model', y='tokens', title='Token Usage by Model',
                 labels={'tokens': 'Total Tokens', 'model': 'Model'})
    st.plotly_chart(fig, use_container_width=True)

# Usage Timeline
st.header("📈 Usage Over Time")

# Generate sample time series data
dates = pd.date_range(end=datetime.now(), periods=30, freq='D')
costs = [0.1 + (i * 0.05) + ((-1) ** i * 0.02) for i in range(30)]
df_timeline = pd.DataFrame({
    'date': dates,
    'cost': costs,
    'cumulative': pd.Series(costs).cumsum()
})

fig = go.Figure()
fig.add_trace(go.Scatter(x=df_timeline['date'], y=df_timeline['cost'],
                         mode='lines+markers', name='Daily Cost'))
fig.add_trace(go.Scatter(x=df_timeline['date'], y=df_timeline['cumulative'],
                         mode='lines', name='Cumulative Cost', yaxis='y2'))

fig.update_layout(
    title='Cost Trends (Last 30 Days)',
    xaxis=dict(title='Date'),
    yaxis=dict(title='Daily Cost ($)', side='left'),
    yaxis2=dict(title='Cumulative Cost ($)', side='right', overlaying='y'),
    hovermode='x unified'
)

st.plotly_chart(fig, use_container_width=True)

# Token Usage Breakdown
st.header("🔢 Token Usage Breakdown")

token_data = pd.DataFrame([
    {"type": "Prompt Tokens", "count": 245000, "cost": 0.735},
    {"type": "Completion Tokens", "count": 42000, "cost": 0.630}
])

col1, col2 = st.columns(2)

with col1:
    st.dataframe(token_data, use_container_width=True)

with col2:
    total_tokens = token_data['count'].sum()
    total_cost = token_data['cost'].sum()
    st.metric("Total Tokens", f"{total_tokens:,}")
    st.metric("Total Cost", f"${total_cost:.2f}")
    st.metric("Cost per 1K Tokens", f"${(total_cost / total_tokens * 1000):.4f}")

# Model Pricing
st.header("💵 Model Pricing")

pricing_data = pd.DataFrame([
    {
        "Model": "GPT-4",
        "Prompt (per 1K)": "$0.03",
        "Completion (per 1K)": "$0.06",
        "Use Case": "Complex reasoning"
    },
    {
        "Model": "GPT-4 Turbo",
        "Prompt (per 1K)": "$0.01",
        "Completion (per 1K)": "$0.03",
        "Use Case": "General purpose"
    },
    {
        "Model": "GPT-3.5-Turbo",
        "Prompt (per 1K)": "$0.0015",
        "Completion (per 1K)": "$0.002",
        "Use Case": "High volume"
    },
    {
        "Model": "Claude-3 Opus",
        "Prompt (per 1K)": "$0.015",
        "Completion (per 1K)": "$0.075",
        "Use Case": "Advanced tasks"
    },
    {
        "Model": "Claude-3 Sonnet",
        "Prompt (per 1K)": "$0.003",
        "Completion (per 1K)": "$0.015",
        "Use Case": "Balanced performance"
    }
])

st.dataframe(pricing_data, use_container_width=True)

# Budget Alerts
st.header("🚨 Budget Alerts")

alerts = [
    {"user": "demo-user-basic", "status": "warning", "message": "85% of budget used", "time": "2 hours ago"},
    {"user": "demo-user-pro", "status": "info", "message": "50% of budget used", "time": "5 hours ago"}
]

for alert in alerts:
    if alert['status'] == 'warning':
        st.warning(f"⚠️ **{alert['user']}**: {alert['message']} ({alert['time']})")
    else:
        st.info(f"ℹ️ **{alert['user']}**: {alert['message']} ({alert['time']})")

# Cost Optimization Tips
st.header("💡 Cost Optimization Tips")

col1, col2 = st.columns(2)

with col1:
    st.markdown("""
    **Best Practices:**
    - ✅ Use GPT-3.5-Turbo for simple tasks
    - ✅ Cache frequent queries
    - ✅ Set token limits on responses
    - ✅ Monitor usage with alerts
    - ✅ Use streaming to reduce latency perception
    """)

with col2:
    st.markdown("""
    **Budget Management:**
    - 📊 Set monthly budgets per user/team
    - 📊 Implement pre-flight cost checks
    - 📊 Graceful degradation (GPT-4 → GPT-3.5)
    - 📊 Real-time budget tracking
    - 📊 Automatic task rejection on budget exceed
    """)

# Export Data
st.header("📥 Export Usage Data")

# Prepare export data
export_data = pd.DataFrame([
    {
        "Date": datetime.now().strftime("%Y-%m-%d"),
        "User": user["user"],
        "Tier": user["tier"],
        "Budget": user["budget"],
        "Spent": user["spent"],
        "Remaining": user["budget"] - user["spent"],
        "Usage %": (user["spent"] / user["budget"]) * 100
    }
    for user in users_data
])

# Add model costs to export
export_with_models = pd.concat([
    export_data,
    pd.DataFrame({"Type": ["Model Costs"]}),  # Use list instead of scalar
    model_data.rename(columns={"model": "Model", "requests": "Requests", "tokens": "Tokens", "cost": "Cost"})
], ignore_index=True)

col1, col2, col3 = st.columns(3)

with col1:
    # Export as CSV
    csv_data = export_data.to_csv(index=False)
    st.download_button(
        label="📥 Export CSV",
        data=csv_data,
        file_name=f"cost_tracking_{datetime.now().strftime('%Y%m%d')}.csv",
        mime="text/csv",
        help="Download usage data as CSV file"
    )

with col2:
    # Export as JSON
    import json

    # Convert timeline dates to strings for JSON serialization
    timeline_data = df_timeline.tail(7).copy()
    timeline_data['date'] = timeline_data['date'].dt.strftime('%Y-%m-%d')

    json_data = {
        "export_date": datetime.now().isoformat(),
        "users": users_data,
        "models": model_data.to_dict(orient="records"),
        "timeline": timeline_data.to_dict(orient="records")
    }
    json_str = json.dumps(json_data, indent=2)
    st.download_button(
        label="📥 Export JSON",
        data=json_str,
        file_name=f"cost_tracking_{datetime.now().strftime('%Y%m%d')}.json",
        mime="application/json",
        help="Download detailed data as JSON file"
    )

with col3:
    # Generate summary report as text
    report = f"""COST TRACKING REPORT
Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}
=====================================

USER BUDGETS:
{'-' * 60}
"""
    for user in users_data:
        remaining = user['budget'] - user['spent']
        usage_pct = (user['spent'] / user['budget']) * 100
        report += f"""
{user['user']} ({user['tier']} Tier)
  Budget:    ${user['budget']:.2f}
  Spent:     ${user['spent']:.2f}
  Remaining: ${remaining:.2f}
  Usage:     {usage_pct:.1f}%
"""

    report += f"""
{'=' * 60}
MODEL COSTS:
{'-' * 60}
"""
    for _, row in model_data.iterrows():
        report += f"""
{row['model']}
  Requests: {row['requests']}
  Tokens:   {row['tokens']:,}
  Cost:     ${row['cost']:.2f}
"""

    st.download_button(
        label="📥 Generate Report",
        data=report,
        file_name=f"cost_report_{datetime.now().strftime('%Y%m%d')}.txt",
        mime="text/plain",
        help="Download summary report as text file"
    )
