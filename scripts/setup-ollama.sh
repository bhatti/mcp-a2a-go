#!/bin/bash
#
# Setup Ollama and pull required models

set -e

OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
MODEL="${1:-llama3}"

echo "🦙 Setting up Ollama..."
echo "URL: $OLLAMA_URL"
echo "Model: $MODEL"
echo ""

# Check if Ollama container is running
echo "Checking if Ollama container is running..."
if ! docker ps | grep -q ollama; then
    echo "✗ Ollama container is not running!"
    echo "  Start it with: docker compose up -d ollama"
    exit 1
fi

# Wait for Ollama to be ready (may take 30-60 seconds on first start)
echo "Waiting for Ollama to be ready (this may take up to 60 seconds)..."
for i in {1..60}; do
    if curl -s "$OLLAMA_URL" > /dev/null 2>&1; then
        echo "✓ Ollama is ready!"
        break
    fi
    if [ $i -eq 60 ]; then
        echo "✗ Ollama failed to respond after 60 seconds"
        echo "  Check logs with: docker compose logs ollama"
        exit 1
    fi
    printf "."
    sleep 1
done
echo ""

# Pull the model
echo ""
echo "Pulling model: $MODEL (this may take a few minutes)..."
docker exec ollama ollama pull $MODEL

echo ""
echo "✓ Model $MODEL is ready!"
echo ""
echo "Available models:"
docker exec ollama ollama list

echo ""
echo "Test the model:"
echo "  curl $OLLAMA_URL/api/generate -d '{\"model\":\"$MODEL\",\"prompt\":\"Hello\"}'"
