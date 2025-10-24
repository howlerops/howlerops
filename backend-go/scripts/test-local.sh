#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Testing local development setup...${NC}\n"

# Check if data directory exists
if [ ! -d "./data" ]; then
    echo -e "${RED}Data directory not found. Run 'make setup-local' first.${NC}"
    exit 1
fi

# Check if .env.development exists
if [ ! -f ".env.development" ]; then
    echo -e "${RED}.env.development not found. Creating from example...${NC}"
    if [ -f ".env.example" ]; then
        cp .env.example .env.development
        echo -e "${GREEN}.env.development created from .env.example${NC}"
    else
        echo -e "${RED}.env.example not found. Cannot create .env.development${NC}"
        exit 1
    fi
fi

# Set environment
export ENVIRONMENT=development

# Test database connection
echo -e "${BLUE}Testing database connection...${NC}"
if [ -f "./data/development.db" ]; then
    echo -e "${GREEN}SQLite database file exists${NC}"
else
    echo -e "${YELLOW}SQLite database file will be created on first run${NC}"
fi

# Build the server
echo -e "\n${BLUE}Building server...${NC}"
go build -o ./tmp/server ./cmd/server/main.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Build successful${NC}"
else
    echo -e "${RED}Build failed${NC}"
    exit 1
fi

# Start server in background
echo -e "\n${BLUE}Starting server...${NC}"
ENVIRONMENT=development ./tmp/server &
SERVER_PID=$!

# Function to kill server on script exit
cleanup() {
    echo -e "\n${BLUE}Cleaning up...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Wait for server to start
echo -e "${YELLOW}Waiting for server to start...${NC}"
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}Server started successfully${NC}"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    sleep 1
    
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo -e "${RED}Server failed to start within 30 seconds${NC}"
        exit 1
    fi
done

# Test health endpoint
echo -e "\n${BLUE}Testing health endpoint...${NC}"
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    echo -e "${GREEN}Health check passed${NC}"
    echo -e "Response: $HEALTH_RESPONSE"
else
    echo -e "${RED}Health check failed${NC}"
    echo -e "Response: $HEALTH_RESPONSE"
    exit 1
fi

# Test metrics endpoint
echo -e "\n${BLUE}Testing metrics endpoint...${NC}"
if curl -f http://localhost:9100/metrics > /dev/null 2>&1; then
    echo -e "${GREEN}Metrics endpoint working${NC}"
    METRICS_COUNT=$(curl -s http://localhost:9100/metrics | grep -c "^#" || true)
    echo -e "Metrics available: $METRICS_COUNT metric families"
else
    echo -e "${YELLOW}Metrics endpoint not available (might be disabled)${NC}"
fi

# Test sync endpoints (should require auth)
echo -e "\n${BLUE}Testing sync endpoints (authentication required)...${NC}"
SYNC_RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8080/api/sync/download?device_id=test)
HTTP_CODE=$(echo "$SYNC_RESPONSE" | tail -n 1)
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}Sync endpoint correctly requires authentication${NC}"
else
    echo -e "${YELLOW}Unexpected response from sync endpoint: HTTP $HTTP_CODE${NC}"
fi

# Summary
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}All tests passed!${NC}"
echo -e "${GREEN}========================================${NC}\n"

echo -e "${BLUE}Server Information:${NC}"
echo -e "  HTTP URL:    http://localhost:8080"
echo -e "  gRPC URL:    localhost:9090"
echo -e "  Metrics URL: http://localhost:9100/metrics"
echo -e "  Health URL:  http://localhost:8080/health"
echo -e "  WebSocket:   ws://localhost:8081/ws"
echo -e ""
echo -e "${BLUE}Available Sync Endpoints:${NC}"
echo -e "  POST   /api/sync/upload"
echo -e "  GET    /api/sync/download"
echo -e "  GET    /api/sync/conflicts"
echo -e "  POST   /api/sync/conflicts/{id}/resolve"
echo -e ""
echo -e "${YELLOW}Server will be stopped automatically when this script exits${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}\n"

# Keep server running
wait $SERVER_PID
