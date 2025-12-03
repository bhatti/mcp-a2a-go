#!/bin/bash -x
clear
cd orchestration

# Install dependencies
pip install -r requirements.txt

# Set environment for Ollama
#export USE_OLLAMA=true
#export OLLAMA_URL=http://localhost:11434
#export MCP_SERVER_URL=http://localhost:8080
#export A2A_SERVER_URL=http://localhost:8081
export DEMO_KEYS_DIR=/tmp/demo-keys
# Run all workflows with Ollama
python3 example.py

# Output shows:
# 1. RAG workflow (MCP + llama3)
# 2. Research workflow (A2A + llama3)
# 3. Hybrid workflow (MCP + A2A + llama3)

