#!/bin/bash
# OpenTelemetry Observability Testing Script
# This script demonstrates how to test metrics and tracing

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}OpenTelemetry Observability Test Suite${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Configuration
MCP_URL=${MCP_SERVER_URL:-http://localhost:8080}
A2A_URL=${A2A_SERVER_URL:-http://localhost:8081}
PROMETHEUS_URL=${PROMETHEUS_URL:-http://localhost:9090}
JAEGER_URL=${JAEGER_URL:-http://localhost:16686}

# Test 1: Check if services are running
echo -e "${YELLOW}Test 1: Checking if services are running...${NC}"

check_service() {
    local url=$1
    local name=$2

    if curl -s -f -o /dev/null "$url/health"; then
        echo -e "${GREEN}✓${NC} $name is healthy"
        return 0
    else
        echo -e "${RED}✗${NC} $name is not responding"
        return 1
    fi
}

check_service "$MCP_URL" "MCP Server"
check_service "$A2A_URL" "A2A Server"

# Test 2: Check metrics endpoints
echo -e "\n${YELLOW}Test 2: Checking metrics endpoints...${NC}"

check_metrics() {
    local url=$1
    local name=$2

    if curl -s "$url/metrics" | grep -q "^# TYPE"; then
        echo -e "${GREEN}✓${NC} $name metrics endpoint is working"

        # Count metrics
        metric_count=$(curl -s "$url/metrics" | grep -c "^# TYPE" || true)
        echo -e "  Found ${BLUE}$metric_count${NC} metric types"
        return 0
    else
        echo -e "${RED}✗${NC} $name metrics endpoint failed"
        return 1
    fi
}

check_metrics "$MCP_URL" "MCP Server"
check_metrics "$A2A_URL" "A2A Server"

# Test 3: Check Prometheus
echo -e "\n${YELLOW}Test 3: Checking Prometheus...${NC}"

if curl -s -f -o /dev/null "$PROMETHEUS_URL/-/healthy"; then
    echo -e "${GREEN}✓${NC} Prometheus is healthy"

    # Check if targets are up
    echo -e "\n${BLUE}Checking Prometheus targets:${NC}"
    targets=$(curl -s "$PROMETHEUS_URL/api/v1/targets" | grep -o '"health":"up"' | wc -l || true)
    echo -e "  ${GREEN}$targets${NC} targets are UP"
else
    echo -e "${RED}✗${NC} Prometheus is not responding"
fi

# Test 4: Make test requests to generate metrics
echo -e "\n${YELLOW}Test 4: Generating test metrics...${NC}"

# Make a few MCP requests
echo -e "${BLUE}Making MCP requests...${NC}"
for i in {1..5}; do
    response=$(curl -s -X POST "$MCP_URL/mcp" \
        -H "Content-Type: application/json" \
        -d '{
            "jsonrpc": "2.0",
            "id": '$i',
            "method": "tools/list"
        }')

    if echo "$response" | grep -q "result"; then
        echo -e "${GREEN}✓${NC} Request $i succeeded"
    else
        echo -e "${RED}✗${NC} Request $i failed"
    fi
done

# Test 5: Query Prometheus for metrics
echo -e "\n${YELLOW}Test 5: Querying Prometheus for metrics...${NC}"

query_prometheus() {
    local query=$1
    local name=$2

    result=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=$query" | \
        python3 -c "import sys, json; data=json.load(sys.stdin); print(data['data']['result'][0]['value'][1] if data['data']['result'] else 'N/A')" 2>/dev/null || echo "N/A")

    echo -e "  $name: ${BLUE}$result${NC}"
}

echo -e "${BLUE}MCP Server Metrics:${NC}"
query_prometheus "sum(mcp_request_count)" "Total Requests"
query_prometheus "mcp_request_active" "Active Requests"

echo -e "\n${BLUE}A2A Server Metrics:${NC}"
query_prometheus "sum(a2a_task_count)" "Total Tasks"
query_prometheus "sum(a2a_cost_total)" "Total Cost (USD)"

# Test 6: Check Jaeger
echo -e "\n${YELLOW}Test 6: Checking Jaeger...${NC}"

if curl -s -f -o /dev/null "$JAEGER_URL"; then
    echo -e "${GREEN}✓${NC} Jaeger UI is accessible"

    # Try to get services
    services=$(curl -s "$JAEGER_URL/api/services" | \
        python3 -c "import sys, json; print(len(json.load(sys.stdin)['data']))" 2>/dev/null || echo "0")

    if [ "$services" -gt 0 ]; then
        echo -e "  Found ${BLUE}$services${NC} traced services"
    else
        echo -e "  ${YELLOW}No services found yet. Make some requests to generate traces.${NC}"
    fi
else
    echo -e "${RED}✗${NC} Jaeger UI is not responding"
fi

# Test 7: Verify trace context propagation
echo -e "\n${YELLOW}Test 7: Testing trace context propagation...${NC}"

# Make a request with a custom trace ID
TRACE_ID="00-$(openssl rand -hex 16)-$(openssl rand -hex 8)-01"
echo -e "${BLUE}Injecting trace ID:${NC} $TRACE_ID"

response=$(curl -s -X POST "$MCP_URL/mcp" \
    -H "Content-Type: application/json" \
    -H "traceparent: $TRACE_ID" \
    -d '{
        "jsonrpc": "2.0",
        "id": 999,
        "method": "tools/list"
    }')

if echo "$response" | grep -q "result"; then
    echo -e "${GREEN}✓${NC} Request with custom trace ID succeeded"
    echo -e "  ${BLUE}Check Jaeger UI for this trace ID${NC}"
else
    echo -e "${RED}✗${NC} Request with custom trace ID failed"
fi

# Summary
echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"

echo -e "\n${GREEN}✓ All tests completed!${NC}\n"

echo -e "Next steps:"
echo -e "1. View metrics in Prometheus: ${BLUE}$PROMETHEUS_URL${NC}"
echo -e "2. View traces in Jaeger: ${BLUE}$JAEGER_URL${NC}"
echo -e "3. Open Streamlit UI: ${BLUE}http://localhost:8501${NC}"
echo -e "   - Go to 'Metrics' page to see real-time metrics"
echo -e "   - Go to 'Tracing' page to learn about distributed tracing"
echo -e "\n4. Make some RAG queries in Streamlit to generate more traces"
echo -e "5. Check the testing guide: ${BLUE}docs/TESTING_OBSERVABILITY.md${NC}\n"
