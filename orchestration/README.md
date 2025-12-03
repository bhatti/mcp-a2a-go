# LangGraph Orchestration Workflows

Python orchestration layer using LangGraph for complex AI workflows with **local Ollama support** (no OpenAI API key required!).

## Features

- ✅ **3 Production Workflows**: RAG, Research, Hybrid
- ✅ **Ollama Support**: Run locally without OpenAI API keys
- ✅ **LangFuse Integration**: Complete LLM observability
- ✅ **Multi-tenant**: JWT-based authentication
- ✅ **Cost Control**: Budget enforcement with A2A

## Quick Start - Local with Ollama

```bash
# 1. Start all services (includes Ollama)
docker compose up -d

# 2. Setup Ollama and pull a model (one-time, ~4GB download)
./scripts/setup-ollama.sh llama3

# 3. Install Python dependencies
cd orchestration
pip install -r requirements.txt

# 4. Set environment for Ollama
export USE_OLLAMA=true
export OLLAMA_URL=http://localhost:11434
export MCP_SERVER_URL=http://localhost:8080
export A2A_SERVER_URL=http://localhost:8081

# 5. Run workflows!
python example.py
```

**No OpenAI API key needed!** 🎉

## Quick Start - With OpenAI

```bash
# 1. Start services
docker compose up -d

# 2. Install dependencies
cd orchestration
pip install -r requirements.txt

# 3. Set environment
export OPENAI_API_KEY="your-key-here"
export USE_OLLAMA=false
export MCP_SERVER_URL=http://localhost:8080
export A2A_SERVER_URL=http://localhost:8081

# Optional: LangFuse observability
export LANGFUSE_PUBLIC_KEY="your-key"
export LANGFUSE_SECRET_KEY="your-secret"

# 4. Run workflows
python example.py
```

## Available Ollama Models

Popular models (download with `docker exec ollama ollama pull <model>`):

- **llama3** (8B, ~4.7GB) - Recommended, fast and capable
- **llama3:70b** (70B, ~40GB) - More capable, slower
- **mistral** (7B, ~4.1GB) - Fast, good for coding
- **codellama** (7B, ~3.8GB) - Optimized for code
- **phi3** (3.8B, ~2.3GB) - Smallest, fastest

See all models: https://ollama.com/library

## Workflows

### 1. RAG Workflow (Internal Knowledge Base)

Uses MCP server for hybrid search on internal documents.

```python
from workflows import RAGWorkflow

# With Ollama (local)
workflow = RAGWorkflow(
    model="llama3",
    use_ollama=True
)

# With OpenAI
workflow = RAGWorkflow(
    model="gpt-4",
    use_ollama=False
)

result = workflow.query("What are our security policies?")
print(result['answer'])
```

**Flow:**
1. Search internal docs via MCP (hybrid BM25 + vector)
2. Format retrieved documents as context
3. Generate answer with LLM (Ollama or OpenAI)

### 2. Research Workflow (External with Cost Control)

Uses A2A server for budget-aware external research.

```python
from workflows import ResearchWorkflow

workflow = ResearchWorkflow(
    model="llama3",
    use_ollama=True,
    budget_tier="pro"  # $50/month
)

result = workflow.research("transformer architecture improvements")
print(result['summary'])
print(f"Cost: ${result['cost']:.4f}")
```

**Flow:**
1. Plan research tasks based on topic
2. Execute tasks via A2A with budget checks
3. Synthesize results with LLM

### 3. Hybrid Workflow (Internal + External)

Combines MCP (internal) and A2A (external) for comprehensive answers.

```python
from workflows import HybridWorkflow

workflow = HybridWorkflow(
    model="llama3",
    use_ollama=True
)

result = workflow.query(
    "What are latest ML advances and how do they relate to our strategy?"
)
print(result['answer'])
```

