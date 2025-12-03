cd orchestration

# Install dependencies (if not already done)
pip install -r requirements.txt

# Set environment variables
export DEMO_KEYS_DIR=/tmp/demo-keys
export MCP_SERVER_URL=http://localhost:8080
export A2A_SERVER_URL=http://localhost:8081
export USE_OLLAMA=true
export OLLAMA_URL=http://localhost:11434


python3 -c "
from workflows import RAGWorkflow

# Initialize with Ollama (local LLM)
workflow = RAGWorkflow(
    mcp_url='http://localhost:8080',
    tenant_id='11111111-1111-1111-1111-111111111111',  # acme-corp
    model='llama3',
    use_ollama=True
)

# Execute query
result = workflow.query('What are our security policies?')

print('=== RAG WORKFLOW RESULT ===')
print(f'Answer: {result[\"answer\"]}')
print(f'Documents found: {len(result[\"documents\"])}')
print(f'Error: {result.get(\"error\", \"None\")}')
"

echo "next script====="

python3 -c "
from workflows import HybridWorkflow

# Initialize
workflow = HybridWorkflow(
    mcp_url='http://localhost:8080',
    a2a_url='http://localhost:8081',
    tenant_id='11111111-1111-1111-1111-111111111111',
    user_id='demo-user-pro',
    model='llama3',
    use_ollama=True
)

# Execute hybrid query (internal + external)
result = workflow.query(
    'What are latest ML advances and how do they relate to our strategy?'
)

print('=== HYBRID WORKFLOW RESULT ===')
print(f'Answer: {result[\"answer\"][:200]}...')
print(f'Internal docs: {len(result[\"internal_documents\"])}')
print(f'External research: {len(result[\"external_results\"])}')
print(f'Error: {result.get(\"error\", \"None\")}')
"