**Flow:**
1. Search internal docs (MCP)
2. Research external sources (A2A)
3. Synthesize both with LLM

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `USE_OLLAMA` | `false` | Use Ollama instead of OpenAI |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama server URL |
| `OPENAI_API_KEY` | - | OpenAI API key (if not using Ollama) |
| `MCP_SERVER_URL` | `http://localhost:8080` | MCP server URL |
| `A2A_SERVER_URL` | `http://localhost:8081` | A2A server URL |
| `LANGFUSE_PUBLIC_KEY` | - | LangFuse API key (optional) |
| `LANGFUSE_SECRET_KEY` | - | LangFuse secret (optional) |

## Model Selection

**For Ollama:**
```python
# Fast and capable (recommended)
workflow = RAGWorkflow(model="llama3", use_ollama=True)

# Best quality (if you have GPU)
workflow = RAGWorkflow(model="llama3:70b", use_ollama=True)

# Fastest (smaller model)
workflow = RAGWorkflow(model="phi3", use_ollama=True)
```

**For OpenAI:**
```python
# Best quality
workflow = RAGWorkflow(model="gpt-4", use_ollama=False)

# Faster and cheaper
workflow = RAGWorkflow(model="gpt-3.5-turbo", use_ollama=False)
```

## Testing

```bash
# Test RAG workflow
python -c "
from workflows import RAGWorkflow
wf = RAGWorkflow(model='llama3', use_ollama=True)
result = wf.query('What are our policies?')
print(result['answer'])
"

# Test Research workflow
python -c "
from workflows import ResearchWorkflow
wf = ResearchWorkflow(model='llama3', use_ollama=True)
result = wf.research('AI trends')
print(result['summary'])
"

# Test Hybrid workflow
python -c "
from workflows import HybridWorkflow
wf = HybridWorkflow(model='llama3', use_ollama=True)
result = wf.query('Latest AI and our strategy?')
print(result['answer'])
"
```

## Performance Tips

**Ollama:**
- First run downloads model (~4-40GB depending on model)
- GPU acceleration highly recommended
- Smaller models (llama3, mistral) work well on CPU
- Use `phi3` for fastest inference on CPU

**OpenAI:**
- Faster initial response (no model download)
- `gpt-3.5-turbo` is fast and cheap
- `gpt-4` is slower but more capable

## Troubleshooting

### Ollama Not Responding

```bash
# Check if Ollama is running
curl http://localhost:11434

# Check logs
docker logs ollama

# Restart Ollama
docker restart ollama
```

### Model Not Found

```bash
# List available models
docker exec ollama ollama list

# Pull missing model
docker exec ollama ollama pull llama3
```

### Out of Memory

- Use smaller model: `phi3` (2.3GB) instead of `llama3` (4.7GB)
- Reduce concurrent requests
- Increase Docker memory limit

### Slow Performance

- Use GPU if available (add nvidia runtime to docker compose)
- Use smaller model (`phi3` or `mistral`)
- Switch to OpenAI for faster response

## Project Structure

```
orchestration/
├── workflows/
│   ├── __init__.py
│   ├── rag_workflow.py        # MCP-based internal search
│   ├── research_workflow.py   # A2A-based external research
│   └── hybrid_workflow.py     # Combined MCP + A2A
├── clients/
│   ├── __init__.py
│   ├── mcp_client.py          # JSON-RPC client
│   ├── a2a_client.py          # REST + SSE client
│   └── auth.py                # JWT generation
├── example.py                 # Demo all workflows
├── requirements.txt           # Python dependencies
└── README.md                  # This file
```

## Next Steps

1. **Try Different Models**: Test llama3, mistral, phi3
2. **Add Custom Workflows**: Extend existing workflows
3. **Enable LangFuse**: Track LLM performance
4. **Production Deployment**: See k8s/ for Kubernetes

## Support

- **Ollama Docs**: https://ollama.com/
- **LangGraph**: https://langchain-ai.github.io/langgraph/
- **LangFuse**: https://langfuse.com/
